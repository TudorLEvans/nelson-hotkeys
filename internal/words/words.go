// Package words holds the vocabulary packs. They are embedded, so one binary
// file is the whole game.
package words

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"sort"
	"strings"
)

//go:embed *.txt
var packs embed.FS

// markers are the characters a word file may contain beyond A-Z and 0-9.
//
// Power-up punctuation is deliberately NOT here. Suffixes are appended by the
// spawner, so a trailing '.' in a pack file is a typo rather than an intent, and
// silently accepting it would put an unearned power-up character on an ordinary
// word.
const markers = "!?+*~"

// Packs lists the pack names.
func Packs() []string {
	entries, err := packs.ReadDir(".")
	if err != nil {
		panic(err)
	}
	out := make([]string, 0, len(entries)+1)
	for _, e := range entries {
		out = append(out, strings.TrimSuffix(e.Name(), ".txt"))
	}
	out = append(out, "english")
	sort.Strings(out)
	return out
}

// Load returns a pack by name. A name containing a path separator or ending in
// .txt is read from disk instead, so custom packs work without a rebuild.
func Load(name string) ([]string, error) {
	if strings.ContainsRune(name, os.PathSeparator) || strings.HasSuffix(name, ".txt") {
		f, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return parse(f, name)
	}
	if name == "english" {
		return englishPack()
	}
	f, err := packs.Open(name + ".txt")
	if err != nil {
		return nil, fmt.Errorf("no word pack %q (have %v)", name, Packs())
	}
	defer f.Close()
	return parse(f, name)
}

type reader interface{ Read([]byte) (int, error) }

func parse(r reader, name string) ([]string, error) {
	var out []string
	sc := bufio.NewScanner(r)
	for line := 1; sc.Scan(); line++ {
		s := strings.TrimSpace(sc.Text())
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		if strings.ContainsAny(s, " \t") {
			return nil, fmt.Errorf("%s:%d: %q contains whitespace; words are single tokens", name, line, s)
		}
		// Restrict to the renderable single-byte set. This is not fussiness: it
		// is the invariant that lets Word.Typed be a plain byte index into the
		// word, and it turns an unrenderable character into a startup error
		// rather than a '?' appearing mid-game in a word nobody can type.
		for i := 0; i < len(s); i++ {
			c := s[i]
			ok := (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
				strings.IndexByte(markers, c) >= 0
			if !ok {
				return nil, fmt.Errorf("%s:%d: %q contains %q; allowed are A-Z, 0-9 and %q",
					name, line, s, rune(c), markers)
			}
		}
		out = append(out, s)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: no words", name)
	}
	return out, nil
}
