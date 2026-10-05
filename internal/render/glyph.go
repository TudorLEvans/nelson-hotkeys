package render

import (
	"strings"

	"nelson/internal/engine"
)

// The block characters the game draws with. All are Unicode East Asian
// Ambiguous, which means they measure two cells wide under a CJK width setting.
// main forces the width setting off; glyph_test asserts the result.
const (
	Full  = '█'
	Upper = '▀'
	Lower = '▄'
	Blank = ' '
)

// Ramp is the explosion decay sequence, brightest first.
var Ramp = []rune{'█', '▓', '▒', '░', '·', ' '}

// brailleBit maps a dot at (row, col) inside a braille cell to its bit. A
// braille cell is 2 dots wide and 4 tall, which is four times the payload of a
// half block and is why words drawn this way are half the size.
var brailleBit = [4][2]byte{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

// Bitmap renders text to a dot grid at the tier's scale. The font itself lives in
// engine, because glyph widths decide word widths and word widths decide
// placement and collision; render only turns dots into cells.
func Bitmap(text string, tier engine.Tier) [][]bool {
	return engine.BitmapFor(text, tier.Scale())
}

// Pack turns a dot grid into terminal cells under the active metrics, with the
// grid shifted down by subOffset dots so a word can sit part way through a cell.
// Everything about vertical placement runs through this one function: sub-cell
// motion, the oversized title tier and the two drawing styles all come out of
// it, which is why there is no separate code path for any of them.
func Pack(dots [][]bool, subOffset int) [][]rune {
	if len(dots) == 0 {
		return nil
	}
	m := engine.M
	if subOffset < 0 || subOffset >= m.DotsY {
		panic("render: subOffset outside the cell")
	}
	dotW := len(dots[0])
	dotH := len(dots) + subOffset

	at := func(y, x int) bool {
		y -= subOffset
		if y < 0 || y >= len(dots) || x < 0 || x >= dotW {
			return false
		}
		return dots[y][x]
	}

	rows := (dotH + m.DotsY - 1) / m.DotsY
	cols := (dotW + m.DotsX - 1) / m.DotsX
	out := make([][]rune, rows)
	for r := 0; r < rows; r++ {
		row := make([]rune, cols)
		for c := 0; c < cols; c++ {
			row[c] = packCell(at, r*m.DotsY, c*m.DotsX, m)
		}
		out[r] = row
	}
	return out
}

func packCell(at func(y, x int) bool, y0, x0 int, m engine.Metrics) rune {
	if m.DotsX == 2 && m.DotsY == 4 {
		var b byte
		for dy := 0; dy < 4; dy++ {
			for dx := 0; dx < 2; dx++ {
				if at(y0+dy, x0+dx) {
					b |= brailleBit[dy][dx]
				}
			}
		}
		if b == 0 {
			return Blank
		}
		return rune(0x2800 + int(b))
	}
	// Block characters: one dot wide, two tall.
	top, bottom := at(y0, x0), at(y0+1, x0)
	switch {
	case top && bottom:
		return Full
	case top:
		return Upper
	case bottom:
		return Lower
	}
	return Blank
}

// Rasterize is Bitmap plus Pack, the whole path from text to cells.
func Rasterize(text string, tier engine.Tier, subOffset int) [][]rune {
	return Pack(Bitmap(text, tier), subOffset)
}

// Render is Rasterize as a string, for golden files and debugging.
func Render(text string, tier engine.Tier, subOffset int) string {
	var b strings.Builder
	for _, row := range Rasterize(text, tier, subOffset) {
		b.WriteString(string(row))
		b.WriteByte('\n')
	}
	return b.String()
}
