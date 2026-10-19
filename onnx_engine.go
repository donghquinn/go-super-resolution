package main

// ONNX Runtime inference for image-to-image super-resolution models
// (e.g. 4x-UltraSharpV2): NCHW float32 RGB input in [0, 1], output scaled by the model's factor.

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"runtime"

	ort "github.com/yalue/onnxruntime_go"
	"gocv.io/x/gocv"
	"google.golang.org/protobuf/encoding/protowire"
)

const (
	onnxTileSize    = 256 // input tile edge fed to the model; every call uses this exact shape
	onnxTileOverlap = 32  // context pixels on each side of a tile, discarded from the output
)

// onnxRuntimeLibrary finds the ONNX Runtime shared library: $SR_ONNXRUNTIME_LIB, next to the
// executable (packaged app), then the platform's usual install location.
func onnxRuntimeLibrary() string {
	if p := os.Getenv("SR_ONNXRUNTIME_LIB"); p != "" {
		return p
	}
	var name string
	var fallbacks []string
	switch runtime.GOOS {
	case "darwin":
		name = "libonnxruntime.dylib"
		fallbacks = []string{"/opt/homebrew/lib/" + name, "/usr/local/lib/" + name}
	case "windows":
		name = "onnxruntime.dll"
	default:
		name = "libonnxruntime.so"
		fallbacks = []string{"/usr/local/lib/" + name, "/usr/lib/" + name}
	}
	if exe, err := os.Executable(); err == nil {
		fallbacks = append([]string{filepath.Join(filepath.Dir(exe), name)}, fallbacks...)
	}
	for _, p := range fallbacks {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return name // let the OS loader search its default paths
}

func newONNXSession(modelPath string, mode InferenceMode) (*ort.DynamicAdvancedSession, error) {
	ort.SetSharedLibraryPath(onnxRuntimeLibrary())
	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to load ONNX Runtime (set SR_ONNXRUNTIME_LIB to libonnxruntime): %v", err)
	}

	options, err := ort.NewSessionOptions()
	if err != nil {
		return nil, err
	}
	defer options.Destroy()

	// Hardware providers are best effort: fall back to CPU if the provider isn't available.
	switch mode {
	case ModeGPU:
		if cuda, err := ort.NewCUDAProviderOptions(); err != nil {
			warnf("CUDA unavailable, using CPU: %v", err)
		} else {
			if err := options.AppendExecutionProviderCUDA(cuda); err != nil {
				warnf("CUDA unavailable, using CPU: %v", err)
			}
			cuda.Destroy()
		}
	case ModeMPS:
		if err := options.AppendExecutionProviderCoreMLV2(map[string]string{
			"ModelFormat":    "MLProgram",
			"MLComputeUnits": "ALL",
		}); err != nil {
			warnf("CoreML unavailable, using CPU: %v", err)
		}
	}

	model, err := os.ReadFile(modelPath)
	if err != nil {
		return nil, err
	}
	// Tiles always have the same shape; pinning it lets ORT/CoreML plan a static graph.
	if fixed, err := withStaticInputShape(model, []int64{1, 3, onnxTileSize, onnxTileSize}); err == nil {
		model = fixed
	}
	return ort.NewDynamicAdvancedSessionWithONNXData(model, []string{"input"}, []string{"output"}, options)
}

// withStaticInputShape rewrites the dynamic dimensions of the model's 4-D graph inputs
// (ModelProto.graph.input[].type.tensor_type.shape) to dims, like ORT's free-dimension overrides.
func withStaticInputShape(model []byte, dims []int64) ([]byte, error) {
	setShape := func(shape []byte) ([]byte, error) {
		count, dynamic := 0, false
		if _, err := rewriteMessageField(shape, 1, func(dim []byte) ([]byte, error) {
			count++
			dynamic = dynamic || !hasField(dim, 1) // no dim_value: dim_param or unknown
			return dim, nil
		}); err != nil || count != len(dims) || !dynamic {
			return shape, err
		}
		i := 0
		return rewriteMessageField(shape, 1, func([]byte) ([]byte, error) {
			dim := protowire.AppendTag(nil, 1, protowire.VarintType)
			dim = protowire.AppendVarint(dim, uint64(dims[i]))
			i++
			return dim, nil
		})
	}
	return rewriteMessageField(model, 7, func(graph []byte) ([]byte, error) { // ModelProto.graph
		return rewriteMessageField(graph, 11, func(input []byte) ([]byte, error) { // GraphProto.input
			return rewriteMessageField(input, 2, func(typ []byte) ([]byte, error) { // ValueInfoProto.type
				return rewriteMessageField(typ, 1, func(tensor []byte) ([]byte, error) { // TypeProto.tensor_type
					return rewriteMessageField(tensor, 2, setShape) // Tensor.shape
				})
			})
		})
	})
}

// rewriteMessageField returns msg with every length-delimited field num replaced by fn(value).
func rewriteMessageField(msg []byte, num protowire.Number, fn func([]byte) ([]byte, error)) ([]byte, error) {
	out := make([]byte, 0, len(msg))
	for len(msg) > 0 {
		n, typ, tagLen := protowire.ConsumeTag(msg)
		if tagLen < 0 {
			return nil, protowire.ParseError(tagLen)
		}
		valLen := protowire.ConsumeFieldValue(n, typ, msg[tagLen:])
		if valLen < 0 {
			return nil, protowire.ParseError(valLen)
		}
		if n == num && typ == protowire.BytesType {
			value, _ := protowire.ConsumeBytes(msg[tagLen:])
			value, err := fn(value)
			if err != nil {
				return nil, err
			}
			out = protowire.AppendTag(out, n, protowire.BytesType)
			out = protowire.AppendBytes(out, value)
		} else {
			out = append(out, msg[:tagLen+valLen]...)
		}
		msg = msg[tagLen+valLen:]
	}
	return out, nil
}

func hasField(msg []byte, num protowire.Number) bool {
	for len(msg) > 0 {
		n, typ, tagLen := protowire.ConsumeTag(msg)
		if tagLen < 0 {
			return false
		}
		if n == num {
			return true
		}
		valLen := protowire.ConsumeFieldValue(n, typ, msg[tagLen:])
		if valLen < 0 {
			return false
		}
		msg = msg[tagLen+valLen:]
	}
	return false
}

// upscaleWithONNX runs the model over img tile by tile and returns the result at the model's
// native scale. progress (may be nil) is called with the completed fraction after each tile.
func upscaleWithONNX(img gocv.Mat, modelPath string, mode InferenceMode, progress func(float64)) (gocv.Mat, error) {
	session, err := newONNXSession(modelPath, mode)
	if err != nil {
		return gocv.Mat{}, err
	}
	defer session.Destroy()
	defer ort.DestroyEnvironment()

	width, height := img.Cols(), img.Rows()
	src, err := img.DataPtrUint8() // BGR, row-major
	if err != nil {
		return gocv.Mat{}, err
	}

	const tile, pad = onnxTileSize, onnxTileOverlap
	core := tile - 2*pad
	tilesX, tilesY := (width+core-1)/core, (height+core-1)/core

	input := make([]float32, 3*tile*tile)
	inputTensor, err := ort.NewTensor(ort.NewShape(1, 3, tile, tile), input)
	if err != nil {
		return gocv.Mat{}, err
	}
	defer inputTensor.Destroy()

	var dst []byte
	scale := 0

	for ty := 0; ty < tilesY; ty++ {
		for tx := 0; tx < tilesX; tx++ {
			x0, y0 := tx*core-pad, ty*core-pad

			// Fill the tile as planar RGB, mirroring pixels that fall outside the image.
			for y := 0; y < tile; y++ {
				sy := reflectIndex(y0+y, height)
				for x := 0; x < tile; x++ {
					i := (sy*width + reflectIndex(x0+x, width)) * 3
					p := y*tile + x
					input[p] = float32(src[i+2]) / 255
					input[tile*tile+p] = float32(src[i+1]) / 255
					input[2*tile*tile+p] = float32(src[i]) / 255
				}
			}

			outputs := []ort.Value{nil}
			if err := session.Run([]ort.Value{inputTensor}, outputs); err != nil {
				return gocv.Mat{}, fmt.Errorf("inference failed: %v", err)
			}
			outTensor, ok := outputs[0].(*ort.Tensor[float32])
			if !ok {
				outputs[0].Destroy()
				return gocv.Mat{}, fmt.Errorf("model output is not a float32 tensor")
			}
			shape := outTensor.GetShape()
			if len(shape) != 4 || shape[1] != 3 || shape[2] != shape[3] || shape[2]%tile != 0 {
				outTensor.Destroy()
				return gocv.Mat{}, fmt.Errorf("unsupported model output shape %v for input [1 3 %d %d]", shape, tile, tile)
			}

			if dst == nil {
				scale = int(shape[2]) / tile
				dst = make([]byte, width*scale*height*scale*3)
			}

			// Copy only the tile's core (overlap discarded) into the BGR output image.
			out := outTensor.GetData()
			outTile, outWidth := tile*scale, width*scale
			coreW, coreH := min(core, width-tx*core)*scale, min(core, height-ty*core)*scale
			for y := 0; y < coreH; y++ {
				oy := pad*scale + y
				row := ((ty*core*scale+y)*outWidth + tx*core*scale) * 3
				for x := 0; x < coreW; x++ {
					p := oy*outTile + pad*scale + x
					d := row + x*3
					dst[d] = toByte(out[2*outTile*outTile+p])
					dst[d+1] = toByte(out[outTile*outTile+p])
					dst[d+2] = toByte(out[p])
				}
			}
			outTensor.Destroy()

			if progress != nil {
				progress(float64(ty*tilesX+tx+1) / float64(tilesX*tilesY))
			}
		}
	}

	return gocv.NewMatFromBytes(height*scale, width*scale, gocv.MatTypeCV8UC3, dst)
}

// reflectIndex maps i into [0, n) by mirroring at the edges (OpenCV's BORDER_REFLECT_101).
func reflectIndex(i, n int) int {
	if n == 1 {
		return 0
	}
	for i < 0 || i >= n {
		if i < 0 {
			i = -i
		} else {
			i = 2*(n-1) - i
		}
	}
	return i
}

func toByte(v float32) byte {
	v = v*255 + 0.5
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return byte(v)
}

// resizeTo resizes img to size, using area interpolation when shrinking and cubic when enlarging.
func resizeTo(img gocv.Mat, size image.Point) gocv.Mat {
	result := gocv.NewMat()
	interpolation := gocv.InterpolationCubic
	if size.X < img.Cols() && size.Y < img.Rows() {
		interpolation = gocv.InterpolationArea
	}
	gocv.Resize(img, &result, size, 0, 0, interpolation)
	return result
}

func warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}
