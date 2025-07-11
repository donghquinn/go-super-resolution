package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"

	"gocv.io/x/gocv"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: sr-go <input_image> <output_image>")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	outputPath := os.Args[2]

	sr := NewSuperResolution()
	defer sr.Close()

	if err := sr.ProcessImage(inputPath, outputPath); err != nil {
		log.Fatalf("Error processing image: %v", err)
	}

	fmt.Printf("Super-resolution completed: %s -> %s\n", inputPath, outputPath)
}

type SuperResolution struct {
	net gocv.Net
}

func NewSuperResolution() *SuperResolution {
	// Load ESRGAN ONNX model
	// modelPath := "models/esrgan.onnx"
	modelPath := "models/Real-ESRGAN-x4plus.onnx"
	net := gocv.ReadNetFromONNX(modelPath)
	if net.Empty() {
		log.Fatalf("Failed to load ONNX model from %s", modelPath)
	}

	// Set backend and target
	// For Mac, try different backends in order of preference
	net.SetPreferableBackend(gocv.NetBackendOpenCV)
	net.SetPreferableTarget(gocv.NetTargetCPU)
	
	fmt.Println("Using CPU backend for inference")

	return &SuperResolution{net: net}
}

func (sr *SuperResolution) Close() {
	sr.net.Close()
}

func (sr *SuperResolution) ProcessImage(inputPath, outputPath string) error {
	// Load input image
	img := gocv.IMRead(inputPath, gocv.IMReadColor)
	if img.Empty() {
		return fmt.Errorf("failed to load image: %s", inputPath)
	}
	defer img.Close()

	// Preprocess image
	blob := sr.preprocessImage(img)
	defer blob.Close()

	// Run inference
	sr.net.SetInput(blob, "")
	output := sr.net.Forward("")
	defer output.Close()

	// Post-process and save result
	result := sr.postprocessImage(output)
	defer result.Close()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Save result
	if !gocv.IMWrite(outputPath, result) {
		return fmt.Errorf("failed to save result image: %s", outputPath)
	}

	return nil
}

func (sr *SuperResolution) preprocessImage(img gocv.Mat) gocv.Mat {
	// Convert to float32 and normalize to [0, 1]
	imgFloat := gocv.NewMat()
	img.ConvertTo(&imgFloat, gocv.MatTypeCV32F)
	imgFloat.DivideFloat(255.0)

	// Create blob from image (NCHW format)
	size := img.Size()
	blob := gocv.BlobFromImage(imgFloat, 1.0, image.Pt(size[1], size[0]), gocv.NewScalar(0, 0, 0, 0), true, false)
	
	imgFloat.Close()
	return blob
}

func (sr *SuperResolution) postprocessImage(output gocv.Mat) gocv.Mat {
	// Convert from blob format back to image
	// Output is in NCHW format, we need to convert to HWC
	
	// Get dimensions
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