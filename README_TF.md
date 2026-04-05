# TensorFlow Engine

> **Note**: The TensorFlow engine is part of the main application. See [README.md](README.md) for build and usage instructions.

`tensorflow_engine.go` implements the `TensorFlowEngine` struct used when the **TensorFlow** engine is selected in the GUI.

## Pipeline

```
Input image (BGR Mat)
    │
    ▼
preprocessImage()       — BGR → RGB, float32, normalize [0,1]
    │
    ▼
runInference()          — upscale via Resize (simulated; replace with real TF session)
    │
    ▼
postprocessImage()      — denormalize [0,255], uint8, RGB → BGR
    │
    ▼
Output image (BGR Mat)
```

## Struct

```go
type TensorFlowEngine struct {
    config      Config   // ScaleFactor, TargetWidth/Height, Mode, ModelPath
    inputName   string   // default: "input"
    outputName  string   // default: "output"
    initialized bool
}
```

## Key Methods

| Method | Description |
|--------|-------------|
| `NewTensorFlowEngine(config)` | Creates and initializes the engine |
| `Inference(img)` | Full preprocess → infer → postprocess pipeline |
| `GetModelInfo()` | Returns metadata map |
| `ValidateModel()` | Placeholder model validation |
| `Close()` | Cleanup |

## Inference Modes

| Mode | Backend |
|------|---------|
| `cpu` | CPU session |
| `gpu` | CUDA |
| `mps` | Metal Performance Shaders (macOS) |

## Development Status

- ✅ Engine structure and pipeline
- ✅ Image preprocessing / postprocessing
- ✅ TargetWidth/TargetHeight and ScaleFactor support
- ⚠️ Inference is currently simulated with `gocv.Resize`
- ⚠️ Actual TF session / model loading not yet implemented

## Implementing Real Inference

Replace the body of `runInference()` in `tensorflow_engine.go` with:

1. Load the `.pb` model using the TensorFlow Go API
2. Create a session with CPU/GPU/MPS configuration
3. Feed the preprocessed `Mat` as an input tensor
4. Run the session and retrieve the output tensor
5. Return the output tensor as a `gocv.Mat`

## Model Format

Expects TensorFlow SavedModel in `.pb` (protobuf) format.

### ONNX → TensorFlow
```python
import onnx
import onnx_tf

onnx_model = onnx.load('esrgan.onnx')
tf_model = onnx_tf.backend.prepare(onnx_model)
tf_model.export_graph('models/esrgan.pb')
```

### PyTorch → TensorFlow
```python
import torch
import tensorflow as tf

# ... conversion via ONNX as intermediate format
tf.saved_model.save(converted_model, 'models/esrgan.pb')
```
