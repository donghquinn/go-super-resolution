package main

// ONNX Runtime inference for image-to-image super-resolution models
// (e.g. 4x-UltraSharpV2): NCHW RGB input in [0, 1], output scaled by the model's factor.

import (
	"encoding/binary"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"runtime"

	ort "github.com/yalue/onnxruntime_go"
	"gocv.io/x/gocv"
	"google.golang.org/protobuf/encoding/protowire"
)

const (
	onnxTileSize    = 256 // tile edge for models that accept any input size
	onnxTileOverlap = 32  // max context pixels on each side of a tile, discarded from the output
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

// onnxUpscaler runs one image-to-image SR model on fixed-size tiles. It accepts models with any
// tensor names, float32 or float16 precision, and dynamic or fixed input sizes.
type onnxUpscaler struct {
	model        []byte   // ONNX model with its input shape pinned to the tile size
	chain        []Device // devices still to try if the current one fails
	device       Device   // device the session runs on
	warnings     []string // why faster devices were skipped
	succeeded    bool     // a run has completed on the current device
	session      *ort.DynamicAdvancedSession
	inputName    string
	outputName   string
	half         bool // model uses float16 tensors
	fixedSize    bool // model only accepts one input size
	tileW, tileH int  // input tile size: the model's fixed input size, or onnxTileSize
	scale        int  // detected on the first run

	input    []float32 // planar RGB tile, filled by the caller before run()
	output   []float32 // planar RGB result of the last run()
	inTensor ort.Value
	inHalf   []byte
	outHalf  *ort.CustomDataTensor
}

// ModelInfo describes a model as seen by the backend (see sr-backend -inspect).
type ModelInfo struct {
	Input      string `json:"input"`
	Output     string `json:"output"`
	Precision  string `json:"precision"`
	TileWidth  int    `json:"tileWidth"`
	TileHeight int    `json:"tileHeight"`
	FixedSize  bool   `json:"fixedSize"`
	Scale      int    `json:"scale"`
	Device     Device `json:"device"`
}

// onnxRun reports how an upscale ran.
type onnxRun struct {
	Device   Device
	Warnings []string
}

func newONNXUpscaler(modelPath string, mode InferenceMode) (*onnxUpscaler, error) {
	if err := initONNXRuntime(); err != nil {
		return nil, err
	}
	u := &onnxUpscaler{}
	if err := u.init(modelPath, mode); err != nil {
		u.Close()
		return nil, err
	}
	return u, nil
}

func (u *onnxUpscaler) init(modelPath string, mode InferenceMode) error {
	model, err := os.ReadFile(modelPath)
	if err != nil {
		return err
	}
	inputs, outputs, err := ort.GetInputOutputInfoWithONNXData(model)
	if err != nil {
		return fmt.Errorf("not a valid ONNX model: %v", err)
	}
	if len(inputs) != 1 || len(outputs) < 1 {
		return fmt.Errorf("expected a model with 1 image input, got %d inputs and %d outputs", len(inputs), len(outputs))
	}
	in, out := inputs[0], outputs[0]
	u.inputName, u.outputName = in.Name, out.Name
	dims := in.Dimensions
	if in.OrtValueType != ort.ONNXTypeTensor || len(dims) != 4 || (dims[1] != 3 && dims[1] > 0) {
		return fmt.Errorf("expected an RGB image input [N, 3, H, W], got %s %v", in.Name, dims)
	}
	switch {
	case in.DataType == ort.TensorElementDataTypeFloat && out.DataType == ort.TensorElementDataTypeFloat:
	case in.DataType == ort.TensorElementDataTypeFloat16 && out.DataType == ort.TensorElementDataTypeFloat16:
		u.half = true
	default:
		return fmt.Errorf("unsupported tensor types %s -> %s (need float32 or float16)", in.DataType, out.DataType)
	}

	u.tileW, u.tileH = onnxTileSize, onnxTileSize
	if dims[2] > 0 && dims[3] > 0 {
		u.tileH, u.tileW, u.fixedSize = int(dims[2]), int(dims[3]), true
	}
	u.chain = deviceChain(mode, hasAttention(model))
	// Tiles always have the same shape; pinning it lets ORT/CoreML plan a static graph.
	u.model = model
	if fixed, err := withStaticInputShape(model, []int64{1, 3, int64(u.tileH), int64(u.tileW)}); err == nil {
		u.model = fixed
	}

	shape := ort.NewShape(1, 3, int64(u.tileH), int64(u.tileW))
	u.input = make([]float32, 3*u.tileW*u.tileH)
	if u.half {
		u.inHalf = make([]byte, 2*len(u.input))
		u.inTensor, err = ort.NewCustomDataTensor(shape, u.inHalf, ort.TensorElementDataTypeFloat16)
	} else {
		u.inTensor, err = ort.NewTensor(shape, u.input)
	}
	if err != nil {
		return err
	}
	return u.openSession()
}

// openSession creates a session on the first device in u.chain that works.
func (u *onnxUpscaler) openSession() error {
	err := fmt.Errorf("no execution device to try")
	for len(u.chain) > 0 {
		d := u.chain[0]
		u.chain = u.chain[1:]
		var options *ort.SessionOptions
		if options, err = sessionOptionsFor(d); err == nil {
			u.session, err = ort.NewDynamicAdvancedSessionWithONNXData(u.model, []string{u.inputName}, []string{u.outputName}, options)
			options.Destroy()
		}
		if err == nil {
			u.device = d
			return nil
		}
		u.skipDevice(d, err)
	}
	return err
}

func (u *onnxUpscaler) skipDevice(d Device, err error) {
	if len(u.chain) == 0 {
		return // nothing left to fall back to; the caller reports err itself
	}
	msg := fmt.Sprintf("%s unavailable, using %s: %s", deviceNames[d], deviceNames[u.chain[0]], firstLine(err.Error()))
	warnf("%s", msg)
	u.warnings = append(u.warnings, msg)
}

// run upscales u.input into u.output. If the first run fails on an accelerator, it retries on
// the next device in the chain (some providers only reject a model once it runs).
func (u *onnxUpscaler) run() error {
	err := u.runOnce()
	if err == nil {
		u.succeeded = true
		return nil
	}
	if u.succeeded || len(u.chain) == 0 {
		return err
	}
	u.skipDevice(u.device, err)
	u.session.Destroy()
	u.session = nil
	if u.outHalf != nil {
		u.outHalf.Destroy()
		u.outHalf, u.scale = nil, 0
	}
	if err := u.openSession(); err != nil {
		return err
	}
	return u.run()
}

// runOnce upscales u.input into u.output, detecting the model's scale on the first call.
func (u *onnxUpscaler) runOnce() error {
	if !u.half {
		outputs := []ort.Value{nil}
		if err := u.session.Run([]ort.Value{u.inTensor}, outputs); err != nil {
			return fmt.Errorf("inference failed: %v", err)
		}
		defer outputs[0].Destroy()
		out, ok := outputs[0].(*ort.Tensor[float32])
		if !ok {
			return fmt.Errorf("model output is not a float32 tensor")
		}
		if err := u.checkOutputShape(out.GetShape()); err != nil {
			return err
		}
		u.output = append(u.output[:0], out.GetData()...)
		return nil
	}

	for i, v := range u.input {
		binary.LittleEndian.PutUint16(u.inHalf[2*i:], float32ToFloat16(v))
	}
	if u.outHalf == nil {
		// ONNX Runtime can't hand back float16 outputs it allocates itself, so the output
		// buffer must be preallocated: try common scales until the model accepts one.
		var lastErr error
		for _, scale := range []int{4, 2, 8, 1, 3} {
			shape := ort.NewShape(1, 3, int64(u.tileH*scale), int64(u.tileW*scale))
			out, err := ort.NewCustomDataTensor(shape, make([]byte, 2*shape.FlattenedSize()), ort.TensorElementDataTypeFloat16)
			if err != nil {
				return err
			}
			if lastErr = u.session.Run([]ort.Value{u.inTensor}, []ort.Value{out}); lastErr == nil {
				u.outHalf, u.scale = out, scale
				break
			}
			out.Destroy()
		}
		if u.outHalf == nil {
			return fmt.Errorf("inference failed: %v", lastErr)
		}
	} else if err := u.session.Run([]ort.Value{u.inTensor}, []ort.Value{u.outHalf}); err != nil {
		return fmt.Errorf("inference failed: %v", err)
	}
	data := u.outHalf.GetData()
	u.output = u.output[:0]
	for i := 0; i < len(data); i += 2 {
		u.output = append(u.output, float16ToFloat32(binary.LittleEndian.Uint16(data[i:])))
	}
	return nil
}

func (u *onnxUpscaler) checkOutputShape(shape ort.Shape) error {
	if len(shape) == 4 && shape[0] == 1 && shape[1] == 3 && shape[2] > 0 && shape[2]%int64(u.tileH) == 0 {
		scale := int(shape[2]) / u.tileH
		if shape[3] == int64(u.tileW*scale) && (u.scale == 0 || u.scale == scale) {
			u.scale = scale
			return nil
		}
	}
	return fmt.Errorf("unsupported model output shape %v for input [1 3 %d %d]", shape, u.tileH, u.tileW)
}

func (u *onnxUpscaler) Close() {
	if u.inTensor != nil {
		u.inTensor.Destroy()
	}
	if u.outHalf != nil {
		u.outHalf.Destroy()
	}
	if u.session != nil {
		u.session.Destroy()
	}
	ort.DestroyEnvironment()
}

// inspectONNXModel loads the model and runs one tile to check it works and find its scale.
func inspectONNXModel(modelPath string, mode InferenceMode) (*ModelInfo, error) {
	u, err := newONNXUpscaler(modelPath, mode)
	if err != nil {
		return nil, err
	}
	defer u.Close()
	for i := range u.input {
		u.input[i] = 0.5
	}
	if err := u.run(); err != nil {
		return nil, err
	}
	info := &ModelInfo{
		Input:      u.inputName,
		Output:     u.outputName,
		Precision:  "float32",
		TileWidth:  u.tileW,
		TileHeight: u.tileH,
		FixedSize:  u.fixedSize,
		Scale:      u.scale,
		Device:     u.device,
	}
	if u.half {
		info.Precision = "float16"
	}
	return info, nil
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
func upscaleWithONNX(img gocv.Mat, modelPath string, mode InferenceMode, progress func(float64)) (gocv.Mat, *onnxRun, error) {
	u, err := newONNXUpscaler(modelPath, mode)
	if err != nil {
		return gocv.Mat{}, nil, err
	}
	defer u.Close()

	width, height := img.Cols(), img.Rows()
	src, err := img.DataPtrUint8() // BGR, row-major
	if err != nil {
		return gocv.Mat{}, nil, err
	}

	tileW, tileH := u.tileW, u.tileH
	pad := min(onnxTileOverlap, min(tileW, tileH)/8)
	coreW, coreH := tileW-2*pad, tileH-2*pad
	tilesX, tilesY := (width+coreW-1)/coreW, (height+coreH-1)/coreH
	plane := tileW * tileH

	var dst []byte
	for ty := 0; ty < tilesY; ty++ {
		for tx := 0; tx < tilesX; tx++ {
			x0, y0 := tx*coreW-pad, ty*coreH-pad

			// Fill the tile as planar RGB, mirroring pixels that fall outside the image.
			for y := 0; y < tileH; y++ {
				sy := reflectIndex(y0+y, height)
				for x := 0; x < tileW; x++ {
					i := (sy*width + reflectIndex(x0+x, width)) * 3
					p := y*tileW + x
					u.input[p] = float32(src[i+2]) / 255
					u.input[plane+p] = float32(src[i+1]) / 255
					u.input[2*plane+p] = float32(src[i]) / 255
				}
			}

			if err := u.run(); err != nil {
				return gocv.Mat{}, nil, err
			}
			scale := u.scale
			if dst == nil {
				dst = make([]byte, width*scale*height*scale*3)
			}

			// Copy only the tile's core (overlap discarded) into the BGR output image.
			out := u.output
			outW, outPlane, outWidth := tileW*scale, plane*scale*scale, width*scale
			cw, ch := min(coreW, width-tx*coreW)*scale, min(coreH, height-ty*coreH)*scale
			for y := 0; y < ch; y++ {
				oy := pad*scale + y
				row := ((ty*coreH*scale+y)*outWidth + tx*coreW*scale) * 3
				for x := 0; x < cw; x++ {
					p := oy*outW + pad*scale + x
					d := row + x*3
					dst[d] = toByte(out[2*outPlane+p])
					dst[d+1] = toByte(out[outPlane+p])
					dst[d+2] = toByte(out[p])
				}
			}

			if progress != nil {
				progress(float64(ty*tilesX+tx+1) / float64(tilesX*tilesY))
			}
		}
	}

	result, err := gocv.NewMatFromBytes(height*u.scale, width*u.scale, gocv.MatTypeCV8UC3, dst)
	return result, &onnxRun{Device: u.device, Warnings: u.warnings}, err
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

// float32ToFloat16 converts to IEEE 754 half precision, rounding half away from zero.
func float32ToFloat16(f float32) uint16 {
	bits := math.Float32bits(f)
	sign := uint16(bits>>16) & 0x8000
	exp := int(bits>>23&0xff) - 127 + 15
	mant := bits & 0x7fffff
	switch {
	case bits&0x7fffffff > 0x7f800000: // NaN
		return sign | 0x7e00
	case exp >= 31: // overflow or Inf
		return sign | 0x7c00
	case exp <= 0: // subnormal half (or zero)
		if exp < -10 {
			return sign
		}
		mant |= 0x800000
		shift := uint(14 - exp)
		h := uint16(mant >> shift)
		if mant>>(shift-1)&1 == 1 {
			h++
		}
		return sign | h
	}
	h := sign | uint16(exp)<<10 | uint16(mant>>13)
	if mant&0x1000 != 0 {
		h++ // a carry into the exponent is still the correctly rounded value
	}
	return h
}

func float16ToFloat32(h uint16) float32 {
	sign := uint32(h&0x8000) << 16
	exp := uint32(h>>10) & 0x1f
	mant := uint32(h & 0x3ff)
	switch {
	case exp == 0x1f:
		return math.Float32frombits(sign | 0x7f800000 | mant<<13)
	case exp == 0:
		if mant == 0 {
			return math.Float32frombits(sign)
		}
		e := uint32(127 - 15 + 1)
		for mant&0x400 == 0 {
			mant <<= 1
			e--
		}
		return math.Float32frombits(sign | e<<23 | (mant&0x3ff)<<13)
	}
	return math.Float32frombits(sign | (exp+127-15)<<23 | mant<<13)
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
