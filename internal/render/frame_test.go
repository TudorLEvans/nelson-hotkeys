package render

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"keyboardwarrior/internal/engine"
)

var update = flag.Bool("update", false, "rewrite golden files")

// dump renders a SimulationScreen to text. tcell's simulation backend is why the
// whole renderer is testable with no terminal at all.
func dump(scr tcell.SimulationScreen) string {
	cells, w, h := scr.GetContents()
	var b strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			runes := cells[y*w+x].Runes
			if len(runes) == 0 || runes[0] == 0 {
				b.WriteByte(' ')
				continue
			}
			b.WriteRune(runes[0])
		}
		b.WriteString("|\n") // pin trailing space so it survives editors
	}
	return b.String()
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".txt")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log("wrote", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run: go test ./... -update)", err)
	}
	if got != string(want) {
		t.Errorf("output differs from %s\n--- got ---\n%s", path, got)
	}
}

func simScreen(t *testing.T, w, h int) tcell.SimulationScreen {
	t.Helper()
	scr := tcell.NewSimulationScreen("UTF-8")
	if err := scr.Init(); err != nil {
		t.Fatal(err)
	}
	scr.SetSize(w, h)
	t.Cleanup(scr.Fini)
	return scr
}

// goldenTuning is pinned rather than taken from DefaultTuning so that changing a
// balance number does not rewrite every golden file. Goldens exist to catch
// layout and font regressions, not tuning edits.
func goldenTuning() *engine.Tuning {
	tune := engine.DefaultTuning()
	tune.FallSpeed = 1.0
	tune.SpeedScale = 1.0
	tune.Lives = 3
	return tune
}

// goldens are rendered in braille, the default drawing style. Tests that want the
// block style set it themselves.
// useBlocks pins the drawing style so a change of default does not churn every
// golden file and expectation.
func useBlocks(t *testing.T) {
	t.Helper()
	prev := engine.M
	engine.SetMetrics(engine.MetricsBlocks)
	t.Cleanup(func() { engine.SetMetrics(prev) })
}

// offsetWord sits part way through a cell, to exercise sub-cell placement.
func offsetWord(text string, col, row, subDots int) engine.Word {
	w := dotWord(text, col, row, 0, 0)
	w.DotY += subDots
	return w
}

// shiftedWord sits one letter advance to the right of col.
func shiftedWord(text string, col, row int) engine.Word {
	w := dotWord(text, col, row, 0, 0)
	w.DotX += engine.AdvanceDotsFor(rune(text[0]))
	return w
}

// dotWord builds a word at a cell column and row, in dot units.
func dotWord(text string, col, row int, typed, kick int) engine.Word {
	return engine.Word{
		Text: text, Typed: typed, Tier: engine.TierNormal,
		DotX: col * engine.M.DotsX, DotY: row * engine.M.DotsY,
		Kick: kick,
	}
}

func fixedState(t *testing.T, l Layout, w engine.Word) *engine.State {
	t.Helper()
	tune := goldenTuning()
	st := engine.NewState(l.PlayW, l.PlayH, tune, engine.NewRNG(1), []string{w.Text})
	st.Words = append(st.Words, w)
	return st
}

func TestGoldenShortWord(t *testing.T) {
	useBlocks(t)
	scr := simScreen(t, 80, 24)
	l := ComputeLayout(80, 24, goldenTuning())
	st := fixedState(t, l, dotWord("RATIO", 20, 4, 0, 0))
	Compose(scr, st, l, nil, nil)
	scr.Show()
	golden(t, "word_short", dump(scr))
}

func TestGoldenLongWord(t *testing.T) {
	useBlocks(t)
	scr := simScreen(t, 80, 24)
	l := ComputeLayout(80, 24, goldenTuning())
	st := fixedState(t, l, dotWord("WHATABOUT", 8, 6, 0, 0))
	Compose(scr, st, l, nil, nil)
	scr.Show()
	golden(t, "word_long", dump(scr))
}

// TestGoldenHalfRowOffset is the visual proof of smooth motion: the same word at
// y and y+0.5 must differ, and the difference must be half-blocks.
func TestGoldenHalfRowOffset(t *testing.T) {
	useBlocks(t)
	scr := simScreen(t, 80, 24)
	l := ComputeLayout(80, 24, goldenTuning())
	st := fixedState(t, l, offsetWord("RATIO", 20, 4, 1))
	Compose(scr, st, l, nil, nil)
	scr.Show()
	golden(t, "word_halfoffset", dump(scr))
}

func TestGoldenTooSmall(t *testing.T) {
	useBlocks(t)
	scr := simScreen(t, 60, 20)
	l := ComputeLayout(60, 20, goldenTuning())
	if !l.TooSmall {
		t.Fatal("60x20 should be too small")
	}
	DrawTooSmall(scr, l)
	scr.Show()
	golden(t, "too_small", dump(scr))
}

// TestTypedPrefixDoesNotReflow is a rule the whole targeting model depends on:
// destroying a letter must not move the letters that remain, or the player's eye
// and fingers have to re-track mid-word.
//
// The check is a whole-screen comparison rather than a leftmost-ink probe. An
// earlier version of this test measured the first inked column, which moves by
// 7 rather than 6 between "RATIO" and "ATIO" purely because R has ink in the
// left pixel column of its top row and A does not. That measured glyph shape,
// not layout.
func TestTypedPrefixDoesNotReflow(t *testing.T) {
	useBlocks(t)
	l := ComputeLayout(80, 24, goldenTuning())

	// "RATIO" with R eaten must be pixel-identical to "ATIO" drawn one letter
	// advance to the right.
	eaten := dotWord("RATIO", 20, 4, 1, 0)
	shifted := shiftedWord("ATIO", 20, 4)

	shot := func(w engine.Word) string {
		scr := simScreen(t, 80, 24)
		st := fixedState(t, l, w)
		Compose(scr, st, l, nil, nil)
		scr.Show()
		return dump(scr)
	}

	if got, want := shot(eaten), shot(shifted); got != want {
		t.Errorf("an eaten prefix reflowed the remaining letters\n--- got ---\n%s", got)
	}
}

// TestGoldenFieldOfWords is the density check the font change was for. Four
// words at 3 rows each fit the 80x24 minimum with room between them; at the
// original 6-rows-per-word they did not.
func TestGoldenFieldOfWords(t *testing.T) {
	useBlocks(t)
	scr := simScreen(t, 80, 24)
	l := ComputeLayout(80, 24, goldenTuning())
	tune := goldenTuning()
	st := engine.NewState(l.PlayW, l.PlayH, tune, engine.NewRNG(1), []string{"X"})
	st.Words = append(st.Words,
		dotWord("RATIO", 2, 0, 0, 0),
		dotWord("SOURCE", 30, 4, 2, 2),
		dotWord("WHATABOUT", 6, 8, 0, 5),
		dotWord("PEDANT", 50, 12, 0, 1),
	)
	Compose(scr, st, l, nil, nil)
	scr.Show()
	golden(t, "field_of_words", dump(scr))
}

// TestGoldenGameOver shows the killer word in the huge tier, which is the only
// place one pixel per full row is used.
func TestGoldenGameOver(t *testing.T) {
	useBlocks(t)
	scr := simScreen(t, 80, 24)
	l := ComputeLayout(80, 24, goldenTuning())
	tune := goldenTuning()
	st := engine.NewState(l.PlayW, l.PlayH, tune, engine.NewRNG(1), []string{"X"})
	st.GameOver = true
	st.Killer = "PEDANT"
	st.Lives = 0
	st.Score = 1284
	st.Hits, st.Keys, st.WordsDestroyed, st.BestStreak, st.BestChain = 240, 260, 31, 47, 4
	st.Elapsed = 113
	Compose(scr, st, l, nil, nil)
	scr.Show()
	golden(t, "game_over", dump(scr))
}

// TestContestedMarker is the cue that replaces the lock. Two words waiting on the
// same letter must show a marker under exactly one of them: the one that will
// actually receive the keypress.
func TestContestedMarker(t *testing.T) {
	useBlocks(t)
	words := []engine.Word{
		dotWord("SOURCE", 2, 2, 0, 0),
		dotWord("STRAWMAN", 40, 9, 0, 0),
		dotWord("PEDANT", 20, 5, 0, 0),
	}
	marked := contestedLetters(words)
	if len(marked) != 1 {
		t.Fatalf("marked %d words, want exactly 1", len(marked))
	}
	if !marked[1] {
		t.Errorf("marked %v, want the lower of the two S words (index 1)", marked)
	}
}

func TestNoMarkerWhenUncontested(t *testing.T) {
	useBlocks(t)
	words := []engine.Word{
		dotWord("SOURCE", 2, 2, 0, 0),
		dotWord("PEDANT", 40, 9, 0, 0),
	}
	if got := contestedLetters(words); len(got) != 0 {
		t.Errorf("marked %v, want nothing: no letter is contested", got)
	}
}

// TestPowerupWordNamesItself. The mark used to be the only thing saying what a
// power-up did, and a mark teaches nobody: there was no point on screen where
// '/' was ever connected to BLAST. The name rides above the word, with the key
// to press, so the set does not have to be memorised between runs.
func TestPowerupWordNamesItself(t *testing.T) {
	useBlocks(t)
	scr := simScreen(t, 80, 24)
	l := ComputeLayout(80, 24, goldenTuning())

	w := dotWord("SOURCE/", 20, 6, 0, 0)
	w.Power = engine.PowerBlast
	st := fixedState(t, l, w)
	Compose(scr, st, l, nil, nil)
	scr.Show()

	out := dump(scr)
	if !strings.Contains(out, "BLAST /") {
		t.Errorf("no name on the power-up word; screen was:\n%s", out)
	}

	// And only on a power-up. An ordinary word gets no label, or the cue would
	// carry no information.
	scr2 := simScreen(t, 80, 24)
	plain := fixedState(t, l, dotWord("SOURCE", 20, 6, 0, 0))
	Compose(scr2, plain, l, nil, nil)
	scr2.Show()
	if strings.Contains(dump(scr2), "BLAST") {
		t.Error("an ordinary word was labelled")
	}
}

// TestInstantPowerNamesItselfInTheStatusRow. FREEZE has a countdown to announce
// it; REWIND, BLAST and REPAIR changed the field and said nothing at all.
func TestInstantPowerNamesItselfInTheStatusRow(t *testing.T) {
	useBlocks(t)
	scr := simScreen(t, 80, 24)
	l := ComputeLayout(80, 24, goldenTuning())

	w := dotWord("RATIO+", 20, 6, 5, 0)
	w.Power = engine.PowerRepair
	st := fixedState(t, l, w)
	st.Lives = 1
	if h := st.Type('+'); !h.Completed {
		t.Fatal("the suffix did not complete the word")
	}

	Compose(scr, st, l, nil, nil)
	scr.Show()
	if !strings.Contains(dump(scr), "REPAIR") {
		t.Errorf("REPAIR fired without naming itself; screen was:\n%s", dump(scr))
	}
}

// gameOverScreen renders one finished run.
func gameOverScreen(t *testing.T, w, h int, killer string) (tcell.SimulationScreen, Layout) {
	t.Helper()
	useBlocks(t)
	scr := simScreen(t, w, h)
	l := ComputeLayout(w, h, goldenTuning())
	st := engine.NewState(l.PlayW, l.PlayH, goldenTuning(), engine.NewRNG(1), []string{"X"})
	st.GameOver = true
	st.Killer = killer
	st.Lives = 0
	st.Score, st.Hits, st.Keys = 12840, 553, 594
	st.WordsDestroyed, st.BestStreak, st.BestChain = 72, 47, 6
	st.Elapsed = 271
	Compose(scr, st, l, nil, nil)
	scr.Show()
	return scr, l
}

// TestKillerWordStaysInsideTheField. It was rasterized at the huge tier but
// placed with engine.TextWidth, which measures the normal one, so a long word
// came out half a word left of centre and ran off the right border. Words too
// long for even the small tier fall back to plain text.
func TestKillerWordStaysInsideTheField(t *testing.T) {
	for _, killer := range []string{"PEDANT", "PERSPICACIOUS", "INCOMPREHENSIBILITY"} {
		scr, l := gameOverScreen(t, 80, 24, killer)
		cells, w, _ := scr.GetContents()
		for y := l.Y; y < l.Y+l.H; y++ {
			for _, x := range []int{l.X, l.X + l.W - 1} {
				switch cells[y*w+x].Runes[0] {
				case Full, Upper, Lower:
					t.Errorf("%s: glyph ink on the border at %d,%d", killer, x, y)
				}
			}
		}

		// The word is the top band of the summary: the rows from the first with
		// any content down to the next blank one.
		blank := func(y int) bool {
			return strings.TrimSpace(strings.Trim(rowText(scr, y), "│")) == ""
		}
		top := l.PlayY
		for top <= l.PlayY+l.PlayH-1 && blank(top) {
			top++
		}
		lead, trail := l.PlayW, l.PlayW
		for y := top; y <= l.PlayY+l.PlayH-1 && !blank(y); y++ {
			row := []rune(rowText(scr, y))
			for x := l.PlayX; x < l.PlayX+l.PlayW; x++ {
				if row[x] == ' ' {
					continue
				}
				lead = min(lead, x-l.PlayX)
				trail = min(trail, l.PlayX+l.PlayW-1-x)
			}
		}
		if lead == l.PlayW {
			t.Errorf("%s: the killer word was not drawn", killer)
			continue
		}
		if d := lead - trail; d < -1 || d > 1 {
			t.Errorf("%s: margins are %d left and %d right", killer, lead, trail)
		}
	}
}

// TestGameOverHidesTheHUD. The HUD carries the score, the letter count, the
// accuracy and the clock, all of which the summary repeats a few rows below, so
// leaving it up printed every figure on the screen twice.
func TestGameOverHidesTheHUD(t *testing.T) {
	scr, l := gameOverScreen(t, 80, 24, "PEDANT")
	hud := rowText(scr, l.HUDY)
	if strings.TrimSpace(strings.Trim(hud, "│")) != "" {
		t.Errorf("the HUD is still drawn behind the game over screen: %q", hud)
	}
	floor := rowText(scr, l.FloorY)
	if strings.ContainsRune(floor, '═') || strings.ContainsRune(floor, '╌') {
		t.Error("the floor is still drawn behind the game over screen")
	}
}

// TestGameOverFiguresLineUp: labels flush left, values flush right, one block.
func TestGameOverFiguresLineUp(t *testing.T) {
	scr, l := gameOverScreen(t, 120, 40, "PEDANT")
	labels := []string{"SCORE", "WORDS", "ACCURACY", "BEST CHAIN", "BEST STREAK", "SURVIVED"}
	lefts, rights := map[int]bool{}, map[int]bool{}
	for _, label := range labels {
		left, right := -1, -1
		for y := l.PlayY; y <= l.PlayY+l.PlayH-1; y++ {
			x := columnOf(scr, y, label+" ")
			if x < 0 {
				continue
			}
			left = x
			row := []rune(rowText(scr, y))
			for c := l.PlayX; c < l.PlayX+l.PlayW; c++ {
				if row[c] != ' ' {
					right = c
				}
			}
			break
		}
		if left < 0 {
			t.Fatalf("%q is missing from the summary", label)
		}
		lefts[left] = true
		rights[right] = true
	}
	if len(lefts) != 1 {
		t.Errorf("labels start in %d columns, want one", len(lefts))
	}
	if len(rights) != 1 {
		t.Errorf("values end in %d columns, want one", len(rights))
	}
}

// TestGameOverIsVerticallyBalanced. The summary was pinned nine rows below the
// top of the playfield, which left it hugging the ceiling on a tall terminal.
func TestGameOverIsVerticallyBalanced(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}} {
		scr, l := gameOverScreen(t, size[0], size[1], "PEDANT")
		blank := func(y int) bool {
			return strings.TrimSpace(strings.Trim(rowText(scr, y), "│")) == ""
		}
		above, below := 0, 0
		for y := l.PlayY; y <= l.PlayY+l.PlayH-1 && blank(y); y++ {
			above++
		}
		for y := l.PlayY + l.PlayH - 1; y >= l.PlayY && blank(y); y-- {
			below++
		}
		if d := above - below; d < -1 || d > 1 {
			t.Errorf("%dx%d: %d blank rows above the summary and %d below",
				size[0], size[1], above, below)
		}
	}
}
