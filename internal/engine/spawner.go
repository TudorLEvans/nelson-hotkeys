package engine

// Pacing beyond the schedule.
//
// Monotone escalation is not tension. The schedule in difficulty.go rises
// smoothly forever, which is correct as a floor but gives the run no shape: no
// moment to breathe and no moment that stands out. Two additions give it one.

// breathing reports whether the spawner is deliberately holding off.
func (s *State) breathing() bool { return s.Elapsed < s.breatheUntil }

// noteDestroyed hooks the word counter, so a breath can be earned by progress
// rather than handed out on a timer. A pause every N words lands after a burst of
// clearing, which is when it reads as a reward.
func (s *State) noteDestroyed() {
	if s.Tune.BreatheEvery <= 0 || s.Tune.BreatheSeconds <= 0 {
		return
	}
	if s.WordsDestroyed%int(s.Tune.BreatheEvery) != 0 {
		return
	}
	s.breatheUntil = s.Elapsed + float32(s.Tune.BreatheSeconds)
}

// dueSwarm reports whether it is time for a swarm.
func (s *State) dueSwarm() bool {
	if s.Tune.SwarmEvery <= 0 || s.LevelF() < s.Tune.SwarmFromLevel {
		return false
	}
	return s.Elapsed-s.lastSwarm >= float32(s.Tune.SwarmEvery)
}

// spawnSwarm drops several short words at once, all starting with the same
// letter.
//
// It is a designed showcase for free fire. Under the targeting rule one letter
// pressed N times takes the head off N words, lowest first, so a swarm is the one
// arrangement where the mechanic pays off spectacularly rather than incidentally.
// It is also the moment a player describes to someone else.
//
// Returns how many words actually landed on the field; placement can refuse.
func (s *State) spawnSwarm() int {
	want := int(s.Tune.SwarmSize)
	if want < 2 {
		return 0
	}
	room := s.MaxWordsCap - len(s.Words)
	if room < 2 {
		return 0
	}
	if want > room {
		want = room
	}

	// Pick a letter that plenty of short words start with, so the swarm is
	// actually uniform rather than three words and a shrug.
	lo, hi := s.bombLenRange()
	lo, hi = lo-2, hi-2
	if lo < int(s.Tune.WordLenMin) {
		lo = int(s.Tune.WordLenMin)
	}
	if hi < lo {
		hi = lo
	}

	var letter byte
	var pool []string
	for attempt := 0; attempt < 12 && len(pool) < want; attempt++ {
		letter = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"[s.Rng.Intn(26)]
		pool = pool[:0]
		for _, w := range s.Vocab {
			if len(w) >= lo && len(w) <= hi && w[0] == letter {
				pool = append(pool, w)
			}
		}
	}
	if len(pool) < 2 {
		return 0
	}

	placed := 0
	for i := 0; i < want; i++ {
		text := pool[s.Rng.Intn(len(pool))]
		x, ok := s.placeX(text)
		if !ok {
			break
		}
		s.Words = append(s.Words, Word{
			Text:  text,
			Tier:  TierNormal,
			DotX:  x,
			DotY:  -GlyphHeightDots,
			Age:   s.nextAge,
			Swarm: true,
		})
		s.nextAge++
		placed++
	}
	if placed > 0 {
		s.lastSwarm = s.Elapsed
		s.SwarmLetter = letter
		s.swarmUntil = s.Elapsed + 1.5
	}
	return placed
}

// SwarmBanner reports the letter to announce, if a swarm just arrived.
func (s *State) SwarmBanner() (byte, bool) {
	if s.Elapsed < s.swarmUntil {
		return s.SwarmLetter, true
	}
	return 0, false
}
