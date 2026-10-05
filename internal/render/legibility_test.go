package render

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"keyboardwarrior/internal/engine"
)

// cellsOf renders one character to the cell grid the player actually sees.
func cellsOf(ch rune) []string {
	var out []string
	for _, row := range Rasterize(string(ch), engine.TierNormal, 0) {
		out = append(out, string(row))
	}
	return out
}

// difference counts cells that differ between two rendered characters. Glyphs of
// unequal width are padded, since a width difference is itself a strong visual
// cue and should count in their favour.
func difference(a, b []string) int {
	diff := 0
	rows := len(a)
	if len(b) > rows {
		rows = len(b)
	}
	for y := 0; y < rows; y++ {
		var ra, rb []rune
		if y < len(a) {
			ra = []rune(a[y])
		}
		if y < len(b) {
			rb = []rune(b[y])
		}
		w := len(ra)
		if len(rb) > w {
			w = len(rb)
		}
		for x := 0; x < w; x++ {
			ca, cb := ' ', ' '
			if x < len(ra) {
				ca = ra[x]
			}
			if x < len(rb) {
				cb = rb[x]
			}
			if ca != cb {
				diff++
			}
		}
	}
	return diff
}

// TestLettersAreTellableApart is the test that should have existed from the
// start. An exact-equality check passed happily while H, M and W rendered as
// three variants of "two verticals plus some middle ink" and were, in play,
// impossible to distinguish. Equality is the wrong bar: the requirement is that
// a player can see which letter is falling, because they have to know which key
// to press.
//
// Every pair of letters must differ in at least minDiff drawn cells.
func TestLettersAreTellableApart(t *testing.T) {
	useBlocks(t)
	const minDiff = 3

	letters := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	rendered := map[rune][]string{}
	for _, ch := range letters {
		rendered[ch] = cellsOf(ch)
	}

	type pair struct {
		a, b rune
		d    int
	}
	var worst []pair
	for i := 0; i < len(letters); i++ {
		for j := i + 1; j < len(letters); j++ {
			a, b := letters[i], letters[j]
			worst = append(worst, pair{a, b, difference(rendered[a], rendered[b])})
		}
	}
	sort.Slice(worst, func(i, j int) bool { return worst[i].d < worst[j].d })

	for _, p := range worst {
		if p.d >= minDiff {
			break
		}
		t.Errorf("%q and %q differ in only %d cells, want at least %d:\n%s",
			p.a, p.b, p.d, minDiff, sideBySide(rendered[p.a], rendered[p.b], p.a, p.b))
	}

	var b strings.Builder
	for _, p := range worst[:6] {
		fmt.Fprintf(&b, " %q/%q:%d", p.a, p.b, p.d)
	}
	t.Logf("closest pairs:%s", b.String())
}

// TestDigitsAndLettersAreTellableApart: digits share the field with letters once
// power-up markers and numbers appear, so O against 0 matters as much as any
// letter pair.
func TestDigitsAndLettersAreTellableApart(t *testing.T) {
	useBlocks(t)
	const minDiff = 3
	for _, d := range "0123456789" {
		for _, l := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			if got := difference(cellsOf(d), cellsOf(l)); got < minDiff {
				t.Errorf("%q and %q differ in only %d cells, want at least %d:\n%s",
					d, l, got, minDiff, sideBySide(cellsOf(d), cellsOf(l), d, l))
			}
		}
	}
}

// TestHMWSpecifically pins the reported bug so it cannot come back quietly.
func TestHMWSpecifically(t *testing.T) {
	useBlocks(t)
	for _, p := range [][2]rune{{'H', 'M'}, {'H', 'W'}, {'M', 'W'}} {
		if got := difference(cellsOf(p[0]), cellsOf(p[1])); got < 4 {
			t.Errorf("%q and %q differ in only %d cells; these three were reported as "+
				"indistinguishable in play and need more separation than the minimum",
				p[0], p[1], got)
		}
	}
}

// TestProportionalWidthsAreDeliberate documents which letters are wider and why,
// so nobody flattens the font back to a fixed width and reintroduces the H/M/W
// collision.
func TestProportionalWidthsAreDeliberate(t *testing.T) {
	want := map[rune]int{'M': 5, 'W': 5, 'T': 5, 'I': 3}
	for _, ch := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		w := len(engine.GlyphSource()[ch][0])
		exp := 4
		if v, ok := want[ch]; ok {
			exp = v
		}
		if w != exp {
			t.Errorf("%q is %d dots wide, want %d", ch, w, exp)
		}
	}
}

func sideBySide(a, b []string, ca, cb rune) string {
	var out strings.Builder
	rows := len(a)
	if len(b) > rows {
		rows = len(b)
	}
	fmt.Fprintf(&out, "    %-8c %c\n", ca, cb)
	for y := 0; y < rows; y++ {
		la, lb := "", ""
		if y < len(a) {
			la = a[y]
		}
		if y < len(b) {
			lb = b[y]
		}
		fmt.Fprintf(&out, "    %-8s %s\n", la, lb)
	}
	return out.String()
}

// punctuation is every non-alphanumeric glyph the font carries.
const punctuation = ".,-/;:'=()[]!?+*~#@&%"

// TestPunctuationIsTellableApart uses a lower bar than letters do, and that is
// deliberate rather than lax. Punctuation carries little ink by design, so the
// absolute cell difference between two marks is always small; they are separated
// by silhouette and by width instead. Two cells is the point at which two marks
// read differently in a three-row cell.
func TestPunctuationIsTellableApart(t *testing.T) {
	useBlocks(t)
	const minDiff = 2
	runes := []rune(punctuation)
	for i := 0; i < len(runes); i++ {
		for j := i + 1; j < len(runes); j++ {
			a, b := runes[i], runes[j]
			if got := difference(cellsOf(a), cellsOf(b)); got < minDiff {
				t.Errorf("%q and %q differ in only %d cells, want at least %d:\n%s",
					a, b, got, minDiff, sideBySide(cellsOf(a), cellsOf(b), a, b))
			}
		}
	}
}

// TestPunctuationNotConfusableWithLetters: a power-up suffix must never be
// mistaken for a letter, or the player types the wrong key and feeds nothing.
func TestPunctuationNotConfusableWithLetters(t *testing.T) {
	useBlocks(t)
	const minDiff = 3
	for _, p := range punctuation {
		for _, l := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" {
			if got := difference(cellsOf(p), cellsOf(l)); got < minDiff {
				t.Errorf("%q and %q differ in only %d cells, want at least %d",
					p, l, got, minDiff)
			}
		}
	}
}

// TestEveryPowerupSuffixIsDrawable derives its list from the actual power set,
// which is the whole point. The previous version hardcoded the marks and went
// stale the moment two power-ups were added: both rendered their suffix as the
// fallback '?', so the game was showing the player a character that did not exist
// and could not be typed.
//
// Any test that restates data instead of reading it will eventually lie.
func TestEveryPowerupSuffixIsDrawable(t *testing.T) {
	for _, p := range engine.Powers {
		ch := rune(p)
		if !engine.HasGlyph(ch) {
			t.Errorf("%v uses suffix %q, which the font cannot draw; it will render "+
				"as '?' and the player cannot know what to press", p, ch)
		}
	}
}

// TestPowerupSuffixesAreNotTheFallback is the same failure caught from the other
// side: a glyph that exists but is identical to '?' is just as useless.
func TestPowerupSuffixesAreNotTheFallback(t *testing.T) {
	useBlocks(t)
	fallback := cellsOf('?')
	for _, p := range engine.Powers {
		ch := rune(p)
		if difference(cellsOf(ch), fallback) < 3 {
			t.Errorf("%v suffix %q is indistinguishable from the fallback '?'", p, ch)
		}
	}
}

// TestPowerupSuffixesAreLegibleAgainstEachOther. A player has to tell one mark
// from another at a glance while the word is falling, and a misread here is a
// wrong keypress rather than a moment's confusion: press the wrong mark and it
// is a typo, which is the one way a bomb can be fed. ',' and '.' failed this in
// spirit if not by the cell count, which is part of why SLOW went.
func TestPowerupSuffixesAreLegibleAgainstEachOther(t *testing.T) {
	useBlocks(t)
	const minDiff = 3
	for i, a := range engine.Powers {
		for _, b := range engine.Powers[i+1:] {
			ca, cb := rune(a), rune(b)
			if got := difference(cellsOf(ca), cellsOf(cb)); got < minDiff {
				t.Errorf("%v (%q) and %v (%q) differ in only %d cells:\n%s",
					a, ca, b, cb, got, sideBySide(cellsOf(ca), cellsOf(cb), ca, cb))
			}
		}
	}
}

// TestStemsAreOneDotWide. A letter whose stem is two dots thick reads as a slab
// rather than a stroke, which is what T looked like at four dots wide: the only
// way to centre a stem in an even width is to double it or offset it, and both
// look wrong. The letters with a single central stem are all odd-width for this
// reason.
func TestStemsAreOneDotWide(t *testing.T) {
	for _, ch := range []rune{'T', 'I'} {
		rows := engine.GlyphSource()[ch]
		w := len(rows[0])
		if w%2 == 0 {
			t.Errorf("%q is %d dots wide; a centred single stem needs an odd width", ch, w)
		}
		// The rows below the top bar must each carry exactly one lit dot.
		for y := 1; y < len(rows)-1; y++ {
			n := 0
			for _, c := range rows[y] {
				if c == '#' {
					n++
				}
			}
			if n != 1 {
				t.Errorf("%q row %d has %d lit dots, want a single-dot stem", ch, y, n)
			}
		}
	}
}
