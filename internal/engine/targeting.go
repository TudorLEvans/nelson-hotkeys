package engine

// Free fire. There is no lock and no selected word: every keypress resolves
// against the whole field, one letter at a time. The player can leave a word
// half-eaten, jump to another, and come back.
//
// Two words starting with S means pressing S twice takes the S off both, lower
// first. Six words sharing a letter and six presses clears six first letters.
// This is why no spawn ambiguity guard is needed: shared letters stop being a
// bug and become an opportunity.

// Resolve returns the index of the word that a keypress consumes a letter from,
// or -1 if no word on the field wants that letter.
//
// Priority is the word furthest down the screen. Not the word in most danger:
// with variable speed a fast word higher up can die sooner, but the player's
// only tool for aiming a letter is the order they press keys, and that tool
// needs a rule they can read off the screen instantly. Screen position is
// readable; time-to-floor is not. The fix for the difference is a better
// display, not a cleverer rule.
//
// Tie-break is float bottom edge, then leftmost x, then word age. Deterministic
// because players will learn it and rely on it.
func Resolve(words []Word, key byte) int {
	// Safe words first. A bomb is only ever reached when nothing else wants the
	// letter, which is what makes bombs impossible to be forced into: no
	// arrangement of words can make a bomb letter unavoidable, because any safe
	// word wanting the same letter always wins. See bomb.go.
	if i := resolveAmong(words, key, false); i >= 0 {
		return i
	}
	return resolveAmong(words, key, true)
}

func resolveAmong(words []Word, key byte, bombs bool) int {
	best := -1
	for i := range words {
		w := &words[i]
		if w.Bomb != bombs {
			continue
		}
		if w.Typed >= len(w.Text) || w.Text[w.Typed] != key {
			continue
		}
		if best < 0 || betterTarget(&words[i], &words[best]) {
			best = i
		}
	}
	return best
}

// betterTarget reports whether a should receive the keypress instead of b.
// Ordering is total, so Resolve never depends on iteration luck.
func betterTarget(a, b *Word) bool {
	if ae, be := a.BottomDot(), b.BottomDot(); ae != be {
		return ae > be
	}
	if a.DotX != b.DotX {
		return a.DotX < b.DotX
	}
	return a.Age < b.Age // older word wins, and Age counts up from the first spawn
}
