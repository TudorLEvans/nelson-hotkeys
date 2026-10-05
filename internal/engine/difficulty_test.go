package engine

import "testing"

func schedState(t *testing.T) *State {
	t.Helper()
	st := NewState(78, 18, DefaultTuning(), NewRNG(1), []string{"RATIO"})
	st.MaxWordsCap = 6
	return st
}

// TestScheduleIsDrivenByTheClock is the requested behaviour: difficulty advances
// on elapsed time and on nothing else.
func TestScheduleIsDrivenByTheClock(t *testing.T) {
	st := schedState(t)
	if st.Level() != 0 {
		t.Errorf("level %d at the start, want 0", st.Level())
	}
	st.Elapsed = float32(st.Tune.LevelBase) - 0.01
	if st.Level() != 0 {
		t.Errorf("level %d just before the first boundary, want 0", st.Level())
	}
	st.Elapsed = float32(st.Tune.LevelBase) + 0.01
	if st.Level() != 1 {
		t.Errorf("level %d just after the first boundary, want 1", st.Level())
	}
}

// TestLevelsGetLonger is the requested pacing change: the bottom of the range is
// trivial for anyone who can type, so early levels are short and later ones are
// not.
func TestLevelsGetLonger(t *testing.T) {
	st := schedState(t)

	// Time at which each level begins.
	start := func(n int) float64 {
		lo, hi := 0.0, 100000.0
		for i := 0; i < 60; i++ {
			mid := (lo + hi) / 2
			st.Elapsed = float32(mid)
			if st.Level() < n {
				lo = mid
			} else {
				hi = mid
			}
		}
		return hi
	}

	var durations []float64
	for n := 1; n <= 12; n++ {
		durations = append(durations, start(n+1)-start(n))
	}
	for i := 1; i < len(durations); i++ {
		if durations[i] <= durations[i-1] {
			t.Errorf("level %d lasts %.1fs, level %d lasts %.1fs; levels must get longer",
				i+1, durations[i-1], i+2, durations[i])
		}
	}
	t.Logf("level 1 lasts %.1fs, level 12 lasts %.1fs; level 5 begins at %.0fs, level 10 at %.0fs",
		durations[0], durations[len(durations)-1], start(5), start(10))

	if durations[0] > 10 {
		t.Errorf("level 1 lasts %.1fs; the opening should be short", durations[0])
	}
}

// TestProgressionIsContinuous is the other requested change. Speed and spawn
// interval must creep every frame, not jump at level boundaries: a jump is a jolt
// the player has to re-read the field for, and it makes the last second of a level
// easier than the first second of the next for no reason.
func TestProgressionIsContinuous(t *testing.T) {
	st := schedState(t)

	// Straddle a level boundary and check nothing steps.
	boundary := float32(st.Tune.LevelBase)
	st.Elapsed = boundary - 0.05
	beforeSpeed, beforeGap := st.FallSpeedNow(), st.SpawnIntervalNow()
	st.Elapsed = boundary + 0.05
	afterSpeed, afterGap := st.FallSpeedNow(), st.SpawnIntervalNow()

	if d := afterSpeed - beforeSpeed; d > 0.03 {
		t.Errorf("speed jumped %.3f across a level boundary; it should creep", d)
	}
	if d := beforeGap - afterGap; d > 0.03 {
		t.Errorf("spawn interval jumped %.3f across a level boundary; it should creep", d)
	}
	if afterSpeed <= beforeSpeed {
		t.Error("speed did not increase at all")
	}

	// And it must be strictly increasing frame to frame, not flat between levels.
	prev := 0.0
	for f := 0; f < 600; f++ {
		st.Elapsed = float32(f) / 30
		v := st.FallSpeedNow()
		if f > 0 && v < prev {
			t.Fatalf("frame %d: speed fell from %.4f to %.4f", f, prev, v)
		}
		prev = v
	}
	// Over 20 seconds it should have moved a long way, given short early levels.
	st.Elapsed = 0
	base := st.FallSpeedNow()
	st.Elapsed = 20
	if st.FallSpeedNow() < base*1.3 {
		t.Errorf("only %.2f -> %.2f over 20s; the early ramp is too slow",
			base, st.FallSpeedNow())
	}
}

// TestScheduleIgnoresPlayerPerformance is the point of the change. A player who
// destroys a hundred words must be at exactly the same difficulty as one who
// destroys none, given the same clock.
func TestScheduleIgnoresPlayerPerformance(t *testing.T) {
	idle := schedState(t)
	busy := schedState(t)
	busy.WordsDestroyed = 100
	busy.Score = 5000
	busy.Hits = 700
	busy.BestStreak = 90

	idle.Elapsed, busy.Elapsed = 95, 95
	if idle.Level() != busy.Level() {
		t.Errorf("levels differ: idle %d, busy %d", idle.Level(), busy.Level())
	}
	if idle.FallSpeedNow() != busy.FallSpeedNow() {
		t.Errorf("speeds differ: idle %.2f, busy %.2f", idle.FallSpeedNow(), busy.FallSpeedNow())
	}
	if idle.SpawnIntervalNow() != busy.SpawnIntervalNow() {
		t.Errorf("intervals differ: idle %.2f, busy %.2f",
			idle.SpawnIntervalNow(), busy.SpawnIntervalNow())
	}
	if idle.MaxWordsNow() != busy.MaxWordsNow() {
		t.Errorf("concurrency differs: idle %d, busy %d", idle.MaxWordsNow(), busy.MaxWordsNow())
	}
}

// TestSpeedRisesAndCaps: monotonic, and bounded so the game stays a game.
func TestSpeedRisesAndCaps(t *testing.T) {
	st := schedState(t)
	prev := 0.0
	for lvl := 0; lvl < 200; lvl++ {
		st.Elapsed = float32(float64(lvl) * st.Tune.LevelBase)
		v := st.FallSpeedNow()
		if v < prev {
			t.Fatalf("level %d: speed fell from %.2f to %.2f", lvl, prev, v)
		}
		if v > st.Tune.MaxFallSpeed*st.Tune.SpeedScale+1e-9 {
			t.Fatalf("level %d: speed %.2f exceeds the cap %.2f", lvl, v, st.Tune.MaxFallSpeed)
		}
		prev = v
	}
	if prev < st.Tune.MaxFallSpeed*st.Tune.SpeedScale-1e-9 {
		t.Errorf("speed topped out at %.2f, never reached the cap %.2f", prev, st.Tune.MaxFallSpeed)
	}
}

// TestSpawnIntervalFallsAndFloors: words must arrive more often, down to a floor.
func TestSpawnIntervalFallsAndFloors(t *testing.T) {
	st := schedState(t)
	prev := 0.0
	for lvl := 0; lvl < 200; lvl++ {
		st.Elapsed = float32(float64(lvl) * st.Tune.LevelBase)
		v := st.SpawnIntervalNow()
		if lvl > 0 && v > prev {
			t.Fatalf("level %d: interval rose from %.2f to %.2f", lvl, prev, v)
		}
		if v < st.Tune.MinSpawnInterval-1e-9 {
			t.Fatalf("level %d: interval %.3f below the floor %.2f", lvl, v, st.Tune.MinSpawnInterval)
		}
		prev = v
	}
	if prev > st.Tune.MinSpawnInterval+1e-9 {
		t.Errorf("interval bottomed at %.3f, never reached the floor %.2f", prev, st.Tune.MinSpawnInterval)
	}
}

// TestConcurrencyGrowsWithinThePlayfield: more words over time, but never more
// than the terminal can show without them overlapping.
func TestConcurrencyGrowsWithinThePlayfield(t *testing.T) {
	st := schedState(t)
	st.MaxWordsCap = 4
	if got := st.MaxWordsNow(); got != int(st.Tune.StartWords) {
		t.Errorf("start concurrency %d, want %v", got, st.Tune.StartWords)
	}
	st.Elapsed = float32(levelStartTime(st, int(st.Tune.StepsPerExtraWord)))
	if got := st.MaxWordsNow(); got != int(st.Tune.StartWords)+1 {
		t.Errorf("after one increase: %d, want %v", got, st.Tune.StartWords+1)
	}
	st.Elapsed = 100000
	if got := st.MaxWordsNow(); got != st.MaxWordsCap {
		t.Errorf("late concurrency %d, want the playfield cap %d", got, st.MaxWordsCap)
	}
}

// TestLiveWordsTakeTheNewSpeed: a schedule step speeds up the whole field, not
// only words spawned after it. Otherwise a level change has no effect until
// everything on screen has been cleared, which makes the ramp feel laggy.
func TestLiveWordsTakeTheNewSpeed(t *testing.T) {
	st := schedState(t)
	st.Spawn()
	w := &st.Words[0]
	before := st.StepEvery(w)

	st.Elapsed = float32(st.Tune.LevelBase) * 10
	after := st.StepEvery(w)
	if after >= before {
		t.Errorf("a live word still steps every %d ticks after ten levels, was %d; "+
			"the schedule must accelerate words already falling", after, before)
	}
	if st.RowsPerSecond(w) <= 0 {
		t.Error("speed came out as zero")
	}
}

// TestCollisionKickIsAFloorNotAnOverride: a word kicked by a faster one above it
// keeps that speed, but the schedule can still overtake it later.
func TestCollisionKickIsAFloorNotAnOverride(t *testing.T) {
	st := schedState(t)
	st.Spawn()
	w := &st.Words[0]

	scheduled := st.StepEvery(w)
	w.Kick = scheduled - 1 // kicked faster than the schedule
	if got := st.StepEvery(w); got != scheduled-1 {
		t.Errorf("kicked word steps every %d, want %d", got, scheduled-1)
	}

	// Much later, the schedule is faster than the kick and should win.
	st.Elapsed = float32(st.Tune.LevelBase) * 40
	if got := st.StepEvery(w); got > w.Kick {
		t.Errorf("schedule gave %d, slower than the stale kick %d", got, w.Kick)
	}
}

// TestScheduleRampIsPlayableEarly guards against a curve that is brutal from the
// first minute; the first level must still leave a beginner time to read a word.
func TestScheduleRampIsPlayableEarly(t *testing.T) {
	st := schedState(t)
	if v := st.FallSpeedNow(); v > 1.5 {
		t.Errorf("level 1 speed %.2f is too fast to start on", v)
	}
	st.Elapsed = float32(st.Tune.LevelBase) * 3 // one minute in
	if v := st.FallSpeedNow(); v < 1.3 {
		t.Errorf("after a minute the speed is still %.2f; the ramp is too flat", v)
	}
}

// TestUncappedSpeedEventuallyEndsTheRun documents what max_fall_speed = 0 means.
// With a ceiling the schedule stops biting and a strong player survives forever;
// without one, every run is eventually unwinnable. Both are legitimate, so the
// choice is a tunable rather than a hardcoded curve.
func TestUncappedSpeedEventuallyEndsTheRun(t *testing.T) {
	st := schedState(t)
	st.Tune.MaxFallSpeed = 0

	// A sustained typist manages about 5 characters a second, so a six-letter word
	// needs roughly 1.2 s of typing plus time to read it. Call the game beaten when
	// the field crosses in under 2 s.
	// Walk elapsed time, not level index: levels get longer, so a level count no
	// longer maps to a duration.
	const playRows = 18
	const hardCross = 2.0
	for mins := 0; mins < 240; mins++ {
		st.Elapsed = float32(mins * 60)
		if playRows/st.FallSpeedNow() < hardCross {
			t.Logf("uncapped: crossing under %.0fs at %d minutes, level %d",
				hardCross, mins, st.Level()+1)
			return
		}
	}
	t.Error("with no speed ceiling the game never becomes unwinnable; the linear " +
		"tail (speed_late) may have been removed")
}

// TestCappedSpeedPlateaus is the same fact from the other side, so the tradeoff
// is written down rather than discovered.
func TestCappedSpeedPlateaus(t *testing.T) {
	st := schedState(t)
	st.Elapsed = 60 * 60
	a := st.FallSpeedNow()
	st.Elapsed = 60 * 60 * 2
	if b := st.FallSpeedNow(); b != a {
		t.Errorf("capped speed still changing between one and two hours: %.2f then %.2f", a, b)
	}
}

// levelStartTime finds when a level begins, by search, since the level function
// is a closed-form inverse rather than a counter.
func levelStartTime(st *State, n int) float64 {
	lo, hi := 0.0, 100000.0
	saved := st.Elapsed
	defer func() { st.Elapsed = saved }()
	for i := 0; i < 60; i++ {
		mid := (lo + hi) / 2
		st.Elapsed = float32(mid)
		if st.Level() < n {
			lo = mid
		} else {
			hi = mid
		}
	}
	return hi
}
