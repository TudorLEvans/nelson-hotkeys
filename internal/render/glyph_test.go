package render

import (
	"strings"
	"testing"

	"github.com/rivo/uniseg"
	"keyboardwarrior/internal/engine"
)

// TestGlyphWidths is the regression test for the East Asian ambiguous width
// bug. Every block character the game draws is Unicode Ambiguous, so under a
// CJK width setting it measures two cells: words would render double-width, the
// playfield geometry would be wrong and half-row motion would break.
//
// The game forces uniseg.EastAsianAmbiguousWidth = 1 at startup. This test
// asserts the forced value produces width 1, and documents that the unforced
// value does not, so nobody "simplifies" the force away.
func TestGlyphWidths(t *testing.T) {
	drawn := []rune{Full, Upper, Lower}
	drawn = append(drawn, Ramp...)
	drawn = append(drawn, '▔')                // the contested-letter marker
	drawn = append(drawn, LifeFull, LifeGone) // HUD lives

	orig := uniseg.EastAsianAmbiguousWidth
	defer func() { uniseg.EastAsianAmbiguousWidth = orig }()

	uniseg.EastAsianAmbiguousWidth = 1
	for _, r := range drawn {
		if w := uniseg.StringWidth(string(r)); w != 1 {
			t.Errorf("U+%04X %q: width %d, want 1 with the force applied", r, r, w)
		}
	}

	// Prove the hazard is real rather than theoretical.
	uniseg.EastAsianAmbiguousWidth = 2
	ambiguous := 0
	for _, r := range drawn {
		if uniseg.StringWidth(string(r)) == 2 {
			ambiguous++
		}
	}
	if ambiguous == 0 {
		t.Fatal("no glyph is East Asian Ambiguous any more; the startup force may " +
			"now be unnecessary, but verify against the current uniseg tables before removing it")
	}
	t.Logf("%d of %d glyphs are East Asian Ambiguous; the startup force is load-bearing",
		ambiguous, len(drawn))
}

func TestFontComplete(t *testing.T) {
	for ch := 'A'; ch <= 'Z'; ch++ {
		if !engine.HasGlyph(ch) {
			t.Errorf("missing glyph for %q", ch)
		}
	}
	for ch := '0'; ch <= '9'; ch++ {
		if !engine.HasGlyph(ch) {
			t.Errorf("missing glyph for %q", ch)
		}
	}
	for _, ch := range "!?+*~ " {
		if !engine.HasGlyph(ch) {
			t.Errorf("missing glyph for %q", ch)
		}
	}
}

func TestFontGlyphsWellFormed(t *testing.T) {
	for ch, rows := range engine.GlyphSource() {
		if len(rows) != engine.GlyphRows {
			t.Errorf("glyph %q: %d rows, want %d", ch, len(rows), engine.GlyphRows)
		}
		w := len(rows[0])
		for y, row := range rows {
			if len(row) != w {
				t.Errorf("glyph %q row %d: %d wide, want %d (ragged glyph)", ch, y, len(row), w)
			}
			if n := strings.Count(row, "#") + strings.Count(row, "."); n != len(row) {
				t.Errorf("glyph %q row %d: characters other than # and .", ch, y)
			}
		}
	}
}

// TestFontGlyphsDistinct guards against a copy-paste slip making two letters
// identical, which would be invisible in play but make a word untypeable by
// sight.
// TestFontGlyphsDistinct catches exact duplicates. It is the weak version of the
// requirement; see legibility_test.go for the check that actually matters.
func TestFontGlyphsDistinct(t *testing.T) {
	seen := map[string]rune{}
	for ch, rows := range engine.GlyphSource() {
		if ch == ' ' {
			continue
		}
		key := strings.Join(rows, "/")
		if other, dup := seen[key]; dup {
			t.Errorf("glyphs %q and %q are identical", ch, other)
		}
		seen[key] = ch
	}
}

func TestTextWidth(t *testing.T) {
	prev := engine.M
	defer engine.SetMetrics(prev)

	// Dot widths are the same whichever style draws them; cell widths are not.
	engine.SetMetrics(engine.MetricsBraille)
	braille := []struct {
		text string
		want int
	}{
		{"", 0}, {"A", 2}, {"AB", 5}, {"ACTUALLY", 20},
	}
	for _, c := range braille {
		if got := engine.TextWidth(c.text); got != c.want {
			t.Errorf("braille TextWidth(%q) = %d, want %d", c.text, got, c.want)
		}
	}

	engine.SetMetrics(engine.MetricsBlocks)
	blocks := []struct {
		text string
		want int
	}{
		{"", 0}, {"A", 4}, {"AB", 9}, {"ACTUALLY", 40},
	}
	for _, c := range blocks {
		if got := engine.TextWidth(c.text); got != c.want {
			t.Errorf("blocks TextWidth(%q) = %d, want %d", c.text, got, c.want)
		}
	}
}

// TestBrailleHalvesTheFootprint is the measurement that justified the second
// drawing style: the same glyphs, packed four to a cell instead of two.
func TestBrailleHalvesTheFootprint(t *testing.T) {
	prev := engine.M
	defer engine.SetMetrics(prev)

	engine.SetMetrics(engine.MetricsBlocks)
	bw, bh := engine.TextWidth("SOURCE"), len(Rasterize("SOURCE", engine.TierNormal, 0))
	engine.SetMetrics(engine.MetricsBraille)
	rw, rh := engine.TextWidth("SOURCE"), len(Rasterize("SOURCE", engine.TierNormal, 0))

	t.Logf("blocks %dx%d cells, braille %dx%d cells", bw, bh, rw, rh)
	if rw*rh*2 > bw*bh {
		t.Errorf("braille uses %d cells against blocks' %d; expected roughly half",
			rw*rh, bw*bh)
	}
}

func TestRasterizeDimensions(t *testing.T) {
	prev := engine.M
	defer engine.SetMetrics(prev)

	cases := []struct {
		metrics   engine.Metrics
		tier      engine.Tier
		subOffset int
		wantRows  int
		wantCols  int
	}{
		// A 3x6 glyph: braille packs it 2x4 to a cell, blocks 1x2.
		// A 3x6 glyph: braille packs it 2x4 to a cell, blocks 1x2.
		// "A" is 4 dots wide, 6 tall.
		{engine.MetricsBraille, engine.TierNormal, 0, 2, 2},
		{engine.MetricsBraille, engine.TierNormal, 3, 3, 2},
		{engine.MetricsBraille, engine.TierHuge, 0, 3, 4},
		{engine.MetricsBlocks, engine.TierNormal, 0, 3, 4},
		{engine.MetricsBlocks, engine.TierNormal, 1, 4, 4},
		{engine.MetricsBlocks, engine.TierHuge, 0, 6, 8},
	}
	for _, c := range cases {
		engine.SetMetrics(c.metrics)
		grid := Rasterize("A", c.tier, c.subOffset)
		if len(grid) != c.wantRows {
			t.Errorf("%s %v offset %d: %d rows, want %d", c.metrics.Name, c.tier,
				c.subOffset, len(grid), c.wantRows)
		}
		if len(grid) > 0 && len(grid[0]) != c.wantCols {
			t.Errorf("%s %v offset %d: %d cols, want %d", c.metrics.Name, c.tier,
				c.subOffset, len(grid[0]), c.wantCols)
		}
	}
}

// TestBlockRasterUsesOnlyBlockRunes: with the block style, a glyph on a cell
// boundary must be full blocks, never halves. Halves there would mean the
// vertical scale is wrong.
func TestBlockRasterUsesOnlyBlockRunes(t *testing.T) {
	prev := engine.M
	defer engine.SetMetrics(prev)
	engine.SetMetrics(engine.MetricsBlocks)
	for _, row := range Rasterize("H", engine.TierHuge, 0) {
		for x, ch := range row {
			if ch != Full && ch != Blank {
				t.Fatalf("col %d: got %q, want %q or blank", x, ch, Full)
			}
		}
	}
}

// TestBrailleRasterIsBraille checks the braille packer emits braille and nothing
// else, so a stray block character cannot creep into a braille frame and break
// the alignment.
func TestBrailleRasterIsBraille(t *testing.T) {
	prev := engine.M
	defer engine.SetMetrics(prev)
	engine.SetMetrics(engine.MetricsBraille)
	for _, row := range Rasterize("SOURCE", engine.TierNormal, 0) {
		for x, ch := range row {
			if ch == Blank {
				continue
			}
			if ch < 0x2800 || ch > 0x28FF {
				t.Fatalf("col %d: got U+%04X, want a braille rune", x, ch)
			}
		}
	}
}

// TestSubOffsetShifts is the core of sub-cell motion: shifting by one dot must
// move the ink, or a word only ever appears on whole-cell boundaries and its fall
// is a slideshow.
func TestSubOffsetShifts(t *testing.T) {
	prev := engine.M
	defer engine.SetMetrics(prev)
	for _, m := range []engine.Metrics{engine.MetricsBraille, engine.MetricsBlocks} {
		engine.SetMetrics(m)
		seen := map[string]bool{}
		for off := 0; off < m.DotsY; off++ {
			key := Render("H", engine.TierNormal, off)
			if seen[key] {
				t.Errorf("%s: offset %d renders identically to an earlier offset; "+
					"sub-cell motion is not being used", m.Name, off)
			}
			seen[key] = true
		}
		if len(seen) != m.DotsY {
			t.Errorf("%s: %d distinct sub-positions, want %d", m.Name, len(seen), m.DotsY)
		}
	}
}

func TestRasterizeUnknownRuneFallsBack(t *testing.T) {
	got := Render("é", engine.TierNormal, 0)
	want := Render("?", engine.TierNormal, 0)
	if got != want {
		t.Error("unknown rune did not fall back to '?'")
	}
}

func TestRasterizeRejectsBadOffset(t *testing.T) {
	prev := engine.M
	defer engine.SetMetrics(prev)
	engine.SetMetrics(engine.MetricsBraille)
	defer func() {
		if recover() == nil {
			t.Error("an offset outside the cell should panic")
		}
	}()
	Rasterize("A", engine.TierNormal, engine.M.DotsY)
}
