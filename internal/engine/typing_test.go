package engine

import (
	"sort"
	"testing"
)

// TestSameFingerRepeatsAreHardest pins the ranking the scores exist to express.
func TestSameFingerRepeatsAreHardest(t *testing.T) {
	cases := []struct{ easier, harder string }{
		// Alternating hands beats staying on one.
		{"AUTHENTIC", "SWEATERS"},
		// A same-finger repeat beats mere length.
		{"MINIMUM", "DEEDED"},
		// Home row beats reaching.
		{"ALASKA", "QUIZZED"},
	}
	for _, c := range cases {
		e, h := TypingDifficulty(c.easier), TypingDifficulty(c.harder)
		if e >= h {
			t.Errorf("%q scores %.2f but %q scores %.2f; the second should be harder",
				c.easier, e, c.harder, h)
		}
	}
}

func TestShortWordsScoreZero(t *testing.T) {
	for _, w := range []string{"", "A"} {
		if got := TypingDifficulty(w); got != 0 {
			t.Errorf("%q scored %.2f, want 0", w, got)
		}
	}
}

// TestDifficultyIsLengthIndependent is the point of having it at all: it must
// rank an awkward short word above a comfortable long one, or it is just length
// again under another name.
func TestDifficultyIsLengthIndependent(t *testing.T) {
	shortAwkward := TypingDifficulty("POLKA")
	longSmooth := TypingDifficulty("ENTITLEMENT")
	if shortAwkward <= longSmooth {
		t.Errorf("POLKA %.2f is not above ENTITLEMENT %.2f; the score is tracking "+
			"length rather than awkwardness", shortAwkward, longSmooth)
	}
}

// TestSelectionGetsHarderOverTime is the behaviour, measured over many draws
// rather than asserted about one.
func TestSelectionGetsHarderOverTime(t *testing.T) {
	pool := []string{
		"ALIENATE", "AUDITION", "ROTATION", "ENTITLE", "ISOLATE", "NOTABLE",
		"MINIMUM", "DEEDED", "POLKA", "SWERVED", "CRAFTS", "PUMPKIN",
		"BUZZARD", "JUGGLED", "AWKWARD", "MONOPOLY",
	}
	mean := func(elapsed float32) float64 {
		tune := DefaultTuning()
		st := NewState(78, 18, tune, NewRNG(19), pool)
		st.MaxWordsCap = 0
		st.Elapsed = elapsed
		lo, hi := 5, 12
		total := 0.0
		const n = 400
		for i := 0; i < n; i++ {
			total += TypingDifficulty(st.pickWord(lo, hi))
		}
		return total / n
	}

	early, late := mean(0), mean(600)
	t.Logf("mean typing difficulty: level 1 %.2f, ten minutes in %.2f", early, late)
	if late <= early {
		t.Errorf("late words score %.2f against %.2f early; selection is not getting "+
			"harder", late, early)
	}
}

// TestDifficultyOffMeansUniform: the whole thing assumes QWERTY, so it has to be
// possible to turn off without changing anything else.
func TestDifficultyOffMeansUniform(t *testing.T) {
	pool := []string{"MINIMUM", "ALIENATE", "POLKA", "ROTATION", "BUZZARD", "ISOLATE"}
	tune := DefaultTuning()
	tune.TypingDifficulty = 0
	st := NewState(78, 18, tune, NewRNG(23), pool)
	st.MaxWordsCap = 0
	st.Elapsed = 600

	seen := map[string]int{}
	for i := 0; i < 600; i++ {
		seen[st.pickWord(5, 12)]++
	}
	if len(seen) != len(pool) {
		t.Errorf("only %d of %d words appeared with weighting off", len(seen), len(pool))
	}
	counts := make([]int, 0, len(seen))
	for _, n := range seen {
		counts = append(counts, n)
	}
	sort.Ints(counts)
	if counts[len(counts)-1] > counts[0]*3 {
		t.Errorf("with weighting off the spread is %d to %d; it should be roughly flat",
			counts[0], counts[len(counts)-1])
	}
}

// TestNoRepeatsSurviveWeighting: the anti-repeat history has to keep working
// while difficulty is steering the choice, or the late game becomes a handful of
// awkward words on rotation.
func TestNoRepeatsSurviveWeighting(t *testing.T) {
	list, err := loadBigPool()
	if err != nil {
		t.Skip(err)
	}
	tune := DefaultTuning()
	st := NewState(78, 18, tune, NewRNG(29), list)
	st.MaxWordsCap = 0
	st.Elapsed = 600

	var recent []string
	for i := 0; i < 300; i++ {
		w := st.pickWord(6, 10)
		for _, r := range recent {
			if r == w {
				t.Fatalf("%q repeated inside the no-repeat window at draw %d", w, i)
			}
		}
		recent = append(recent, w)
		if len(recent) > 12 {
			recent = recent[1:]
		}
	}
}

func loadBigPool() ([]string, error) {
	var out []string
	for _, base := range []string{
		"ALIENATE", "AUDITION", "ROTATION", "ENTITLE", "ISOLATE", "NOTABLE",
		"MINIMUM", "DEEDED", "POLKA", "SWERVED", "CRAFTS", "PUMPKIN",
		"BUZZARD", "JUGGLED", "AWKWARD", "MONOPOLY", "TANGENT", "REALISE",
		"SEVERAL", "OUTLINE", "PROBLEM", "QUANTUM", "VERSION", "WESTERN",
		"YOUNGER", "ZEALOUS", "BRACKET", "CADENCE", "DESKTOP", "EXHIBIT",
		"FASTEST", "GLIMPSE", "HANDLED", "IMPULSE", "JOURNAL", "KEYNOTE",
	} {
		out = append(out, base)
	}
	return out, nil
}
