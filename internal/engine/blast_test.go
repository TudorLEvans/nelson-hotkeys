package engine

import "testing"

func blastState(t *testing.T) *State {
	t.Helper()
	st := NewState(78, 18, DefaultTuning(), NewRNG(2), []string{"RATIO"})
	st.MaxWordsCap = 0
	return st
}

// TestBlastHitsOnlyWhatIsNear is the point of the effect: a CLEAR with a radius,
// so where you finish the word matters rather than only that you finished it.
func TestBlastHitsOnlyWhatIsNear(t *testing.T) {
	st := blastState(t)
	st.Words = append(st.Words,
		Word{Text: "NEARBY", Tier: TierNormal, DotX: 10 * M.DotsX, DotY: 9 * M.DotsY},
		Word{Text: "FARAWAY", Tier: TierNormal, DotX: 60 * M.DotsX, DotY: 2 * M.DotsY},
	)
	centre := Rect{X: 12, Y: 9, W: 4, H: 3}

	boxes := st.applyPowerAt(PowerBlast, centre)
	if len(boxes) == 0 {
		t.Error("no explosion boxes returned")
	}
	if len(st.Words) != 1 {
		t.Fatalf("%d words survived, want 1", len(st.Words))
	}
	if st.Words[0].Text != "FARAWAY" {
		t.Errorf("the surviving word is %q; the distant one should have been spared",
			st.Words[0].Text)
	}
}

// TestBlastSparesBombs. Detonating one for the player is the single thing they
// have no defence against, and the bomb rules promise only their own keystrokes
// can set one off.
func TestBlastSparesBombs(t *testing.T) {
	st := blastState(t)
	st.Words = append(st.Words,
		Word{Text: "BOGUS", Tier: TierNormal, Bomb: true, DotX: 10 * M.DotsX, DotY: 9 * M.DotsY},
		Word{Text: "SOURCE", Tier: TierNormal, DotX: 14 * M.DotsX, DotY: 9 * M.DotsY},
	)
	st.applyPowerAt(PowerBlast, Rect{X: 12, Y: 9, W: 4, H: 3})

	if len(st.Words) != 1 || !st.Words[0].Bomb {
		t.Errorf("field is %v; the bomb should be the only survivor", st.Words)
	}
	if st.Lives != int(st.Tune.Lives) || st.GameOver {
		t.Error("the blast set the bomb off")
	}
}

// TestBlastIsNotAWholeFieldWipe. It replaced CLEAR, and it must not have become
// CLEAR in the process: the radius is the entire point, because it is what makes
// WHERE the word is finished matter.
func TestBlastIsNotAWholeFieldWipe(t *testing.T) {
	st := blastState(t)
	for i := 0; i < 6; i++ {
		st.Words = append(st.Words, Word{
			Text: "SOURCE", Tier: TierNormal,
			DotX: i * 12 * M.DotsX, DotY: i * 3 * M.DotsY,
		})
	}
	st.applyPowerAt(PowerBlast, Rect{X: 0, Y: 0, W: 4, H: 3})

	if len(st.Words) == 0 {
		t.Error("BLAST wiped the whole field; it is meant to be the radius version")
	}
	if len(st.Words) == 6 {
		t.Error("BLAST destroyed nothing at all")
	}
	t.Logf("of 6 spread words, BLAST left %d", len(st.Words))
}

// TestBlastScoresNothing: it destroys words the player did not type, and a button
// that pays out is a button pressed mindlessly.
func TestBlastScoresNothing(t *testing.T) {
	st := blastState(t)
	st.Words = append(st.Words,
		Word{Text: "SOURCE", Tier: TierNormal, DotX: 10 * M.DotsX, DotY: 9 * M.DotsY})
	before := st.Score
	st.applyPowerAt(PowerBlast, Rect{X: 10, Y: 9, W: 4, H: 3})
	if st.Score != before {
		t.Errorf("BLAST paid out %d points", st.Score-before)
	}
}

// TestRepairIsTheRarest. Of the four that survived the cut it is the only one
// that hands back a life, so rarity has to track that.
func TestRepairIsTheRarest(t *testing.T) {
	tune := DefaultTuning()
	for _, p := range Powers {
		if p == PowerRepair {
			continue
		}
		if p.weight(tune) <= tune.WeightRepair {
			t.Errorf("%v weight %v is not above REPAIR %v", p, p.weight(tune), tune.WeightRepair)
		}
	}
}
