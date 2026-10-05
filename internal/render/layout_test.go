package render

import (
	"testing"

	"keyboardwarrior/internal/engine"
)

func TestLayoutSizeGate(t *testing.T) {
	tune := engine.DefaultTuning()
	cases := []struct {
		w, h     int
		tooSmall bool
	}{
		{79, 24, true},
		{80, 23, true},
		{80, 24, false},
		{200, 60, false},
	}
	for _, c := range cases {
		if got := ComputeLayout(c.w, c.h, tune).TooSmall; got != c.tooSmall {
			t.Errorf("%dx%d: TooSmall = %v, want %v", c.w, c.h, got, c.tooSmall)
		}
	}
}

// TestLayoutCaps is the point of having a maximum: a huge terminal must not give
// the player a wider field to spread words across, or scores stop comparing.
func TestLayoutCaps(t *testing.T) {
	tune := engine.DefaultTuning()
	l := ComputeLayout(300, 100, tune)
	if l.W != int(tune.MaxCols) || l.H != int(tune.MaxRows) {
		t.Errorf("playfield %dx%d, want capped at %vx%v", l.W, l.H, tune.MaxCols, tune.MaxRows)
	}
	if l.X <= 0 || l.Y <= 0 {
		t.Errorf("playfield not centred: X=%d Y=%d", l.X, l.Y)
	}
}

// TestPlayableRows pins the row budget the geometry arguments rest on.
//
// The maximum is 34 again. It was 31 while Johnson paced a band above the field,
// which cost three rows; he is in the intro now and the band went back to the
// playfield.
func TestPlayableRows(t *testing.T) {
	tune := engine.DefaultTuning()
	if got := ComputeLayout(80, 24, tune).PlayH; got != 18 {
		t.Errorf("minimum size: %d playable rows, want 18", got)
	}
	if got := ComputeLayout(120, 40, tune).PlayH; got != 34 {
		t.Errorf("maximum size: %d playable rows, want 34", got)
	}
}

// TestLayoutRowsDoNotCollide checks the chrome budget actually adds up, so the
// floor never lands on top of the play area or the status row.
func TestLayoutRowsDoNotCollide(t *testing.T) {
	tune := engine.DefaultTuning()
	for _, h := range []int{24, 30, 40, 100} {
		l := ComputeLayout(100, h, tune)
		playBottom := l.PlayY + l.PlayH - 1
		if playBottom >= l.FloorY {
			t.Errorf("h=%d: play area ends at %d, floor at %d", h, playBottom, l.FloorY)
		}
		if l.FloorY >= l.StatusY {
			t.Errorf("h=%d: floor %d, status %d", h, l.FloorY, l.StatusY)
		}
		if l.StatusY >= l.Y+l.H {
			t.Errorf("h=%d: status row %d outside box ending %d", h, l.StatusY, l.Y+l.H-1)
		}
		if l.HUDY >= l.PlayY {
			t.Errorf("h=%d: HUD %d overlaps play area starting %d", h, l.HUDY, l.PlayY)
		}
	}
}

// TestMaxConcurrent is the corrected version of a claim the first spec draft got
// wrong: six words of up to 11 letters at 80x24 is geometrically impossible.
func TestMaxConcurrent(t *testing.T) {
	tune := engine.DefaultTuning()
	if got := ComputeLayout(80, 24, tune).MaxConcurrent(); got != 4 {
		t.Errorf("minimum size: cap %d, want 4", got)
	}
	if got := ComputeLayout(120, 40, tune).MaxConcurrent(); got != 6 {
		t.Errorf("maximum size: cap %d, want 6", got)
	}
}
