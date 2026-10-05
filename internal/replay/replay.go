// Package replay records and plays back a run.
//
// The point is reproducing a bug, not watching a demo. A player who hits
// something odd can send a file that is a few kilobytes of text, and it will
// produce the identical run on another machine.
//
// That works only because of decisions taken much earlier: the simulation reads a
// fixed timestep rather than a measured one, positions are integers, and the PRNG
// is hand-rolled with a pinned sequence rather than the standard library's. A run
// is therefore a pure function of (seed, tuning, keys-by-frame), and this file is
// just that triple written down.
package replay

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Recording is a full run: what it started from and every key, by frame.
type Recording struct {
	Seed    uint64
	Font    string
	Pack    string
	Keys    map[int][]rune // frame -> keys pressed that frame
	Frames  int
	Version int
}

// FormatVersion guards against a recording made by a build whose rules differ.
// A replay that silently diverges is worse than one that refuses to run.
const FormatVersion = 1

// NewRecording starts a fresh recording.
func NewRecording(seed uint64, font, pack string) *Recording {
	return &Recording{
		Seed: seed, Font: font, Pack: pack,
		Keys: map[int][]rune{}, Version: FormatVersion,
	}
}

// Record notes a key pressed on a frame.
func (r *Recording) Record(frame int, key rune) {
	r.Keys[frame] = append(r.Keys[frame], key)
	if frame > r.Frames {
		r.Frames = frame
	}
}

// At returns the keys for a frame.
func (r *Recording) At(frame int) []rune { return r.Keys[frame] }

// Write serialises the recording. The format is plain text and deliberately
// readable: a bug report you can inspect without a tool is a bug report people
// actually send.
func (r *Recording) Write(w io.Writer) error {
	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, "nelson-replay %d\n", r.Version)
	fmt.Fprintf(bw, "seed %d\n", r.Seed)
	fmt.Fprintf(bw, "font %s\n", r.Font)
	fmt.Fprintf(bw, "pack %s\n", r.Pack)
	fmt.Fprintf(bw, "frames %d\n", r.Frames)

	frames := make([]int, 0, len(r.Keys))
	for f := range r.Keys {
		frames = append(frames, f)
	}
	sort.Ints(frames)
	for _, f := range frames {
		fmt.Fprintf(bw, "%d %s\n", f, string(r.Keys[f]))
	}
	return bw.Flush()
}

// Save writes the recording to a file.
func (r *Recording) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := r.Write(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Load reads a recording back.
func Load(path string) (*Recording, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := &Recording{Keys: map[int][]rune{}}
	sc := bufio.NewScanner(f)
	line := 0
	for sc.Scan() {
		line++
		text := sc.Text()
		if text == "" {
			continue
		}
		head, rest, _ := strings.Cut(text, " ")
		switch head {
		case "nelson-replay", "keyboardwarrior-replay": // the latter from before the rename
			v, err := strconv.Atoi(rest)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: bad version %q", path, line, rest)
			}
			if v != FormatVersion {
				return nil, fmt.Errorf(
					"%s: recorded by format version %d, this build speaks %d; "+
						"the rules have changed and the replay would diverge",
					path, v, FormatVersion)
			}
			r.Version = v
		case "seed":
			r.Seed, err = strconv.ParseUint(rest, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: bad seed", path, line)
			}
		case "font":
			r.Font = rest
		case "pack":
			r.Pack = rest
		case "frames":
			r.Frames, _ = strconv.Atoi(rest)
		default:
			frame, err := strconv.Atoi(head)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: expected a frame number, got %q", path, line, head)
			}
			r.Keys[frame] = []rune(rest)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if r.Version == 0 {
		return nil, fmt.Errorf("%s: not a replay file", path)
	}
	return r, nil
}
