#!/usr/bin/env python3
"""Render an asciinema cast file to an animated GIF.

Usage: python3 docs/cast2gif.py docs/demo.cast docs/demo.gif

Requires: pip install pyte pillow
Fonts: Menlo (ASCII) plus a CJK face (PingFang or Hiragino), bundled with
macOS.
"""
import argparse
import json
import os
import sys

import pyte
from PIL import Image, ImageDraw, ImageFont

# Dracula-inspired palette
BG = (40, 42, 54)
FG = (248, 248, 242)
PALETTE = {
    "black": (40, 42, 54),
    "red": (255, 85, 85),
    "green": (80, 250, 123),
    "yellow": (241, 250, 140),
    "brown": (241, 250, 140),  # pyte >= 0.9 names SGR 33 "brown"
    "blue": (98, 114, 164),
    "magenta": (255, 121, 198),
    "cyan": (139, 233, 253),
    "white": (248, 248, 242),
    "brightblack": (98, 114, 164),
    "brightred": (255, 110, 103),
    "brightgreen": (91, 250, 140),
    "brightyellow": (255, 255, 150),
    "brightblue": (109, 149, 252),
    "brightmagenta": (255, 148, 203),
    "brightcyan": (148, 226, 213),
    "brightwhite": (255, 255, 255),
}

FONT_SIZE = 17
FPS = 15
ASCIITTC = "/System/Library/Fonts/Menlo.ttc"
CJKTTCS = [
    "/System/Library/Fonts/PingFang.ttc",
    "/System/Library/Fonts/Hiragino Sans GB.ttc",
    "/System/Library/Fonts/STHeiti Light.ttc",
]


def rgb(attr):
    """Map a pyte color attribute to an RGB tuple, or None for default."""
    if attr == "default":
        return None
    if attr.startswith("#") and len(attr) == 7:
        return tuple(int(attr[i:i + 2], 16) for i in (1, 3, 5))
    return PALETTE.get(attr)


BLANK = (" ", "default", "default", False)


def snapshot(screen, cols, rows):
    """Return a hashable snapshot of the screen buffer."""
    # pyte >= 0.9 keeps the buffer as a sparse {row: {col: Char}} dict
    out = []
    for y in range(rows):
        line = screen.buffer.get(y, {})
        out.append(tuple(
            (c.data, c.fg, c.bg, c.bold) if (c := line.get(x)) else BLANK
            for x in range(cols)
        ))
    return tuple(out)


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("cast")
    ap.add_argument("gif")
    ap.add_argument("--fps", type=int, default=FPS)
    ap.add_argument("--font-size", type=int, default=FONT_SIZE)
    args = ap.parse_args()

    with open(args.cast, encoding="utf-8") as f:
        header = json.loads(f.readline())
        events = []
        # asciicast v3 event times are deltas from the previous event;
        # v2 times are cumulative wall-clock offsets.
        cumulative = header.get("version", 3) < 3
        clock = 0.0
        for line in f:
            t, kind, data = json.loads(line)
            if cumulative:
                clock = t
            else:
                clock += t
            if kind == "o":
                events.append((clock, data))

    term = header.get("term", {})
    cols = term.get("cols", header.get("width", 80))
    rows = term.get("rows", header.get("height", 24))
    screen = pyte.Screen(cols, rows)
    stream = pyte.ByteStream(screen)
    # pyte >= 0.9 feeds str, pyte 0.8 feeds bytes
    try:
        stream.feed("")
        feed_str = True
    except TypeError:
        feed_str = False

    # Pick the first CJK font from the candidate list that exists on this
    # machine; all candidates are known to carry CJK glyphs.
    cjk = None
    for path in CJKTTCS:
        if os.path.exists(path):
            cjk = ImageFont.truetype(path, args.font_size, index=0)
            break
    if cjk is None:
        sys.exit(f"no CJK face found in {CJKTTCS}")
    ascii_font = ImageFont.truetype(ASCIITTC, args.font_size, index=0)
    cell_w = ascii_font.getlength("M")
    line_h = int(args.font_size * 1.6)

    def draw_frame():
        img = Image.new("RGB", (int(cols * cell_w), rows * line_h), BG)
        d = ImageDraw.Draw(img)
        for y in range(rows):
            line = screen.buffer.get(y, {})
            x = 0.0
            for cx in range(cols):
                ch = line.get(cx)
                if ch is None or ch.data == " ":
                    x += cell_w
                    continue
                fg = rgb(ch.fg) or FG
                bg = rgb(ch.bg)
                if bg:
                    d.rectangle(
                        [x, y * line_h, x + cell_w, (y + 1) * line_h], fill=bg
                    )
                wide = not ch.data.isascii()
                font = cjk if wide else ascii_font
                d.text((x, y * line_h), ch.data, font=font, fill=fg)
                if ch.bold:  # fake bold by double-striking
                    d.text((x + 0.8, y * line_h), ch.data, font=font, fill=fg)
                x += 2 * cell_w if wide else cell_w
        return img

    # Feed events up to each frame time; static frames extend the previous
    # frame's duration instead of duplicating pixels.
    frames, durations, last_snap = [], [], None
    total = events[-1][0] if events else 0
    frame_t, ei = 0.0, 0
    while frame_t <= total:
        while ei < len(events) and events[ei][0] <= frame_t:
            data = events[ei][1]
            stream.feed(data if feed_str else data.encode())
            ei += 1
        snap = snapshot(screen, cols, rows)
        if snap != last_snap:
            frames.append(draw_frame())
            durations.append(1000 / args.fps)
            last_snap = snap
        elif frames:
            durations[-1] += 1000 / args.fps
        frame_t += 1.0 / args.fps
    if frames:
        durations[-1] += 1000  # hold the final screen briefly
        frames[0].save(
            args.gif,
            save_all=True,
            append_images=frames[1:],
            duration=durations,
            loop=0,
            optimize=False,
        )
        print(f"wrote {args.gif}: {len(frames)} frames, {sum(durations) / 1000:.1f}s")
    else:
        sys.exit("no output events in cast file")


if __name__ == "__main__":
    main()
