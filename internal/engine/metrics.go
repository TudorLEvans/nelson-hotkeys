package engine

// Everything geometric in this game is measured in glyph DOTS, not terminal
// cells, because how many dots fit in a cell depends on which characters are
// used to draw them:
//
//	blocks  █ ▀ ▄        1 dot wide, 2 dots tall per cell
//	braille ⠿            2 dots wide, 4 dots tall per cell
//
// A cell is about twice as tall as it is wide, so both of these give roughly
// square dots. Braille packs four times as many, which makes words smaller and
// quarters the vertical movement step.
//
// Dots are the engine's unit so that switching between them changes only how the
// same world is drawn, never how it behaves. Cell conversion happens in render.

// Metrics is the dot-to-cell mapping of the active drawing style. It is a
// package variable rather than plumbing because it is a display constant chosen
// once at startup, before any State exists. Tests set it directly.
type Metrics struct {
	Name  string
	DotsX int // dots per cell horizontally
	DotsY int // dots per cell vertically
}

var (
	MetricsBlocks  = Metrics{Name: "blocks", DotsX: 1, DotsY: 2}
	MetricsBraille = Metrics{Name: "braille", DotsX: 2, DotsY: 4}
)

// M is the active metrics. The binary sets this from --font, which defaults to
// braille: the finer movement step and the smaller words are what the game is
// tuned around, and blocks stays available for a terminal or a font where
// braille renders badly. Blocks is the value here only so that a test which
// never calls SetMetrics gets the simpler geometry.
var M = MetricsBlocks

// SetMetrics selects the drawing style. Call once, before building a State.
func SetMetrics(m Metrics) { M = m }

// MetricsByName resolves a --font value.
func MetricsByName(name string) (Metrics, bool) {
	switch name {
	case "braille":
		return MetricsBraille, true
	case "blocks":
		return MetricsBlocks, true
	}
	return M, false
}

// ceilDiv divides rounding up, for positive divisors.
func ceilDiv(n, d int) int {
	if n <= 0 {
		return 0
	}
	return (n + d - 1) / d
}

// floorDiv divides rounding toward negative infinity, which Go's / does not do
// for negative operands. Words sit at negative positions while entering the
// field, so this matters: a truncating divide put them a row too low.
func floorDiv(n, d int) int {
	q := n / d
	if n%d != 0 && (n < 0) != (d < 0) {
		q--
	}
	return q
}

// TextWidth is the rendered width of text in cells, under the active metrics.
func TextWidth(text string) int { return ceilDiv(TextWidthDots(text), M.DotsX) }

// GlyphHeightDots is the vertical extent of a word in dots.
const GlyphHeightDots = GlyphRows

// HeightCells is how many terminal rows a word covers when its top sits on a
// cell boundary. A sub-cell offset can add one.
func HeightCells() int { return ceilDiv(GlyphHeightDots, M.DotsY) }

// CellsForGlyph is the width of one glyph in cells, used for the next-letter
// highlight and the contested marker. Proportional font, so it depends on which
// letter.
func CellsForGlyph(ch rune) int { return ceilDiv(GlyphFor(ch).W, M.DotsX) }

// FloorDivX converts a horizontal dot offset to cells.
func FloorDivX(dots int) int { return floorDiv(dots, M.DotsX) }

// FloorDivY converts a vertical dot position to a cell row, rounding toward
// negative infinity so words entering from above land correctly.
func FloorDivY(dots int) int { return floorDiv(dots, M.DotsY) }
