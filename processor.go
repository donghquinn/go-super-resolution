package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gocv.io/x/gocv"
)

type SuperResolutionProcessor struct {
	config    Config
	net       gocv.Net          // cached ONNX net; valid when netLoaded == true
	netLoaded bool
	tfEngine  *TensorFlowEngine // cached TF engine; valid when non-nil
}

func NewSuperResolutionProcessor(config Config) (*SuperResolutionProcessor, error) {
	if _, err := os.Stat(config.InputPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("input file does not exist: %s", config.InputPath)
	}

	if config.Engine != EngineOpenCV {
		if _, err := os.Stat(config.ModelPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("model file does not exist: %s", config.ModelPath)
		}
	}

	if err := os.MkdirAll(filepath.Dir(config.OutputPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	processor := &SuperResolutionProcessor{config: config}

	// Load engine once so ProcessImage() pays no model-loading cost.
	switch config.Engine {
	case EngineONNX:
		net := gocv.ReadNetFromONNX(config.ModelPath)
		if net.Empty() {
			return nil, fmt.Errorf("failed to load ONNX model: %s", config.ModelPath)
		}
		switch config.Mode {
		case ModeCPU:
			net.SetPreferableBackend(gocv.NetBackendOpenCV)
			net.SetPreferableTarget(gocv.NetTargetCPU)
		case ModeGPU:
			net.SetPreferableBackend(gocv.NetBackendCUDA)
			net.SetPreferableTarget(gocv.NetTargetCUDA)
		case ModeMPS:
			net.SetPreferableBackend(gocv.NetBackendOpenCV)
			net.SetPreferableTarget(gocv.NetTargetCPU) // MPS not directly supported
		}
		processor.net = net
		processor.netLoaded = true

	case EngineTensorFlow:
		engine, err := NewTensorFlowEngine(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create TensorFlow engine: %v", err)
		}
		processor.tfEngine = engine
	}

	fmt.Printf("✓ Processor initialized with %s engine in %s mode\n", config.Engine, config.Mode)
	return processor, nil
}

func (p *SuperResolutionProcessor) ProcessImage() error {
	img := gocv.IMRead(p.config.InputPath, gocv.IMReadColor)
	if img.Empty() {
		return fmt.Errorf("failed to load image: %s", p.config.InputPath)
	}
	defer img.Close()

	fmt.Printf("✓ Loaded image: %dx%d\n", img.Cols(), img.Rows())

	var result gocv.Mat
	var err error

	switch p.config.Engine {
	case EngineOpenCV:
		result = p.processWithOpenCV(img)
	case EngineONNX:
		result, err = p.processWithONNX(img)
		if err != nil {
			return fmt.Errorf("ONNX processing failed: %v", err)
		}
	case EngineTensorFlow:
		result, err = p.processWithTensorFlow(img)
		if err != nil {
			return fmt.Errorf("TensorFlow processing failed: %v", err)
		}
	case EngineTFLite:
		result, err = p.processWithTFLite(img)
		if err != nil {
			return fmt.Errorf("TFLite processing failed: %v", err)
		}
	default:
		return fmt.Errorf("unsupported engine: %s", p.config.Engine)
	}
	defer result.Close()

	if !gocv.IMWrite(p.config.OutputPath, result) {
		return fmt.Errorf("failed to save result: %s", p.config.OutputPath)
	}

	fmt.Printf("✓ Saved result: %dx%d\n", result.Cols(), result.Rows())
	return nil
}

func (p *SuperResolutionProcessor) processWithOpenCV(img gocv.Mat) gocv.Mat {
	fmt.Printf("Processing with OpenCV engine (%s mode)\n", p.config.Mode)

	newSize := p.calculateTargetSize(img)
	if p.config.TargetWidth > 0 || p.config.TargetHeight > 0 {
		fmt.Printf("Target size: %dx%d\n", newSize.X, newSize.Y)
	} else {
		fmt.Printf("Scale factor: %dx (from %dx%d to %dx%d)\n",
			p.config.ScaleFactor, img.Cols(), img.Rows(), newSize.X, newSize.Y)
	}

	result := gocv.NewMat()

	var interpolation gocv.InterpolationFlags
	switch p.config.Mode {
	case ModeCPU:
		interpolation = gocv.InterpolationCubic
	case ModeGPU:
		interpolation = gocv.InterpolationLanczos4
	case ModeMPS:
		interpolation = gocv.InterpolationCubic
	}

	gocv.Resize(img, &result, newSize, 0, 0, interpolation)
	return result
}

func (p *SuperResolutionProcessor) processWithONNX(img gocv.Mat) (gocv.Mat, error) {
	fmt.Printf("Processing with ONNX engine (%s mode)\n", p.config.Mode)

	blob := p.preprocessImage(img)
	defer blob.Close()

	p.net.SetInput(blob, "")
	output := p.net.Forward("")
	defer output.Close()

	result := p.postprocessImage(output)
	return result, nil
}

func (p *SuperResolutionProcessor) processWithTensorFlow(img gocv.Mat) (gocv.Mat, error) {
	fmt.Printf("Processing with TensorFlow engine (%s mode)\n", p.config.Mode)

	result, err := p.tfEngine.Inference(img)
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("TensorFlow inference failed: %v", err)
	}

	return result, nil
}

func (p *SuperResolutionProcessor) processWithTFLite(img gocv.Mat) (gocv.Mat, error) {
	fmt.Printf("Processing with TensorFlow Lite engine (%s mode)\n", p.config.Mode)
	fmt.Println("⚠️  TensorFlow Lite inference simulation")
	fmt.Printf("Model: %s\n", p.config.ModelPath)

	newSize := p.calculateTargetSize(img)
	if p.config.TargetWidth > 0 || p.config.TargetHeight > 0 {
		fmt.Printf("Target size: %dx%d\n", newSize.X, newSize.Y)
	} else {
		fmt.Printf("Scale factor: %dx\n", p.config.ScaleFactor)
	}

	result := gocv.NewMat()

	var interpolation gocv.InterpolationFlags
	switch p.config.Mode {
	case ModeCPU:
		interpolation = gocv.InterpolationCubic
	case ModeGPU:
		interpolation = gocv.InterpolationLanczos4
	case ModeMPS:
		interpolation = gocv.InterpolationCubic
	}

	gocv.Resize(img, &result, newSize, 0, 0, interpolation)
	return result, nil
}

// calculateTargetSize computes the output dimensions based on Config.
// Uses TargetWidth/TargetHeight when set, otherwise applies ScaleFactor.
// A missing dimension is derived from the other while preserving aspect ratio.
func (p *SuperResolutionProcessor) calculateTargetSize(img gocv.Mat) image.Point {
	if p.config.TargetWidth > 0 || p.config.TargetHeight > 0 {
		width := p.config.TargetWidth
		height := p.config.TargetHeight

		if width == 0 {
			aspectRatio := float64(img.Cols()) / float64(img.Rows())
			width = int(float64(height) * aspectRatio)
		}
		if height == 0 {
			aspectRatio := float64(img.Rows()) / float64(img.Cols())
			height = int(float64(width) * aspectRatio)
		}

		return image.Pt(width, height)
	}

	return image.Pt(img.Cols()*p.config.ScaleFactor, img.Rows()*p.config.ScaleFactor)
}

func (p *SuperResolutionProcessor) preprocessImage(img gocv.Mat) gocv.Mat {
	// BlobFromImage handles float32 conversion and [0,1] normalization in a
	// single pass via its scalefactor argument, avoiding an intermediate Mat.
	size := img.Size()
	return gocv.BlobFromImage(img, 1.0/255.0, image.Pt(size[1], size[0]), gocv.NewScalar(0, 0, 0, 0), true, false)
}

func (p *SuperResolutionProcessor) postprocessImage(output gocv.Mat) gocv.Mat {
	// Convert from blob format back to image.
	// Output is in NCHW format; convert to HWC.

	size := output.Size()
	if len(size) != 4 {
		log.Fatalf("Unexpected output dimensions: %v", size)
	}

	// Reshape to 3D (CHW)
	channels := size[1]
	height := size[2]
	width := size[3]

	reshaped := output.Reshape(1, channels*height*width)
	defer reshaped.Close()

	// Convert CHW to HWC
	var hwc gocv.Mat
	gocv.CvtColor(reshaped, &hwc, gocv.ColorBGRToRGB)

	// Denormalize from [0, 1] to [0, 255]
	hwc.MultiplyFloat(255.0)

	// Convert to uint8
	var result gocv.Mat
	hwc.ConvertTo(&result, gocv.MatTypeCV8U)

	hwc.Close()
	return result
}

func (p *SuperResolutionProcessor) Close() {
	if p.netLoaded {
		p.net.Close()
		p.netLoaded = false
	}
	if p.tfEngine != nil {
		p.tfEngine.Close()
		p.tfEngine = nil
	}
}

func detectImageFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return "jpeg"
	case ".png":
		return "png"
	default:
		return "jpeg"
	}
}
