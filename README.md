# Super Resolution - AI Image Upscaler

A cross-platform GUI application for AI-powered image super-resolution. Enhance image quality using multiple AI engines including ESRGAN, with support for various hardware acceleration options.

![GUI Demo](gui-demo.png)

## Features

### **Multiple AI Engines**
- **OpenCV**: Fast traditional upscaling algorithms — no model required
- **ONNX**: AI model inference via GoCV's DNN module (`.onnx`)
- **TensorFlow**: TensorFlow model support (`.pb`)
- **TensorFlow Lite**: Lightweight inference for edge devices (`.tflite`)

### **Hardware Acceleration**
- **CPU**: Works on all platforms
- **GPU**: CUDA acceleration for NVIDIA graphics cards
- **MPS**: Metal Performance Shaders for macOS (Apple Silicon & Intel)

### **Flexible Scaling**
- **Scale Factors**: 2x, 4x, 8x
- **Custom Dimensions**: Specify exact output width/height
- **Aspect Ratio**: Single-dimension input preserves proportions

### **GUI Features**
- Drag-and-drop image loading
- Side-by-side original vs. result comparison
- Real-time progress tracking
- Browse and select AI model files
- Export result to any location

## Quick Start

```bash
# Build
go build -o sr-gui .

# Run
./sr-gui
```

1. **Select Image** — drag & drop or click Browse Files
2. **Configure Settings** — engine, mode, scale, and model path
3. **Process** — click "🚀 Process Image"
4. **Save** — click "💾 Save Result"

## Installation

### Prerequisites

- **Go 1.21+**
- **OpenCV** with pkg-config

#### macOS
```bash
brew install opencv pkg-config
```

#### Ubuntu/Debian
```bash
sudo apt-get install libopencv-dev pkg-config
```

#### Windows
```bash
# Download OpenCV from https://opencv.org/releases/
# Or: vcpkg install opencv
```

### Build from Source

```bash
git clone <repository>
cd go-super-resolution
go mod tidy
go build -o sr-gui .
./sr-gui
```

### Creating App Bundles

```bash
# Install fyne packaging tool
go install fyne.io/fyne/v2/cmd/fyne@latest

# macOS
fyne package -os darwin -name "Super Resolution" -icon icon.png

# Windows
fyne package -os windows -name "Super Resolution" -icon icon.png

# Linux
fyne package -os linux -name "Super Resolution" -icon icon.png
```

## Engine Configuration

| Engine | Model Required | Speed | Notes |
|--------|---------------|-------|-------|
| **OpenCV** | No | ⭐⭐⭐⭐⭐ | Cubic/Lanczos4 interpolation |
| **ONNX** | `.onnx` | ⭐⭐⭐ | Recommended for AI quality |
| **TensorFlow** | `.pb` | ⭐⭐⭐ | Full TensorFlow support |
| **TensorFlow Lite** | `.tflite` | ⭐⭐⭐⭐ | Lightweight/edge devices |

## Hardware Modes

| Mode | Platform | Notes |
|------|----------|-------|
| **CPU** | All | Universal compatibility |
| **GPU** | NVIDIA CUDA | Requires CUDA toolkit |
| **MPS** | macOS only | Apple Silicon & Intel Macs |

## Model Files

### Where to Get Models

- **Real-ESRGAN**: https://github.com/xinntao/Real-ESRGAN
- **ONNX Model Zoo**: https://github.com/onnx/models

### Converting Models

#### PyTorch → ONNX
```python
import torch

model = torch.load('esrgan.pth')
model.eval()

dummy_input = torch.randn(1, 3, 64, 64)
torch.onnx.export(model, dummy_input, "models/esrgan.onnx",
                  opset_version=11,
                  input_names=['input'],
                  output_names=['output'])
```

#### TensorFlow → TensorFlow Lite
```python
import tensorflow as tf

converter = tf.lite.TFLiteConverter.from_saved_model('esrgan_savedmodel')
tflite_model = converter.convert()

with open('models/esrgan.tflite', 'wb') as f:
    f.write(tflite_model)
```

## Project Structure

```
go-super-resolution/
├── main.go                 # Entry point — launches GUI
├── types.go                # Type definitions (InferenceMode, UpscaleEngine, Config)
├── processor.go            # SuperResolutionProcessor — all engine logic
├── tensorflow_engine.go    # TensorFlowEngine — TF inference pipeline
├── gui.go                  # GUI struct and all UI logic (Fyne)
├── check_backends.go       # Diagnostic tool (go:build ignore)
├── test_build.go           # Build verification tool (go:build ignore)
├── go.mod
├── gui-demo.png
└── models/
    ├── Real-ESRGAN-x4plus.onnx
    ├── esrgan.pb
    └── esrgan.tflite
```

## Troubleshooting

### "OpenCV not found" / "pkg-config not found"
```bash
# macOS
brew install opencv pkg-config

# Ubuntu
sudo apt-get install libopencv-dev pkg-config

# Verify
pkg-config --modversion opencv4
```

### "Model file does not exist"
- Verify the model path in the Model field
- Download the appropriate model for the selected engine
- ONNX engine requires `.onnx`, TensorFlow requires `.pb`

### "CUDA not available"
- Install the NVIDIA CUDA toolkit
- Fall back to CPU mode if CUDA is unavailable

### "MPS not supported"
- MPS requires macOS with Metal support (Apple Silicon or Intel)
- Use CPU mode on other platforms

### Memory / Segmentation Fault
- Try smaller input images first
- Use a scale factor instead of large custom dimensions
- Verify OpenCV installation integrity

## Diagnostic Tools

`check_backends.go` and `test_build.go` are standalone diagnostic programs excluded from the normal build. Run them with:

```bash
go run check_backends.go   # List available DNN backends and targets
go run test_build.go       # Verify GoCV and ONNX installation
```

## License

MIT License — see [LICENSE](LICENSE) for details.

## Acknowledgments

- **Real-ESRGAN** — state-of-the-art pre-trained models
- **OpenCV / GoCV** — image processing and DNN inference
- **Fyne** — cross-platform GUI framework
