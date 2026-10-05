package engine

import "testing"

func bombState(t *testing.T) *State {
	t.Helper()
	st := NewState(78, 18, DefaultTuning(), NewRNG(11), []string{"SOURCE"})
	st.MaxWordsCap = 0
	return st
}

// TestBombNeverTakesPriority is the rule the whole mechanic rests on. Any safe
// word wanting the letter wins, so no arrangement of words can force the player
// into a bomb.
func TestBombNeverTakesPriority(t *testing.T) {
	// The bomb is far lower, which under the ordinary rule would make it the
	// target. It must still lose to the safe word.
	words := []Word{
		{Text: "SOURCE", Tier: TierNormal, DotY: 4},
		{Text: "STRAWMAN", Tier: TierNormal, DotX: 200, DotY: 60, Bomb: true},
	}
	if got := Resolve(words, 'S'); got != 0 {
		t.Errorf("Resolve picked word %d; a bomb must never beat a safe word", got)
	}

	// Only when nothing safe wants the letter does the bomb receive it.
	words[0].Typed = 1 // now wants O
	if got := Resolve(words, 'S'); got != 1 {
		t.Errorf("Resolve picked %d; with no safe candidate the bomb should take it", got)
	}
}

// TestBombCannotBeForced is the same property stated as the player experiences
// it: every letter of a word can be typed with a bomb on the field, whatever the
// bomb says and wherever it sits.
func TestBombCannotBeForced(t *testing.T) {
	st := bombState(t)
	st.Words = append(st.Words,
		Word{Text: "SOURCE", Tier: TierNormal, DotY: 4},
		Word{Text: "SOURCE", Tier: TierNormal, DotX: 200, DotY: 60, Bomb: true},
	)
	for _, ch := range "SOURCE" {
		h := st.Type(byte(ch))
		if !h.Ok {
			t.Fatalf("%q missed", ch)
		}
		if h.Bomb {
			t.Fatalf("%q was fed to the bomb while a safe word wanted it", ch)
		}
	}
	if st.BombsHit != 0 {
		t.Error("the bomb took damage during a clean word")
	}
}

// TestOnlyTyposFeedABomb: the only way to feed one is to press a letter nothing
// else wants, which is what a mistake looks like.
func TestOnlyTyposFeedABomb(t *testing.T) {
	st := bombState(t)
	st.Words = append(st.Words,
		Word{Text: "RATIO", Tier: TierNormal, DotY: 4},
		Word{Text: "BOGUS", Tier: TierNormal, DotX: 200, DotY: 60, Bomb: true},
	)
	// B is wanted by nothing safe, so it lands on the bomb. Under the old scoring
	// this was a free point; now it is damage.
	h := st.Type('B')
	if !h.Ok {
		t.Fatal("B should have reached the bomb")
	}
	if st.Words[1].Typed != 1 {
		t.Error("the bomb did not take the letter")
	}
}

func TestCompletingABombCostsALife(t *testing.T) {
	st := bombState(t)
	st.Words = append(st.Words, Word{Text: "BOGUS", Tier: TierNormal, DotY: 4, Bomb: true})
	st.Streak = 12

	lives := st.Lives
	var h Hit
	for _, ch := range "BOGUS" {
		h = st.Type(byte(ch))
	}
	if !h.Completed || !h.Bomb {
		t.Fatal("the bomb did not report as completed")
	}
	if st.Lives != lives-1 {
		t.Errorf("lives %d, want %d", st.Lives, lives-1)
	}
	if st.Streak != 0 {
		t.Errorf("combo survived a bomb: %d", st.Streak)
	}
	if st.BombsHit != 1 {
		t.Errorf("BombsHit = %d, want 1", st.BombsHit)
	}
}

// TestBombLandsHarmlessly: ignoring a bomb is always valid, and usually right.
func TestBombLandsHarmlessly(t *testing.T) {
	tune := DefaultTuning()
	tune.SpawnPause = 0
	st := NewState(78, 18, tune, NewRNG(3), []string{"RATIO"})
	st.MaxWordsCap = 0
	st.Words = append(st.Words, Word{
		Text: "BOGUS", Tier: TierNormal, Bomb: true, DotY: 18 * M.DotsY,
	})

	res := st.Step(1.0 / 30)
	if res.BombsDodged != 1 {
		t.Errorf("BombsDodged = %d, want 1", res.BombsDodged)
	}
	if res.LostLife || st.Lives != int(tune.Lives) {
		t.Error("a dodged bomb cost a life")
	}
	if len(st.Words) != 0 {
		t.Error("the bomb stayed on the field")
	}
}

// TestBombsAndPowerupsNeverCoexist: both are exceptions to ordinary play, and two
// exceptions on screen at once is unreadable.
func TestBombsAndPowerupsNeverCoexist(t *testing.T) {
	tune := DefaultTuning()
	tune.BombFromLevel = 0
	tune.BombChance = 0.5
	tune.PowerupFromLevel = 0
	tune.PowerupChance = 0.5
	st := NewState(78, 18, tune, NewRNG(21), []string{
		"RATIO", "SOURCE", "PEDANT", "CITATION", "STRAWMAN", "ANECDOTAL", "CONSENSUS",
	})
	st.MaxWordsCap = 6

	for i := 0; i < 400; i++ {
		st.Spawn()
		bombs, powers := 0, 0
		for j := range st.Words {
			if st.Words[j].Bomb {
				bombs++
			}
			if st.Words[j].Power != PowerNone {
				powers++
			}
		}
		if bombs > 1 {
			t.Fatalf("%d bombs on screen", bombs)
		}
		if bombs > 0 && powers > 0 {
			t.Fatalf("a bomb and a power-up on screen together")
		}
		if len(st.Words) > 5 {
			st.Words = st.Words[:0]
		}
	}
}

func TestNoBombsBeforeTheirLevel(t *testing.T) {
	tune := DefaultTuning()
	tune.BombChance = 1
	tune.BombFromLevel = 4
	st := NewState(78, 18, tune, NewRNG(6), []string{"RATIO", "SOURCE", "PEDANT"})
	st.MaxWordsCap = 6
	for i := 0; i < 30; i++ {
		st.Words = st.Words[:0]
		st.Spawn()
		if len(st.Words) > 0 && st.Words[0].Bomb {
			t.Fatalf("bomb at level %.2f, gate is %v", st.LevelF(), tune.BombFromLevel)
		}
	}
}
