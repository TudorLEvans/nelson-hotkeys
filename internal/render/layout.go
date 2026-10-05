package render

import "keyboardwarrior/internal/engine"

// Layout is the resolved geometry for one terminal size. The playfield is
// clamped to a maximum and centred, so difficulty is comparable across
// displays: a 200-column terminal must not hand the player a wider field to
// spread words across.
type Layout struct {
	TooSmall bool
	TermW    int
	TermH    int

	// Outer box, absolute terminal coordinates, including the border.
	X, Y, W, H int

	// Inner playfield, absolute coordinates of the top-left play cell.
	PlayX, PlayY     int
	PlayW, PlayH     int
	HUDY, FloorY     int
	StatusY          int
	MinCols, MinRows int
}

// Row budget inside the outer box: top border, HUD, separator, play area,
// floor, status, bottom border.
const chromeRows = 6

// ComputeLayout resolves the geometry, or reports the terminal is too small.
func ComputeLayout(termW, termH int, t *engine.Tuning) Layout {
	l := Layout{
		TermW: termW, TermH: termH,
		MinCols: int(t.MinCols), MinRows: int(t.MinRows),
	}
	if termW < l.MinCols || termH < l.MinRows {
		l.TooSmall = true
		return l
	}
	l.W = min(termW, int(t.MaxCols))
	l.H = min(termH, int(t.MaxRows))
	l.X = (termW - l.W) / 2

	top := (termH - l.H) / 2
	if top < 0 {
		top = 0
	}
	l.Y = top

	l.HUDY = l.Y + 1
	l.PlayX = l.X + 1
	l.PlayY = l.Y + 3
	l.PlayW = l.W - 2
	l.PlayH = l.H - chromeRows
	l.FloorY = l.Y + l.H - 3
	l.StatusY = l.Y + l.H - 2
	return l
}

// MaxConcurrent is how many words the field can hold without overlap, derived
// from the playfield rather than asserted. A compact word plus a gap is four
// rows, so 18 playable rows give four bands.
func (l Layout) MaxConcurrent() int {
	return min(6, l.PlayH/4)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
