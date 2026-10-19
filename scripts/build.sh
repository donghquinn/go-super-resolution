#!/bin/bash
set -e

# Build script for the Super Resolution desktop app (Electron + Go backend).
# Requires OpenCV 4 (gocv), ONNX Runtime, Go, and Node.js.

cd "$(dirname "$0")/../electron"

# gocv needs OpenCV 4; Homebrew installs it keg-only as opencv@4.
if command -v brew >/dev/null && brew --prefix opencv@4 >/dev/null 2>&1; then
  export PKG_CONFIG_PATH="$(brew --prefix opencv@4)/lib/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
fi

echo "Installing Electron dependencies..."
npm install

# Builds the Go backend into electron/bin/ and packages the app for the current OS
# into electron/dist/. The backend links OpenCV via cgo, so build on each target OS.
echo "Building backend and packaging app..."
npm run dist

echo "Done! Packages are in electron/dist/"
