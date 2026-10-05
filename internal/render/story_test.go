package render

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"keyboardwarrior/internal/engine"
)

func storyScreen(t *testing.T, w, h int) (tcell.SimulationScreen, Layout) {
	t.Helper()
	prev := engine.M
	engine.SetMetrics(engine.MetricsBlocks)
	t.Cleanup(func() { engine.SetMetrics(prev) })
	scr := tcell.NewSimulationScreen("UTF-8")
	if err := scr.Init(); err != nil {
		t.Fatal(err)
	}
	scr.SetSize(w, h)
	t.Cleanup(scr.Fini)
	return scr, ComputeLayout(w, h, engine.DefaultTuning())
}

// TestIntroRunsAndEnds, and stays inside its frame while it does.
func TestIntroRunsAndEnds(t *testing.T) {
	scr, l := storyScreen(t, 80, 24)
	total := IntroSeconds()
	if total < 10 || total > 25 {
		t.Errorf("the intro runs %.1fs; it should be long enough to read and short "+
			"enough to sit through", total)
	}
	for e := 0.0; e < total; e += 0.25 {
		if !DrawIntro(scr, l, e, int(e*30)) {
			t.Fatalf("the intro ended early at %.2fs of %.2f", e, total)
		}
		scr.Show()
		assertInsideBorder(t, scr, l, e)
	}
	if DrawIntro(scr, l, total+0.1, 0) {
		t.Error("the intro never finished")
	}
}

func TestDefeatRunsAndEnds(t *testing.T) {
	scr, l := storyScreen(t, 80, 24)
	for e := 0.0; e < DefeatSeconds; e += 0.1 {
		if !DrawDefeat(scr, l, e, "PEDANTRY") {
			t.Fatalf("the defeat sequence ended early at %.2fs", e)
		}
		scr.Show()
		assertInsideBorder(t, scr, l, e)
	}
	if DrawDefeat(scr, l, DefeatSeconds, "PEDANTRY") {
		t.Error("the defeat sequence never finished")
	}
}

// TestDefeatIsShort. It sits between death and the score screen, which is
// friction by construction; brevity is half the defence and the skip is the other.
func TestDefeatIsShort(t *testing.T) {
	if DefeatSeconds > 3 {
		t.Errorf("the defeat sequence runs %.1fs; three is the outside limit", DefeatSeconds)
	}
}

func TestDefeatSaysTheLine(t *testing.T) {
	scr, l := storyScreen(t, 80, 24)
	DrawDefeat(scr, l, DefeatSeconds-0.2, "PEDANTRY")
	scr.Show()
	cells, w, h := scr.GetContents()
	var all strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := cells[y*w+x].Runes
			if len(r) == 0 || r[0] == 0 {
				all.WriteByte(' ')
				continue
			}
			all.WriteRune(r[0])
		}
	}
	if !strings.Contains(all.String(), "KISS ME, HARDY") {
		t.Error("Nelson does not get his line")
	}
}

func assertInsideBorder(t *testing.T, scr tcell.SimulationScreen, l Layout, at float64) {
	t.Helper()
	cells, w, _ := scr.GetContents()
	y := l.Y + l.H - 1
	for x := l.X; x < l.X+l.W; x++ {
		r := cells[y*w+x].Runes
		if len(r) == 0 {
			continue
		}
		switch r[0] {
		case '─', '└', '┘', ' ', 0:
		default:
			t.Fatalf("at %.2fs content spilled onto the bottom border: %q", at, r[0])
		}
	}
}

// TestNothingCollidesWithTheSkipHint. A cutscene whose exit instruction is buried
// under the animation is a cutscene people cannot leave.
func TestNothingCollidesWithTheSkipHint(t *testing.T) {
	scr, l := storyScreen(t, 80, 27)
	total := IntroSeconds()
	hintRow := l.PlayY + l.PlayH - 1
	for e := 0.0; e < total; e += 0.2 {
		scr.Clear()
		DrawIntro(scr, l, e, int(e*30))
		scr.Show()
		cells, w, _ := scr.GetContents()
		var row []rune
		for x := l.PlayX; x < l.PlayX+l.PlayW; x++ {
			r := cells[hintRow*w+x].Runes
			if len(r) == 0 || r[0] == 0 {
				row = append(row, ' ')
				continue
			}
			row = append(row, r[0])
		}
		if !strings.Contains(string(row), "any key to skip") {
			t.Fatalf("at %.1fs the skip hint is obscured: %q", e, strings.TrimSpace(string(row)))
		}
	}
}

// TestRainStaysOutOfTheSkyline. Two kinds of dense content in the same cells is
// not two things, it is noise.
func TestRainStaysOutOfTheSkyline(t *testing.T) {
	scr, l := storyScreen(t, 80, 27)
	for e := 0.6; e < DefeatSeconds; e += 0.1 {
		scr.Clear()
		DrawDefeat(scr, l, e, "PEDANTRY")
		scr.Show()
		cells, w, _ := scr.GetContents()
		// The bottom skylineRows rows of the field, less the hint row, must be
		// buildings and column only: no letters.
		for y := l.PlayY + l.PlayH - 1 - skylineRows; y < l.PlayY+l.PlayH-1; y++ {
			for x := l.PlayX; x < l.PlayX+l.PlayW; x++ {
				r := cells[y*w+x].Runes
				if len(r) == 0 || r[0] == 0 {
					continue
				}
				if r[0] >= 'A' && r[0] <= 'Z' {
					t.Fatalf("at %.1fs a word letter %q is inside the skyline at row %d",
						e, r[0], y)
				}
			}
		}
	}
}

// TestSkylineIsBuildingsNotBars. Single columns of varying height read as a bar
// chart; a city needs contiguous blocks.
func TestSkylineIsBuildingsNotBars(t *testing.T) {
	scr, l := storyScreen(t, 80, 27)
	scr.Clear()
	DrawDefeat(scr, l, 0.2, "")
	scr.Show()
	cells, w, _ := scr.GetContents()

	y := l.PlayY + l.PlayH - 3 // one row above the base
	runs, current := 0, 0
	for x := l.PlayX; x < l.PlayX+l.PlayW; x++ {
		r := cells[y*w+x].Runes
		filled := len(r) > 0 && r[0] == '█'
		if filled {
			current++
			continue
		}
		if current >= 2 {
			runs++
		}
		current = 0
	}
	if current >= 2 {
		runs++
	}
	if runs < 3 {
		t.Errorf("only %d contiguous runs of two or more cells; that is a bar chart", runs)
	}
}

// TestPortraitDataIsWellFormed guards the generated file. It is the one thing
// here nobody reads by eye, and a bad index would panic in the middle of the
// intro rather than at build time.
func TestPlateDataIsWellFormed(t *testing.T) {
	for name, art := range allPlates() {
		for i, p := range art.Tiers {
			if len(p.Pixels) != p.H*2 {
				t.Fatalf("%s tier %d: %d pixel rows for %d cell rows; a cell is two pixels tall",
					name, i, len(p.Pixels), p.H)
			}
			for y, row := range p.Pixels {
				if len(row) != p.W {
					t.Fatalf("%s tier %d row %d is %d pixels wide, want %d", name, i, y, len(row), p.W)
				}
				for x := 0; x < len(row); x++ {
					if n := int(row[x] - '0'); n < 0 || n >= len(art.Ramp) {
						t.Fatalf("%s tier %d pixel %d,%d is %q, outside the %d-tone ramp",
							name, i, x, y, row[x], len(art.Ramp))
					}
				}
			}
		}
	}
}

// TestPlateTiersAreOrderedAndProportioned. The intro takes the first tier that
// fits, so the order is load-bearing; and a pixel is roughly square, so a tier
// whose shape does not match its source crop is a stretched picture. Wide art is
// checked against a wider band because that is what it is.
func TestPlateTiersAreOrderedAndProportioned(t *testing.T) {
	for name, art := range allPlates() {
		for i, p := range art.Tiers {
			if i > 0 && (p.W >= art.Tiers[i-1].W || p.H >= art.Tiers[i-1].H) {
				t.Errorf("%s tier %d (%dx%d) is not smaller than tier %d (%dx%d); largest must come first",
					name, i, p.W, p.H, i-1, art.Tiers[i-1].W, art.Tiers[i-1].H)
			}
			aspect := float64(p.W) / float64(p.H*2)
			if art.Wide != (aspect > 1.6) {
				t.Errorf("%s tier %d has aspect %.2f but Wide is %v", name, i, aspect, art.Wide)
			}
			// Every tier of one image must be the same shape as the others.
			if want := float64(art.Tiers[0].W) / float64(art.Tiers[0].H*2); aspect < want-0.12 || aspect > want+0.12 {
				t.Errorf("%s tier %d aspect %.2f, want near %.2f like the largest",
					name, i, aspect, want)
			}
		}
	}
}

// TestEveryPlateDrawsAtTheMinimumTerminal. If one does not, an 80x24 player
// silently gets words where everyone else gets a picture, and nobody notices
// until they are on one.
func TestEveryPlateDrawsAtTheMinimumTerminal(t *testing.T) {
	scr, l := storyScreen(t, 80, 24)
	if scr.Colors() < 256 {
		t.Skipf("the simulation screen reports %d colours", scr.Colors())
	}
	for name, p := range map[string]introPlate{
		"abbey": introPlates.abbey, "johnson": introPlates.johnson,
		"lambeth": introPlates.lambeth, "nelson": introPlates.nelson,
	} {
		scr.Clear()
		if !p.draw(scr, l, 1) {
			t.Errorf("%s: no tier fits 80x24 beside its text", name)
			continue
		}
		scr.Show()
		if countArtCells(scr) == 0 {
			t.Errorf("%s: draw reported success but painted nothing", name)
		}
	}
}

// TestPlateRevealComesUpFromDark. The reveal is the only movement in a beat, so
// "it actually starts dark and ends lit" is the thing to pin.
func TestPlateRevealComesUpFromDark(t *testing.T) {
	scr, l := storyScreen(t, 120, 40)
	if scr.Colors() < 256 {
		t.Skipf("the simulation screen reports %d colours", scr.Colors())
	}
	bright := func(at float64) int {
		scr.Clear()
		introPlates.johnson.draw(scr, l, at)
		scr.Show()
		cells, w, h := scr.GetContents()
		var lit int
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				fg, _, _ := cells[y*w+x].Style.Decompose()
				if fg == rampPurple[len(rampPurple)-1] {
					lit++
				}
			}
		}
		return lit
	}
	if start, end := bright(0), bright(revealSeconds); start >= end {
		t.Errorf("the plate has %d bright cells at t=0 and %d once revealed; it should come up out of the dark",
			start, end)
	}
	if bright(revealSeconds) != bright(revealSeconds*3) {
		t.Error("the plate is still changing after the reveal should have finished")
	}
}

func allPlates() map[string]*plateArt {
	return map[string]*plateArt{
		"abbey": &plateAbbey, "johnson": &plateJohnson,
		"lambeth": &plateLambeth, "nelson": &plateNelson,
	}
}

func countArtCells(scr tcell.SimulationScreen) int {
	cells, w, h := scr.GetContents()
	var n int
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if _, bg, _ := cells[y*w+x].Style.Decompose(); bg != tcell.ColorDefault {
				n++
			}
		}
	}
	return n
}

// TestPlateFallsBackToWords. Eight tones of one hue is the least that keeps a
// subject readable, and the four ANSI greys cannot do it at any threshold, so
// below 256 colours the words alone are not a degraded plate but the only thing
// that works. The same fallback covers a playfield too small for the art.
func TestPlateFallsBackToWords(t *testing.T) {
	scr, l := storyScreen(t, 100, 34)
	narrow := l
	narrow.PlayW = 24 // too narrow for any tier beside its text
	scr.Clear()
	if introPlates.nelson.draw(scr, narrow, 1) {
		t.Fatal("a plate drew in a playfield too narrow for its smallest tier")
	}
	drawPlateText(scr, narrow, introPlates.nelson)
	scr.Show()
	if n := countArtCells(scr); n != 0 {
		t.Errorf("the fallback painted %d picture cells", n)
	}
	if !strings.Contains(rowText(scr, l.PlayY+(l.PlayH-3)/2), "ADMIRAL LORD NELSON") {
		t.Error("the fallback dropped the picture without keeping the words")
	}
}
