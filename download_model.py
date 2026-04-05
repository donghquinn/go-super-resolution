#!/usr/bin/env python3
"""
Helper script to download and convert ESRGAN model to ONNX format.
This is a one-time setup script.
"""

import torch
import torch.onnx
import os
from pathlib import Path

def download_realesrgan_model():
    """Download Real-ESRGAN model and convert to ONNX"""

    # Create models directory
    models_dir = Path("models")
    models_dir.mkdir(exist_ok=True)

    print("Creating a simple super-resolution model for demonstration...")

    # Simple CNN-based super-resolution model.
    # bias=False on all Conv2d layers: PyTorch exports Conv2d with bias as a
    # 3-input ONNX Conv node (input + weight + bias), which some OpenCV DNN
    # versions fail to parse and leave a null pointer, causing a SIGSEGV on
    # the first forward pass.  2-input Conv nodes (no bias) are universally
    # supported.
    class SimpleSR(torch.nn.Module):
        def __init__(self):
            super(SimpleSR, self).__init__()
            self.conv1 = torch.nn.Conv2d(3, 64, 3, padding=1, bias=False)
            self.conv2 = torch.nn.Conv2d(64, 64, 3, padding=1, bias=False)
            self.conv3 = torch.nn.Conv2d(64, 3, 3, padding=1, bias=False)
            self.upsample = torch.nn.Upsample(scale_factor=2, mode='nearest')

        def forward(self, x):
            x = torch.relu(self.conv1(x))
            x = torch.relu(self.conv2(x))
            x = self.upsample(x)
            x = torch.sigmoid(self.conv3(x))
            return x

    # Create model
    model = SimpleSR()
    model.eval()

    # Create dummy input (batch_size=1, channels=3, height=64, width=64)
    dummy_input = torch.randn(1, 3, 64, 64)

    # Export to ONNX.
    # opset 9: best compatibility with OpenCV DNN — all core ops are stable.
    # opset 11+ adds Resize coordinate modes (e.g. pytorch_half_pixel) that
    # OpenCV DNN may not support, and can produce 3-input Conv nodes.
    # Static shapes (no dynamic_axes): avoids shape-inference issues in the
    # OpenCV ONNX importer.
    onnx_path = models_dir / "esrgan.onnx"
    with torch.no_grad():
        torch.onnx.export(
            model,
            dummy_input,
            str(onnx_path),
            export_params=True,
            opset_version=9,
            do_constant_folding=True,
            input_names=['input'],
            output_names=['output'],
        )

    print(f"Model saved to {onnx_path}")
    print("Note: This is a simple demonstration model.")
    print("For real super-resolution, use a pre-trained ESRGAN model.")

    return str(onnx_path)

if __name__ == "__main__":
    try:
        model_path = download_realesrgan_model()
        print(f"✓ Model ready at: {model_path}")
    except Exception as e:
        print(f"✗ Error: {e}")
        print("Please install PyTorch: uv run --with torch,torchvision download_model.py")
