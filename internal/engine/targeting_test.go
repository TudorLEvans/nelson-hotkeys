package engine

import "testing"

// word takes a cell column and a row for readability and converts to the dot
// units the engine actually uses.
func word(text string, col, row float32, typed int) Word {
	return Word{
		Text: text, Typed: typed, Tier: TierNormal,
		DotX: int(col) * M.DotsX,
		DotY: int(row * float32(M.DotsY)),
	}
}

func newTestState(t *testing.T, words ...Word) *State {
	t.Helper()
	tune := DefaultTuning()
	st := NewState(78, 18, tune, NewRNG(1), []string{"RATIO"})
	st.Words = append(st.Words, words...)
	return st
}

// TestResolveCaseTable is the full behaviour of the free fire rule. The player
// will learn this and rely on it, so every branch is pinned.
func TestResolveCaseTable(t *testing.T) {
	cases := []struct {
		name  string
		words []Word
		key   byte
		want  int
	}{
		{"no words", nil, 'S', -1},
		{"no candidate", []Word{word("RATIO", 0, 5, 0)}, 'S', -1},
		{"single candidate", []Word{word("SOURCE", 0, 5, 0)}, 'S', 0},
		{"lower of two wins", []Word{
			word("SOURCE", 0, 2, 0),
			word("STRAWMAN", 20, 9, 0),
		}, 'S', 1},
		{"lower of two wins regardless of order", []Word{
			word("STRAWMAN", 20, 9, 0),
			word("SOURCE", 0, 2, 0),
		}, 'S', 0},
		{"only untyped letters count", []Word{
			word("SOURCE", 0, 9, 1), // next letter is O, not S
			word("STRAWMAN", 20, 2, 0),
		}, 'S', 1},
		{"matches mid-word letter", []Word{word("SOURCE", 0, 5, 2)}, 'U', 0},
		{"exhausted word is not a candidate", []Word{
			word("SO", 0, 9, 2),
			word("SOURCE", 20, 2, 0),
		}, 'S', 1},
		{"tie on row broken by leftmost x", []Word{
			word("SOURCE", 40, 5, 0),
			word("STRAWMAN", 5, 5, 0),
		}, 'S', 1},
	}
	for _, c := range cases {
		if got := Resolve(c.words, c.key); got != c.want {
			t.Errorf("%s: Resolve(%q) = %d, want %d", c.name, c.key, got, c.want)
		}
	}
}

// TestResolveTieBrokenByAge is the last tie-break. Two words at the same place
// cannot happen in play, but the ordering must still be total or Resolve would
// depend on iteration luck.
func TestResolveTieBrokenByAge(t *testing.T) {
	a := word("SOURCE", 5, 5, 0)
	a.Age = 1
	b := word("STRAWMAN", 5, 5, 0)
	b.Age = 0 // older
	if got := Resolve([]Word{a, b}, 'S'); got != 1 {
		t.Errorf("Resolve = %d, want 1 (the older word)", got)
	}
	if got := Resolve([]Word{b, a}, 'S'); got != 0 {
		t.Errorf("reversed slice: Resolve = %d, want 0 (the same older word)", got)
	}
}

// TestSixWordsSharingALetter is the behaviour the user asked for by name: no
// lock, so pressing one letter six times takes it off six different words,
// lowest first.
func TestSixWordsSharingALetter(t *testing.T) {
	var words []Word
	for i := 0; i < 6; i++ {
		words = append(words, word("SOURCE", float32(i*6), float32(2+i*2), 0))
	}
	st := newTestState(t, words...)

	for press := 0; press < 6; press++ {
		h := st.Type('S')
		if !h.Ok {
			t.Fatalf("press %d missed; every word still starts with S", press+1)
		}
		// Lowest first: words were placed with increasing y, so the expected
		// order is last to first.
		if want := 5 - press; h.Word != want {
			t.Errorf("press %d hit word %d, want %d (lowest first)", press+1, h.Word, want)
		}
	}
	for i := range st.Words {
		if st.Words[i].Typed != 1 {
			t.Errorf("word %d has %d letters eaten, want 1", i, st.Words[i].Typed)
		}
	}
	// A seventh press finds nothing: every word now waits on O.
	if h := st.Type('S'); h.Ok {
		t.Error("seventh S hit something; all six words should now want O")
	}
}

// TestTypoDoesNotStealFromAnotherWord: free fire means the field, not a lock,
// decides. A key no word wants is a miss and nothing more.
func TestMissCostsOnlyTheCombo(t *testing.T) {
	st := newTestState(t, word("SOURCE", 0, 5, 0))
	st.Type('S')
	st.Type('O')
	if st.Streak != 2 {
		t.Fatalf("combo %d, want 2", st.Streak)
	}
	h := st.Type('Z')
	if h.Ok {
		t.Error("Z hit something")
	}
	if st.Streak != 0 {
		t.Errorf("combo %d after a miss, want 0", st.Streak)
	}
	if st.Misses != 1 || st.Hits != 2 {
		t.Errorf("hits %d misses %d, want 2 and 1", st.Hits, st.Misses)
	}
	if st.Words[0].Typed != 2 {
		t.Error("a miss must not consume a letter")
	}
	if st.Lives != int(st.Tune.Lives) {
		t.Error("a miss must not cost a life")
	}
}

func TestCompletionRemovesWord(t *testing.T) {
	st := newTestState(t, word("SO", 0, 5, 0))
	st.Type('S')
	h := st.Type('O')
	if !h.Completed {
		t.Fatal("last letter did not complete the word")
	}
	if len(st.Words) != 0 {
		t.Errorf("%d words left, want 0", len(st.Words))
	}
	if st.WordsDestroyed != 1 {
		t.Errorf("WordsDestroyed = %d, want 1", st.WordsDestroyed)
	}
}

// TestDepthMultiplierZones pins the score dial. The renderer colours words from
// this same function, so a change here changes what the player sees too.
func TestDepthMultiplierZones(t *testing.T) {
	const playH = 18
	cases := []struct{ row, want int }{
		{0, 1}, {5, 1},
		{6, 2}, {11, 2},
		{12, 3}, {15, 3},
		{16, 5}, {17, 5}, {18, 5},
	}
	for _, c := range cases {
		if got := DepthMultiplier(c.row, playH); got != c.want {
			t.Errorf("row %d of %d: multiplier %d, want %d", c.row, playH, got, c.want)
		}
	}
}

// TestDeepLetterScoresMore is the reason the multiplier exists: it turns every
// word into a bet rather than a chore.
func TestDeepLetterScoresMore(t *testing.T) {
	shallow := newTestState(t, word("SOURCE", 0, 0, 0))
	shallow.Type('S')
	deep := newTestState(t, word("SOURCE", 0, 15, 0))
	deep.Type('S')
	if deep.Score <= shallow.Score {
		t.Errorf("deep letter scored %d, shallow %d; depth must pay more",
			deep.Score, shallow.Score)
	}
	if deep.Score != 5 || shallow.Score != 1 {
		t.Errorf("scores %d and %d, want 5 and 1", deep.Score, shallow.Score)
	}
}

// TestChainMultiplierTiers. The scale runs past the old ceiling of five: capping
// there stopped the reward at exactly the moment the best players were doing the
// most impressive thing.
func TestChainMultiplierTiers(t *testing.T) {
	for chain, want := range map[int]int{
		0: 1, 1: 1, 2: 2, 3: 3, 4: 5, 5: 8, 6: 8, 7: 12, 9: 12, 10: 20, 15: 20,
	} {
		if got := ChainMultiplier(chain); got != want {
			t.Errorf("chain %d: x%d, want x%d", chain, got, want)
		}
	}
	// Monotonic, so a longer chain is never worth less.
	for c := 1; c < 30; c++ {
		if ChainMultiplier(c) < ChainMultiplier(c-1) {
			t.Fatalf("chain %d pays less than %d", c, c-1)
		}
	}
}

// TestChainPaysForFreeFire is the payoff that makes chipping worth the risk.
// Two words finished inside the window must beat the same two finished apart.
func TestChainPaysForFreeFire(t *testing.T) {
	build := func() *State {
		st := newTestState(t, word("SO", 0, 5, 0), word("AT", 20, 5, 0))
		// Get combo above the chain gate first, on a third word.
		st.Words = append(st.Words, word("EEEEE", 40, 1, 0))
		for i := 0; i < 5; i++ {
			st.Type('E')
		}
		return st
	}

	chained := build()
	chained.Type('S')
	chained.Type('O')
	chained.Type('A')
	chained.Type('T')

	apart := build()
	apart.Type('S')
	apart.Type('O')
	// Let the chain window lapse.
	for apart.Elapsed <= float32(apart.Tune.ChainWindow)+0.1 {
		apart.Step(1.0 / 30)
	}
	apart.Type('A')
	apart.Type('T')

	if chained.Score <= apart.Score {
		t.Errorf("chained %d vs apart %d; finishing inside the window must pay more",
			chained.Score, apart.Score)
	}
	if chained.BestChain < 2 {
		t.Errorf("BestChain = %d, want at least 2", chained.BestChain)
	}
}

// TestChainNeedsCombo stops mashing from stumbling into the payout.
func TestChainNeedsCombo(t *testing.T) {
	st := newTestState(t, word("SO", 0, 5, 0), word("AT", 20, 5, 0))
	st.Type('S')
	st.Type('O') // combo 2, below the gate of 5
	st.Type('A')
	st.Type('T')
	if st.Chain != 1 {
		t.Errorf("chain reached %d on a combo of 4; the gate is %v",
			st.Chain, st.Tune.ChainMinStreak)
	}
}

func TestLandingCostsALifeAndShockwaves(t *testing.T) {
	tune := DefaultTuning()
	tune.FallSpeed = 1
	st := NewState(78, 18, tune, NewRNG(1), []string{"RATIO"})
	st.MaxWordsCap = 0 // no new spawns, so the test controls the field exactly
	// A word lands when its bottom edge (top plus the glyph height) reaches the
	// floor, so the landing line is row 15 of 18, not row 18.
	// Horizontally separated on purpose: overlapping spans now trigger collision
	// resolution, which would push the second word up out of the blast radius and
	// quietly invalidate the test.
	st.Words = append(st.Words,
		word("SOURCE", 0, 14.9, 0),  // one step from the floor
		word("PEDANT", 34, 13.5, 0), // bottom row 16, inside the 3-row shockwave
		word("RATIO", 62, 2, 0),     // safely high
	)

	var res StepResult
	for i := 0; i < 30*20 && !res.LostLife; i++ {
		res = st.Step(1.0 / 30)
	}
	if !res.LostLife {
		t.Fatal("word never landed")
	}
	if st.Lives != int(tune.Lives)-1 {
		t.Errorf("lives %d, want %d", st.Lives, int(tune.Lives)-1)
	}
	if res.Cleared == 0 {
		t.Error("the impact shockwave cleared nothing; a cascade would end the run instantly")
	}
	if len(st.Words) != 1 || st.Words[0].Text != "RATIO" {
		t.Errorf("field is %v, want only the high word to survive", st.Words)
	}
	if st.Killer != "SOURCE" {
		t.Errorf("killer %q, want SOURCE", st.Killer)
	}
}

func TestGameOverAfterThreeLives(t *testing.T) {
	tune := DefaultTuning()
	tune.FallSpeed = 8
	tune.SpawnPause = 0
	st := NewState(78, 18, tune, NewRNG(3), []string{"RATIO"})
	st.Spawn()
	for i := 0; i < 30*120 && !st.GameOver; i++ {
		st.Step(1.0 / 30)
	}
	if !st.GameOver {
		t.Fatal("never reached game over")
	}
	if st.Lives != 0 {
		t.Errorf("lives %d at game over, want 0", st.Lives)
	}
	// Nothing must move after the run ends.
	before := len(st.Words)
	st.Step(1.0 / 30)
	if len(st.Words) != before {
		t.Error("the field changed after game over")
	}
	if h := st.Type('R'); h.Ok {
		t.Error("typing still worked after game over")
	}
}

func TestResetClearsEverything(t *testing.T) {
	st := newTestState(t, word("SO", 0, 5, 0))
	st.Type('S')
	st.Type('O')
	st.GameOver = true
	st.Killer = "SO"
	st.Reset()

	if st.Score != 0 || st.Hits != 0 || st.Keys != 0 || st.WordsDestroyed != 0 {
		t.Error("counters survived a reset")
	}
	if st.GameOver || st.Killer != "" {
		t.Error("game over state survived a reset")
	}
	if st.Lives != int(st.Tune.Lives) {
		t.Errorf("lives %d after reset, want %v", st.Lives, st.Tune.Lives)
	}
	if len(st.Words) != 0 {
		t.Error("field survived a reset")
	}
}

// TestSpawnerFillsTheField checks the spawner actually keeps words coming, since
// without it there is nothing to type.
func TestSpawnerFillsTheField(t *testing.T) {
	tune := DefaultTuning()
	tune.SpawnInterval = 0.5
	st := NewState(78, 18, tune, NewRNG(11), []string{"RATIO", "PEDANT", "SOURCE"})
	st.MaxWordsCap = 4
	peak := 0
	for i := 0; i < 30*20; i++ {
		st.Step(1.0 / 30)
		if len(st.Words) > peak {
			peak = len(st.Words)
		}
		if len(st.Words) > st.MaxWordsNow() {
			t.Fatalf("frame %d: %d words on screen, cap is %d", i, len(st.Words), st.MaxWordsNow())
		}
	}
	if peak < 2 {
		t.Errorf("peaked at %d words in 20 s; the field never filled", peak)
	}
}

// TestAccuracyAndWPM guards the two numbers the HUD shows.
func TestAccuracyAndWPM(t *testing.T) {
	st := newTestState(t, word("SOURCE", 0, 5, 0))
	if st.Accuracy() != 100 {
		t.Errorf("fresh accuracy %v, want 100", st.Accuracy())
	}
	st.Type('S')
	st.Type('Z')
	if got := st.Accuracy(); got != 50 {
		t.Errorf("accuracy %v after 1 hit and 1 miss, want 50", got)
	}
	st.Elapsed = 60
	if got := st.WPM(); got < 0.19 || got > 0.21 {
		t.Errorf("WPM %v for 1 letter in a minute, want 0.2", got)
	}
}

// TestOpeningReactionTime turns a balance target into a test. The spec asks for
// about 15 seconds on the first word: much longer and the opening screen sits
// empty for so long that the game looks like it is not responding, much shorter
// and a beginner never gets started.
func TestOpeningReactionTime(t *testing.T) {
	tune := DefaultTuning()
	const playH = 18 // the 80x24 minimum
	st := NewState(78, playH, tune, NewRNG(1), []string{"RATIO"})
	st.MaxWordsCap = 1
	st.Spawn()

	frames := 0
	for len(st.Words) > 0 && frames < 30*120 {
		st.Step(1.0 / 30)
		frames++
	}
	// The band is wide because the ramp is steep early: the word is accelerating
	// while it falls, so it lands sooner than its spawn speed alone implies.
	secs := float64(frames) / 30
	if secs < 9 || secs > 16 {
		t.Errorf("first word reaches the floor in %.1f s, want 9-16 "+
			"(fall_speed is %.2f)", secs, tune.FallSpeed)
	}

	// And it must become visible quickly, or the opening looks broken.
	st2 := NewState(78, playH, tune, NewRNG(1), []string{"RATIO"})
	st2.MaxWordsCap = 1
	st2.Spawn()
	visible := 0
	for st2.Words[0].BottomRow() < 0 && visible < 30*10 {
		st2.Step(1.0 / 30)
		visible++
	}
	if v := float64(visible) / 30; v > 1.5 {
		t.Errorf("first word takes %.1f s to appear on screen, want under 1.5", v)
	}
}
