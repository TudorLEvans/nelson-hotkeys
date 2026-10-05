package engine

import "fmt"

// The font lives in engine, not render, because glyph widths decide how wide a
// word is, and word width decides placement, collision and targeting. Render
// only packs these dots into terminal cells.
//
// The font is PROPORTIONAL: 3 dots for I and T, 5 for M and W, 4 for everything
// else. That is not a flourish, it is what legibility costs.
//
// A fixed 3-dot font was tried and does not work. With one middle column, M and W
// can only be "two verticals plus some middle ink", which is also what H is, and
// the three were reported as indistinguishable in play:
//
//	H = █ █    M = █▄█    W = █ █
//	    █▄█        █▀█        █▄█
//	    █ █        █ █        █▀█
//
// Widening H, M and W alone was not enough either. A pairwise check over the
// whole alphabet then found K against X differing by a single cell, P against R
// by one, and A/R, H/K, H/U, U/V, I/T and J/V by two. Three dots cannot separate
// 26 capitals, whatever care goes into the individual shapes.
//
// A player who cannot tell which letter is falling cannot know which key to
// press, so this is a correctness requirement, not a matter of taste. It is
// enforced by TestLettersAreTellableApart rather than left to judgement. Keeping
// the narrow letters narrow is what makes the wide ones affordable.
//
// glyphSrc rows are given as one string per pixel row; a glyph's width is the
// length of its rows, and every row of a glyph must be the same length.
var glyphSrc = map[rune][]string{
	// Four dots wide is the working width: three cannot separate 26 capitals.
	'A': {".##.", "#..#", "#..#", "####", "#..#", "#..#"},
	'B': {"###.", "#..#", "###.", "#..#", "#..#", "###."},
	'C': {".###", "#...", "#...", "#...", "#...", ".###"},
	'D': {"##..", "#.#.", "#..#", "#..#", "#.#.", "##.."},
	'E': {"####", "#...", "###.", "#...", "#...", "####"},
	'F': {"####", "#...", "###.", "#...", "#...", "#..."},
	'G': {".###", "#...", "#.##", "#..#", "#..#", ".###"},
	'H': {"#..#", "#..#", "#..#", "####", "#..#", "#..#"},
	'J': {"..##", "...#", "...#", "...#", "#..#", ".##."},
	'K': {"#..#", "#.#.", "##..", "##..", "#.#.", "#..#"},
	'L': {"#...", "#...", "#...", "#...", "#...", "####"},
	'N': {"#..#", "##.#", "#.##", "#..#", "#..#", "#..#"},
	'O': {".##.", "#..#", "#..#", "#..#", "#..#", ".##."},
	'P': {"###.", "#..#", "#..#", "###.", "#...", "#..."},
	'Q': {".##.", "#..#", "#..#", "#..#", ".##.", "..##"},
	'R': {"###.", "#..#", "###.", "#.#.", "#..#", "#..#"},
	'S': {".###", "#...", ".##.", "...#", "...#", "###."},
	// U ends flat, V tapers. They differ nowhere else, so this is load-bearing.
	'U': {"#..#", "#..#", "#..#", "#..#", "#..#", "####"},
	'V': {"#..#", "#..#", "#..#", "#..#", ".##.", ".##."},
	'X': {"#..#", "#..#", ".##.", ".##.", "#..#", "#..#"},
	'Y': {"#..#", "#..#", ".##.", "..#.", "..#.", "..#."},
	'Z': {"####", "...#", "..#.", ".#..", "#...", "####"},

	// Five dots wide, so the diagonals actually fit. At four or fewer, M and W
	// can only be "two verticals plus middle ink", which is also what H is, and
	// the three were reported as indistinguishable in play.
	'M': {"#...#", "##.##", "#.#.#", "#...#", "#...#", "#...#"},
	'W': {"#...#", "#...#", "#.#.#", "#.#.#", "##.##", "#...#"},
	// T joins them at five dots so its trunk can be a single column and centred.
	// At four it had to be either two dots thick, which read as a slab, or one dot
	// off-centre, which read as a mistake. An odd width is the only way to centre
	// a one-dot stem.
	'T': {"#####", "..#..", "..#..", "..#..", "..#..", "..#.."},

	// Three dots wide: the one genuinely narrow letter. Keeping it narrow is what
	// makes the five-dot letters affordable.
	'I': {"###", ".#.", ".#.", ".#.", ".#.", "###"},

	'0': {".##.", "#.##", "#.##", "##.#", "##.#", ".##."},
	'1': {".#..", "##..", ".#..", ".#..", ".#..", ".#.."},
	'2': {".##.", "#..#", "...#", "..#.", ".#..", "####"},
	'3': {"###.", "...#", ".###", "...#", "...#", "###."},
	'4': {"..##", ".#.#", "#..#", "####", "...#", "...#"},
	'5': {"####", "#...", "##..", "..#.", "...#", "###."},
	'6': {".##.", "#...", "###.", "#..#", "#..#", ".##."},
	'7': {"####", "...#", "..#.", ".#..", ".#..", ".#.."},
	'8': {".##.", "#..#", ".##.", "#..#", "#..#", ".##."},
	'9': {".##.", "#..#", "#..#", ".###", "...#", "###."},

	// Punctuation. These are typeable suffixes, not signage, which is a change
	// from the first design: that made markers display-only because '!' costs a
	// shift and reaching for shift mid-chain breaks the flow the scoring rewards.
	// But most punctuation is UNSHIFTED on a Latin layout, so a word ending in
	// '.' or ',' or '-' costs exactly one ordinary keystroke. Only the unshifted
	// set is used for power-ups; the rest exist so real words with apostrophes and
	// hyphens can be drawn.
	//
	// Deliberately narrow. Punctuation carries little ink, so it is separated by
	// silhouette and width rather than by detail.
	// The three low marks are power-up suffixes and are drawn heavier and further
	// apart than typography would want. As punctuation in a sentence a period and
	// a comma differing by one dot is fine; as the label on an effect the player
	// has to identify while it falls, it is not.
	'.':  {"..", "..", "..", "..", "##", "##"},
	',':  {"..", "..", "..", "##", "##", ".#"},
	'-':  {"...", "...", "###", "...", "...", "..."},
	'/':  {"...", "..#", "..#", ".#.", "#..", "#.."},
	';':  {"..", "##", "..", "..", "##", ".#"},
	':':  {"..", "##", "..", "##", "..", ".."},
	'\'': {".#", ".#", "..", "..", "..", ".."},
	'=':  {"...", "###", "...", "###", "...", "..."},
	'(':  {".#", "#.", "#.", "#.", "#.", ".#"},
	')':  {"#.", ".#", ".#", ".#", ".#", "#."},
	// Square brackets are power-up suffixes, so they are drawn heavy rather than
	// as hairlines. A thin bracket at this size is a couple of lit dots in a
	// column and reads as noise; the player has to know which key to press.
	'[': {"###", "##.", "##.", "##.", "##.", "###"},
	']': {"###", ".##", ".##", ".##", ".##", "###"},
	'!': {"#", "#", "#", ".", ".", "#"},
	'?': {"###", "..#", ".##", ".#.", "...", ".#."},
	'+': {"...", ".#.", ".#.", "###", ".#.", ".#."},
	'*': {"...", "#.#", ".#.", "###", ".#.", "#.#"},
	'~': {"....", "....", ".##.", "..##", "....", "...."},
	'#': {".#.#", "####", ".#.#", "####", ".#.#", "...."},
	'@': {".##.", "#..#", "#.##", "####", "#...", ".##."},
	'&': {".##.", "#..#", ".#..", "#.#.", "#..#", ".###"},
	'%': {"#..#", "#.#.", "..#.", ".#..", ".#.#", "#..#"},
	' ': {"...", "...", "...", "...", "...", "..."},
}

// Glyph is a variable-width dot bitmap, indexed [row][col].
type Glyph struct {
	W    int
	Dots [][]bool
}

var glyphs map[rune]Glyph

const (
	// GlyphRows is the height of every glyph, in dots.
	GlyphRows = 6
	// KernDots is the blank space between adjacent glyphs, in dots.
	KernDots = 1
	// FallbackGlyph stands in for anything the font cannot draw.
	FallbackGlyph = '?'
)

func init() {
	glyphs = make(map[rune]Glyph, len(glyphSrc))
	for ch, rows := range glyphSrc {
		if len(rows) != GlyphRows {
			panic(fmt.Sprintf("engine: glyph %q has %d rows, want %d", ch, len(rows), GlyphRows))
		}
		w := len(rows[0])
		g := Glyph{W: w, Dots: make([][]bool, GlyphRows)}
		for y, row := range rows {
			if len(row) != w {
				panic(fmt.Sprintf("engine: glyph %q row %d is %d wide, want %d", ch, y, len(row), w))
			}
			g.Dots[y] = make([]bool, w)
			for x := 0; x < w; x++ {
				switch row[x] {
				case '#':
					g.Dots[y][x] = true
				case '.':
				default:
					panic(fmt.Sprintf("engine: glyph %q: want '#' or '.', got %q", ch, row[x]))
				}
			}
		}
		glyphs[ch] = g
	}
}

// GlyphFor returns a glyph, falling back for anything unmapped.
func GlyphFor(ch rune) Glyph {
	if g, ok := glyphs[ch]; ok {
		return g
	}
	return glyphs[FallbackGlyph]
}

// HasGlyph reports whether a rune can be drawn.
func HasGlyph(ch rune) bool { _, ok := glyphs[ch]; return ok }

// GlyphRunes lists every drawable rune. Used by tests.
func GlyphRunes() []rune {
	out := make([]rune, 0, len(glyphs))
	for ch := range glyphs {
		out = append(out, ch)
	}
	return out
}

// GlyphSource exposes the raw rows for tests that check the table itself.
func GlyphSource() map[rune][]string { return glyphSrc }

// AdvanceDotsFor is the horizontal step from one letter to the next.
func AdvanceDotsFor(ch rune) int { return GlyphFor(ch).W + KernDots }

// TextWidthDots is the rendered width of text in dots.
func TextWidthDots(text string) int {
	w := 0
	for _, ch := range text {
		w += AdvanceDotsFor(ch)
	}
	if w > 0 {
		w -= KernDots // no trailing kern
	}
	return w
}

// LetterOffsetDots is where letter i starts, in dots from the word's left edge.
// Because the font is proportional this cannot be i times a constant, and getting
// it wrong would put explosions and the next-letter highlight on the wrong letter.
func LetterOffsetDots(text string, i int) int {
	off := 0
	for n, ch := range text {
		if n >= i {
			break
		}
		off += AdvanceDotsFor(ch)
	}
	return off
}

// BitmapFor renders text to a dot grid at the given scale.
func BitmapFor(text string, scale int) [][]bool {
	if scale < 1 {
		scale = 1
	}
	w := TextWidthDots(text) * scale
	h := GlyphRows * scale
	if w <= 0 {
		return nil
	}
	grid := make([][]bool, h)
	for i := range grid {
		grid[i] = make([]bool, w)
	}
	for i, ch := range []rune(text) {
		g := GlyphFor(ch)
		originX := LetterOffsetDots(text, i) * scale
		for py := 0; py < GlyphRows; py++ {
			for px := 0; px < g.W; px++ {
				if !g.Dots[py][px] {
					continue
				}
				for sy := 0; sy < scale; sy++ {
					for sx := 0; sx < scale; sx++ {
						x, y := originX+px*scale+sx, py*scale+sy
						if x < w && y < h {
							grid[y][x] = true
						}
					}
				}
			}
		}
	}
	return grid
}
