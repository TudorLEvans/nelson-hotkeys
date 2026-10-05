package engine

import "math"

// Typo heat: mistakes make the field faster, and the effect decays.
//
// Every miss adds HeatPerMiss to a running value that bleeds away with a
// half-life. The whole field moves faster in proportion, so sloppiness is
// punished continuously rather than at one moment, and a clean patch of play
// earns the pressure back.
//
// This is the one thing in the game that responds to the player, and it is
// deliberately one-directional: heat can only ever make the game HARDER, never
// easier. The schedule is a clock precisely so that two runs of the same length
// are the same run and scores compare (see difficulty.go). Letting clean play dip
// below the schedule would break that, because the fastest run would no longer be
// the canonical one. Keeping heat additive means the clock is the best case and
// flawless typing traces it exactly, which is the reward.
//
// It is applied to the global movement tick rather than to each word's step
// count. Step counts are integers, so a 9% change often rounds to nothing at low
// speeds and saturates at high ones; scaling the shared clock is smooth at every
// speed and keeps every word in lockstep.

// TypoHeat is the current penalty, 0 for clean play. 0.25 means everything falls
// 25% faster.
func (s *State) TypoHeat() float64 { return float64(s.typoHeat) }

// addTypoHeat records a miss.
func (s *State) addTypoHeat() {
	s.typoHeat += float32(s.Tune.HeatPerMiss)
	if max := float32(s.Tune.HeatMax); s.typoHeat > max {
		s.typoHeat = max
	}
}

// decayTypoHeat bleeds heat away over dt, on a half-life so the shape is the same
// whatever the frame rate.
func (s *State) decayTypoHeat(dt float32) {
	if s.typoHeat <= 0 {
		s.typoHeat = 0
		return
	}
	hl := s.Tune.HeatHalfLife
	if hl <= 0 {
		s.typoHeat = 0
		return
	}
	s.typoHeat *= float32(math.Exp2(-float64(dt) / hl))
	if s.typoHeat < 0.001 {
		s.typoHeat = 0
	}
}

// heatFactor is what the fall speed is multiplied by.
func (s *State) heatFactor() float64 { return 1 + s.TypoHeat() }

// EffectiveSpeedNow is the speed the player actually experiences: the schedule
// plus whatever their mistakes have added. FallSpeedNow is the clean schedule
// value, and the gap between the two is what the speed trails draw.
func (s *State) EffectiveSpeedNow() float64 {
	return s.FallSpeedNow() * s.heatFactor()
}

// HeatEquilibrium is the heat a player sustains at a given miss rate, for tuning.
// Exponential decay against a constant input settles at rate * per_miss *
// halflife / ln 2, which is how the defaults were chosen rather than by feel:
// about +3% for a 95%-accurate typist at four keys a second, about +14% at 80%,
// and the cap for a masher.
//
// An earlier set ran at roughly twice these numbers and was too heavy in play. A
// typo should be a nudge you notice, not a change of gear.
func (t *Tuning) HeatEquilibrium(missesPerSecond float64) float64 {
	if t.HeatHalfLife <= 0 {
		return 0
	}
	v := missesPerSecond * t.HeatPerMiss * t.HeatHalfLife / math.Ln2
	return math.Min(v, t.HeatMax)
}
