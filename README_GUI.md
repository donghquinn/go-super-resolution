# Super Resolution GUI Application

A cross-platform GUI application for AI-powered image super-resolution with drag-and-drop support and side-by-side comparison.

## Features

✅ **Cross-Platform**: Native executables for Windows, macOS, and Linux  
✅ **Drag & Drop**: Simply drag image files into the application  
✅ **Multiple Engines**: OpenCV, ONNX, TensorFlow, TensorFlow Lite  
✅ **Multiple Modes**: CPU, GPU, MPS (Metal Performance Shaders)  
✅ **Side-by-Side Comparison**: View original and enhanced images together  
✅ **Flexible Scaling**: Scale factors (2x, 4x, 8x) or custom dimensions  
✅ **Model Selection**: Browse and select different AI models  
✅ **Progress Tracking**: Real-time progress indicators  
✅ **Export Options**: Save results in various formats  

## Screenshots

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│ Super Resolution - AI Image Upscaler                                   [─][□][×] │
├─────────────────────────────────────────────────────────────────────────────────┤
│ 📁 Select Image          │ 🖼️ Image Comparison                                    │
│ ┌─────────────────────┐   │ ┌─────────────────┐ ┌─────────────────┐               │
│ │ 📎 Drag and drop an │   │ │    Original     │ │ Super Resolution│               │
│ │ image file here     │   │ │                 │ │                 │               │
│ │                     │   │ │   [Image]       │ │   [Image]       │               │
│ │  📁 Browse Files    │   │ │                 │ │                 │               │
│ └─────────────────────┘   │ └─────────────────┘ └─────────────────┘               │
│                           │                                                       │
│ ⚙️ Settings                │ Status: Ready - Select an image to get started        │
│ Engine: [OpenCV    ▼]     │ ████████████████████████████████████████████████████ │
│ Mode:   [CPU       ▼]     │                                                       │
│ Scale:  [4x        ▼]     │                                                       │
│ Model:  [Browse...]       │                                                       │
│                           │                                                       │
│ 🚀 Process Image          │                                                       │
│ 💾 Save Result  🗑️ Clear   │                                                       │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## Installation

### Option 1: Download Pre-built Executables

1. Download the appropriate executable for your platform:
   - **macOS**: `sr-gui-macos`
   - **Windows**: `sr-gui-windows.exe`
   - **Linux**: `sr-gui-linux`

2. Run the executable directly

### Option 2: Build from Source

1. **Prerequisites**:
   ```bash
   # Install Go (1.21 or later)
   # Install OpenCV
   # macOS: brew install opencv pkg-config
   # Ubuntu: sudo apt-get install libopencv-dev pkg-config
   ```

2. **Clone and build**:
   ```bash
   git clone <repository>
   cd sr-go
   go mod tidy
   go build -o sr-gui sr-gui.go types.go
   ```

3. **Run**:
   ```bash
   ./sr-gui
   ```

## Usage

### Basic Workflow

1. **Select Image**:
   - Drag and drop an image file into the drop zone
   - Or click "📁 Browse Files" to select an image

2. **Configure Settings**:
   - **Engine**: Choose processing engine (OpenCV, ONNX, TensorFlow, TFLite)
   - **Mode**: Select hardware acceleration (CPU, GPU, MPS)
   - **Scale**: Choose scale factor (2x, 4x, 8x) or custom dimensions
   - **Model**: Browse and select AI model file (for ONNX/TensorFlow engines)

3. **Process Image**:
   - Click "🚀 Process Image" to start super-resolution
   - Watch the progress bar and status updates

4. **Compare Results**:
   - View original and enhanced images side by side
   - Zoom and pan to examine details

5. **Save Result**:
   - Click "💾 Save Result" to export the enhanced image
   - Choose output format and location

### Engine Options

| Engine | Description | Model Required | Performance |
|--------|-------------|----------------|-------------|
| **OpenCV** | Simple upscaling algorithms | ❌ No | Fast |
| **ONNX** | AI model inference via ONNX Runtime | ✅ .onnx file | Medium |
| **TensorFlow** | AI model inference via TensorFlow | ✅ .pb file | Medium |
| **TensorFlow Lite** | Lightweight AI inference | ✅ .tflite file | Fast |

### Mode Options

| Mode | Description | Availability |
|------|-------------|--------------|
| **CPU** | CPU-only processing | All platforms |
| **GPU** | CUDA GPU acceleration | NVIDIA GPUs |
| **MPS** | Metal Performance Shaders | macOS only |

### Scale Options

| Scale | Description | Output Size |
|-------|-------------|-------------|
| **2x** | Double resolution | 2× width & height |
| **4x** | Quadruple resolution | 4× width & height |
| **8x** | Octuple resolution | 8× width & height |
| **Custom** | Specific dimensions | User-defined W×H |

## Model Files

### Where to Get Models

1. **Real-ESRGAN**: 
   - GitHub: https://github.com/xinntao/Real-ESRGAN
   - HuggingFace: https://huggingface.co/models?search=esrgan

2. **ESRGAN ONNX Models**:
   - ONNX Model Zoo: https://github.com/onnx/models
   - Convert PyTorch models to ONNX

3. **TensorFlow Models**:
   - TensorFlow Hub: https://tfhub.dev/
   - Convert to TensorFlow Lite for mobile

### Model Formats

| Format | Extension | Engine | Description |
|--------|-----------|---------|-------------|
| **ONNX** | .onnx | ONNX | Cross-platform AI models |
| **TensorFlow** | .pb | TensorFlow | TensorFlow SavedModel |
| **TensorFlow Lite** | .tflite | TFLite | Lightweight models |

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| **Ctrl+O** | Open image file |
| **Ctrl+S** | Save result |
| **Ctrl+R** | Process image |
| **Ctrl+C** | Clear images |
| **F5** | Refresh |

## Troubleshooting

### Common Issues

1. **"Failed to load image"**:
   - Ensure image format is supported (JPG, PNG, BMP, TIFF)
   - Check file permissions

2. **"Model file does not exist"**:
   - Verify model file path is correct
   - Download appropriate model for selected engine

3. **"OpenCV not found"**:
   - Install OpenCV development libraries
   - Set proper environment variables

4. **"CUDA not available"**:
   - Install NVIDIA CUDA toolkit
   - Use CPU mode as fallback

5. **"MPS not supported"**:
   - MPS only works on macOS with Apple Silicon or Intel with Metal
   - Use CPU mode on other platforms

### Performance Tips

1. **For best quality**: Use ONNX or TensorFlow engines with pre-trained models
2. **For speed**: Use OpenCV engine or TensorFlow Lite
3. **For large images**: Use GPU or MPS acceleration
4. **Memory issues**: Process smaller images or use scale factors instead of custom dimensions

## Development

### Building App Bundles

```bash
# Install fyne tool
go install fyne.io/fyne/v2/cmd/fyne@latest

# Create app bundle
fyne package -os darwin -name "Super Resolution" -icon icon.png
fyne package -os windows -name "Super Resolution" -icon icon.png
fyne package -os linux -name "Super Resolution" -icon icon.png
```

### Cross-Platform Build

```bash
# Use the build script
./build.sh

# Or build manually
GOOS=windows GOARCH=amd64 go build -o sr-gui-windows.exe sr-gui.go types.go
GOOS=linux GOARCH=amd64 go build -o sr-gui-linux sr-gui.go types.go
GOOS=darwin GOARCH=amd64 go build -o sr-gui-macos sr-gui.go types.go
```

## License

MIT License - See LICENSE file for details.

## Contributing

1. Fork the repository
2. Create feature branch
3. Make changes
4. Test on multiple platforms
5. Submit pull request

## Support

For issues, feature requests, or questions:
- Open an issue on GitHub
- Check troubleshooting section
- Review documentation