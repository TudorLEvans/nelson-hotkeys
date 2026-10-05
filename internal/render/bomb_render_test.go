package render

import (
	"testing"

	"nelson/internal/engine"
)

// TestBombBracketMovesWithItsLetters is the regression test for a bug that only
// showed up in motion. The brackets were positioned by cell row, so they ticked
// once per whole cell while the letters moved every dot via the sub-cell offset.
// The two slid against each other as the word fell.
//
// The fix was to put the brackets in the dot bitmap so one Pack call positions
// everything. This checks the property directly: across a full cell of sub-cell
// positions, the gap between the top bracket and the first glyph row never
// changes.
func TestBombBracketMovesWithItsLetters(t *testing.T) {
	useBlocks(t)
	l := ComputeLayout(80, 24, goldenTuning())

	var gaps []int
	for dot := 0; dot < 4*engine.M.DotsY; dot++ {
		scr := simScreen(t, 80, 24)
		st := engine.NewState(l.PlayW, l.PlayH, goldenTuning(), engine.NewRNG(1), []string{"X"})
		st.Words = append(st.Words, engine.Word{
			Text: "BOGUS", Tier: engine.TierNormal, Bomb: true,
			DotX: 20 * engine.M.DotsX, DotY: 8*engine.M.DotsY + dot,
		})
		Compose(scr, st, l, nil, nil)
		scr.Show()

		cells, w, h := scr.GetContents()
		top, firstGlyph := -1, -1
		for y := 0; y < h && firstGlyph < 0; y++ {
			for x := 0; x < w; x++ {
				r := cells[y*w+x].Runes
				if len(r) == 0 || r[0] == ' ' || r[0] == 0 {
					continue
				}
				if x < l.PlayX || x >= l.PlayX+l.PlayW || y < l.PlayY {
					continue
				}
				if top < 0 {
					top = y
				} else if y > top {
					firstGlyph = y
					break
				}
			}
		}
		if top < 0 || firstGlyph < 0 {
			continue
		}
		gaps = append(gaps, firstGlyph-top)
	}

	if len(gaps) < 4 {
		t.Fatalf("only %d positions produced ink", len(gaps))
	}
	for i, g := range gaps {
		if g != gaps[0] {
			t.Errorf("at sub-position %d the bracket sits %d rows from the letters, "+
				"but %d at position 0; the bracket is not moving with the word",
				i, g, gaps[0])
		}
	}
}

// TestBombFuseGrowsAsItIsEaten: the top edge is solid over letters already
// destroyed and dashed over the rest, so accumulated danger reads without
// counting letters, and as a shape rather than only a colour.
func TestBombFuseGrowsAsItIsEaten(t *testing.T) {
	useBlocks(t)
	w := engine.Word{Text: "BOGUS", Tier: engine.TierNormal, Bomb: true}

	solidAt := func(typed int) int {
		w.Typed = typed
		dots, _, _ := bombFrame(&w, engine.BitmapFor(w.Remaining(), 1))
		n := 0
		for x := 0; x < len(dots[0])-1; x++ {
			if dots[0][x] && dots[0][x+1] {
				n++ // two adjacent lit dots means solid, not dashed
			}
		}
		return n
	}

	none, some, most := solidAt(0), solidAt(2), solidAt(4)
	t.Logf("solid fuse runs: 0 letters eaten %d, 2 eaten %d, 4 eaten %d", none, some, most)
	if !(none < some && some < most) {
		t.Errorf("fuse did not grow with letters eaten: %d, %d, %d", none, some, most)
	}
}

// TestBombKeepsItsFootprint: the bracket spans the whole original word, not just
// what is left, so the hazard does not appear to shrink as it is fed.
func TestBombKeepsItsFootprint(t *testing.T) {
	useBlocks(t)
	w := engine.Word{Text: "BOGUS", Tier: engine.TierNormal, Bomb: true}
	full := engine.TextWidthDots("BOGUS")
	for typed := 0; typed < 5; typed++ {
		w.Typed = typed
		dots, _, dotX := bombFrame(&w, engine.BitmapFor(w.Remaining(), 1))
		if len(dots[0]) != full {
			t.Errorf("%d eaten: frame is %d dots wide, want %d", typed, len(dots[0]), full)
		}
		if dotX != w.DotX {
			t.Errorf("%d eaten: frame moved to %d", typed, dotX)
		}
	}
}
