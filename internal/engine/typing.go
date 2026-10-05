package engine

// How hard a word is to TYPE, as distinct from how long it is.
//
// Length is the crude proxy the difficulty schedule already uses, and it is
// wrong in both directions: MINIMUM is eleven letters and rolls off the hand,
// while POLKA is five and asks the same finger to do two jobs in a row. Picking
// by length alone means the late game is full of long comfortable words and the
// player never meets the awkward ones.
//
// This scores the awkwardness so the schedule can bias toward it, which raises
// difficulty without making words longer, and makes the late game feel different
// rather than merely more.
//
// It assumes QWERTY. On Dvorak or Colemak the scores are wrong, so the weighting
// is a tunable that can be turned off rather than something baked into selection.

type keyPos struct {
	hand   int // 0 left, 1 right
	finger int // 1 index .. 4 little
	row    int // 0 home, 1 top, 2 bottom
}

var keyMap = map[byte]keyPos{}

func init() {
	rows := []struct {
		keys   string
		row    int
		digits []int // finger per key, left to right
		hands  []int
	}{
		{"QWERTYUIOP", 1,
			[]int{4, 3, 2, 1, 1, 1, 1, 2, 3, 4},
			[]int{0, 0, 0, 0, 0, 1, 1, 1, 1, 1}},
		{"ASDFGHJKL", 0,
			[]int{4, 3, 2, 1, 1, 1, 1, 2, 3},
			[]int{0, 0, 0, 0, 0, 1, 1, 1, 1}},
		{"ZXCVBNM", 2,
			[]int{4, 3, 2, 1, 1, 1, 2},
			[]int{0, 0, 0, 0, 0, 1, 1}},
	}
	for _, r := range rows {
		for i := 0; i < len(r.keys); i++ {
			keyMap[r.keys[i]] = keyPos{hand: r.hands[i], finger: r.digits[i], row: r.row}
		}
	}
}

// TypingDifficulty scores a word from 0 (easy) upward. The scale is relative;
// only the ordering matters.
//
// The costs are ranked by what actually slows a typist down, hardest first:
// hitting the same finger twice in a row forces a full lift and replace, staying
// on one hand denies the natural alternation, and reaching off the home row costs
// a little each time.
func TypingDifficulty(word string) float64 {
	if len(word) < 2 {
		return 0
	}
	var score float64
	var prev keyPos
	var havePrev bool

	for i := 0; i < len(word); i++ {
		p, ok := keyMap[word[i]]
		if !ok {
			havePrev = false
			continue
		}
		switch p.row {
		case 1:
			score += 0.3
		case 2:
			score += 0.5
		}
		if p.finger == 4 {
			score += 0.4 // little finger
		}
		if havePrev {
			switch {
			case prev.hand == p.hand && prev.finger == p.finger:
				// Same finger twice: the slowest thing a hand can be asked to do,
				// and worse still when it also has to change row.
				score += 2.0
				if prev.row != p.row {
					score += 1.0
				}
			case prev.hand == p.hand:
				score += 0.6 // same hand, no alternation
			}
		}
		prev, havePrev = p, true
	}
	return score / float64(len(word))
}

// difficultyTarget is the awkwardness the schedule is aiming for at the current
// level, from 0 at the start to 1 late on.
func (s *State) difficultyTarget() float64 {
	if s.Tune.TypingDifficulty <= 0 {
		return 0
	}
	reach := s.Tune.TypingDifficultyBy
	if reach <= 0 {
		return s.Tune.TypingDifficulty
	}
	f := s.LevelF() / reach
	if f > 1 {
		f = 1
	}
	return f * s.Tune.TypingDifficulty
}
