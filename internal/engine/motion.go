package engine

// Motion is quantised, deliberately.
//
// A word can only be drawn at whole dot positions, and a cell holds 2 dots
// vertically with block characters or 4 with braille. Over the 18 playable rows
// of the 80x24 minimum that is 36 or 72 positions, so a word crossing the field
// in 15 seconds moves once every 0.42 or 0.21 seconds however the code is
// written. Sub-dot placement is not renderable at any font size.
//
// The first version let every word hold its own fractional position and cross
// those boundaries whenever it happened to. Each word then jumped at a different
// moment, which read as words animating independently and as stutter.
//
// So positions are integer dots and every word advances on one shared clock. A
// word's speed is how many global ticks pass between its own advances, so slow
// and fast words still land on the same grid and the field moves as one. The
// granularity then reads as a deliberate cadence rather than as jitter, which is
// what stepped sprite movement has always looked like.

// StepEvery is how many global ticks pass between a word's advances. Lower is
// faster.
//
// It is read from the schedule every tick rather than fixed when the word
// spawned, so a level change speeds up the whole field at once, including words
// already falling.
//
// Two things can make a single word faster than the schedule: chip acceleration,
// which scales with letters destroyed, and a collision kick inherited from a
// faster word above. Both are floors rather than replacements, so the schedule
// can still overtake them later.
func (s *State) StepEvery(w *Word) int {
	e := BaseStepFor(s.FallSpeedNow(), s.Tune)
	if w.Kick > 0 && w.Kick < e {
		e = w.Kick
	}
	if w.Typed > 0 {
		e = int(float64(e)/(1+s.Tune.ChipAccel*float64(w.Typed)) + 0.5)
	}
	if e < 1 {
		e = 1
	}
	return e
}

// BaseStepFor converts a human-facing speed in rows per second into a tick
// interval. fall_speed stays the knob people reason about; the quantisation is
// an implementation detail of how it is drawn.
func BaseStepFor(rowsPerSecond float64, t *Tuning) int {
	if rowsPerSecond <= 0 || t.StepInterval <= 0 {
		return 1
	}
	dotsPerSecond := rowsPerSecond * float64(M.DotsY)
	e := int(1/(t.StepInterval*dotsPerSecond) + 0.5)
	if e < 1 {
		e = 1
	}
	return e
}

// RowsPerSecond is a word's current speed, for the HUD and the speed trail. It
// includes typo heat and any slow, so the trails visibly grow when the player is
// being sloppy and shrink when the field is slowed.
func (s *State) RowsPerSecond(w *Word) float64 {
	e := s.StepEvery(w)
	if e < 1 || s.Tune.StepInterval <= 0 {
		return 0
	}
	return s.heatFactor() * (1 - s.SlowFactor()) /
		(s.Tune.StepInterval * float64(e) * float64(M.DotsY))
}

// spans reports whether two words overlap horizontally. Words are never allowed
// to overlap on screen: two block-glyph words on the same cells are unreadable,
// and readability is the whole game.
func spansOverlap(a, b *Word) bool {
	ax0, ax1 := a.DotX, a.DotX+a.WidthDots()-1
	bx0, bx1 := b.DotX, b.DotX+b.WidthDots()-1
	return ax0 <= bx1 && bx0 <= ax1
}

// resolveCollisions keeps words from ever interpenetrating.
//
// When a word catches the word below it, the lower one inherits the faster
// speed. That is not a patch on top of a separate anti-overlap check: once the
// lower word is at least as fast, the two cannot close on each other again, so
// the guarantee falls out of the mechanic. It also gives chipping a visible
// cost, since accelerating a word kicks whatever is under it into gear, and it
// is not chip-specific: a fast word catching a slow one behaves the same way.
func (s *State) resolveCollisions() {
	// Process lowest first so a stack settles from the floor upward in one pass.
	order := make([]int, len(s.Words))
	for i := range order {
		order[i] = i
	}
	for i := 1; i < len(order); i++ {
		for j := i; j > 0 && s.Words[order[j]].DotY > s.Words[order[j-1]].DotY; j-- {
			order[j], order[j-1] = order[j-1], order[j]
		}
	}

	for a := 0; a < len(order); a++ {
		lower := &s.Words[order[a]]
		for b := a + 1; b < len(order); b++ {
			upper := &s.Words[order[b]]
			if !spansOverlap(lower, upper) {
				continue
			}
			if upper.BottomDot() <= lower.DotY {
				continue // already clear
			}
			// Rest the upper word on top of the lower one, and pass the speed
			// down so they travel together rather than merging.
			upper.DotY = lower.DotY - upper.HeightDots()
			if ue := s.StepEvery(upper); ue < s.StepEvery(lower) {
				lower.Kick = ue
			}
		}
	}
}

// placeX picks a horizontal position for a new word that does not overlap
// anything already near the top of the field. Without this, words spawn on top
// of each other and the field is unreadable from the first second.
//
// Returns false when there is no room, in which case the spawn is skipped and
// retried on the next interval rather than forced.
func (s *State) placeX(text string) (int, bool) {
	span := s.PlayW*M.DotsX - TextWidthDots(text)
	if span < 0 {
		return 0, false
	}

	// Only words still in the upper part of the field can collide with a spawn.
	entryDepth := 3 * GlyphHeightDots
	var blockers []*Word
	for i := range s.Words {
		if s.Words[i].DotY < entryDepth {
			blockers = append(blockers, &s.Words[i])
		}
	}

	probe := Word{Text: text, Tier: TierNormal}
	for attempt := 0; attempt < 32; attempt++ {
		probe.DotX = s.Rng.Intn(span + 1)
		clear := true
		for _, b := range blockers {
			if spansOverlap(&probe, b) {
				clear = false
				break
			}
		}
		if clear {
			return probe.DotX, true
		}
	}
	return 0, false
}
