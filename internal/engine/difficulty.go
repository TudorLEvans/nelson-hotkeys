package engine

import "math"

// Difficulty runs on a clock, not on the player.
//
// It is deliberately not adaptive. An earlier design advanced the level once per
// ten words destroyed, which meant a fast typist reached level 14 while a slow one
// reached level 6 in the same time. That reads as fair but it makes difficulty a
// function of the player, so two runs of the same length are not the same run and
// scores stop comparing. A clock gives everyone the identical sequence of
// pressure, which is what makes "I got to four minutes" worth saying.
//
// Two properties of the shape matter:
//
// Progression is CONTINUOUS. Speed and spawn interval come from a real number,
// not a step counter, so they creep up every frame instead of jumping at level
// boundaries. A discrete jump is a jolt the player has to re-read the field for,
// and it makes the last second of a level easier than the first second of the next
// for no reason. The integer level is kept and displayed, because a player wants
// to know which rung they are on, but nothing in the simulation reads it.
//
// Levels get LONGER as they go. Level 1 lasts 8 seconds, level 20 about 46. The
// bottom of the range is trivial for anyone who can already type, so spending
// equal time there wastes the opening.

// LevelF is the continuous progression value: 0 at the start, 1.0 when level 1 is
// complete, fractional in between.
//
// Level durations grow linearly, so level k lasts LevelBase*(1 + k*LevelGrowth)
// and the cumulative time to finish n levels is quadratic in n. This is the
// closed-form inverse. Closed form rather than an accumulating counter is what
// keeps the value continuous and keeps a seeded run reproducible whatever the
// frame timing.
func (s *State) LevelF() float64 {
	t := float64(s.Elapsed)
	if t <= 0 {
		return 0
	}
	p0 := s.Tune.LevelBase
	if p0 <= 0 {
		return 0
	}
	g := s.Tune.LevelGrowth
	if g <= 0 {
		return t / p0 // constant-length levels
	}
	b := 1 - g/2
	return (-b + math.Sqrt(b*b+2*g*t/p0)) / g
}

// TimeForLevel is the inverse of LevelF: when a given level begins. Used by
// --level, which moves the clock rather than faking a level so that everything
// derived from the clock stays consistent.
func TimeForLevel(t *Tuning, level float64) float64 {
	if level <= 0 || t.LevelBase <= 0 {
		return 0
	}
	g := t.LevelGrowth
	if g <= 0 {
		return level * t.LevelBase
	}
	return t.LevelBase * (level + g*level*(level-1)/2)
}

// Level is the whole level. The HUD shows this plus one.
func (s *State) Level() int { return int(s.LevelF()) }

// LevelProgress is how far through the current level, 0 to 1.
func (s *State) LevelProgress() float64 {
	f := s.LevelF()
	return f - math.Floor(f)
}

// rampCurve is progression raised to an exponent. Below 1 it is concave: most of
// the increase lands early and later levels add little.
//
// Both ramps use it, and both were linear first. Linear is the wrong shape at
// both ends. Early on it lingers on three and four letter words, which only a
// beginner finds difficult. Later it walks steadily into the ceiling, so the
// interesting middle of the curve passes too quickly and then stops changing.
func rampCurve(level, exponent float64) float64 {
	if level <= 0 {
		return 0
	}
	if exponent <= 0 {
		return 1
	}
	return math.Pow(level, exponent)
}

// FallSpeedNow is the current fall speed in rows per second. It applies to every
// word on screen, not only newly spawned ones, so the ramp accelerates the whole
// field; see StepEvery.
func (s *State) FallSpeedNow() float64 {
	return s.fallSpeedAt(s.LevelF())
}

func (s *State) fallSpeedAt(lvl float64) float64 {
	// Concave term for the shape, plus a small linear term so the curve never goes
	// truly flat. Without the linear term a square-root ramp needs over half an
	// hour to reach a speed anyone would lose to, which makes "no ceiling"
	// indistinguishable from a ceiling.
	v := s.Tune.FallSpeed +
		s.Tune.SpeedGrowth*rampCurve(lvl, s.Tune.SpeedCurve) +
		s.Tune.SpeedLate*lvl
	if s.Tune.MaxFallSpeed > 0 {
		v = math.Min(v, s.Tune.MaxFallSpeed)
	}
	return v * s.Tune.SpeedScale
}

// SpawnIntervalNow is the gap between spawns, in seconds. Geometric in the
// continuous level, so it shrinks smoothly rather than in steps.
func (s *State) SpawnIntervalNow() float64 {
	v := s.Tune.SpawnInterval * math.Pow(s.Tune.IntervalFactor, s.LevelF())
	return math.Max(v, s.Tune.MinSpawnInterval)
}

// MaxWordsNow is how many words may be on screen. Necessarily discrete, and
// capped by what the playfield can hold without overlap, which is a property of
// the terminal size rather than of the difficulty.
func (s *State) MaxWordsNow() int {
	n := int(s.Tune.StartWords)
	if s.Tune.StepsPerExtraWord > 0 {
		n += int(s.LevelF() / s.Tune.StepsPerExtraWord)
	}
	if n > s.MaxWordsCap {
		n = s.MaxWordsCap
	}
	// Clamp up only when the playfield can hold something. An explicit cap of zero
	// means zero: it is what the too-small-terminal path and the tests use, and
	// silently spawning one word anyway made "no words" impossible to ask for.
	if n < 1 && s.MaxWordsCap >= 1 {
		n = 1
	}
	if n < 0 {
		n = 0
	}
	return n
}
