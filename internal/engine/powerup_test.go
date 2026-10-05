package engine

import (
	"strings"
	"testing"
)

func powerState(t *testing.T) *State {
	t.Helper()
	tune := DefaultTuning()
	tune.PowerupFromLevel = 0
	tune.PowerupChance = 1 // every spawn carries one
	tune.BombChance = 0    // bombs suppress power-ups, so keep them out of the way
	st := NewState(78, 18, tune, NewRNG(9), []string{
		"ERM", "WELL", "RATIO", "SOURCE", "PEDANT", "CITATION", "STRAWMAN",
		"ANECDOTAL", "CONSENSUS", "DEFLECTION", "OBFUSCATION", "WHATABOUTISM",
	})
	st.MaxWordsCap = 6
	return st
}

// TestPowerupsRideLongerWords is the requested rule. A power-up on a short word
// would be a free gift; on a long one it is a commitment, and because chip
// acceleration scales with letters destroyed, a long word accelerates hard as it
// is eaten. Taking a power-up should mean taking on the most dangerous thing on
// the field.
func TestPowerupsRideLongerWords(t *testing.T) {
	st := powerState(t)
	st.Elapsed = 90 // mid-run, so the ordinary band is not already at the ceiling

	ordLo, ordHi := st.WordLenRange()
	powLo, powHi := st.powerLenRange()
	if powHi <= ordHi {
		t.Errorf("power-up band is %d-%d, ordinary band %d-%d; power-ups must be longer",
			powLo, powHi, ordLo, ordHi)
	}
	t.Logf("at %.0fs: ordinary %d-%d, power-up %d-%d", st.Elapsed, ordLo, ordHi, powLo, powHi)

	// And in practice, not only in the band arithmetic.
	for i := 0; i < 60; i++ {
		st.Words = st.Words[:0]
		if !st.Spawn() {
			continue
		}
		w := st.Words[0]
		if w.Power == PowerNone {
			t.Fatal("chance is 1, so every spawn should carry a power-up")
		}
		base := strings.TrimSuffix(w.Text, string(rune(w.Power)))
		if len(base) < powLo {
			t.Errorf("power-up word %q has %d letters, below the band floor %d",
				base, len(base), powLo)
		}
	}
}

// TestPowerupSuffixIsTyped: the suffix is part of the word, so free fire needs no
// special case and the mark is simply the last letter.
func TestPowerupSuffixIsTyped(t *testing.T) {
	st := powerState(t)
	st.Elapsed = 60
	st.Words = st.Words[:0]
	st.Words = append(st.Words, Word{Text: "RATIO.", Tier: TierNormal, Power: PowerFreeze, DotY: 20})

	for _, ch := range "RATIO" {
		if h := st.Type(byte(ch)); !h.Ok {
			t.Fatalf("%q missed", ch)
		}
	}
	if len(st.Words) != 1 {
		t.Fatal("word vanished before its suffix was typed")
	}
	if st.Frozen() {
		t.Error("effect fired before the word was finished")
	}
	h := st.Type('.')
	if !h.Ok || !h.Completed {
		t.Fatal("the suffix did not complete the word")
	}
	if h.Power != PowerFreeze {
		t.Errorf("hit reported power %q, want FREEZE", h.Power)
	}
	if !st.Frozen() {
		t.Error("FREEZE did not take effect")
	}
}

// TestInstantEffectsNameThemselves. A timed effect announces itself with its
// countdown. The instant ones had no readout anywhere: they fired, changed the
// field, and never said what they were, so the only way to learn a mark was to
// press it and infer from what happened.
func TestInstantEffectsNameThemselves(t *testing.T) {
	fire := func(p Power, suffix byte) *State {
		t.Helper()
		st := powerState(t)
		st.MaxWordsCap = 0
		st.Words = st.Words[:0]
		st.Words = append(st.Words, Word{
			Text: "RATIO" + string(suffix), Tier: TierNormal, Power: p, Typed: 5, DotY: 20,
		})
		if h := st.Type(suffix); !h.Completed {
			t.Fatalf("%q did not complete the word", suffix)
		}
		return st
	}

	for _, tc := range []struct {
		p      Power
		suffix byte
	}{{PowerRepair, '+'}, {PowerRewind, '-'}, {PowerBlast, '/'}} {
		st := fire(tc.p, tc.suffix)
		got, ok := st.FiredPower()
		if !ok || got != tc.p {
			t.Errorf("%v fired without naming itself: got %v, shown %v", tc.p, got, ok)
		}
		// And the flash expires, so it is naming what just happened.
		st.Elapsed += FiredSeconds + 0.1
		if _, ok := st.FiredPower(); ok {
			t.Errorf("%v is still named %.0fs later", tc.p, FiredSeconds+0.1)
		}
	}

	// FREEZE runs for a time, so it belongs to the countdown and not the flash.
	// Two readouts of one effect is one too many.
	st := fire(PowerFreeze, '.')
	if _, ok := st.FiredPower(); ok {
		t.Error("FREEZE took the instant flash as well as its countdown")
	}
	if fx := st.ActiveEffects(); len(fx) != 1 || fx[0].Power != PowerFreeze {
		t.Errorf("FREEZE is not in the running effects: %v", fx)
	}
}

// TestPunctuationCannotBeStolen: punctuation appears only on power-up words, so a
// suffix keypress is unambiguous under free fire by construction.
func TestPunctuationCannotBeStolen(t *testing.T) {
	st := powerState(t)
	st.Words = st.Words[:0]
	st.Words = append(st.Words,
		Word{Text: "SOURCE.", Tier: TierNormal, Power: PowerFreeze, DotY: 8, Typed: 6},
		Word{Text: "PEDANT", Tier: TierNormal, DotX: 200, DotY: 40}, // much lower
	)
	h := st.Type('.')
	if !h.Ok || h.Word != 0 {
		t.Errorf("'.' resolved to word %d; only the power-up word can want it", h.Word)
	}
}

func TestFreezeStopsMovement(t *testing.T) {
	st := powerState(t)
	st.Words = st.Words[:0]
	st.MaxWordsCap = 0
	st.Words = append(st.Words, Word{Text: "RATIO", Tier: TierNormal})
	st.freezeUntil = 1.0

	before := st.Words[0].DotY
	for i := 0; i < 20; i++ {
		st.Step(1.0 / 30)
	}
	if st.Words[0].DotY != before {
		t.Errorf("word moved %d dots while frozen", st.Words[0].DotY-before)
	}
	// The clock keeps running, so a freeze buys room without pausing the run.
	if st.Elapsed == 0 {
		t.Error("the clock stopped too")
	}
	for i := 0; i < 40; i++ {
		st.Step(1.0 / 30)
	}
	if st.Words[0].DotY == before {
		t.Error("word never resumed after the freeze expired")
	}
}

// TestSlowAppliesToTheSharedClock: like typo heat, the slow scales the global
// movement tick rather than each word's integer step count, so a 20% change
// registers instead of rounding away.
func TestSlowAppliesToTheSharedClock(t *testing.T) {
	st := powerState(t)
	w := Word{Text: "RATIO", Tier: TierNormal}
	perWord := st.StepEvery(&w)

	st.slowStack = 0.2
	if got := st.StepEvery(&w); got != perWord {
		t.Errorf("slow changed a word's step count (%d to %d); it belongs on the "+
			"shared clock, where a small change cannot round away", perWord, got)
	}
	if st.RowsPerSecond(&w) >= 1/(st.Tune.StepInterval*float64(perWord)*float64(M.DotsY)) {
		t.Error("reported speed did not drop")
	}
}

func TestShieldAbsorbsOneLanding(t *testing.T) {
	tune := DefaultTuning()
	tune.SpawnPause = 0
	st := NewState(78, 18, tune, NewRNG(2), []string{"RATIO"})
	st.MaxWordsCap = 0
	st.Shield = true
	st.Words = append(st.Words, Word{Text: "RATIO", Tier: TierNormal, DotY: 18 * M.DotsY})

	res := st.Step(1.0 / 30)
	if !res.Shielded {
		t.Fatal("landing was not shielded")
	}
	if res.LostLife || st.Lives != int(tune.Lives) {
		t.Errorf("shield did not save the life: lives %d", st.Lives)
	}
	if st.Shield {
		t.Error("shield was not consumed")
	}
}

func TestRepairAndRewind(t *testing.T) {
	st := powerState(t)
	st.Lives = 1
	st.applyPower(PowerRepair)
	if st.Lives != 2 {
		t.Errorf("REPAIR gave %d lives, want 2", st.Lives)
	}
	st.Lives = int(st.Tune.MaxLives)
	st.applyPower(PowerRepair)
	if st.Lives != int(st.Tune.MaxLives) {
		t.Error("REPAIR went past the cap")
	}

	st.Words = st.Words[:0]
	st.Words = append(st.Words, Word{Text: "RATIO", Tier: TierNormal, DotY: 40, Kick: 2})
	st.applyPower(PowerRewind)
	if st.Words[0].DotY != 40-int(st.Tune.RewindRows)*M.DotsY {
		t.Errorf("REWIND left the word at %d", st.Words[0].DotY)
	}
	if st.Words[0].Kick != 0 {
		t.Error("REWIND should drop a collision kick; nothing is beneath the word now")
	}
}

// TestOnePowerupAtATime: two markers on the field is unreadable, and stacking
// effects raises questions nobody needs answered.
func TestOnePowerupAtATime(t *testing.T) {
	st := powerState(t)
	st.Elapsed = 60
	for i := 0; i < 40; i++ {
		st.Spawn()
		live := 0
		for j := range st.Words {
			if st.Words[j].Power != PowerNone {
				live++
			}
		}
		if live > 1 {
			t.Fatalf("%d power-up words on screen", live)
		}
	}
}

func TestNoPowerupsBeforeTheirLevel(t *testing.T) {
	tune := DefaultTuning()
	tune.PowerupChance = 1
	tune.PowerupFromLevel = 3
	st := NewState(78, 18, tune, NewRNG(5), []string{"RATIO", "SOURCE", "CITATION"})
	st.MaxWordsCap = 6
	for i := 0; i < 40; i++ {
		st.Words = st.Words[:0]
		st.Spawn()
		if len(st.Words) > 0 && st.Words[0].Power != PowerNone {
			t.Fatalf("power-up at level %d, gate is %v", st.Level()+1, tune.PowerupFromLevel)
		}
	}
}

// TestMissedPowerupCostsNothing is what makes taking one a decision rather than
// an obligation. Power-ups ride the longest words on the field, so charging a
// life for ignoring one would be a punishment dressed as a reward.
func TestMissedPowerupCostsNothing(t *testing.T) {
	tune := DefaultTuning()
	tune.SpawnPause = 0
	st := NewState(78, 18, tune, NewRNG(4), []string{"RATIO"})
	st.MaxWordsCap = 0
	st.Words = append(st.Words, Word{
		Text: "OBFUSCATION.", Tier: TierNormal, Power: PowerFreeze,
		DotY: 18 * M.DotsY,
	})

	livesBefore := st.Lives
	res := st.Step(1.0 / 30)

	if res.PowerMissed != 1 {
		t.Errorf("PowerMissed = %d, want 1", res.PowerMissed)
	}
	if res.LostLife || st.Lives != livesBefore {
		t.Errorf("a missed power-up cost a life: %d -> %d", livesBefore, st.Lives)
	}
	if len(res.Landed) != 0 {
		t.Errorf("missed power-up reported as a landing: %v", res.Landed)
	}
	if len(st.Words) != 0 {
		t.Error("missed power-up stayed on the field")
	}
	if st.Frozen() {
		t.Error("the effect fired even though the word was never finished")
	}
	if st.Killer != "" {
		t.Error("a missed power-up was recorded as the killer word")
	}
}

// TestMissedOrdinaryWordStillCostsALife: only special words are free to ignore.
func TestMissedOrdinaryWordStillCostsALife(t *testing.T) {
	tune := DefaultTuning()
	tune.SpawnPause = 0
	st := NewState(78, 18, tune, NewRNG(4), []string{"RATIO"})
	st.MaxWordsCap = 0
	st.Words = append(st.Words, Word{Text: "RATIO", Tier: TierNormal, DotY: 18 * M.DotsY})

	res := st.Step(1.0 / 30)
	if !res.LostLife {
		t.Error("an ordinary word reached the floor without costing a life")
	}
	if st.Lives != int(tune.Lives)-1 {
		t.Errorf("lives %d, want %d", st.Lives, int(tune.Lives)-1)
	}
}
