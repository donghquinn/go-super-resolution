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

The desktop app is an **Electron** frontend (`electron/`) that runs a headless Go/OpenCV
backend (`sr-backend.go`, `types.go`, `onnx_engine.go`) for each job. The `onnx` engine runs
models with [ONNX Runtime](https://onnxruntime.ai/).

### Build from Source

1. **Prerequisites**:
   ```bash
   # Go (1.23 or later), Node.js (20 or later)
   # OpenCV 4 (gocv does not support OpenCV 5) and ONNX Runtime:
   #   macOS:  brew install opencv@4 pkg-config onnxruntime
   #           export PKG_CONFIG_PATH="$(brew --prefix opencv@4)/lib/pkgconfig"
   #   Ubuntu: sudo apt-get install libopencv-dev pkg-config
   #           + ONNX Runtime from https://github.com/microsoft/onnxruntime/releases
   ```

   The backend looks for the ONNX Runtime library next to `sr-backend`, then in the usual
   system locations. Set `SR_ONNXRUNTIME_LIB=/path/to/libonnxruntime.{dylib,so}` to override.

2. **Run in development** (builds the Go backend into `electron/bin/` and launches Electron):
   ```bash
   cd electron
   npm install
   npm run dev
   ```

   Set `SR_BACKEND=/path/to/sr-backend` to use a backend binary from somewhere else.

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

### Built-in Models

The **Model** menu offers two [4x-UltraSharpV2](https://huggingface.co/Kim2091/UltraSharpV2)
models by Kim2091 (May 2025). They are downloaded on first use: into `models/` in development,
or the app's user-data folder when packaged. To fetch them ahead of time, run
`python3 download_model.py`.

| Model | Architecture | Size | Notes |
|-------|--------------|------|-------|
| **UltraSharpV2** | DAT2 transformer | 52 MB | Best quality. Always runs on CPU in `mps` mode (slower with CoreML) |
| **UltraSharpV2 Lite** | RealPLKSR CNN | 30 MB | ~4x faster on CPU, much faster with CoreML (`mps`) |

Both are licensed **CC-BY-NC-SA-4.0** (non-commercial use only).

Both models upscale 4x natively. For 2x, 8x, or custom sizes, the 4x result is resized with
area or cubic interpolation. Images are processed in 256×256 tiles with 32 px overlap, so large
images don't run out of memory.

### Custom Models

Pick **Custom file…** to use any ONNX image-to-image SR model with an `input`/`output` tensor
(NCHW, RGB, values 0–1). Its scale factor is detected automatically.

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

### Architecture

- `electron/src/main.js` - window, native file dialogs, and IPC; spawns `sr-backend` per job
- `electron/src/preload.js` - the `window.sr` bridge (context isolation, sandboxed renderer)
- `electron/src/renderer/` - UI (HTML/CSS/JS)
- `sr-backend.go` - CLI that processes one image and prints a JSON result line

### Packaging

```bash
./scripts/build.sh        # or: cd electron && npm run dist
```

Packages for the current OS are written to `electron/dist/`. The backend links OpenCV via
cgo, so build on each target OS. The OpenCV shared libraries must be available on the
target machine (or bundled alongside `sr-backend`).

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