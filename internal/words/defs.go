package words

import (
	"bufio"
	"compress/gzip"
	"embed"
	"fmt"
	"strings"
	"sync"
)

// english.tsv.gz is WORD<TAB>pos<TAB>definition, one line per word, built from
// WordNet 3.1 intersected with the system dictionary.
//
// Two sources because each fixes the other's problem. The system dictionary is
// Webster's Second (1934) and carries 175,000 headwords, but most of them are
// Victorian obscurities: a random sample reads HALLAGE, PEDULE, PERPERA,
// SURDELINE. Fine as typing practice, useless for learning a word. WordNet is
// modern and every lemma has a gloss, but includes forms the dictionary rejects.
// The intersection is 47,541 words that are real, current, and defined.
//
// It also means the word list and the definitions cannot drift apart: they are
// the same file, so every spawnable word has something to show.
//
// WordNet 3.1 is Princeton University's, used under its BSD-style licence. See
// LICENSE-WORDNET.
//
//go:embed english.tsv.gz
var defsFS embed.FS

// Definition is one entry.
type Definition struct {
	POS  string // n, v, adj, adv
	Text string
}

var (
	defsOnce sync.Once
	defsMap  map[string]Definition
	defsList []string
	defsErr  error
)

// loadDefs parses the embedded table once. Roughly 47k lines out of a megabyte of
// gzip, so it is worth doing off the frame path; see Preload.
func loadDefs() {
	f, err := defsFS.Open("english.tsv.gz")
	if err != nil {
		defsErr = err
		return
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		defsErr = err
		return
	}
	defer zr.Close()

	defsMap = make(map[string]Definition, 48000)
	defsList = make([]string, 0, 48000)
	sc := bufio.NewScanner(zr)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		word, rest, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		pos, text, _ := strings.Cut(rest, "\t")
		defsMap[word] = Definition{POS: pos, Text: text}
		defsList = append(defsList, word)
	}
	defsErr = sc.Err()
}

// Preload parses the table in the background, so the first definition lookup does
// not cost a frame. Safe to call more than once.
func Preload() {
	go func() { defsOnce.Do(loadDefs) }()
}

// Lookup returns a definition, if the word has one. Any pack's words can be
// looked up, not only the English pack: a curated word that WordNet happens to
// know still gets a gloss.
func Lookup(word string) (Definition, bool) {
	defsOnce.Do(loadDefs)
	d, ok := defsMap[word]
	return d, ok
}

// englishPack is the word list, taken from the definition table so the two cannot
// disagree.
func englishPack() ([]string, error) {
	defsOnce.Do(loadDefs)
	if defsErr != nil {
		return nil, fmt.Errorf("english pack: %w", defsErr)
	}
	out := make([]string, len(defsList))
	copy(out, defsList)
	return out, nil
}
