package main

// Headless backend used by the Electron desktop app (see electron/).
// Runs a single super-resolution job and reports the result as one JSON line on stdout.
// While running, "progress <0-1>" lines are written to stderr.
// With -inspect, it instead checks that -model is a usable ONNX model and reports its details.
//
// Build: go build -o sr-backend sr-backend.go types.go onnx_engine.go

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"gocv.io/x/gocv"
)

type backendResult struct {
	OK         bool       `json:"ok"`
	OutputPath string     `json:"outputPath,omitempty"`
	Width      int        `json:"width,omitempty"`
	Height     int        `json:"height,omitempty"`
	Model      *ModelInfo `json:"model,omitempty"`
	Error      string     `json:"error,omitempty"`
}

func main() {
	var (
		inputPath    = flag.String("input", "", "Input image path")
		outputPath   = flag.String("output", "", "Output image path")
		modelPath    = flag.String("model", "models/4x-UltraSharpV2_fp32_op17.onnx", "Model path")
		mode         = flag.String("mode", "cpu", "Inference mode: cpu, gpu, mps")
		engine       = flag.String("engine", "onnx", "Upscale engine: opencv, onnx, tensorflow, tflite")
		scaleFactor  = flag.Int("scale", 4, "Scale factor")
		targetWidth  = flag.Int("width", 0, "Target width (0 = use scale factor)")
		targetHeight = flag.Int("height", 0, "Target height (0 = use scale factor)")
		inspect      = flag.Bool("inspect", false, "Check the ONNX -model and print its details instead of processing")
	)
	flag.Parse()

	if *inspect {
		info, err := inspectONNXModel(*modelPath, InferenceMode(*mode))
		if err != nil {
			fail(err)
		}
		emit(backendResult{OK: true, Model: info})
		return
	}

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
	processor.OnProgress = func(f float64) { fmt.Fprintf(os.Stderr, "progress %.4f\n", f) }

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
