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
backend (`sr-backend.go`, `types.go`, `onnx_engine.go`, `devices.go`) for each job. The `onnx` engine runs
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
   - **Model**: Pick a model (**Manage…** downloads more), **Basic resize (no AI)**, or **Custom file…**
   - **Engine**: Set automatically from the model (see below)
   - **Use GPU when available**: on by default; GPUs are detected automatically
   - **Scale**: Choose scale factor (2x, 4x, 8x) or custom dimensions

3. **Process Image**:
   - Click "🚀 Process Image" to start super-resolution
   - Watch the progress bar and status updates

4. **Compare Results**:
   - View original and enhanced images side by side
   - Zoom and pan to examine details

5. **Inspect and save**:
   - Each image has **Zoom** and **Download** buttons. You can also double-click an image to zoom.
   - The zoom viewer shows the **Original**, the **Result**, or both **Side by side** with zoom
     and pan kept in sync, so you can compare the same spot. The original is drawn as sharp pixels
     when magnified, so you see what the model started from.
   - **Download** (or **Save Result**) saves a copy of the source or the result, in the same
     format as the file.

### Engine Options

The engine is chosen automatically from the **Model** menu, so it always matches the model:

| Model choice | Engine |
|--------------|--------|
| Any managed model (catalog or added from a URL) | **ONNX Runtime** |
| **Basic resize (no AI)** | **OpenCV** bicubic resize |
| **Custom file…** `.onnx` | **ONNX Runtime** |
| **Custom file…** `.pb` / `.tflite` | **TensorFlow** / **TensorFlow Lite**: not implemented yet, the image is resized without AI |

### GPU Acceleration

The app detects usable GPUs at startup (`sr-backend -devices`) and uses them automatically:
- an **NVIDIA GPU via CUDA** on Windows/Linux;
- the **Apple GPU / Neural Engine via CoreML** on macOS.

The line under **Use GPU when available** shows what was detected. Clear the checkbox to run
everything on CPU; the choice is remembered. If no GPU is usable, the checkbox is disabled.
The status bar shows the device each job ran on.

If a GPU fails for a model, either when the session is created or on the first tile, the job
falls back to the next device instead of failing, and the status bar says why.

On macOS, transformer models with attention layers run on CPU even with GPU on. This covers
DAT/DAT2 (UltraSharpV2, RealWebPhoto), HAT and SwinIR, and is detected from the model itself.
These models measured 2–10x *slower* on CoreML than on CPU, while convolutional models such as
RealPLKSR measured 5–9x faster. The status bar notes when this happens.

The command-line backend still accepts `-mode auto|cpu|gpu|mps` (`gpu` = CUDA, `mps` = CoreML)
to force a device.

**CUDA setup:** Homebrew's and the default ONNX Runtime downloads are CPU builds (plus CoreML on
macOS). For NVIDIA GPUs, install the GPU build of ONNX Runtime 1.31 (`onnxruntime-gpu`) with
CUDA 12 and cuDNN 9. Point the backend at it with
`SR_ONNXRUNTIME_LIB=/path/to/libonnxruntime.so`, or place the library next to `sr-backend`.
`sr-backend -devices` shows whether CUDA is usable and, if not, why.

### Scale Options

| Scale | Description | Output Size |
|-------|-------------|-------------|
| **2x** | Double resolution | 2× width & height |
| **4x** | Quadruple resolution | 4× width & height |
| **8x** | Octuple resolution | 8× width & height |
| **Custom** | Specific dimensions | User-defined W×H |

## Model Files

### Managing Models

Click **Manage…** next to the Model menu to download, use and delete models. Models are stored
in `models/` in development, or the app's user-data folder when packaged (**Open Folder**
shows it). A catalog model you pick in the menu that isn't downloaded yet is fetched when you
click **Process Image**.

Catalog (every model checked against ONNX Runtime with this backend):

| Model | Scale | Architecture | Size | License | Best for |
|-------|-------|--------------|------|---------|----------|
| **UltraSharpV2** (default) | 4x | DAT2 | 52 MB | CC-BY-NC-SA-4.0 | Best overall quality |
| **UltraSharpV2 Lite** | 4x | RealPLKSR | 30 MB | CC-BY-NC-SA-4.0 | Fast version of UltraSharpV2 |
| **RealWebPhoto v4** | 4x | DAT2 | 49 MB | CC-BY-4.0 | Web photos with JPEG artifacts/noise |
| **Nomos2 RealPLKSR** | 4x | RealPLKSR | 30 MB | CC-BY-4.0 | Fast photo upscaling |
| **RealPLKSR 2x** | 2x | RealPLKSR | 30 MB | MIT | Native 2x, faithful (non-GAN) |
| **RealPLKSR 4x** | 4x | RealPLKSR | 30 MB | MIT | Faithful (non-GAN) 4x |

CC-BY-NC-SA models are for non-commercial use only. On Apple Silicon with GPU on, RealPLKSR
models run on CoreML (5–9x faster than CPU), and DAT2 models run on CPU (see
[GPU Acceleration](#gpu-acceleration)).

`python3 download_model.py` pre-downloads the two UltraSharpV2 models.

### Adding Your Own Models

Paste a link to any `.onnx` model under **Add a model from a URL**, e.g. from
[OpenModelDB](https://openmodeldb.info/) or Hugging Face. Hugging Face and GitHub page links
(`/blob/`) are converted to direct downloads. After downloading, the app checks the model by
running `sr-backend -inspect`, and removes it again if it can't be used. You can also drop
`.onnx` files into the models folder yourself.

Supported models take one RGB image in and produce one RGB image out (values 0–1):
- **precision:** float32 or float16;
- **input size:** any size, or a fixed one (the fixed size is used as the tile size);
- **tensor names:** any.

The scale factor is detected automatically. For 2x, 8x or custom sizes, the model's native
result is resized with area or cubic interpolation. Images are processed in tiles with up to
32 px overlap, so large images don't run out of memory.

To check a model from the command line:

```bash
electron/bin/sr-backend -inspect -model path/to/model.onnx
# {"ok":true,"model":{"input":"input","output":"output","precision":"float16","tileWidth":256,"tileHeight":256,"fixedSize":false,"scale":4}}
```

### Model Formats

| Format | Extension | Engine | Description |
|--------|-----------|---------|-------------|
| **ONNX** | .onnx | ONNX | Cross-platform AI models |
| **TensorFlow** | .pb | TensorFlow | TensorFlow SavedModel |
| **TensorFlow Lite** | .tflite | TFLite | Lightweight models |

## Zoom Viewer Controls

| Input | Action |
|-------|--------|
| Scroll wheel / trackpad pinch | Zoom around the cursor |
| Drag | Pan |
| Double-click | Zoom to 100% at the cursor, or back to fit |
| **+** / **−** | Zoom in / out |
| **0** | Fit to window |
| **1** | 100% (actual pixels of the shown image) |
| Arrow keys | Pan |
| **Esc** | Close |

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
- `electron/src/renderer/` - UI (HTML/CSS/JS); `viewer.js` is the zoom viewer
- `electron/src/models.js` - model catalog, downloads, URL import
- `sr-backend.go` - CLI that processes one image (or `-inspect`s a model) and prints a JSON result line
- `devices.go` - execution device selection (CUDA / CoreML / CPU) and `-devices` probing
- `onnx_engine.go` - tiled ONNX Runtime inference (`go test sr-backend.go types.go onnx_engine.go devices.go onnx_engine_test.go`)

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