# Intro art

Four images, all public domain or CC0. `generate.py` turns them into
`internal/render/plates_data.go`.

| File | Subject | Source | Rights |
|---|---|---|---|
| `westminster-abbey.jpg` | West Front of Westminster Abbey, John Bluck, aquatint | Yale Center for British Art, B1977.14.16428 | CC0 |
| `johnson-reynolds.jpg` | Samuel Johnson, Joshua Reynolds | Tate, via Wikimedia Commons | Public domain (Reynolds d. 1792) |
| `lambeth-palace.jpg` | Lambeth Palace from the Thames, Samuel Wale | Yale Center for British Art, B1986.29.251 | CC0 |
| `nelson.jpg` | Rear-Admiral Sir Horatio Nelson | Royal Museums Greenwich, BHC2890 | Public domain |

Nothing but the generated Go source is compiled in. The game decodes no images
at runtime and links no image packages, for the same reason the music is note
data rather than recordings. The jpegs are kept here so the bitmaps can be
regenerated, not so the game can read them.

    python3 -m venv venv && ./venv/bin/pip install pillow
    ./venv/bin/python generate.py > ../../internal/render/plates_data.go

## What the settings in ART are for

Each image carries its own crop, ramp, inversion, and tone controls, because the
sources are not alike - two aquatints, a pen-and-wash drawing and two oils - and
one set of numbers flattered none of them. Every value was arrived at by
rendering it and looking, not by theory.

The buildings are inverted. Both prints are dark stone against a bright sky,
which on a dark terminal fills the frame with lit cells and reads as a grey
wall. Inverted, the stone glows against a night sky, which is also the right
hour for the fiction.

Nelson needs the most work. The painting is dark, and left alone the uniform and
the background merge into one murk with a pale oval floating in it. Lifting the
midtones alone washed the face out; what worked was clipping harder
(`cutoff=3`), a little added contrast, and only a mild gamma lift.
