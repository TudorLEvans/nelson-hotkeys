package engine

// Word choice has two jobs: pick from the length band the schedule currently
// calls for, and avoid handing the player the same word twice in quick
// succession.
//
// Uniform random with replacement is not good enough for the second. With a
// 200-word pack and a spawn every two seconds, the birthday paradox alone puts a
// repeat inside the first dozen spawns, and repeats read as a bug even when the
// list is large. So a short history of what has already been used is kept and
// re-drawn against.

// WordLenRange is the length band for the current level. It starts short and
// grows, which raises difficulty twice over: longer words take longer to type,
// and chip acceleration scales with letters destroyed, so an eleven-letter word
// reaches over twice its spawn speed by its last letter where a four-letter word
// can only reach 1.4x.
//
// The ceiling climbs on a concave curve, so three and four letter words are gone
// inside the first minute. There is no point dwelling there: only someone who
// cannot type finds them hard, and for everyone else it is dead time.
func (s *State) WordLenRange() (minLen, maxLen int) {
	return s.lenRangeAt(s.LevelF())
}

// lenRangeAt is the band at an arbitrary progression value, so power-ups can ask
// for the band a couple of levels ahead of the current one.
func (s *State) lenRangeAt(level float64) (minLen, maxLen int) {
	t := s.Tune
	maxLen = int(t.WordLenStart + t.WordLenGrowth*rampCurve(level, t.WordLenCurve))
	if hard := int(t.WordLenMax); maxLen > hard {
		maxLen = hard
	}
	minLen = maxLen - int(t.WordLenSpread)
	if floor := int(t.WordLenMin); minLen < floor {
		minLen = floor
	}
	if minLen > maxLen {
		minLen = maxLen
	}
	return minLen, maxLen
}

// pickWord chooses the next word to spawn, from the given length band.
func (s *State) pickWord(minLen, maxLen int) string {

	// Rebuild the eligible pool when the band moves. Reused between spawns so
	// this is not a per-spawn allocation.
	if s.poolMin != minLen || s.poolMax != maxLen || s.pool == nil {
		s.pool = s.pool[:0]
		for _, w := range s.Vocab {
			if n := len(w); n >= minLen && n <= maxLen {
				s.pool = append(s.pool, w)
			}
		}
		// A band with nothing in it must not deadlock the spawner. Fall back to
		// the whole vocabulary rather than refusing to spawn.
		if len(s.pool) == 0 {
			s.pool = append(s.pool, s.Vocab...)
		}
		s.poolMin, s.poolMax = minLen, maxLen
	}

	// Keep the history shorter than the pool, or every draw would be rejected.
	want := len(s.pool) / 2
	if want > 24 {
		want = 24
	}
	if want != len(s.recent) {
		s.recent = make([]string, want)
		s.recentAt = 0
	}

	// Draw a handful of candidates and keep the one whose typing difficulty best
	// matches what the schedule is asking for. Sampling rather than sorting keeps
	// the word list varied: picking the single hardest word every time would make
	// the late game a short list of the same tongue-twisters.
	target := s.difficultyTarget()
	best, bestGap := "", 0.0
	for try := 0; try < 16; try++ {
		w := s.pool[s.Rng.Intn(len(s.pool))]
		if s.usedRecently(w) {
			continue
		}
		if target <= 0 {
			s.remember(w)
			return w
		}
		gap := TypingDifficulty(w) - target*s.Tune.TypingDifficultyScale
		if gap < 0 {
			gap = -gap
		}
		if best == "" || gap < bestGap {
			best, bestGap = w, gap
		}
		if try >= 5 && best != "" {
			break
		}
	}
	if best != "" {
		s.remember(best)
		return best
	}
	// Pathologically small pool; take whatever comes.
	w := s.pool[s.Rng.Intn(len(s.pool))]
	s.remember(w)
	return w
}

func (s *State) usedRecently(w string) bool {
	for _, r := range s.recent {
		if r == w {
			return true
		}
	}
	return false
}

func (s *State) remember(w string) {
	if len(s.recent) == 0 {
		return
	}
	s.recent[s.recentAt] = w
	s.recentAt = (s.recentAt + 1) % len(s.recent)
}
