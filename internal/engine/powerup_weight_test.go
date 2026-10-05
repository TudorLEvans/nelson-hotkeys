package engine

import (
	"sort"
	"testing"
)

// TestRarityTracksStrength is the requested shape: mild effects often enough to
// plan around, strong ones rare enough to be a moment.
func TestRarityTracksStrength(t *testing.T) {
	tune := DefaultTuning()
	tune.PowerupFromLevel = 0
	tune.PowerupChance = 1
	tune.BombChance = 0
	st := NewState(78, 18, tune, NewRNG(77), []string{
		"RATIO", "SOURCE", "PEDANT", "CITATION", "STRAWMAN", "ANECDOTAL",
		"CONSENSUS", "DEFLECTION", "OBFUSCATION",
	})
	st.MaxWordsCap = 6

	counts := map[Power]int{}
	const draws = 20000
	for i := 0; i < draws; i++ {
		st.Words = st.Words[:0]
		if p := st.pickPowerup(); p != PowerNone {
			counts[p]++
		}
	}

	// Every power must actually be reachable.
	for _, p := range Powers {
		if counts[p] == 0 {
			t.Errorf("%v never appeared in %d draws", p, draws)
		}
	}

	// Observed frequency must follow the declared weights.
	type row struct {
		p    Power
		n    int
		want float64
	}
	var rows []row
	total := 0.0
	for _, p := range Powers {
		total += p.weight(tune)
	}
	for _, p := range Powers {
		rows = append(rows, row{p, counts[p], p.weight(tune) / total})
	}
	for _, r := range rows {
		got := float64(r.n) / draws
		if got < r.want*0.85 || got > r.want*1.15 {
			t.Errorf("%v drawn %.1f%% of the time, want about %.1f%%",
				r.p, got*100, r.want*100)
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	for _, r := range rows {
		t.Logf("%-9v %5.1f%%", r.p, float64(r.n)/draws*100)
	}

	// And the ordering that matters: the mild effect is common, the one that hands
	// back a life is rare.
	if counts[PowerBlast] <= counts[PowerRepair]*2 {
		t.Errorf("BLAST %d vs REPAIR %d; the mild effect should be far commoner",
			counts[PowerBlast], counts[PowerRepair])
	}
}

// TestTheSetIsFour records the cut. Nine power-ups meant nine marks, and six of
// them said nothing about what they did, so the player was memorising a table
// instead of reading the field.
//
// The four that stayed are the ones whose mark can be guessed at. Anything added
// later has to clear the same bar, which is what this test is for.
func TestTheSetIsFour(t *testing.T) {
	want := map[Power]string{
		PowerFreeze: "FREEZE",
		PowerRewind: "REWIND",
		PowerBlast:  "BLAST",
		PowerRepair: "REPAIR",
	}
	if len(Powers) != len(want) {
		t.Errorf("%d spawnable power-ups, want %d: %v", len(Powers), len(want), Powers)
	}
	for _, p := range Powers {
		name, ok := want[p]
		if !ok {
			t.Errorf("%q (%v) is spawnable and should not be", rune(p), p)
			continue
		}
		if p.String() != name {
			t.Errorf("%q is %q, want %q", rune(p), p, name)
		}
	}

	// Every mark must be typeable, and every typeable mark must be a power-up.
	for _, p := range Powers {
		if !Typeable(rune(p)) {
			t.Errorf("%q is a power-up mark the input layer would drop", rune(p))
		}
	}
	if len(powerSuffixes) != len(Powers) {
		t.Errorf("powerSuffixes is %q but there are %d powers; a mark no word carries "+
			"is a key that can only ever be a typo", powerSuffixes, len(Powers))
	}
	for _, ch := range powerSuffixes {
		found := false
		for _, p := range Powers {
			found = found || rune(p) == ch
		}
		if !found {
			t.Errorf("%q is typeable but no power-up uses it", ch)
		}
	}
}

// TestCutPowersStayCut. STEADY, AUTOCHIP and CLEAR are gone outright; SLOW and
// SHIELD survive only as chain milestones, so they must not come back as spawns.
func TestCutPowersStayCut(t *testing.T) {
	for _, name := range []string{"STEADY", "AUTOCHIP", "CLEAR", "SLOW", "SHIELD"} {
		for _, p := range Powers {
			if p.String() == name {
				t.Errorf("%s is spawnable again", name)
			}
		}
	}
	for _, ch := range ",;'=[]" {
		if Typeable(ch) {
			t.Errorf("%q is still typeable; nothing on the field can want it, so it "+
				"can only ever register as a typo", ch)
		}
	}
	// Chip acceleration is unconditional now that nothing suppresses it.
	st := NewState(78, 18, DefaultTuning(), NewRNG(1), []string{"RATIO"})
	clean := Word{Text: "OBFUSCATION", Tier: TierNormal}
	eaten := Word{Text: "OBFUSCATION", Tier: TierNormal, Typed: 8}
	if st.StepEvery(&eaten) >= st.StepEvery(&clean) {
		t.Error("an eaten word is not falling faster than an untouched one")
	}
}

// TestChainStillPaysInSlowAndShield: the effects outlived the power-ups that
// used to grant them, and the milestones are now the only source.
func TestChainStillPaysInSlowAndShield(t *testing.T) {
	tune := DefaultTuning()
	st := NewState(78, 18, tune, NewRNG(3), []string{"RATIO"})
	st.MaxWordsCap = 0

	if r := st.chainRewardFor(int(tune.ChainSlowAt)); r == nil {
		t.Fatalf("no reward at chain %v", tune.ChainSlowAt)
	}
	if !st.Slowed() {
		t.Error("the chain milestone did not slow the field")
	}
	if got, want := st.SlowFactor(), tune.SlowPerPickup; got < want-1e-6 || got > want+1e-6 {
		t.Errorf("milestone gave %.2f of slow, want %.2f", got, want)
	}

	if r := st.chainRewardFor(int(tune.ChainShieldAt)); r == nil {
		t.Fatalf("no reward at chain %v", tune.ChainShieldAt)
	}
	if !st.Shield {
		t.Error("the chain milestone did not grant a shield")
	}
}

// TestSlowStacksAndCaps. A flat halving was too strong; one grant should be a
// nudge, and the old effect should have to be earned by collecting several.
func TestSlowStacksAndCaps(t *testing.T) {
	tune := DefaultTuning()
	st := NewState(78, 18, tune, NewRNG(5), []string{"RATIO"})
	st.MaxWordsCap = 0

	if st.SlowFactor() != 0 {
		t.Fatalf("slowed before any pickup: %v", st.SlowFactor())
	}

	st.applyPower(PowerSlow)
	one := st.SlowFactor()
	if d := one - tune.SlowPerPickup; d > 1e-6 || d < -1e-6 {
		t.Errorf("one pickup gives %.2f, want %.2f", one, tune.SlowPerPickup)
	}
	if one >= 0.5 {
		t.Errorf("one pickup already halves the speed (%.2f); it should be a nudge", one)
	}

	st.applyPower(PowerSlow)
	if got := st.SlowFactor(); got <= one {
		t.Errorf("a second pickup did not stack: %.2f then %.2f", one, got)
	}

	for i := 0; i < 20; i++ {
		st.applyPower(PowerSlow)
	}
	if got := st.SlowFactor(); got > tune.SlowMax+1e-6 {
		t.Errorf("stack reached %.2f, cap is %.2f", got, tune.SlowMax)
	}
	if got := st.SlowFactor(); got < tune.SlowMax-1e-6 {
		t.Errorf("stack only reached %.2f, cap is %.2f", got, tune.SlowMax)
	}
}

// TestSlowActuallySlowsTheField, and by the declared proportion.
func TestSlowActuallySlowsTheField(t *testing.T) {
	fall := func(slow float32) int {
		st := NewState(78, 18, DefaultTuning(), NewRNG(5), []string{"RATIO"})
		st.MaxWordsCap = 0
		st.Words = append(st.Words, Word{Text: "RATIO", Tier: TierNormal})
		for i := 0; i < 120; i++ {
			st.slowStack = slow // hold it against the drain
			st.Step(1.0 / 30)
		}
		return st.Words[0].DotY
	}

	full, nudged, halved := fall(0), fall(0.2), fall(0.5)
	t.Logf("4 seconds of falling: full speed %d dots, -20%% %d, -50%% %d",
		full, nudged, halved)

	if nudged >= full {
		t.Error("a 20% slow did not slow anything")
	}
	if halved >= nudged {
		t.Error("the cap was not slower than one pickup")
	}
	// -50% should be about half the distance; quantisation makes it approximate.
	// Approximate: movement is integer dots on a quantised tick, so a window this
	// short loses part of a step to the remainder.
	if r := float64(halved) / float64(full); r < 0.38 || r > 0.60 {
		t.Errorf("at the cap the field moved %.2f of full speed, want about 0.5", r)
	}
}

// TestSlowDrainsAway at a rate that makes the status countdown honest.
func TestSlowDrainsAway(t *testing.T) {
	tune := DefaultTuning()
	st := NewState(78, 18, tune, NewRNG(5), []string{"RATIO"})
	st.MaxWordsCap = 0
	st.applyPower(PowerSlow)

	left := st.SlowSecondsLeft()
	if d := float64(left) - tune.SlowSeconds; d > 0.1 || d < -0.1 {
		t.Errorf("one pickup reports %.1fs left, want %.1f", left, tune.SlowSeconds)
	}

	for i := 0; i < int(tune.SlowSeconds*30)+2; i++ {
		st.Step(1.0 / 30)
	}
	if st.Slowed() {
		t.Errorf("still slowed after %.0fs: %.3f", tune.SlowSeconds, st.SlowFactor())
	}

	// Two pickups last twice as long, so each is worth its own slice of time.
	st.applyPower(PowerSlow)
	st.applyPower(PowerSlow)
	if got := st.SlowSecondsLeft(); float64(got) < tune.SlowSeconds*1.8 {
		t.Errorf("two pickups report %.1fs, want about %.1f", got, tune.SlowSeconds*2)
	}
}

// TestRollbackIsGone: removed as too strong and, being the only thing that
// touched the difficulty clock, hard for a player to read. BLAST took its place
// as the effect that does something a player can see happen, and later took
// CLEAR's '/' when the set was cut to four.
func TestRollbackIsGone(t *testing.T) {
	for _, p := range Powers {
		if p.String() == "ROLLBACK" {
			t.Error("ROLLBACK is still in the spawnable set")
		}
	}
	if got := Power('/').String(); got != "BLAST" {
		t.Errorf("'/' maps to %q, want BLAST", got)
	}
	// Nothing may read the difficulty clock's old discount any more.
	st := NewState(78, 18, DefaultTuning(), NewRNG(1), []string{"RATIO"})
	st.MaxWordsCap = 0
	st.Elapsed = 200
	before := st.LevelF()
	for _, p := range Powers {
		st.applyPower(p)
	}
	if st.LevelF() != before {
		t.Errorf("a power-up moved the difficulty clock: %.3f then %.3f", before, st.LevelF())
	}
}
