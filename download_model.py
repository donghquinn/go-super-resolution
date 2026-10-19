#!/usr/bin/env python3
"""
Download the built-in ONNX super-resolution models into models/.

The Electron app downloads these on first use; run this to fetch them ahead of time.
Models: 4x-UltraSharpV2 and 4x-UltraSharpV2 Lite by Kim2091 (CC-BY-NC-SA-4.0, non-commercial).
"""

import sys
import urllib.request
from pathlib import Path

BASE_URL = "https://huggingface.co/Kim2091/UltraSharpV2/resolve/main/"
MODELS = [
    "4x-UltraSharpV2_fp32_op17.onnx",       # DAT2, best quality (52 MB)
    "4x-UltraSharpV2_Lite_fp32_op17.onnx",  # RealPLKSR, faster (30 MB)
]


def download(name: str, models_dir: Path) -> None:
    dest = models_dir / name
    if dest.exists():
        print(f"✓ {name} (already downloaded)")
        return
    part = dest.with_suffix(dest.suffix + ".part")

    def report(blocks: int, block_size: int, total: int) -> None:
        if total > 0:
            print(f"\r  {name}: {min(blocks * block_size / total, 1):.0%}", end="", flush=True)

    try:
        urllib.request.urlretrieve(BASE_URL + name, part, report)
    except BaseException:
        part.unlink(missing_ok=True)
        raise
    part.rename(dest)
    print(f"\r✓ {name}          ")


def main() -> int:
    models_dir = Path(__file__).resolve().parent / "models"
    models_dir.mkdir(exist_ok=True)
    try:
        for name in MODELS:
            download(name, models_dir)
    except Exception as e:
        print(f"\n✗ Download failed: {e}")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
