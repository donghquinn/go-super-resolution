#!/bin/bash
set -e

# Build script for the Super Resolution desktop app (Electron + Go backend).
# Requires OpenCV (gocv), Go, and Node.js.

cd "$(dirname "$0")/../electron"

echo "Installing Electron dependencies..."
npm install

# Builds the Go backend into electron/bin/ and packages the app for the current OS
# into electron/dist/. The backend links OpenCV via cgo, so build on each target OS.
echo "Building backend and packaging app..."
npm run dist

echo "Done! Packages are in electron/dist/"
