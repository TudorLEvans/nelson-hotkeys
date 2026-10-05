"""Turn the ship below into internal/render/ship_data.go.

A first-rate ship of the line, close-hauled with the wind on her quarter, for
the "England expects" screen. Bow to the right, because the signal was flown
from Victory's masts and the eye should end up at the words under her.

Like the monument, and unlike the plates, there is no source image. A
photograph of Victory at sixty pixels wide is a brown smear; what the screen
wants is a silhouette, and a silhouette is drawn rather than sampled.

Unlike the monument, the silhouette is BUILT rather than typed out. A ship is
three masts of stacked trapezoids, and the thing an editor wants to change is
"the main mast is too tall" or "the courses are too wide", not two hundred
characters of ASCII that have to stay consistent with each other. Every
proportion below is a number, and the drawing follows from it.

Four materials, because a ship is not one substance and shading it as if it
were is what makes rigged silhouettes read as a bush:

    #  canvas   lit, the brightest thing on screen
    %  hull     dark, the ship's weight
    |  timber   masts, yards and bowsprit, flat mid-tone
    *  gunports dark port lids along her lit strake, Victory's chequer
    ~  sea      one dim line she sits in

Run:     python3 assets/ship/generate.py > internal/render/ship_data.go
Preview: python3 assets/ship/generate.py --preview
"""
import sys

W, H = 60, 20  # pixels; H must be even, since a cell is two pixel rows

# The ship indexes rampNavy from plates_data.go, the same eight blues the Nelson
# plate is drawn in. One navy for every bit of navy in the game.
RAMP, RAMP_LEN = "rampNavy", 8

# Per material: the tone at the lit edge and at the shaded one. Canvas stops one
# short of the top of the ramp; the lightest blue is near white and would pull
# the eye off the signal, which is the point of the screen.
TONE = {
    "#": (6.0, 3.0),
    "%": (5.0, 1.0),
    "|": (3.0, 3.0),
    "*": (2.0, 2.0),
    "~": (1.0, 1.0),
}

# SOFT stops a narrow run spanning the whole ramp. A three-pixel yard given the
# full range is one bright pixel beside one black one, which is speckle.
SOFT = 4

# Materials graded down the column rather than across the row. Everything else
# is lit from the left, the way the monument is, which makes a sail round. Run
# across a hull it does the opposite: fifty pixels of one run means a bright
# stern fading to a black bow, and she loses her head. Graded downward instead,
# the gunwale catches the light and the keel sits in the water.
VERTICAL = {"%"}

DECK = 15  # the pixel row the deck sits on; the masts stand from here up

# Masts, stern first: x, the row the truck reaches, and the sails on it as
# (yard row, first canvas row, last canvas row, half width at the top, half width
# at the bottom). Square sails are trapezoids and the courses are the widest
# thing on the ship, so every sail widens downward.
#
# Each sail has its yard on the row ABOVE the canvas rather than on its first
# row. With the yard cut into the canvas the sails ran together into one tall
# trapezoid per mast; with a row of timber between them the ship is three masts
# of three sails, which is what it is.
#
# Half widths are set by the gaps, not by the sails. Mizzen reaches x=18, main
# starts at x=21; main reaches x=33, fore starts at x=35. Those two or three
# pixels of sky are the whole reason three masts read as three masts rather than
# as one bush, and they are the first thing to check after moving anything.
#
# The mizzen is a row shorter than the fore and its sails sit a row lower. They
# were identical, which made the ship symmetrical and took the bow off her.
MASTS = [
    (13, 3, [(5, 6, 7, 2.5, 3.5), (8, 9, 10, 3.5, 4.0), (11, 12, 14, 4.0, 5.0)]),  # mizzen
    (27, 0, [(2, 3, 5, 3.0, 4.0), (6, 7, 9, 4.5, 5.0), (10, 11, 13, 5.0, 6.0)]),   # main
    (41, 2, [(4, 5, 6, 2.5, 3.5), (7, 8, 9, 4.0, 4.5), (10, 11, 13, 4.5, 5.5)]),   # fore
]

# The hull, one (row, left, right) per pixel row. She tumbles home: widest at the
# gunwale and narrowing to the keel. Four rows, not five. Five made her a slab
# with a rig balanced on it; a ship of the line shows far more canvas than timber
# and the silhouette has to say so.
HULL = [
    (15, 4, 52),
    (16, 4, 52),
    (17, 6, 50),
    (18, 10, 46),
]

# The headsails, as one triangle on the bowsprit. Three jibs at this size is
# three ragged pixels, and a jib drawn back to the foremast merges with the fore
# course into one mass, which is what took the bow off the first version. Set
# forward of the foremast it reads as headsails and leaves the gap intact.
JIB = ((51, 6), (59, 11), (51, 14))

GUNPORTS = (16, 8, 48, 3)  # row, first x, last x, spacing
BOWSPRIT = ((51, 15), (59, 11))  # from the bow, up and out
PENNANT = (0, 28, 32)  # row, and the run of it, streaming from the main truck
SEA = 19


def blank():
    return [["."] * W for _ in range(H)]


def put(g, x, y, ch):
    if 0 <= x < W and 0 <= y < H:
        g[y][x] = ch


def draw_sail(g, cx, yard, top, bottom, hw_top, hw_bottom):
    """One square sail: a trapezoid of canvas with its yard on the row above."""
    span = max(bottom - top, 1)
    for y in range(top, bottom + 1):
        hw = hw_top + (hw_bottom - hw_top) * (y - top) / span
        for x in range(int(round(cx - hw)), int(round(cx + hw)) + 1):
            put(g, x, y, "#")
    # The yard overhangs the canvas under it by a pixel, which is what says
    # "spar" rather than "the sail has a hard edge". A pixel and a half ran the
    # main and fore course yards into one bar straight across the ship.
    for x in range(int(round(cx - hw_top - 1.0)), int(round(cx + hw_top + 1.0)) + 1):
        put(g, x, yard, "|")


def draw_line(g, a, b, ch):
    (x0, y0), (x1, y1) = a, b
    steps = max(abs(x1 - x0), abs(y1 - y0))
    for i in range(steps + 1):
        t = i / steps
        put(g, int(round(x0 + (x1 - x0) * t)), int(round(y0 + (y1 - y0) * t)), ch)


def draw_triangle(g, a, b, c, ch):
    """Filled triangle, by barycentric sign. Used for the headsails."""
    xs, ys = [p[0] for p in (a, b, c)], [p[1] for p in (a, b, c)]

    def side(p, q, x, y):
        return (q[0] - p[0]) * (y - p[1]) - (q[1] - p[1]) * (x - p[0])

    for y in range(min(ys), max(ys) + 1):
        for x in range(min(xs), max(xs) + 1):
            d1, d2, d3 = side(a, b, x, y), side(b, c, x, y), side(c, a, x, y)
            neg = d1 < 0 or d2 < 0 or d3 < 0
            pos = d1 > 0 or d2 > 0 or d3 > 0
            if not (neg and pos):
                put(g, x, y, ch)


def build():
    g = blank()

    for x in range(W):
        put(g, x, SEA, "~")

    for row, left, right in HULL:
        for x in range(left, right + 1):
            put(g, x, row, "%")

    row, first, last, step = GUNPORTS
    for x in range(first, last + 1, step):
        put(g, x, row, "*")

    draw_line(g, *BOWSPRIT, "|")
    draw_triangle(g, *JIB, "#")
    draw_line(g, *BOWSPRIT, "|")  # the bowsprit runs in front of its own jibs

    for cx, truck, sails in MASTS:
        for y in range(truck, DECK):
            put(g, cx, y, "|")
        for sail in sails:
            draw_sail(g, cx, *sail)

    row, first, last = PENNANT
    for x in range(first, last + 1):
        put(g, x, row, "|")

    return ["".join(r) for r in g]


def tone(hi, lo, i, n):
    """Step i of n along a material's gradient, as a ramp digit."""
    if hi == lo:
        t = hi
    else:
        t = hi - (hi - lo) * i / max(n - 1, SOFT)
    return str(max(0, min(RAMP_LEN - 1, int(round(t)))))


def shade(art):
    """Materials to tone digits, lit from the upper left.

    Each maximal run of one material is graded across its own width, so a sail
    is round and a spar turns away from the light. Materials in VERTICAL are
    graded down their column instead. A material whose two tones are equal comes
    out flat, which is what the gunports and the sea want.
    """
    cells = [list("." * W) for _ in range(H)]

    for y, row in enumerate(art):
        x = 0
        while x < W:
            ch = row[x]
            if ch == "." or ch in VERTICAL:
                x += 1
                continue
            end = x
            while end < W and row[end] == ch:
                end += 1
            hi, lo = TONE[ch]
            for i in range(end - x):
                cells[y][x + i] = tone(hi, lo, i, end - x)
            x = end

    # Graded against the material's own top and bottom row across the whole
    # image, not per column. Per column, a column carrying a gunport has one less
    # row of hull in it and its gradient slips by a step, which banded the
    # planking into vertical stripes under the ports.
    for ch in VERTICAL:
        hi, lo = TONE[ch]
        ys = [y for y in range(H) if ch in art[y]]
        if not ys:
            continue
        top, span = ys[0], len(ys)
        for y in ys:
            for x in range(W):
                if art[y][x] == ch:
                    cells[y][x] = tone(hi, lo, y - top, span)

    return ["".join(r) for r in cells]


def preview(mat, art):
    """Materials, then tones, then half blocks, to look at before shipping."""
    print("\n".join(mat))
    print()
    print("\n".join(art))
    print()
    ramp = " .:-=+*#@"
    for cy in range(H // 2):
        top, bottom = art[cy * 2], art[cy * 2 + 1]
        line = ""
        for cx in range(W):
            t = 0 if top[cx] == "." else int(top[cx]) + 1
            b = 0 if bottom[cx] == "." else int(bottom[cx]) + 1
            line += ramp[max(t, b)]
        print(line)


def main():
    art = build()
    assert len(art) % 2 == 0, "%d pixel rows; cells are two rows tall" % len(art)
    if "--preview" in sys.argv:
        preview(art, shade(art))
        return

    print('''// Code generated by assets/ship/generate.py. DO NOT EDIT.

package render

// shipW is the ship's width in cells. The pixel rows below are one digit per
// pixel: '.' is transparent, '0' to '%d' index shipRamp. Two pixel rows to a
// cell, the same encoding the monument uses.
const shipW = %d

// shipRamp is the water she sails in and the night she sails through.
var shipRamp = &%s

// shipArt is a first-rate under full sail, bow to the right, on the "England
// expects" screen. Transparent where there is no ship, so she flies against the
// terminal's own background rather than inside a black box.
var shipArt = []string{''' % (RAMP_LEN - 1, W, RAMP))
    for r in shade(art):
        print('\t"%s",' % r)
    print("}")


if __name__ == "__main__":
    main()
