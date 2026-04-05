# Super Resolution GUI

> **Note**: This document covers GUI-specific details. See [README.md](README.md) for the full project documentation.

The GUI is the primary interface of this application. It is built with [Fyne](https://fyne.io) and runs natively on Windows, macOS, and Linux.

## Build

```bash
go build -o sr-gui .
./sr-gui
```

## Interface Layout

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│ Super Resolution - AI Image Upscaler                                   [─][□][×] │
├─────────────────────────────────────────────────────────────────────────────────┤
│ 📁 Select Image          │ 🖼️ Image Comparison                                    │
│ ┌─────────────────────┐   │ ┌─────────────────┐ ┌─────────────────┐               │
│ │ 📎 Drag and drop an │   │ │    Original     │ │ Super Resolution│               │
│ │ image file here     │   │ │                 │ │                 │               │
│ │  📁 Browse Files    │   │ │   [Image]       │ │   [Image]       │               │
│ └─────────────────────┘   │ └─────────────────┘ └─────────────────┘               │
│                           │                                                       │
│ ⚙️ Settings                │ Status: Ready - Select an image to get started        │
│ Engine: [OpenCV    ▼]     │ ████████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ │
│ Mode:   [CPU       ▼]     │                                                       │
│ Scale:  [4x        ▼]     │                                                       │
│ Model:  [Browse...]       │                                                       │
│                           │                                                       │
│ 🚀 Process Image          │                                                       │
│ 💾 Save Result  🗑️ Clear   │                                                       │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## Components

**Left panel:**
- **Drop Zone** — drag & drop or browse for image files (JPG, PNG, BMP, TIFF)
- **Settings** — engine, mode, scale, and model path
- **Control buttons** — Process, Save, Clear

**Right panel:**
- **Image Comparison** — side-by-side original and enhanced images
- **Status bar** — real-time status text and progress bar

## Usage

1. Load an image via drag & drop or Browse Files
2. Choose engine and hardware mode
3. Select scale factor (2x/4x/8x) or enter custom width/height
4. Browse for a model file if using ONNX/TensorFlow engines
5. Click **🚀 Process Image**
6. Click **💾 Save Result** to export

## Creating App Bundles

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest

fyne package -os darwin  -name "Super Resolution" -icon icon.png
fyne package -os windows -name "Super Resolution" -icon icon.png
fyne package -os linux   -name "Super Resolution" -icon icon.png
```
