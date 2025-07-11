#!/bin/bash

# Build script for Super Resolution GUI

echo "Building Super Resolution GUI..."

# Build for macOS
echo "Building for macOS..."
go build -o sr-gui-macos sr-gui.go types.go

# Build for Windows (requires CGO for OpenCV)
echo "Building for Windows..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o sr-gui-windows.exe sr-gui.go types.go

# Build for Linux
echo "Building for Linux..."
GOOS=linux GOARCH=amd64 go build -o sr-gui-linux sr-gui.go types.go

echo "Build complete!"
echo "Executables:"
echo "  - sr-gui-macos (macOS)"
echo "  - sr-gui-windows.exe (Windows)"
echo "  - sr-gui-linux (Linux)"

# Package with fyne
echo ""
echo "Creating app bundles..."
go install fyne.io/fyne/v2/cmd/fyne@latest
fyne package -os darwin -name "Super Resolution" -icon icon.png
fyne package -os windows -name "Super Resolution" -icon icon.png
fyne package -os linux -name "Super Resolution" -icon icon.png

echo "Done!"