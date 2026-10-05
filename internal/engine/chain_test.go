package engine

import "testing"

func chainState(t *testing.T) *State {
	t.Helper()
	tune := DefaultTuning()
	tune.BombChance = 0
	tune.PowerupChance = 0
	st := NewState(120, 30, tune, NewRNG(13), []string{"X"})
	st.MaxWordsCap = 0
	return st
}

func loadField(st *State, words ...string) {
	st.Words = st.Words[:0]
	for i, w := range words {
		st.Words = append(st.Words, Word{
			Text: w, Tier: TierNormal,
			DotX: i * 30 * M.DotsX,
			DotY: (3 + i*4) * M.DotsY,
		})
	}
}

// TestLoadedMarksChainFuel. A word one key from done is what a chain is built
// from, and marking it is the single change most likely to make the mechanic
// land: a field with three loaded words should look like an opportunity.
func TestLoadedMarksChainFuel(t *testing.T) {
	st := chainState(t)
	loadField(st, "BEACH", "FJORD", "GUILT")
	if st.LoadedCount() != 0 {
		t.Fatalf("%d loaded before typing", st.LoadedCount())
	}

	for _, ch := range "BEAC" {
		st.Type(byte(ch))
	}
	if st.LoadedCount() != 1 {
		t.Errorf("%d loaded after chipping one word, want 1", st.LoadedCount())
	}
	if !st.Words[0].Loaded() {
		t.Error("the chipped word is not marked loaded")
	}

	for _, ch := range "FJORGUIL" {
		st.Type(byte(ch))
	}
	if st.LoadedCount() != 3 {
		t.Errorf("%d loaded after chipping all three, want 3", st.LoadedCount())
	}

	// Finishing one takes it out of the count.
	st.Type('H')
	if st.LoadedCount() != 2 {
		t.Errorf("%d loaded after firing one, want 2", st.LoadedCount())
	}
}

// TestBombsAreNeverLoaded: the loaded cue means "press this", and a bomb is the
// one word that must never carry an invitation.
func TestBombsAreNeverLoaded(t *testing.T) {
	w := Word{Text: "BOGUS", Tier: TierNormal, Bomb: true, Typed: 4}
	if w.Loaded() {
		t.Error("a bomb one key from completion is marked as chain fuel")
	}
}

// TestChainWindowIsReadable is the other half of the legibility fix: a window the
// player cannot see is a window they cannot aim at.
func TestChainWindowIsReadable(t *testing.T) {
	st := chainState(t)
	loadField(st, "BEACH", "FJORD")
	for _, ch := range "BEACFJOR" {
		st.Type(byte(ch))
	}

	if st.ChainOpen() {
		t.Error("window open before any completion")
	}
	if got := st.NextChainMultiplier(); got != 1 {
		t.Errorf("next multiplier %d with no chain running, want 1", got)
	}

	st.Type('H')
	if !st.ChainOpen() {
		t.Fatal("window did not open on a completion")
	}
	if left := st.ChainSecondsLeft(); left <= 0 || float64(left) > st.Tune.ChainWindow {
		t.Errorf("%.2fs left, want between 0 and %v", left, st.Tune.ChainWindow)
	}
	if got := st.NextChainMultiplier(); got != 2 {
		t.Errorf("next completion shows x%d, want x2", got)
	}

	// It must close on its own.
	for i := 0; i < int(st.Tune.ChainWindow*30)+4; i++ {
		st.Step(1.0 / 30)
	}
	if st.ChainOpen() {
		t.Error("window never closed")
	}
}

// deepChain drives a chain to the requested depth and returns the last hit.
//
// The streak-building word is deliberately longer than the keys pressed into it,
// so it never completes. An earlier version used a word that finished, which
// started the chain a step early and made every depth off by one.
func deepChain(t *testing.T, st *State, depth int) Hit {
	t.Helper()
	st.Words = append(st.Words, Word{Text: "MMMMMMMM", Tier: TierNormal})
	for i := 0; i < 6; i++ {
		st.Type('M')
	}
	if st.Chain != 0 {
		t.Fatalf("chain already at %d before the run", st.Chain)
	}

	var last Hit
	for i := 0; i < depth; i++ {
		st.Words = append(st.Words, Word{Text: "ZZ", Tier: TierNormal, Typed: 1})
		last = st.Type('Z')
		if !last.Completed {
			t.Fatalf("completion %d did not land", i+1)
		}
		if last.Chain != i+1 {
			t.Fatalf("completion %d reported chain %d", i+1, last.Chain)
		}
	}
	return last
}

// TestChainRewardsPayInSurvival is the point of the deep tiers. Past two or three
// minutes the field is fast enough that a score multiplier is a consolation
// prize; the problem is staying alive.
func TestChainRewardsPayInSurvival(t *testing.T) {
	tune := DefaultTuning()

	t.Run("slow", func(t *testing.T) {
		st := chainState(t)
		deepChain(t, st, int(tune.ChainSlowAt))
		if !st.Slowed() {
			t.Errorf("chain %v gave no breathing room", tune.ChainSlowAt)
		}
	})

	t.Run("shield", func(t *testing.T) {
		st := chainState(t)
		deepChain(t, st, int(tune.ChainShieldAt))
		if !st.Shield {
			t.Errorf("chain %v gave no shield", tune.ChainShieldAt)
		}
	})

	t.Run("sweep", func(t *testing.T) {
		st := chainState(t)
		st.Words = append(st.Words,
			Word{Text: "LOWDOWN", Tier: TierNormal, DotY: 24 * M.DotsY},
			Word{Text: "HIGHUP", Tier: TierNormal, DotX: 60 * M.DotsX, DotY: 1 * M.DotsY},
		)
		h := deepChain(t, st, int(tune.ChainSweepAt))
		if h.Reward == nil || h.Reward.Name != "SWEEP" {
			t.Fatalf("chain %v gave %v", tune.ChainSweepAt, h.Reward)
		}
		for i := range st.Words {
			if st.Words[i].Text == "LOWDOWN" {
				t.Error("the low word survived the sweep")
			}
		}
		var high bool
		for i := range st.Words {
			if st.Words[i].Text == "HIGHUP" {
				high = true
			}
		}
		if !high {
			t.Error("the sweep took the upper half too; that is CLEAR's job")
		}
	})

	t.Run("life", func(t *testing.T) {
		st := chainState(t)
		st.Lives = 1
		deepChain(t, st, int(tune.ChainLifeAt))
		if st.Lives != 2 {
			t.Errorf("chain %v left %d lives, want 2", tune.ChainLifeAt, st.Lives)
		}
	})
}

// TestChainRewardsFireOnceEach: a milestone is a milestone, not a per-completion
// payout.
func TestChainRewardsFireOnceEach(t *testing.T) {
	st := chainState(t)
	st.Lives = 1
	fired := map[string]int{}
	st.Words = append(st.Words, Word{Text: "MMMMMMMM", Tier: TierNormal})
	for i := 0; i < 6; i++ {
		st.Type('M')
	}
	for i := 0; i < 14; i++ {
		st.Words = append(st.Words, Word{Text: "ZZ", Tier: TierNormal, Typed: 1})
		if h := st.Type('Z'); h.Reward != nil {
			fired[h.Reward.Name]++
		}
	}
	for name, n := range fired {
		if n != 1 {
			t.Errorf("%s fired %d times", name, n)
		}
	}
	if len(fired) != 4 {
		t.Errorf("%d distinct rewards fired, want 4: %v", len(fired), fired)
	}
}

// TestSweepSparesBombs, like every other area effect in the game.
func TestSweepSparesBombs(t *testing.T) {
	st := chainState(t)
	st.Words = append(st.Words,
		Word{Text: "BOGUS", Tier: TierNormal, Bomb: true, DotY: 24 * M.DotsY})
	deepChain(t, st, int(st.Tune.ChainSweepAt))
	var found bool
	for i := range st.Words {
		if st.Words[i].Bomb {
			found = true
		}
	}
	if !found {
		t.Error("the sweep destroyed a bomb")
	}
}
