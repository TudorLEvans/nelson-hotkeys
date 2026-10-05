package words

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEmbeddedPacksLoad is the test that catches a bad word list at build time
// instead of at play time. A word with a character the font cannot draw becomes
// a '?' on screen, and a '?' is untypeable, so the player loses a life to a
// typo in a text file.
func TestEmbeddedPacksLoad(t *testing.T) {
	names := Packs()
	if len(names) == 0 {
		t.Fatal("no embedded word packs")
	}
	for _, name := range names {
		list, err := Load(name)
		if err != nil {
			t.Errorf("pack %q: %v", name, err)
			continue
		}
		t.Logf("pack %q: %d words", name, len(list))
		for _, w := range list {
			if len(w) < 2 {
				t.Errorf("pack %q: %q is too short to be a target", name, w)
			}
			if len(w) > 12 {
				t.Errorf("pack %q: %q is %d letters; the widest word that fits "+
					"the minimum playfield is 12", name, w, len(w))
			}
		}
	}
}

// TestPackWordsAreSingleByte pins the invariant Word.Typed relies on: it is a
// byte index into the word, which is only correct while every character is one
// byte.
func TestPackWordsAreSingleByte(t *testing.T) {
	for _, name := range Packs() {
		list, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range list {
			if len([]rune(w)) != len(w) {
				t.Errorf("pack %q: %q is multi-byte; Word.Typed would index into "+
					"the middle of a character", name, w)
			}
		}
	}
}

func TestLoadRejects(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"lowercase":    "ratio\n",
		"whitespace":   "BAD FAITH\n",
		"unrenderable": "CAFÉ\n",
		"empty":        "# only a comment\n",
		"punctuation":  "WHAT.\n",
	}
	for name, body := range cases {
		path := filepath.Join(dir, name+".txt")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Errorf("%s: %q was accepted", name, strings.TrimSpace(body))
		}
	}
}

func TestLoadAcceptsMarkers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.txt")
	os.WriteFile(path, []byte("FREEZE!\nNUKE!!\nREPAIR+\nFRENZY*\nSLOW~\nMYSTERY?\n"), 0o644)
	list, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 6 {
		t.Errorf("got %d words, want 6", len(list))
	}
}

func TestLoadUnknownPack(t *testing.T) {
	if _, err := Load("nosuchpack"); err == nil {
		t.Error("unknown pack name accepted")
	}
}

// TestNoDuplicates matters more than it looks. Repeats in the file make the same
// word more likely to be drawn, which is exactly the "same words again and again"
// problem the spawner's no-repeat history exists to solve.
func TestNoDuplicates(t *testing.T) {
	for _, name := range Packs() {
		list, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for _, w := range list {
			if seen[w] {
				t.Errorf("pack %q: %q appears more than once", name, w)
			}
			seen[w] = true
		}
	}
}

// TestLengthCoverage checks the pack can actually feed the difficulty schedule.
// Word length grows over time, so a pack clustered in the middle would leave the
// early game repeating a handful of short words and the late game a handful of
// long ones.
func TestLengthCoverage(t *testing.T) {
	for _, name := range Packs() {
		list, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		byLen := map[int]int{}
		for _, w := range list {
			byLen[len(w)]++
		}
		for n := 3; n <= 12; n++ {
			if byLen[n] < 8 {
				t.Errorf("pack %q has only %d words of %d letters; want at least 8 "+
					"so the spawner is not forced to repeat", name, byLen[n], n)
			}
		}
		t.Logf("pack %q: %d words, lengths %v", name, len(list), byLen)
	}
}
