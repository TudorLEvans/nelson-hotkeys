#!/usr/bin/env python3
"""Pull a single melodic line out of a MIDI file and print it in the notation
internal/audio/music.go parses: "D4/4 E4/8 F4/8", where the denominator is the
note value (4 = crotchet, 8 = quaver, 4. = dotted crotchet) and R is a rest.

The game does not read MIDI. This turns a reference file in this directory into
Go source you paste into music.go, the way assets/art/generate.py emits
plates_data.go. The point is that a tune in the game should come from a score
rather than from memory.

Usage:
    generate.py FILE [--name SUB | --track N] [--voice top|bottom]
                     [--bars A:B] [--flats] [--limit N] [--grid N]

    --name SUB     use the track whose name contains SUB (case-insensitive)
    --track N      use track N (0-based, as printed by --list)
    --list         print the tracks and exit
    --voice        which note to keep when the track is chordal (default top)
    --bars A:B     keep bars A to B, 1-based, half-open
    --flats        spell black notes as Bb rather than A#
    --limit N      stop after N notes
    --grid N       quantise to 1/N notes (default 16)
    --min-rest N   fold rests shorter than N grid slots into the note before
    --transpose N  shift by N semitones

Enharmonic spelling is not something the script can decide: a MIDI file records
pitch, not notation, so --flats gives Gb where the score says F#. Fix those by
hand when you paste the result in.
"""

import argparse
import struct
import sys

SEMIS = {0: "C", 2: "D", 4: "E", 5: "F", 7: "G", 9: "A", 11: "B"}
SHARPS = ["C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"]
FLATS = ["C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "Bb", "B"]

# Durations parse() can express, counted in crotchets, which is what the
# denominator is relative to: "/4" is one crotchet. A dotted note is written
# "/4." and lasts half as long again.
DURATIONS = [
    (4.0, "1"), (3.0, "2."), (2.0, "2"), (1.5, "4."), (1.0, "4"),
    (0.75, "8."), (0.5, "8"), (0.375, "16."), (0.25, "16"), (0.125, "32"),
]


def vlq(b, i):
    v = 0
    while True:
        c = b[i]
        i += 1
        v = (v << 7) | (c & 0x7F)
        if not c & 0x80:
            return v, i


def read(path):
    """Return (division, tempo, tracks), each track a dict of name and notes.

    A note is (start_tick, end_tick, pitch). Running status is honoured because
    plenty of the hobbyist sequences in SOURCES.md rely on it.
    """
    b = open(path, "rb").read()
    if b[:4] != b"MThd":
        sys.exit(f"{path}: not a MIDI file")
    _, ntrks, div = struct.unpack(">HHH", b[8:14])
    if div & 0x8000:
        sys.exit(f"{path}: SMPTE timing is not supported")
    tempo = 500000
    tracks = []
    i = 14
    for _ in range(ntrks):
        if b[i:i + 4] != b"MTrk":
            break
        ln = struct.unpack(">I", b[i + 4:i + 8])[0]
        end, j, t, status = i + 8 + ln, i + 8, 0, 0
        name, open_notes, notes = "", {}, []
        while j < end:
            d, j = vlq(b, j)
            t += d
            if j >= end:
                break
            ev = b[j]
            if ev == 0xFF:
                typ = b[j + 1]
                ln2, j2 = vlq(b, j + 2)
                data = b[j2:j2 + ln2]
                if typ == 0x03 and not name:
                    name = data.decode("latin-1").strip()
                if typ == 0x51 and ln2 == 3 and tempo == 500000:
                    tempo = int.from_bytes(data, "big")
                j = j2 + ln2
                continue
            if ev in (0xF0, 0xF7):
                ln2, j2 = vlq(b, j + 1)
                j = j2 + ln2
                continue
            if ev & 0x80:
                status = ev
                j += 1
            kind, pitch = status & 0xF0, b[j]
            if kind == 0x90 and b[j + 1] > 0:
                open_notes.setdefault(pitch, []).append(t)
            elif kind in (0x80, 0x90):
                starts = open_notes.get(pitch)
                if starts:
                    notes.append((starts.pop(0), t, pitch))
            j += 1 if 0xC0 <= status < 0xE0 else 2
        tracks.append({"name": name, "notes": sorted(notes)})
        i = end
    return div, tempo, tracks


def pick(tracks, name, index):
    if index is not None:
        return index
    if name:
        for n, tr in enumerate(tracks):
            if name.lower() in tr["name"].lower():
                return n
    # Failing an explicit choice, the melody is usually the busiest track that
    # also sits highest, so rank on mean pitch and ignore the near-empty ones.
    scored = [(sum(p for _, _, p in tr["notes"]) / len(tr["notes"]), n)
              for n, tr in enumerate(tracks) if len(tr["notes"]) >= 8]
    if not scored:
        sys.exit("no track has enough notes to be a melody")
    return max(scored)[1]


def monophonic(notes, div, grid, voice, min_rest):
    """Flatten a chordal track to one note per grid slot, then run-length it.

    Rests shorter than min_rest slots are swallowed by the note before them.
    Sequencers write detached quavers as note-plus-gap, and taken literally that
    doubles the token count and makes the tune stutter."""
    step = max(1, div * 4 // grid)
    if not notes:
        return []
    last = max(e for _, e, _ in notes)
    slots = []
    for t in range(0, last, step):
        mid = t + step // 2
        sounding = [p for s, e, p in notes if s <= mid < e]
        slots.append((max(sounding) if voice == "top" else min(sounding))
                     if sounding else None)
    runs = []
    for p in slots:
        if runs and runs[-1][0] == p:
            runs[-1][1] += 1
        else:
            runs.append([p, 1])
    merged = []
    for p, count in runs:
        if p is None and count < min_rest and merged:
            merged[-1][1] += count
        else:
            merged.append([p, count])
    return merged


def spell(pitch, flats):
    table = FLATS if flats else SHARPS
    octave = pitch // 12 - 1
    if not 0 <= octave <= 9:
        return None
    return f"{table[pitch % 12]}{octave}"


def emit(runs, grid, flats, limit):
    """Turn runs of grid slots into tokens, splitting durations parse() cannot
    write into a tied pair rather than rounding them away."""
    out = []
    for pitch, count in runs:
        if pitch is None and not out:
            continue  # no leading rest
        beats = count / grid * 4  # grid slots are 1/grid of a semibreve
        name = "R" if pitch is None else spell(pitch, flats)
        if name is None:
            continue
        while beats > 0.01:
            for value, denom in DURATIONS:
                if value <= beats + 1e-9:
                    out.append(f"{name}/{denom}")
                    beats -= value
                    break
            else:
                break
        if limit and len(out) >= limit:
            return out[:limit]
    while out and out[-1].startswith("R"):
        out.pop()
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("file")
    ap.add_argument("--name")
    ap.add_argument("--track", type=int)
    ap.add_argument("--list", action="store_true")
    ap.add_argument("--voice", choices=["top", "bottom"], default="top")
    ap.add_argument("--bars")
    ap.add_argument("--flats", action="store_true")
    ap.add_argument("--limit", type=int, default=0)
    ap.add_argument("--grid", type=int, default=16)
    ap.add_argument("--min-rest", type=int, default=2,
                    help="rests shorter than this many grid slots are folded "
                         "into the note before them (default 2)")
    ap.add_argument("--transpose", type=int, default=0,
                    help="shift by N semitones, for sequences not in the "
                         "usual key")
    a = ap.parse_args()

    div, tempo, tracks = read(a.file)
    if a.list:
        for n, tr in enumerate(tracks):
            notes = tr["notes"]
            mean = sum(p for _, _, p in notes) / len(notes) if notes else 0
            print(f"{n}\t{len(notes):5d} notes\tmean pitch {mean:5.1f}\t{tr['name']}")
        return

    n = pick(tracks, a.name, a.track)
    notes = [(s, e, p + a.transpose) for s, e, p in tracks[n]["notes"]]
    if a.bars:
        lo, hi = (int(x) for x in a.bars.split(":"))
        # Bars are assumed to be four crotchets. Nothing in the MIDI says so,
        # and a piece in three will need the range adjusting by hand.
        start, end = (lo - 1) * div * 4, (hi - 1) * div * 4
        notes = [(s - start, e - start, p) for s, e, p in notes
                 if start <= s < end]
    runs = monophonic(notes, div, a.grid, a.voice, a.min_rest)
    tokens = emit(runs, a.grid, a.flats, a.limit)

    beat_ms = round(tempo / 1000)
    print(f"// {a.file}, track {n} ({tracks[n]['name'] or 'unnamed'}), "
          f"{len(tokens)} notes, {beat_ms}ms to the crotchet", file=sys.stderr)
    print(f'parse({beat_ms}, Square,')
    for i in range(0, len(tokens), 8):
        line = " ".join(tokens[i:i + 8])
        tail = '")' if i + 8 >= len(tokens) else '"+'
        print(f'\t"{line} '.rstrip() + tail)


if __name__ == "__main__":
    main()
