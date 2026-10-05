package engine

// Bombs are words you must not finish.
//
// The obvious problem with putting one on a free fire field is that free fire has
// no lock: a keypress goes to whichever word wants that letter and is furthest
// down. If a bomb is the lowest word wanting a letter you need, you are forced to
// feed it, and with several words on screen that becomes unavoidable rather than
// difficult. A mechanic the player cannot avoid by playing well is not a mechanic.
//
// One rule fixes it:
//
//	A bomb never takes priority. A keypress reaches a bomb only when no safe
//	word on the field wants that letter.
//
// That makes bombs impossible to be forced into by construction, rather than by
// careful spawn selection. No arrangement of words can make a bomb letter
// unavoidable, because any safe word wanting the same letter always wins.
//
// What it turns bombs into is a tax on typos: the only way to feed one is to press
// a letter nothing else wants, which is what a mistake looks like. It also closes
// the farming hole in scoring, where pressing one letter across the field paid out
// for nothing. With a bomb up, mashing a letter is how you complete the bomb.

// Bombed reports whether a word is a bomb.
func (w *Word) Bombed() bool { return w.Bomb }

// hasBomb reports whether a bomb is already on the field.
func (s *State) hasBomb() bool {
	for i := range s.Words {
		if s.Words[i].Bomb {
			return true
		}
	}
	return false
}

// pickBomb decides whether the next spawn is a bomb.
func (s *State) pickBomb() bool {
	if s.Tune.BombChance <= 0 || s.LevelF() < s.Tune.BombFromLevel {
		return false
	}
	// One at a time, and never alongside a power-up: both are exceptions to
	// ordinary play, and two exceptions on screen at once is unreadable.
	if s.hasBomb() {
		return false
	}
	for i := range s.Words {
		if s.Words[i].Power != PowerNone {
			return false
		}
	}
	return float64(s.Rng.Float32()) < s.Tune.BombChance
}

// bombLenRange keeps bombs mid-length. Short and a couple of stray keys finish
// them; long and they clutter the field for their whole descent.
func (s *State) bombLenRange() (int, int) {
	return int(s.Tune.BombLenMin), int(s.Tune.BombLenMax)
}

// sharesFirstLetter reports whether any live word starts with the same letter.
// Not needed for fairness, since the priority rule already guarantees that, but it
// stops the very first stray key landing on a bomb before the player has taken in
// that one is there.
func (s *State) sharesFirstLetter(text string) bool {
	if text == "" {
		return false
	}
	for i := range s.Words {
		w := &s.Words[i]
		if !w.Done() && w.Text[w.Typed] == text[0] {
			return true
		}
	}
	return false
}

// Detonated is what happens when a bomb is completed: the self-inflicted wound.
// No impact shockwave, because nothing landed, and a consolation prize for a
// mistake would blunt the whole point.
func (s *State) detonate() {
	s.Streak, s.Chain = 0, 0
	if s.Tune.BombCostsLife > 0 && !s.God {
		s.Lives -= int(s.Tune.BombCostsLife)
		if s.Lives <= 0 {
			s.Lives = 0
			s.GameOver = true
		}
	}
}
