package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"gocv.io/x/gocv"
)

type InferenceMode string
type UpscaleEngine string

const (
	// Inference modes
	ModeAuto InferenceMode = "auto" // fastest available: CUDA, then CoreML (macOS), then CPU
	ModeCPU  InferenceMode = "cpu"
	ModeGPU  InferenceMode = "gpu" // CUDA
	ModeMPS  InferenceMode = "mps" // Apple GPU / Neural Engine via CoreML

	// Upscale engines
	EngineOpenCV     UpscaleEngine = "opencv"
	EngineONNX       UpscaleEngine = "onnx"
	EngineTensorFlow UpscaleEngine = "tensorflow"
	EngineTFLite     UpscaleEngine = "tflite"
)

type Config struct {
	InputPath    string
	OutputPath   string
	ModelPath    string
	Mode         InferenceMode
	Engine       UpscaleEngine
	ScaleFactor  int
	TargetWidth  int
	TargetHeight int
}

type SuperResolutionProcessor struct {
	config Config

	// OnProgress, if set, is called with the completed fraction (0-1) during model inference.
	OnProgress func(float64)

	// Set by ProcessImage for model engines: the device that ran the model, and why any
	// faster devices were skipped.
	Device   Device
	Warnings []string
}

func NewSuperResolutionProcessor(config Config) (*SuperResolutionProcessor, error) {
	// Validate input file
	if _, err := os.Stat(config.InputPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("input file does not exist: %s", config.InputPath)
	}

	// Validate model file for ONNX and TensorFlow engines
	if config.Engine != EngineOpenCV {
		if _, err := os.Stat(config.ModelPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("model file does not exist: %s", config.ModelPath)
		}
	}

	// Create output directory if needed
	if err := os.MkdirAll(filepath.Dir(config.OutputPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	processor := &SuperResolutionProcessor{
		config: config,
	}

	return processor, nil
}

func (p *SuperResolutionProcessor) ProcessImage() error {
	// Load image
	img := gocv.IMRead(p.config.InputPath, gocv.IMReadColor)
	if img.Empty() {
		return fmt.Errorf("failed to load image: %s", p.config.InputPath)
	}
	defer img.Close()

	var result gocv.Mat
	var err error

	// Process based on selected engine
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

	// Save result
	if !gocv.IMWrite(p.config.OutputPath, result) {
		return fmt.Errorf("failed to save result: %s", p.config.OutputPath)
	}

	return nil
}

// targetSize returns the requested output size for img: explicit dimensions (keeping the
// aspect ratio when only one is given) or the scale factor.
func (p *SuperResolutionProcessor) targetSize(img gocv.Mat) image.Point {
	width, height := p.config.TargetWidth, p.config.TargetHeight
	if width <= 0 && height <= 0 {
		return image.Pt(img.Cols()*p.config.ScaleFactor, img.Rows()*p.config.ScaleFactor)
	}
	if width <= 0 {
		width = int(float64(height) * float64(img.Cols()) / float64(img.Rows()))
	}
	if height <= 0 {
		height = int(float64(width) * float64(img.Rows()) / float64(img.Cols()))
	}
	return image.Pt(width, height)
}

func (p *SuperResolutionProcessor) processWithOpenCV(img gocv.Mat) gocv.Mat {
	result := gocv.NewMat()

	// Use different interpolation based on mode
	interpolation := gocv.InterpolationCubic
	if p.config.Mode == ModeGPU {
		interpolation = gocv.InterpolationLanczos4
	}

	gocv.Resize(img, &result, p.targetSize(img), 0, 0, interpolation)
	return result
}

// processWithONNX upscales with the model at its native scale, then resizes to the requested size.
func (p *SuperResolutionProcessor) processWithONNX(img gocv.Mat) (gocv.Mat, error) {
	upscaled, run, err := upscaleWithONNX(img, p.config.ModelPath, p.config.Mode, p.OnProgress)
	if err != nil {
		return gocv.Mat{}, err
	}
	p.Device, p.Warnings = run.Device, run.Warnings
	size := p.targetSize(img)
	if upscaled.Cols() == size.X && upscaled.Rows() == size.Y {
		return upscaled, nil
	}
	defer upscaled.Close()
	return resizeTo(upscaled, size), nil
}

func (p *SuperResolutionProcessor) processWithTensorFlow(img gocv.Mat) (gocv.Mat, error) {
	// For demo purposes, just do OpenCV upscaling
	// In real implementation, this would use TensorFlow inference
	return p.processWithOpenCV(img), nil
}

func (p *SuperResolutionProcessor) processWithTFLite(img gocv.Mat) (gocv.Mat, error) {
	// For demo purposes, just do OpenCV upscaling
	// In real implementation, this would use TensorFlow Lite inference
	return p.processWithOpenCV(img), nil
}

func (p *SuperResolutionProcessor) Close() {
	// Cleanup resources
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
