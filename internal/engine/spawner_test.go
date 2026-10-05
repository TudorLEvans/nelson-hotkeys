package engine

import "testing"

func rhythmState(t *testing.T) *State {
	t.Helper()
	tune := DefaultTuning()
	tune.BombChance = 0
	tune.PowerupChance = 0
	st := NewState(78, 18, tune, NewRNG(31), []string{
		"SALT", "SPIN", "SNARK", "SHILL", "SEETHE", "SOURCE", "STRAW", "SNIPE",
		"BEEF", "BIAS", "BOGUS", "BLOCK", "BASE", "BLOC",
		"RATIO", "RANT", "REPLY", "ROUT",
	})
	st.MaxWordsCap = 6
	return st
}

// TestSwarmDropsOneLetter is the designed showcase for free fire: under the
// targeting rule one letter pressed N times takes the head off N words, lowest
// first, so a swarm is where the mechanic pays off spectacularly.
func TestSwarmDropsOneLetter(t *testing.T) {
	st := rhythmState(t)
	st.Elapsed = 200 // past the gate, and past swarm_every
	n := st.spawnSwarm()
	if n < 2 {
		t.Fatalf("swarm placed %d words", n)
	}
	first := st.Words[0].Text[0]
	for i := range st.Words {
		if st.Words[i].Text[0] != first {
			t.Errorf("swarm word %q does not start with %c", st.Words[i].Text, first)
		}
		if !st.Words[i].Swarm {
			t.Error("swarm word not marked as one")
		}
	}
	if letter, ok := st.SwarmBanner(); !ok || letter != first {
		t.Errorf("banner reported %q %v, want %c", letter, ok, first)
	}

	// And the payoff: one key clears one letter off every one of them.
	for range st.Words {
		if h := st.Type(first); !h.Ok {
			t.Fatal("a swarm letter missed")
		}
	}
	for i := range st.Words {
		if st.Words[i].Typed != 1 {
			t.Errorf("word %d has %d letters eaten, want 1", i, st.Words[i].Typed)
		}
	}
}

func TestSwarmRespectsTheFieldCap(t *testing.T) {
	st := rhythmState(t)
	st.Elapsed = 200
	st.MaxWordsCap = 3
	st.spawnSwarm()
	if len(st.Words) > 3 {
		t.Errorf("%d words after a swarm, cap is 3", len(st.Words))
	}
}

func TestNoSwarmsBeforeTheirLevel(t *testing.T) {
	st := rhythmState(t)
	st.Elapsed = 5
	if st.dueSwarm() {
		t.Errorf("swarm due at level %.2f, gate is %v", st.LevelF(), st.Tune.SwarmFromLevel)
	}
}

// TestBreathAfterAStreak: a pause lands just after a burst of clearing, which is
// when it reads as a reward rather than dead air.
func TestBreathAfterAStreak(t *testing.T) {
	st := rhythmState(t)
	st.MaxWordsCap = 0
	every := int(st.Tune.BreatheEvery)

	for i := 0; i < every; i++ {
		st.Words = append(st.Words, Word{Text: "RATIO", Tier: TierNormal})
		for _, ch := range "RATIO" {
			st.Type(byte(ch))
		}
	}
	if st.WordsDestroyed != every {
		t.Fatalf("destroyed %d words, want %d", st.WordsDestroyed, every)
	}
	if !st.breathing() {
		t.Errorf("no breath after %d words", every)
	}

	for i := 0; i < int(st.Tune.BreatheSeconds*30)+2; i++ {
		st.Step(1.0 / 30)
	}
	if st.breathing() {
		t.Error("the breath never ended")
	}
}

// TestTimeForLevelInvertsLevelF guards --level: it moves the clock rather than
// faking a level, so everything derived from the clock stays consistent.
func TestTimeForLevelInvertsLevelF(t *testing.T) {
	tune := DefaultTuning()
	st := NewState(78, 18, tune, NewRNG(1), []string{"RATIO"})
	for _, lvl := range []float64{1, 2, 5, 10, 25} {
		st.Elapsed = float32(TimeForLevel(tune, lvl))
		if got := st.LevelF(); got < lvl-0.01 || got > lvl+0.01 {
			t.Errorf("TimeForLevel(%v) lands at level %.3f", lvl, got)
		}
	}
}

func TestGodModeNeverLosesALife(t *testing.T) {
	tune := DefaultTuning()
	tune.SpawnPause = 0
	st := NewState(78, 18, tune, NewRNG(8), []string{"RATIO"})
	st.MaxWordsCap = 0
	st.God = true
	st.Words = append(st.Words, Word{Text: "RATIO", Tier: TierNormal, DotY: 18 * M.DotsY})

	st.Step(1.0 / 30)
	if st.Lives != int(tune.Lives) {
		t.Errorf("god mode lost a life: %d", st.Lives)
	}

	// Including to a bomb, which is a separate path.
	st.Words = append(st.Words, Word{Text: "BOGUS", Tier: TierNormal, Bomb: true})
	for _, ch := range "BOGUS" {
		st.Type(byte(ch))
	}
	if st.Lives != int(tune.Lives) || st.GameOver {
		t.Errorf("god mode died to a bomb: lives %d over %v", st.Lives, st.GameOver)
	}
}
