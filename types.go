package main

type InferenceMode string
type UpscaleEngine string

const (
	ModeCPU InferenceMode = "cpu"
	ModeGPU InferenceMode = "gpu"
	ModeMPS InferenceMode = "mps"

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
