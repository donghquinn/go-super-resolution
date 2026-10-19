package main

// Headless backend used by the Electron desktop app (see electron/).
// Runs a single super-resolution job and reports the result as one JSON line on stdout.
//
// Build: go build -o sr-backend sr-backend.go types.go

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"gocv.io/x/gocv"
)

type backendResult struct {
	OK         bool   `json:"ok"`
	OutputPath string `json:"outputPath,omitempty"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	Error      string `json:"error,omitempty"`
}

func main() {
	var (
		inputPath    = flag.String("input", "", "Input image path")
		outputPath   = flag.String("output", "", "Output image path")
		modelPath    = flag.String("model", "models/Real-ESRGAN-x4plus.onnx", "Model path")
		mode         = flag.String("mode", "cpu", "Inference mode: cpu, gpu, mps")
		engine       = flag.String("engine", "opencv", "Upscale engine: opencv, onnx, tensorflow, tflite")
		scaleFactor  = flag.Int("scale", 4, "Scale factor")
		targetWidth  = flag.Int("width", 0, "Target width (0 = use scale factor)")
		targetHeight = flag.Int("height", 0, "Target height (0 = use scale factor)")
	)
	flag.Parse()

	if *inputPath == "" || *outputPath == "" {
		fail(fmt.Errorf("both -input and -output are required"))
	}

	config := Config{
		InputPath:    *inputPath,
		OutputPath:   *outputPath,
		ModelPath:    *modelPath,
		Mode:         InferenceMode(*mode),
		Engine:       UpscaleEngine(*engine),
		ScaleFactor:  *scaleFactor,
		TargetWidth:  *targetWidth,
		TargetHeight: *targetHeight,
	}

	processor, err := NewSuperResolutionProcessor(config)
	if err != nil {
		fail(fmt.Errorf("failed to create processor: %v", err))
	}
	defer processor.Close()

	if err := processor.ProcessImage(); err != nil {
		fail(fmt.Errorf("processing failed: %v", err))
	}

	result := backendResult{OK: true, OutputPath: config.OutputPath}
	if img := gocv.IMRead(config.OutputPath, gocv.IMReadColor); !img.Empty() {
		result.Width, result.Height = img.Cols(), img.Rows()
		img.Close()
	}
	emit(result)
}

func emit(r backendResult) {
	json.NewEncoder(os.Stdout).Encode(r)
}

func fail(err error) {
	emit(backendResult{OK: false, Error: err.Error()})
	os.Exit(1)
}
