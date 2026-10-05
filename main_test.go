package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"nelson/internal/engine"
	"nelson/internal/render"
	"nelson/internal/replay"
	"nelson/internal/stats"
	"nelson/internal/words"
)

// useBlocks pins the drawing style so a change of default does not churn every
// golden file and expectation.
func useBlocks(t *testing.T) {
	t.Helper()
	prev := engine.M
	engine.SetMetrics(engine.MetricsBlocks)
	t.Cleanup(func() { engine.SetMetrics(prev) })
}

func sim(t *testing.T, w, h int) tcell.SimulationScreen {
	t.Helper()
	scr := tcell.NewSimulationScreen("UTF-8")
	if err := scr.Init(); err != nil {
		t.Fatal(err)
	}
	scr.SetSize(w, h)
	t.Cleanup(scr.Fini)
	return scr
}

func newTestGame(t *testing.T, w, h int, tune *engine.Tuning) *game {
	t.Helper()
	if tune == nil {
		tune = engine.DefaultTuning()
	}
	g := newGame(sim(t, w, h), tune, []string{"RATIO"}, 1, true)
	enterPlay(g)
	return g
}

// enterPlay drives the real route from the menu into a run, so tests exercise the
// same path a player takes rather than reaching past it.
func enterPlay(g *game) {
	g.press(' ') // PLAY is the first menu item
	g.press(' ') // any key dismisses the ready gate
}

// TestLoopAdvancesWord drives the real loop body. The pty smoke run proves the
// binary starts and exits cleanly, but tcell interleaves cursor moves between
// cells so scraping its output cannot confirm the world is actually moving.
// This can.
func TestLoopAdvancesWord(t *testing.T) {
	useBlocks(t)
	tune := engine.DefaultTuning()
	g := newTestGame(t, 80, 24, tune)

	// The spawner owns the first word, so tick once to get it.
	g.tick()
	if len(g.st.Words) != 1 {
		t.Fatalf("%d words after the first tick, want 1", len(g.st.Words))
	}
	start := g.st.Words[0].DotY

	const frames = 119 // four seconds, less the tick already spent
	for i := 0; i < frames; i++ {
		g.tick()
	}

	// Movement is quantised to a shared tick, so check the distance covered
	// rather than an exact float position.
	elapsed := float64(frames) * float64(dt)
	ticks := elapsed / tune.StepInterval
	want := int(ticks) / g.st.StepEvery(&g.st.Words[0])
	got := g.st.Words[0].DotY - start
	if got < want-1 || got > want+1 {
		t.Errorf("advanced %d half rows in %.1f s, want about %d", got, elapsed, want)
	}
	if g.st.Frame != frames+1 {
		t.Errorf("frame counter %d, want %d", g.st.Frame, frames+1)
	}
}

// TestSubCellMotionIsExercised is the reason the dot machinery exists. A word
// that only ever rendered on whole-cell boundaries would sit still for most of a
// second and then jump, which reads as a slideshow. Over four seconds it must
// visit several sub-cell positions.
func TestSubCellMotionIsExercised(t *testing.T) {
	useBlocks(t)
	g := newTestGame(t, 80, 24, nil)
	g.tick() // the spawner produces the first word
	changes := 0
	prev := g.st.Words[0].SubOffset()
	for i := 0; i < 120; i++ {
		g.tick()
		if o := g.st.Words[0].SubOffset(); o != prev {
			changes++
			prev = o
		}
	}
	if changes < 8 {
		t.Errorf("sub-cell offset changed %d times in 4 s, want at least 8; "+
			"sub-cell motion is not being exercised", changes)
	}
}

// TestFrameCostUnderBudget is the claim the language choice rested on. A
// regression here is a real regression, so it is a test rather than a note.
func TestFrameCostUnderBudget(t *testing.T) {
	useBlocks(t)
	g := newTestGame(t, 120, 40, nil)
	for i := 0; i < 30; i++ {
		g.tick() // warm up, and let the word enter the field
	}
	const n = 300
	start := time.Now()
	for i := 0; i < n; i++ {
		g.tick()
	}
	per := time.Since(start) / n
	budget := tickDur
	t.Logf("mean frame %.3f ms of a %.1f ms budget (%.1f%%)",
		float64(per.Nanoseconds())/1e6, float64(budget.Nanoseconds())/1e6,
		100*float64(per)/float64(budget))
	if per > budget/4 {
		t.Errorf("mean frame %v exceeds a quarter of the %v budget", per, budget)
	}
}

func TestSizeGateSwitchesScene(t *testing.T) {
	useBlocks(t)
	scr := sim(t, 60, 20)
	g := newGame(scr, engine.DefaultTuning(), []string{"RATIO"}, 1, false)
	enterPlay(g)
	if !g.lay.TooSmall {
		t.Fatal("60x20 should be too small")
	}
	if len(g.st.Words) != 0 {
		t.Error("words spawned while the terminal is too small")
	}
	g.tick() // must not panic drawing the resize card

	scr.SetSize(80, 24)
	g.resize()
	if g.lay.TooSmall {
		t.Fatal("80x24 should be playable")
	}
	enterPlay(g)
	g.tick()
	if len(g.st.Words) == 0 {
		t.Error("no word after growing to a playable size")
	}
}

// TestResizeWithinCapKeepsTheRun matters because the playfield is capped: going
// from 200 to 180 columns changes nothing the player can see, so it must not
// throw away their game.
func TestResizeWithinCapKeepsTheRun(t *testing.T) {
	useBlocks(t)
	scr := sim(t, 200, 60)
	g := newGame(scr, engine.DefaultTuning(), []string{"RATIO"}, 1, false)
	enterPlay(g)
	for i := 0; i < 60; i++ {
		g.tick()
	}
	before := g.st.Words[0].DotY
	scr.SetSize(180, 55)
	g.resize()
	if len(g.st.Words) == 0 {
		t.Fatal("resize inside the cap dropped the field")
	}
	if g.st.Words[0].DotY != before {
		t.Errorf("word moved on a cosmetic resize: %v -> %v", before, g.st.Words[0].DotY)
	}
}

// TestTuningHotReload is the feature that justified choosing a compiled
// language: tuning must not need a rebuild.
func TestTuningHotReload(t *testing.T) {
	useBlocks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "tuning.conf")
	if err := os.WriteFile(path, []byte("fall_speed = 0.8\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tune, err := engine.LoadTuning(path)
	if err != nil {
		t.Fatal(err)
	}
	g := newGame(sim(t, 80, 24), tune, []string{"RATIO"}, 1, true)
	enterPlay(g)
	g.tick()

	// Edit the file mid-run, with an mtime the watcher will notice.
	if err := os.WriteFile(path, []byte("fall_speed = 4.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < watchInt*2; i++ {
		g.tick()
	}
	if g.tune.FallSpeed != 4.0 {
		t.Fatalf("fall_speed = %v, want 4.0 after a mid-run edit", g.tune.FallSpeed)
	}
	if g.dbg.Reloads == 0 {
		t.Error("reload not reported in the debug overlay")
	}
}

// TestBadTuningEditDoesNotCrash: a half-written file is normal when someone is
// saving mid-frame, and it must surface as a message, not a crash.
func TestBadTuningEditDoesNotCrash(t *testing.T) {
	useBlocks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "tuning.conf")
	os.WriteFile(path, []byte("fall_speed = 0.8\n"), 0o644)
	tune, err := engine.LoadTuning(path)
	if err != nil {
		t.Fatal(err)
	}
	g := newGame(sim(t, 80, 24), tune, []string{"RATIO"}, 1, true)
	enterPlay(g)
	g.tick()

	os.WriteFile(path, []byte("fall_speed = 0.8\nwhat_is_this = 3\n"), 0o644)
	future := time.Now().Add(2 * time.Second)
	os.Chtimes(path, future, future)
	for i := 0; i < watchInt*2; i++ {
		g.tick()
	}
	if g.tune.FallSpeed != 0.8 {
		t.Errorf("bad edit was partly applied: fall_speed = %v", g.tune.FallSpeed)
	}
	if g.dbg.TuneErr == "" {
		t.Error("bad tuning edit produced no message; a silent typo looks like bad game feel")
	}
}

// TestEscapeGoesBackNotOut. Escape in a run returns to the menu, because quitting
// the whole program is a bigger step than a player pressing escape usually means.
// Escape at the menu does quit, since there is nowhere further back to go.
func TestEscapeGoesBackNotOut(t *testing.T) {
	useBlocks(t)

	g := newTestGame(t, 80, 24, nil)
	g.handleEvent(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if g.quit {
		t.Error("escape during a run quit the program")
	}
	if g.scene != sceneMenu {
		t.Errorf("escape during a run left scene %v, want the menu", g.scene)
	}

	g.handleEvent(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if !g.quit {
		t.Error("escape at the menu did not quit")
	}
}

func TestCtrlCAlwaysQuits(t *testing.T) {
	useBlocks(t)
	for _, s := range []scene{sceneMenu, sceneGuide, sceneReady, scenePlay} {
		g := newTestGame(t, 80, 24, nil)
		g.scene = s
		g.handleEvent(tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone))
		if !g.quit {
			t.Errorf("ctrl-c did not quit from scene %v", s)
		}
	}
}

func TestLetterKeysDoNotQuit(t *testing.T) {
	useBlocks(t)
	g := newTestGame(t, 80, 24, nil)
	g.handleEvent(tcell.NewEventKey(tcell.KeyRune, 'R', tcell.ModNone))
	if g.quit {
		t.Error("a letter key quit the game")
	}
}

// TestWordNeverDrawnOutsideTheField runs a long session and checks nothing
// escapes the border, which is the failure a width or offset bug produces.
func TestWordNeverDrawnOutsideTheField(t *testing.T) {
	useBlocks(t)
	tune := engine.DefaultTuning()
	tune.FallSpeed = 3.0
	scr := sim(t, 80, 24)
	g := newGame(scr, tune, []string{"RATIO", "WHATABOUT", "ANTIDISESTAB"}, 7, false)
	enterPlay(g)

	inked := map[rune]bool{render.Full: true, render.Upper: true, render.Lower: true}
	for i := 0; i < 3000; i++ {
		g.tick()
		cells, w, h := scr.GetContents()
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r := cells[y*w+x].Runes
				if len(r) == 0 || !inked[r[0]] {
					continue
				}
				if x < g.lay.PlayX || x >= g.lay.PlayX+g.lay.PlayW {
					t.Fatalf("frame %d: glyph at column %d, playfield is %d..%d",
						i, x, g.lay.PlayX, g.lay.PlayX+g.lay.PlayW-1)
				}
				if y < g.lay.PlayY || y > g.lay.PlayY+g.lay.PlayH-1 {
					t.Fatalf("frame %d: glyph at row %d, playfield is %d..%d",
						i, y, g.lay.PlayY, g.lay.PlayY+g.lay.PlayH-1)
				}
			}
		}
	}
}

// press feeds a character through the same path a terminal keypress takes.
func (g *game) press(ch rune) {
	g.handleEvent(tcell.NewEventKey(tcell.KeyRune, ch, tcell.ModNone))
}

// TestTypingThroughTheLoop is the test for the complaint that the game did not
// respond to keyboard input. It drives real key events into the real handler.
func TestTypingThroughTheLoop(t *testing.T) {
	useBlocks(t)
	tune := engine.DefaultTuning()
	tune.FallSpeed = 0.6
	g := newGame(sim(t, 80, 24), tune, []string{"RATIO"}, 1, true)
	enterPlay(g)
	for i := 0; i < 40; i++ {
		g.tick() // let the word enter the field
	}
	if len(g.st.Words) == 0 {
		t.Fatal("no word to type")
	}

	for _, ch := range "RATIO" {
		before := g.st.Score
		g.press(ch)
		g.tick()
		if g.st.Score <= before {
			t.Fatalf("pressing %q scored nothing", ch)
		}
	}
	if g.st.Hits != 5 {
		t.Errorf("hits %d, want 5", g.st.Hits)
	}
	if g.st.WordsDestroyed != 1 {
		t.Errorf("words destroyed %d, want 1", g.st.WordsDestroyed)
	}
	if g.dbg.KeysSeen != 5 {
		t.Errorf("debug counter saw %d keys, want 5", g.dbg.KeysSeen)
	}
}

// TestLowercaseAndCapsBothWork: the game is in block capitals but nobody types
// with caps lock on.
func TestLowercaseAndCapsBothWork(t *testing.T) {
	useBlocks(t)
	for _, text := range []string{"ratio", "RATIO", "RaTiO"} {
		tune := engine.DefaultTuning()
		tune.FallSpeed = 0.6
		g := newGame(sim(t, 80, 24), tune, []string{"RATIO"}, 1, false)
		enterPlay(g)
		for i := 0; i < 40; i++ {
			g.tick()
		}
		for _, ch := range text {
			g.press(ch)
		}
		if g.st.WordsDestroyed != 1 {
			t.Errorf("typing %q destroyed %d words, want 1", text, g.st.WordsDestroyed)
		}
	}
}

// TestTypingSpawnsParticles: a letter vanishing with no feedback feels dead, so
// the explosion is part of the mechanic, not decoration.
func TestTypingSpawnsParticles(t *testing.T) {
	useBlocks(t)
	tune := engine.DefaultTuning()
	tune.FallSpeed = 0.6
	g := newGame(sim(t, 80, 24), tune, []string{"RATIO"}, 1, false)
	enterPlay(g)
	for i := 0; i < 40; i++ {
		g.tick()
	}
	if g.fx.Count() != 0 {
		t.Fatalf("%d particles before typing", g.fx.Count())
	}
	g.press('R')
	if g.fx.Count() == 0 {
		t.Error("destroying a letter produced no particles")
	}
	// And they must clear up rather than accumulate for the rest of the run.
	for i := 0; i < 30; i++ {
		g.tick()
	}
	if g.fx.Count() != 0 {
		t.Errorf("%d particles still alive a second later", g.fx.Count())
	}
}

// TestFreeFireAcrossWordsThroughTheLoop is the user's mechanic end to end: leave
// one word half-eaten, work another, come back.
func TestFreeFireAcrossWordsThroughTheLoop(t *testing.T) {
	useBlocks(t)
	tune := engine.DefaultTuning()
	tune.FallSpeed = 0.2
	tune.SpawnInterval = 0.1
	g := newGame(sim(t, 80, 24), tune, []string{"SOURCE", "PEDANT"}, 4, false)
	enterPlay(g)
	for i := 0; i < 90 && len(g.st.Words) < 2; i++ {
		g.tick()
	}
	if len(g.st.Words) < 2 {
		t.Skip("spawner did not produce two words in time")
	}

	// Chip one letter off whichever word each key hits, then verify the field
	// holds partial words rather than forcing one to be finished first.
	g.press(rune(g.st.Words[0].Text[0]))
	g.press(rune(g.st.Words[1].Text[0]))
	partial := 0
	for i := range g.st.Words {
		if g.st.Words[i].Typed > 0 {
			partial++
		}
	}
	if partial < 2 {
		t.Errorf("%d words are part-eaten, want 2; free fire must not force one at a time", partial)
	}
}

// TestGameOverRestartsOnSpace: friction between death and the next attempt costs
// attempts, so this is a requirement rather than a nicety.
func TestGameOverRestartsOnSpace(t *testing.T) {
	useBlocks(t)
	tune := engine.DefaultTuning()
	tune.FallSpeed = 9
	tune.SpawnPause = 0
	g := newGame(sim(t, 80, 24), tune, []string{"RATIO"}, 2, false)
	enterPlay(g)
	for i := 0; i < 30*90 && !g.st.GameOver; i++ {
		g.tick()
	}
	if !g.st.GameOver {
		t.Fatal("never died")
	}
	if g.st.Killer == "" {
		t.Error("no killer word recorded for the game over screen")
	}
	g.tick() // must draw the game over screen without panicking

	g.press('R') // a letter must not restart
	if !g.st.GameOver {
		t.Error("a letter key restarted the run")
	}
	g.press(' ')
	if g.st.GameOver {
		t.Fatal("space did not restart")
	}
	if g.st.Score != 0 || g.st.Lives != int(tune.Lives) {
		t.Errorf("restart left score %d lives %d", g.st.Score, g.st.Lives)
	}
	g.tick()
	if len(g.st.Words) == 0 {
		t.Error("no word after restarting")
	}
}

// TestShippedTuningFileIsValid is the test that should have existed before a key
// was renamed. The strict unknown-key check means a stale tuning.conf makes the
// game report an error on startup, which is exactly what happened when
// ramp_period became level_base and level_growth in code but not in the file.
//
// Checks both directions: nothing in the file is unknown to the code, and nothing
// in the code is missing from the file.
func TestShippedTuningFileIsValid(t *testing.T) {
	const path = "tuning.conf"
	tune, err := engine.LoadTuning(path)
	if err != nil {
		t.Fatalf("shipped %s does not parse: %v", path, err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	inFile := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, _, ok := strings.Cut(line, "=")
		if !ok {
			t.Errorf("%s: unparseable line %q", path, line)
			continue
		}
		inFile[strings.TrimSpace(key)] = true
	}

	for _, key := range tune.Keys() {
		if !inFile[key] {
			t.Errorf("%s does not mention %q; every tunable should be documented "+
				"where people look for it", path, key)
		}
	}
	// The reverse direction is already covered: LoadTuning rejects unknown keys,
	// so a stale name would have failed above.
}

// TestTuningFileMatchesDefaults guards against the file drifting away from the
// built-in values, which would make the defaults a lie: the game would behave one
// way with the file present and another without it.
func TestTuningFileMatchesDefaults(t *testing.T) {
	fromFile, err := engine.LoadTuning("tuning.conf")
	if err != nil {
		t.Fatal(err)
	}
	defaults := engine.DefaultTuning()
	for _, key := range defaults.Keys() {
		a, aok := fromFile.Value(key)
		b, bok := defaults.Value(key)
		if !aok || !bok {
			t.Errorf("%q not readable", key)
			continue
		}
		if a != b {
			t.Errorf("%s: tuning.conf has %v, DefaultTuning has %v", key, a, b)
		}
	}
}

// TestPowerupSuffixReachesTheEngine covers the gap that let power-ups ship
// untypeable. The engine rule had a test, but it called State.Type directly; the
// input layer in between silently dropped every non-letter, so no key press could
// ever finish a power-up word. Test the path the keyboard actually takes.
func TestPowerupSuffixReachesTheEngine(t *testing.T) {
	for _, suffix := range []rune{'.', '-', '/', '+'} {
		tune := engine.DefaultTuning()
		tune.FallSpeed = 0.4
		g := newGame(sim(t, 80, 24), tune, []string{"RATIO"}, 1, false)
		enterPlay(g)
		g.st.Words = g.st.Words[:0]
		g.st.MaxWordsCap = 0
		g.st.Words = append(g.st.Words, engine.Word{
			Text: "RATIO" + string(suffix), Tier: engine.TierNormal,
			Power: engine.Power(suffix), DotY: 8,
		})

		for _, ch := range "RATIO" {
			g.press(ch)
		}
		before := g.st.Hits
		g.press(suffix)
		if g.st.Hits != before+1 {
			t.Errorf("%q never reached the engine: hits stayed at %d", suffix, before)
			continue
		}
		if len(g.st.Words) != 0 {
			t.Errorf("%q did not complete the word", suffix)
		}
	}
}

// TestDigitsAndUnknownKeys: digits are typeable because words may contain them;
// anything else must be ignored rather than counted as a miss, or a stray Tab
// would cost the player speed.
func TestNonTypeableKeysAreIgnored(t *testing.T) {
	useBlocks(t)
	g := newTestGame(t, 80, 24, nil)
	for i := 0; i < 30; i++ {
		g.tick()
	}
	before := g.st.Keys
	for _, ch := range []rune{'£', '~', '@', '\\', '`'} {
		g.press(ch)
	}
	if g.st.Keys != before {
		t.Errorf("%d stray keys were counted; unsupported keys must be ignored",
			g.st.Keys-before)
	}
}

// TestRunOpensWithOneWord. Every run used to start with two on screen at once,
// because newGame seeded one by hand and the spawner's timer also fired on the
// first tick. The spawner owns spawning; nothing else may call it.
func TestRunOpensWithOneWord(t *testing.T) {
	useBlocks(t)
	g := newTestGame(t, 80, 24, nil)
	if len(g.st.Words) != 0 {
		t.Fatalf("%d words before the first tick, want 0", len(g.st.Words))
	}
	g.tick()
	if got := len(g.st.Words); got != 1 {
		t.Fatalf("%d words on the first tick, want exactly 1", got)
	}

	// And the second must wait out the spawn interval rather than piling on.
	gap := g.st.SpawnIntervalNow()
	for i := 0; i < int(gap*30)-4; i++ {
		g.tick()
	}
	if got := len(g.st.Words); got != 1 {
		t.Errorf("%d words just before the interval elapsed, want 1", got)
	}
	for i := 0; i < 8; i++ {
		g.tick()
	}
	if got := len(g.st.Words); got != 2 {
		t.Errorf("%d words after the interval, want 2", got)
	}
}

// TestRestartAlsoOpensWithOneWord: the same for the second run, since a fast
// retry is the common case.
func TestRestartAlsoOpensWithOneWord(t *testing.T) {
	useBlocks(t)
	tune := engine.DefaultTuning()
	tune.FallSpeed = 9
	tune.SpawnPause = 0
	g := newGame(sim(t, 80, 24), tune, []string{"RATIO"}, 2, false)
	enterPlay(g)
	for i := 0; i < 30*90 && !g.st.GameOver; i++ {
		g.tick()
	}
	if !g.st.GameOver {
		t.Fatal("never died")
	}
	g.press(' ')
	g.tick()
	if got := len(g.st.Words); got != 1 {
		t.Errorf("%d words on the first tick after restart, want 1", got)
	}
}

// TestMenuIgnoresLetterKeys is a deliberate rule, not an oversight. The player is
// one keypress from a mode where every letter does something, and a menu that
// also reacts to letters teaches the wrong reflex on the very first screen. J and
// K are the exception because they move rather than select.
func TestMenuIgnoresLetterKeys(t *testing.T) {
	useBlocks(t)
	g := newGame(sim(t, 80, 24), engine.DefaultTuning(), []string{"RATIO"}, 1, false)
	if g.scene != sceneMenu {
		t.Fatalf("game did not open on the menu, got scene %v", g.scene)
	}
	for _, ch := range "PLAYQUITHOWX" {
		if ch == 'J' || ch == 'K' {
			continue
		}
		g.press(ch)
		if g.scene != sceneMenu {
			t.Fatalf("%q left the menu", ch)
		}
		if g.quit {
			t.Fatalf("%q quit the game", ch)
		}
	}
	if g.menuPick != 0 {
		t.Errorf("letters moved the selection to %d", g.menuPick)
	}
}

func TestMenuNavigationWraps(t *testing.T) {
	useBlocks(t)
	g := newGame(sim(t, 80, 24), engine.DefaultTuning(), []string{"RATIO"}, 1, false)
	n := render.MenuCount

	g.handleEvent(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
	if g.menuPick != n-1 {
		t.Errorf("up from the top went to %d, want %d", g.menuPick, n-1)
	}
	g.handleEvent(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	if g.menuPick != 0 {
		t.Errorf("down from the bottom went to %d, want 0", g.menuPick)
	}
	g.press('j')
	if g.menuPick != 1 {
		t.Errorf("j went to %d, want 1", g.menuPick)
	}
	g.press('k')
	if g.menuPick != 0 {
		t.Errorf("k went to %d, want 0", g.menuPick)
	}
}

// TestReadyGateHoldsTheClock is the reason the gate exists. The difficulty
// schedule promises every run gets the same sequence of pressure; a run whose
// clock starts while the player is still reading breaks that promise.
func TestReadyGateHoldsTheClock(t *testing.T) {
	useBlocks(t)
	g := newGame(sim(t, 80, 24), engine.DefaultTuning(), []string{"RATIO"}, 1, false)
	g.press(' ') // PLAY
	if g.scene != sceneReady {
		t.Fatalf("PLAY went to scene %v, want the ready gate", g.scene)
	}

	for i := 0; i < 120; i++ {
		g.tick()
	}
	if g.st.Elapsed != 0 {
		t.Errorf("clock ran for %.1fs while waiting at the ready gate", g.st.Elapsed)
	}
	if len(g.st.Words) != 0 {
		t.Error("words spawned before the player was ready")
	}

	// Any key starts, not a specific one.
	g.press('Q')
	if g.scene != scenePlay {
		t.Fatalf("a key at the ready gate left scene %v", g.scene)
	}
	g.tick()
	if g.st.Elapsed == 0 {
		t.Error("the clock did not start")
	}
}

// TestRestartSkipsTheReadyGate: on a retry the player's hands are already on the
// keys, so the gate would be friction between death and the next attempt.
func TestRestartSkipsTheReadyGate(t *testing.T) {
	useBlocks(t)
	tune := engine.DefaultTuning()
	tune.FallSpeed = 9
	tune.SpawnPause = 0
	g := newGame(sim(t, 80, 24), tune, []string{"RATIO"}, 2, false)
	enterPlay(g)
	for i := 0; i < 30*90 && !g.st.GameOver; i++ {
		g.tick()
	}
	if !g.st.GameOver {
		t.Fatal("never died")
	}
	g.press(' ')
	if g.scene != scenePlay {
		t.Errorf("restart went to scene %v, want straight back into play", g.scene)
	}
}

func TestGuidePagesThenReturns(t *testing.T) {
	useBlocks(t)
	g := newGame(sim(t, 80, 24), engine.DefaultTuning(), []string{"RATIO"}, 1, false)
	g.press('j') // HOW TO PLAY
	g.press(' ')
	if g.scene != sceneGuide {
		t.Fatalf("scene %v, want the guide", g.scene)
	}
	for i := 0; i < render.GuidePages(); i++ {
		g.tick() // each page must draw without panicking
		g.press(' ')
	}
	if g.scene != sceneMenu {
		t.Errorf("paging off the end left scene %v, want the menu", g.scene)
	}

	// Escape returns from anywhere in the guide.
	g.press(' ')
	g.press(' ')
	g.handleEvent(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if g.scene != sceneMenu {
		t.Error("escape did not leave the guide")
	}
}

// TestGuideFitsTheMinimumScreen. A guide that wraps or overflows at 80x24 is
// worse than no guide, and 80x24 is the contract.
func TestGuideFitsTheMinimumScreen(t *testing.T) {
	l := render.ComputeLayout(80, 24, engine.DefaultTuning())
	if bad := render.WrapCheck(l.PlayW); len(bad) > 0 {
		t.Errorf("%d guide lines are too wide for a %d-column playfield: %q",
			len(bad), l.PlayW, bad[0])
	}
	if need := render.GuideRowsNeeded(); need > l.PlayH {
		t.Errorf("the longest guide page needs %d rows, only %d available", need, l.PlayH)
	}
}

func statusContains(scr tcell.SimulationScreen, want string) bool {
	cells, w, h := scr.GetContents()
	for y := 0; y < h; y++ {
		var row []rune
		for x := 0; x < w; x++ {
			r := cells[y*w+x].Runes
			if len(r) == 0 {
				row = append(row, ' ')
				continue
			}
			row = append(row, r[0])
		}
		if strings.Contains(string(row), want) {
			return true
		}
	}
	return false
}

// TestFlaggedRunsAreUnranked. A best that anyone can produce by passing --god is
// not a best, and the same goes for a speed override, a level skip or a forced
// word. Those flags exist for development and do not produce the same game.
func TestFlaggedRunsAreUnranked(t *testing.T) {
	cases := []struct {
		name   string
		o      opts
		ranked bool
	}{
		{"clean", opts{}, true},
		{"god", opts{god: true}, false},
		{"speed", opts{speed: 0.5}, false},
		{"level skip", opts{startLevel: 10}, false},
		{"forced word", opts{oneWord: "RATIO"}, false},
		{"debug only", opts{debug: true}, true},
		{"daily", opts{daily: true}, true},
	}
	for _, c := range cases {
		got := !c.o.god && c.o.speed == 0 && c.o.startLevel == 0 && c.o.oneWord == ""
		if got != c.ranked {
			t.Errorf("%s: ranked = %v, want %v", c.name, got, c.ranked)
		}
	}
}

// TestUnrankedRunDoesNotWriteStats is the same rule checked through the game
// rather than through the expression.
func TestUnrankedRunDoesNotWriteStats(t *testing.T) {
	useBlocks(t)
	t.Setenv("HOME", t.TempDir())

	tune := engine.DefaultTuning()
	tune.FallSpeed = 9
	tune.SpawnPause = 0
	g := newGame(sim(t, 80, 24), tune, []string{"RATIO"}, 2, false)
	g.stats = stats.Load()
	g.ranked = false
	enterPlay(g)
	for i := 0; i < 30*90 && !g.st.GameOver; i++ {
		g.tick()
	}
	if !g.st.GameOver {
		t.Fatal("never died")
	}
	if g.stats.Games != 0 {
		t.Errorf("an unranked run was recorded: %d games", g.stats.Games)
	}
	if stats.Load().Games != 0 {
		t.Error("an unranked run reached the file")
	}
	if !strings.Contains(g.bestLine(), "unranked") {
		t.Errorf("the menu does not say the run is unranked: %q", g.bestLine())
	}
}

// TestRankedRunPersists, end to end through the game.
func TestRankedRunPersists(t *testing.T) {
	useBlocks(t)
	t.Setenv("HOME", t.TempDir())

	tune := engine.DefaultTuning()
	tune.FallSpeed = 9
	tune.SpawnPause = 0
	g := newGame(sim(t, 80, 24), tune, []string{"RATIO"}, 2, false)
	g.stats = stats.Load()
	enterPlay(g)
	// Score some points, then stop typing so the run actually ends.
	for i := 0; i < 30*120 && !g.st.GameOver; i++ {
		g.tick()
		if i < 90 && len(g.st.Words) > 0 {
			w := &g.st.Words[0]
			if !w.Done() {
				g.press(rune(w.Text[w.Typed]))
			}
		}
	}
	if !g.st.GameOver {
		t.Fatal("never died")
	}
	if g.st.Score == 0 {
		t.Fatal("scored nothing, so there is no score to persist")
	}
	if g.stats.Games != 1 {
		t.Fatalf("%d games recorded, want 1", g.stats.Games)
	}
	if !g.highScore {
		t.Error("the first ever run was not a high score")
	}

	reloaded := stats.Load()
	if reloaded.Games != 1 {
		t.Errorf("file has %d games, want 1", reloaded.Games)
	}
	if reloaded.BestScore() != g.st.Score {
		t.Errorf("file score %d, run scored %d", reloaded.BestScore(), g.st.Score)
	}
	if g.bestLine() == "" {
		t.Error("the menu shows no best after a recorded run")
	}
}

// TestBeatClearsOnRestart: the NEW HIGH SCORE banner belongs to the run that earned it.
func TestBeatClearsOnRestart(t *testing.T) {
	useBlocks(t)
	t.Setenv("HOME", t.TempDir())
	g := newTestGame(t, 80, 24, nil)
	g.stats = stats.Load()
	g.highScore = true
	g.st.GameOver = true
	g.press(' ')
	if g.highScore {
		t.Error("the banner survived a restart")
	}
}

// TestStatusRowCarriesDefinitionsOnly. The row sits directly under the playfield
// in a game where the player is reading falling words, so anything permanently
// parked there competes for the same attention. It used to hold the controls and
// a first-run tip; both are gone.
func TestStatusRowCarriesDefinitionsOnly(t *testing.T) {
	useBlocks(t)
	scr := sim(t, 80, 24)
	g := newGame(scr, engine.DefaultTuning(), []string{"RATIO"}, 1, false)
	enterPlay(g)
	for i := 0; i < 30; i++ {
		g.tick()
	}
	for _, junk := range []string{"esc quit", "ctrl-l", "type the words", "LOWEST"} {
		if statusContains(scr, junk) {
			t.Errorf("the status row still carries %q", junk)
		}
	}
}

// TestChainHudIsJustTheNumber. An earlier version showed the multiplier, the
// seconds left and the next tier at once, which is more than anyone can read
// while typing.
func TestChainHudIsJustTheNumber(t *testing.T) {
	useBlocks(t)
	scr := sim(t, 80, 24)
	g := newGame(scr, engine.DefaultTuning(), []string{"AB", "CD"}, 1, false)
	enterPlay(g)
	g.st.MaxWordsCap = 0
	g.st.Words = g.st.Words[:0]
	g.st.Words = append(g.st.Words,
		engine.Word{Text: "MMMMMMMM", Tier: engine.TierNormal},
		engine.Word{Text: "ZZ", Tier: engine.TierNormal, Typed: 1, DotX: 40 * engine.M.DotsX},
	)
	for i := 0; i < 6; i++ {
		g.press('M')
	}
	g.press('Z')
	g.tick()

	if !statusContains(scr, "CHAIN 1") {
		t.Error("the chain number is not on screen")
	}
	for _, junk := range []string{"->", "0.", "1.5s"} {
		if statusContains(scr, junk) {
			t.Errorf("the chain readout still shows %q", junk)
		}
	}
}

// TestReplayReproducesTheRunExactly is the property the whole feature rests on,
// and it is only true because of decisions taken much earlier: a fixed timestep
// rather than a measured one, integer positions, and a hand-rolled PRNG with a
// pinned sequence. A run is a pure function of seed, tuning and keys-by-frame.
func TestReplayReproducesTheRunExactly(t *testing.T) {
	useBlocks(t)
	const seed = 4242

	play := func(rec *replay.Recording, rp *replay.Recording) (*game, string) {
		tune := engine.DefaultTuning()
		g := newGame(sim(t, 80, 24), tune, mustPack(t), seed, false)
		g.rec, g.replay = rec, rp
		g.ranked = false
		if rp != nil {
			g.scene = scenePlay
		} else {
			enterPlay(g)
		}
		for i := 0; i < 900 && !g.st.GameOver; i++ {
			g.tick()
			// A scripted "player": fire the letter the lowest live word wants.
			if rp == nil && i%3 == 0 && len(g.st.Words) > 0 {
				w := &g.st.Words[len(g.st.Words)-1]
				if !w.Done() {
					g.typeKey(rune(w.Text[w.Typed]))
				}
			}
		}
		return g, fingerprint(g)
	}

	rec := replay.NewRecording(seed, "blocks", "english")
	live, want := play(rec, nil)
	if live.st.Hits == 0 {
		t.Fatal("the scripted player never hit anything, so there is nothing to replay")
	}

	_, got := play(nil, rec)
	if got != want {
		t.Errorf("replay diverged from the original run\n live: %s\nreplay: %s", want, got)
	}
	t.Logf("reproduced: %s", want)
}

// fingerprint is everything about a finished run that ought to be reproducible.
func fingerprint(g *game) string {
	var b strings.Builder
	fmt.Fprintf(&b, "score=%d letters=%d words=%d misses=%d lives=%d frame=%d chain=%d streak=%d",
		g.st.Score, g.st.Hits, g.st.WordsDestroyed, g.st.Misses,
		g.st.Lives, g.st.Frame, g.st.BestChain, g.st.BestStreak)
	for i := range g.st.Words {
		w := &g.st.Words[i]
		fmt.Fprintf(&b, " |%s@%d,%d+%d", w.Text, w.DotX, w.DotY, w.Typed)
	}
	return b.String()
}

func mustPack(t *testing.T) []string {
	t.Helper()
	list, err := words.Load("english")
	if err != nil {
		t.Fatal(err)
	}
	return list
}

// TestReplayIsUnranked: it is somebody else's run being reproduced, not yours.
func TestReplayIsUnranked(t *testing.T) {
	useBlocks(t)
	t.Setenv("HOME", t.TempDir())
	rec := replay.NewRecording(1, "blocks", "english")
	g := newGame(sim(t, 80, 24), engine.DefaultTuning(), []string{"RATIO"}, 1, false)
	g.replay = rec
	g.ranked = false
	g.scene = scenePlay
	g.stats = stats.Load()
	for i := 0; i < 60; i++ {
		g.tick()
	}
	g.st.GameOver = true
	g.finish()
	if g.stats.Games != 0 {
		t.Error("a replay was recorded as a played game")
	}
}

// TestAudioTogglesFromTheMenuAndPersists. Audio is on for a fresh install,
// because audio nobody is told about goes unheard. The switch lives on the menu
// and the answer is remembered, so turning it off sticks.
func TestAudioTogglesFromTheMenuAndPersists(t *testing.T) {
	useBlocks(t)
	t.Setenv("HOME", t.TempDir())

	fresh := stats.Load()
	if !fresh.Sound || !fresh.Music {
		t.Fatal("audio is off on a fresh install")
	}

	g := newGame(sim(t, 80, 24), engine.DefaultTuning(), []string{"RATIO"}, 1, false)
	g.stats = fresh
	g.applyAudio(fresh.Sound, fresh.Music)
	if !g.wantSFX || !g.wantTune {
		t.Fatal("the remembered setting did not reach the run")
	}

	g.menuPick = render.MenuMusic
	g.choose()
	if g.wantTune {
		t.Error("choosing MUSIC did not turn it off")
	}
	if stats.Load().Music {
		t.Error("the choice did not reach the file")
	}

	g.menuPick = render.MenuSound
	g.choose()
	if g.wantSFX {
		t.Error("choosing SOUND did not turn it off")
	}

	// And on again.
	g.choose()
	if !g.wantSFX {
		t.Error("choosing SOUND a second time did not turn it back on")
	}
	if !stats.Load().Sound {
		t.Error("turning it back on did not reach the file")
	}
}

// TestRememberedAudioSurvivesRestart, which is the whole point of writing it down.
// Off is the setting worth checking: on is what the default already gives.
func TestRememberedAudioSurvivesRestart(t *testing.T) {
	useBlocks(t)
	t.Setenv("HOME", t.TempDir())

	first := newGame(sim(t, 80, 24), engine.DefaultTuning(), []string{"RATIO"}, 1, false)
	first.stats = stats.Load()
	first.applyAudio(first.stats.Sound, first.stats.Music)
	first.menuPick = render.MenuMusic
	first.choose()
	first.audio.Close()

	reloaded := stats.Load()
	if reloaded.Music {
		t.Fatal("music was turned off but came back on")
	}
	if !reloaded.Sound {
		t.Error("sound was turned off too")
	}
}

// TestOldStatsFileTakesTheNewAudioDefault. A file written before audio defaulted
// on says sound:false, but nobody chose that, so it must not pin audio off.
func TestOldStatsFileTakesTheNewAudioDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.MkdirAll(filepath.Dir(stats.Path()), 0o755); err != nil {
		t.Fatal(err)
	}
	old := `{"games":3,"sound":false,"music":false}`
	if err := os.WriteFile(stats.Path(), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}

	s := stats.Load()
	if s.Games != 3 {
		t.Fatalf("history was lost: games = %d, want 3", s.Games)
	}
	if !s.Sound || !s.Music {
		t.Error("an unchosen false pinned audio off")
	}
}

// TestMenuLabelsShowTheState: a switch that does not say which way it is set is
// not a switch.
func TestMenuLabelsShowTheState(t *testing.T) {
	off := render.MenuLabels(false, false)
	on := render.MenuLabels(true, true)
	if !strings.Contains(off[render.MenuSound], "OFF") {
		t.Errorf("sound off reads %q", off[render.MenuSound])
	}
	if !strings.Contains(on[render.MenuSound], "ON") {
		t.Errorf("sound on reads %q", on[render.MenuSound])
	}
	if !strings.Contains(on[render.MenuMusic], "ON") {
		t.Errorf("music on reads %q", on[render.MenuMusic])
	}
	if len(off) != render.MenuCount {
		t.Errorf("%d labels for %d items", len(off), render.MenuCount)
	}
}
