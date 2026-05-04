#!/usr/bin/env python3
"""Split the 2×4 owl sprite sheet into normalized PNG frames (transparent BG).

Uses gutter-aware column bounds (this sheet is not an exact 256×4 grid).
Requires: pip install pillow
"""

from __future__ import annotations

import argparse
from pathlib import Path

from PIL import Image

# Tuned for web/public/assets/owl/sprite-sheet-original.jpg @ 1024×682
X_SPLITS = (0, 259, 490, 729, 1024)
Y_SPLITS = (0, 340, 682)
BG_RGB_MAX = 11  # inclusive; treat as background (same as RGB < 12)
PAD = 12


def key_background_rgba(im: Image.Image) -> Image.Image:
    rgba = im.convert("RGBA").copy()
    px = rgba.load()
    for y in range(rgba.height):
        for x in range(rgba.width):
            r, g, b, a = px[x, y]
            if r <= BG_RGB_MAX and g <= BG_RGB_MAX and b <= BG_RGB_MAX:
                px[x, y] = (r, g, b, 0)
    return rgba


def main() -> None:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument(
        "--src",
        type=Path,
        default=Path("web/public/assets/owl/sprite-sheet-original.jpg"),
        help="Sprite sheet image path",
    )
    p.add_argument(
        "--out-dir",
        type=Path,
        default=Path("web/public/assets/owl"),
        help="Directory for frame-01.png … frame-08.png",
    )
    args = p.parse_args()

    sheet = Image.open(args.src).convert("RGBA")
    W, H = sheet.size
    if list(X_SPLITS) != [0, 259, 490, 729, W] or list(Y_SPLITS) != [0, 340, H]:
        raise SystemExit(
            f"Unexpected sheet size {W}×{H}; update X_SPLITS/Y_SPLITS in this script for the new asset."
        )

    crops: list[Image.Image] = []
    for row in range(2):
        for col in range(4):
            x0, x1 = X_SPLITS[col], X_SPLITS[col + 1]
            y0, y1 = Y_SPLITS[row], Y_SPLITS[row + 1]
            cell = sheet.crop((x0, y0, x1, y1))
            keyed = key_background_rgba(cell)
            bb = keyed.getbbox()
            if bb is None:
                crops.append(keyed)
            else:
                crops.append(keyed.crop(bb))

    widths = [im.width for im in crops]
    heights = [im.height for im in crops]
    cw = max(widths) + 2 * PAD
    ch = max(heights) + 2 * PAD
    if cw % 2:
        cw += 1
    if ch % 2:
        ch += 1

    args.out_dir.mkdir(parents=True, exist_ok=True)
    for i, im in enumerate(crops, start=1):
        canvas = Image.new("RGBA", (cw, ch), (0, 0, 0, 0))
        ox = (cw - im.width) // 2
        oy = (ch - im.height) // 2
        canvas.paste(im, (ox, oy), im)
        dest = args.out_dir / f"frame-{i:02d}.png"
        canvas.save(dest, optimize=True)
        print(dest, im.width, "x", im.height, "->", cw, "x", ch)


if __name__ == "__main__":
    main()
