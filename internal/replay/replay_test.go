package replay

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	r := NewRecording(12345, "blocks", "english")
	r.Record(10, 'S')
	r.Record(10, 'O')
	r.Record(42, '.')
	r.Record(7, 'Z')

	path := filepath.Join(t.TempDir(), "run.replay")
	if err := r.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Seed != 12345 || got.Font != "blocks" || got.Pack != "english" {
		t.Errorf("header lost: %+v", got)
	}
	if string(got.At(10)) != "SO" {
		t.Errorf("frame 10 keys %q, want SO", string(got.At(10)))
	}
	if string(got.At(42)) != "." {
		t.Errorf("frame 42 keys %q, want a full stop", string(got.At(42)))
	}
	if got.Frames != 42 {
		t.Errorf("frames %d, want 42", got.Frames)
	}
}

// TestFormatIsReadable. A bug report you can inspect without a tool is a bug
// report people actually send.
func TestFormatIsReadable(t *testing.T) {
	r := NewRecording(7, "braille", "argument")
	r.Record(3, 'A')
	var buf bytes.Buffer
	if err := r.Write(&buf); err != nil {
		t.Fatal(err)
	}
	text := buf.String()
	for _, want := range []string{"nelson-replay 1", "seed 7", "font braille", "3 A"} {
		if !strings.Contains(text, want) {
			t.Errorf("the file does not contain %q:\n%s", want, text)
		}
	}
}

// TestFramesAreOrdered so a diff between two recordings is meaningful.
func TestFramesAreOrdered(t *testing.T) {
	r := NewRecording(1, "blocks", "english")
	for _, f := range []int{90, 4, 51, 12} {
		r.Record(f, 'X')
	}
	var buf bytes.Buffer
	r.Write(&buf)
	var last int
	for _, line := range strings.Split(buf.String(), "\n") {
		var f int
		if n, _ := fmt.Sscanf(line, "%d ", &f); n == 1 {
			if f < last {
				t.Errorf("frame %d came after %d", f, last)
			}
			last = f
		}
	}
}

// TestVersionMismatchRefuses. A replay that silently diverges from the recording
// is worse than one that will not run: it sends whoever is debugging after a
// phantom.
func TestVersionMismatchRefuses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.replay")
	body := "nelson-replay 99\nseed 1\nfont blocks\npack english\nframes 0\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("a future format version was accepted")
	}
	if !strings.Contains(err.Error(), "diverge") {
		t.Errorf("the error does not explain the risk: %v", err)
	}
}

func TestGarbageIsRejected(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"empty.replay":  "",
		"prose.replay":  "this is not a replay\n",
		"badseed.repl":  "nelson-replay 1\nseed banana\n",
		"badframe.repl": "nelson-replay 1\nseed 1\nwibble A\n",
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}
