package engine

import (
	"math"
	"testing"
)

func heatState(t *testing.T) *State {
	t.Helper()
	st := NewState(78, 18, DefaultTuning(), NewRNG(3), []string{"SOURCE"})
	st.MaxWordsCap = 0
	return st
}

// TestMissAddsHeat is the mechanic: a typo makes the field faster.
func TestMissAddsHeat(t *testing.T) {
	st := heatState(t)
	st.Words = append(st.Words, Word{Text: "SOURCE", Tier: TierNormal})

	if st.TypoHeat() != 0 {
		t.Fatalf("heat %v before any typing", st.TypoHeat())
	}
	clean := st.EffectiveSpeedNow()

	if h := st.Type('Z'); h.Ok {
		t.Fatal("Z should have missed")
	}
	if got := st.TypoHeat(); math.Abs(got-st.Tune.HeatPerMiss) > 1e-6 {
		t.Errorf("heat %v after one miss, want %v", got, st.Tune.HeatPerMiss)
	}
	if st.EffectiveSpeedNow() <= clean {
		t.Error("a miss did not make the field faster")
	}
	// The clean schedule value must not move: heat is layered on top, so the
	// schedule stays a pure function of the clock.
	if st.FallSpeedNow() != clean {
		t.Error("a miss changed the schedule itself")
	}
}

// TestCorrectKeysAddNoHeat: only mistakes are punished.
func TestCorrectKeysAddNoHeat(t *testing.T) {
	st := heatState(t)
	st.Words = append(st.Words, Word{Text: "SOURCE", Tier: TierNormal})
	for _, ch := range "SOURCE" {
		st.Type(byte(ch))
	}
	if st.TypoHeat() != 0 {
		t.Errorf("heat %v after a flawless word", st.TypoHeat())
	}
}

// TestHeatDecays: the penalty fades, so a clean patch of play earns the pressure
// back rather than the run being ruined by one early slip.
func TestHeatDecays(t *testing.T) {
	st := heatState(t)
	st.Words = append(st.Words, Word{Text: "SOURCE", Tier: TierNormal})
	st.Type('Z')
	start := st.TypoHeat()

	// One half-life should halve it.
	for i := 0; i < int(st.Tune.HeatHalfLife*30); i++ {
		st.Step(1.0 / 30)
	}
	got := st.TypoHeat()
	if math.Abs(got-start/2) > start*0.05 {
		t.Errorf("after one half-life heat is %.4f, want about %.4f", got, start/2)
	}

	// And it reaches zero rather than lingering forever.
	for i := 0; i < 30*60; i++ {
		st.Step(1.0 / 30)
	}
	if st.TypoHeat() != 0 {
		t.Errorf("heat %v after a minute of clean play", st.TypoHeat())
	}
}

// TestHeatIsCapped: a masher should be punished, not executed. Without a ceiling
// the miss-faster-miss loop makes the game unplayable in a second and the penalty
// stops being a penalty.
func TestHeatIsCapped(t *testing.T) {
	st := heatState(t)
	st.Words = append(st.Words, Word{Text: "SOURCE", Tier: TierNormal})
	for i := 0; i < 500; i++ {
		st.Type('Z')
	}
	if got := st.TypoHeat(); got > st.Tune.HeatMax+1e-6 {
		t.Errorf("heat reached %v, cap is %v", got, st.Tune.HeatMax)
	}
	if got := st.TypoHeat(); got < st.Tune.HeatMax-1e-6 {
		t.Errorf("heat only reached %v after 500 misses, cap is %v", got, st.Tune.HeatMax)
	}
}

// TestHeatSpeedsUpTheWholeField, and does so smoothly. Applying the penalty to
// each word's integer step count would round a small penalty away entirely; it
// goes on the shared movement clock instead.
func TestHeatSpeedsUpTheWholeField(t *testing.T) {
	run := func(heat float32) int {
		st := heatState(t)
		st.Words = append(st.Words,
			Word{Text: "SOURCE", Tier: TierNormal},
			Word{Text: "PEDANT", Tier: TierNormal, DotX: 200},
		)
		st.typoHeat = heat
		for i := 0; i < 60; i++ {
			st.Step(1.0 / 30)
			st.typoHeat = heat // hold it steady against decay
		}
		if st.Words[0].DotY != st.Words[1].DotY {
			t.Fatalf("words fell out of lockstep: %d and %d",
				st.Words[0].DotY, st.Words[1].DotY)
		}
		return st.Words[0].DotY
	}

	clean := run(0)
	small := run(0.10)
	big := run(0.50)
	t.Logf("2 seconds of falling: clean %d dots, +10%% heat %d, +50%% heat %d",
		clean, small, big)

	if small <= clean {
		t.Errorf("a 10%% penalty moved words %d dots against a clean %d; small "+
			"penalties must register, not round away", small, clean)
	}
	if big <= small {
		t.Error("a larger penalty was not faster")
	}
}

// TestHeatEquilibriumMatchesTheTuning documents how the defaults were chosen: a
// steady miss rate settles at a predictable penalty rather than climbing forever.
//
// The bands are deliberately tight. The first set of numbers was twice this and
// played too heavy, so these are the ones to argue with if it still feels wrong.
func TestHeatEquilibriumMatchesTheTuning(t *testing.T) {
	tune := DefaultTuning()
	cases := []struct {
		name       string
		accuracy   float64
		keysPerSec float64
		wantLow    float64
		wantHigh   float64
	}{
		{"95% accurate", 0.95, 4, 0.02, 0.07},
		{"80% accurate", 0.80, 4, 0.10, 0.20},
		{"mashing", 0.0, 8, tune.HeatMax, tune.HeatMax},
	}
	for _, c := range cases {
		missRate := c.keysPerSec * (1 - c.accuracy)
		got := tune.HeatEquilibrium(missRate)
		if got < c.wantLow-1e-9 || got > c.wantHigh+1e-9 {
			t.Errorf("%s: settles at +%.0f%%, want %.0f-%.0f%%",
				c.name, got*100, c.wantLow*100, c.wantHigh*100)
		}
		t.Logf("%-14s %.1f misses/sec settles at +%.0f%% speed", c.name, missRate, got*100)
	}
}

// TestHeatOnlyEverPunishes pins the design constraint. The schedule is a clock so
// that runs compare; heat may push above it but must never dip below, or the
// fastest possible run would no longer be the clean one.
func TestHeatOnlyEverPunishes(t *testing.T) {
	st := heatState(t)
	st.Words = append(st.Words, Word{Text: "SOURCE", Tier: TierNormal})
	for i := 0; i < 200; i++ {
		for _, ch := range "SOURCE" {
			st.Type(byte(ch))
		}
		st.Words = append(st.Words, Word{Text: "SOURCE", Tier: TierNormal})
		st.Step(1.0 / 30)
		if st.EffectiveSpeedNow() < st.FallSpeedNow()-1e-9 {
			t.Fatalf("flawless play dropped below the schedule: %.4f < %.4f",
				st.EffectiveSpeedNow(), st.FallSpeedNow())
		}
	}
}

func TestResetClearsHeat(t *testing.T) {
	st := heatState(t)
	st.Words = append(st.Words, Word{Text: "SOURCE", Tier: TierNormal})
	st.Type('Z')
	st.Reset()
	if st.TypoHeat() != 0 {
		t.Errorf("heat %v survived a reset", st.TypoHeat())
	}
}
