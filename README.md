# Super-Resolution Go Application

A pure Go implementation of image super-resolution using ESRGAN and GoCV.

## Prerequisites

1. **Install OpenCV**:
   ```bash
   # macOS
   brew install opencv
   brew install pkg-config
   
   # Ubuntu/Debian
   sudo apt-get install libopencv-dev
   
   # Windows
   # Download OpenCV from https://opencv.org/releases/
   ```

2. **Install GoCV**:
   ```bash
   go mod tidy
   ```

## Setup

1. **Download ESRGAN ONNX Model**:
   
   You need to convert an ESRGAN model to ONNX format. Here are some options:
   
   - Use Real-ESRGAN pre-trained models
   - Convert PyTorch models to ONNX
   - Download from model repositories
   
   Place the model file as `models/esrgan.onnx`
      * I've downloaded from [huggingface](https://huggingface.co/qualcomm/Real-ESRGAN-x4plus/tree/main)
2. **Build the application**:
   ```bash
   go build -o sr-go main.go
   ```

## Usage

```bash
./sr-go input_image.jpg output_image.jpg
```

## Model Conversion

If you have a PyTorch ESRGAN model, you can convert it to ONNX:

```python
import torch
import torch.onnx

# Load your PyTorch model
model = torch.load('esrgan_model.pth')
model.eval()

# Create dummy input
dummy_input = torch.randn(1, 3, 64, 64)

# Export to ONNX
torch.onnx.export(model, dummy_input, "models/esrgan.onnx",
                  export_params=True,
                  opset_version=11,
                  do_constant_folding=True,
                  input_names=['input'],
                  output_names=['output'])
```

## Features

- Pure Go implementation
- CPU and GPU support (when available)
- ONNX model compatibility
- Command-line interface
- Error handling and validation

## Performance Notes

- GPU acceleration requires CUDA-enabled OpenCV
- Processing time depends on input image size and model complexity
- For best performance, use images that are multiples of the model's expected input size