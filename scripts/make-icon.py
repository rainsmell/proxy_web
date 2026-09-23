#!/usr/bin/env python3
"""Generate the fnOS app icons.

Renders the mark at high resolution and downsamples with Lanczos.
Requires Pillow (pip install pillow).

Usage:
    python3 scripts/make-icon.py A            # preview /tmp/icon-A.png
    python3 scripts/make-icon.py A --write     # write into packaging/fnos
"""
import math
import os
import sys
from PIL import Image, ImageDraw, ImageFilter

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT = os.path.join(ROOT, "packaging", "fnos")
N = 1024  # supersampled render size


def lerp(a, b, t):
    return tuple(int(round(a[i] + (b[i] - a[i]) * t)) for i in range(3))


def diagonal_gradient(size, *stops):
    """stops: list of RGB colors, blended along a 45 degree axis."""
    w = 64
    small = Image.new("RGB", (w, w))
    px = small.load()
    n = len(stops) - 1
    for y in range(w):
        for x in range(w):
            t = (x + y) / (2 * (w - 1))
            seg = min(int(t * n), n - 1)
            local = t * n - seg
            px[x, y] = lerp(stops[seg], stops[seg + 1], local)
    return small.resize((size, size), Image.BICUBIC)


def rounded_mask(size, ratio=0.225):
    m = Image.new("L", (size, size), 0)
    ImageDraw.Draw(m).rounded_rectangle([0, 0, size - 1, size - 1], radius=int(size * ratio), fill=255)
    return m


def cap(draw, p, w, fill):
    r = w / 2
    draw.ellipse([p[0] - r, p[1] - r, p[0] + r, p[1] + r], fill=fill)


def arc_rounded(draw, c, r, a0, a1, w, fill):
    draw.arc([c[0] - r, c[1] - r, c[0] + r, c[1] + r], a0, a1, fill=fill, width=w)
    for a in (a0, a1):
        rad = math.radians(a)
        cap(draw, (c[0] + r * math.cos(rad), c[1] + r * math.sin(rad)), w, fill)


def wave_points(x0, x1, cy, amp, cycles, steps=120, phase=0.0):
    pts = []
    for i in range(steps + 1):
        t = i / steps
        pts.append((x0 + t * (x1 - x0), cy + amp * math.sin(2 * math.pi * cycles * t + phase)))
    return pts


def mark_flow(draw):
    """Three flowing strokes: reads as data in motion."""
    x0, x1 = 0.20 * N, 0.80 * N
    for i, (cy, amp, w, alpha) in enumerate([
        (0.36, 0.085, 0.085, 255),
        (0.52, 0.085, 0.085, 190),
        (0.68, 0.085, 0.085, 120),
    ]):
        pts = wave_points(x0, x1, cy * N, amp * N, 1.35, phase=math.pi * 0.5 * i)
        col = (255, 255, 255, alpha)
        draw.line(pts, fill=col, width=int(w * N), joint="curve")
        cap(draw, pts[0], w * N, col)
        cap(draw, pts[-1], w * N, col)


def mark_orbit(draw):
    """An open ring cradling a node: relay / connection."""
    c = (0.5 * N, 0.5 * N)
    r = 0.295 * N
    w = int(0.088 * N)
    arc_rounded(draw, c, r, -75, 255, w, (255, 255, 255, 255))
    a = math.radians(-90)
    node = (c[0] + r * math.cos(a), c[1] + r * math.sin(a))
    nr = 0.112 * N
    draw.ellipse([node[0] - nr, node[1] - nr, node[0] + nr, node[1] + nr], fill=(255, 255, 255, 255))
    inner = 0.045 * N
    draw.ellipse([node[0] - inner, node[1] - inner, node[0] + inner, node[1] + inner], fill=(56, 189, 248, 255))


def mark_merge(draw):
    """Two overlapping discs: merge / sync, abstract and bold."""
    r = 0.175 * N
    left = (0.5 * N - 0.155 * N, 0.5 * N)
    right = (0.5 * N + 0.155 * N, 0.5 * N)
    draw.ellipse([left[0] - r, left[1] - r, left[0] + r, left[1] + r], fill=(255, 255, 255, 255))
    draw.ellipse([right[0] - r, right[1] - r, right[0] + r, right[1] + r], fill=(255, 255, 255, 150))
    inner = 0.055 * N
    draw.ellipse([0.5 * N - inner, 0.5 * N - inner, 0.5 * N + inner, 0.5 * N + inner], fill=(56, 189, 248, 255))


def mark_ribbon(draw):
    """An abstract folded ribbon (M-ish) built from rounded strokes."""
    x0, x1 = 0.235 * N, 0.765 * N
    y0, y1 = 0.285 * N, 0.715 * N
    xm = (x0 + x1) / 2
    ym = y0 + (y1 - y0) * 0.60
    w = int(0.115 * N)
    left = [(x0, y1), (x0, y0), (xm, ym)]
    right = [(xm, ym), (x1, y0), (x1, y1)]
    draw.line(left, fill=(255, 255, 255, 255), width=w, joint="curve")
    cap(draw, left[0], w, (255, 255, 255, 255))
    draw.line(right, fill=(255, 255, 255, 170), width=w, joint="curve")
    cap(draw, right[-1], w, (255, 255, 255, 170))


MARKS = {"A": mark_flow, "B": mark_orbit, "C": mark_ribbon, "D": mark_merge}
GRADIENTS = {
    "A": [(37, 99, 235), (124, 58, 237), (6, 182, 212)],
    "B": [(30, 58, 138), (37, 99, 235), (34, 211, 238)],
    "C": [(76, 29, 149), (37, 99, 235), (14, 165, 233)],
    "D": [(37, 99, 235), (124, 58, 237), (236, 72, 153)],
}
SLANT = {"C": 0.09}


def build(variant):
    tile = diagonal_gradient(N, *GRADIENTS[variant])
    mask = rounded_mask(N)

    # soft highlight in the upper-left for depth
    sheen = Image.new("L", (N, N), 0)
    ImageDraw.Draw(sheen).ellipse([-N * 0.55, -N * 0.85, N * 0.95, N * 0.40], fill=40)
    sheen = sheen.filter(ImageFilter.GaussianBlur(N * 0.07))
    tile = Image.composite(Image.new("RGB", (N, N), (255, 255, 255)), tile, sheen)

    glyph = Image.new("RGBA", (N, N), (0, 0, 0, 0))
    MARKS[variant](ImageDraw.Draw(glyph))
    if variant in SLANT:
        s = SLANT[variant]
        glyph = glyph.transform((N, N), Image.AFFINE, (1, s, -s * N * 0.5, 0, 1, 0), resample=Image.BICUBIC)

    shadow = Image.new("RGBA", (N, N), (0, 0, 0, 0))
    shadow.paste((0, 0, 0, 110), (0, 0), glyph.split()[3])
    shadow = shadow.filter(ImageFilter.GaussianBlur(N * 0.018))

    out = tile.convert("RGBA")
    out = Image.alpha_composite(out, shadow)
    out = Image.alpha_composite(out, glyph)
    out.putalpha(mask)
    return out


def save_all(img):
    for size, names in [
        (256, ["ICON_256.PNG", "app/ui/images/icon_256.png", "app/ui/images/icon_v2_256.png"]),
        (64, ["ICON.PNG", "app/ui/images/icon_64.png", "app/ui/images/icon_v2_64.png"]),
    ]:
        resized = img.resize((size, size), Image.LANCZOS)
        for name in names:
            path = os.path.join(OUT, name)
            os.makedirs(os.path.dirname(path), exist_ok=True)
            resized.save(path, "PNG", optimize=True)
            print("wrote", os.path.relpath(path, ROOT), size)


if __name__ == "__main__":
    variant = (sys.argv[1] if len(sys.argv) > 1 else "B").upper()
    if variant not in MARKS:
        raise SystemExit("variant must be A, B or C")
    img = build(variant)
    img.resize((256, 256), Image.LANCZOS).save("/tmp/icon-%s.png" % variant)
    print("preview /tmp/icon-%s.png" % variant)
    if "--write" in sys.argv[2:]:
        save_all(img)
