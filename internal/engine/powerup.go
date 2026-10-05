package engine

// Power-ups are ordinary words with a punctuation suffix, and the suffix is
// typed.
//
// An earlier design made the marker display-only, reasoning that '!' costs a
// shift and reaching for shift mid-chain breaks the flow the scoring rewards.
// That was right about '!' and wrong about punctuation generally: most of it is
// unshifted on a Latin layout, so '.' or '-' or '/' is one ordinary keystroke.
//
// Because the suffix is part of Word.Text, the whole free fire path needs no
// special case: the mark is simply the last letter. And since punctuation appears
// only on power-up words, pressing '.' can never be stolen by an ordinary word.
// Being last also means a power-up has to be earned by finishing the word.

// Power is a power-up kind, identified by the character that ends its word.
//
// There were nine of these. Nine symbols is more than anyone learns from playing,
// and six of them meant nothing on sight: '[' does not look like "strip a letter
// off everything". Four is a set a player can hold in their head, and each of the
// four marks can be guessed at: a full stop stops the field, a minus pushes it
// back, a slash cuts down what is near, a plus adds a life.
//
// Cut, and why:
//
//	STEADY   the effect was the absence of an effect. Chip acceleration not
//	         happening is not something anyone can see happening.
//	AUTOCHIP ate a letter off every word and sped them all up. A reward that
//	         cannot be told apart from a penalty teaches nothing.
//	CLEAR    BLAST does the same job with a radius, and where you finish the
//	         word matters. Two field-wipes was one too many; the rarer one went.
//	SLOW     the same axis as FREEZE, and ',' is near enough identical to '.'
//	         in the block font that misreading one is a wrong keypress, which is
//	         how a bomb gets fed.
//	SHIELD   the same axis as REPAIR, and it fired silently.
//
// SLOW and SHIELD survive as effects because deep chains still pay in them; see
// chainRewardFor. They are simply no longer spawnable, so their runes are
// identity for the status row rather than a key anyone presses.
type Power rune

const (
	PowerNone   Power = 0
	PowerFreeze Power = '.' // everything stops
	PowerRewind Power = '-' // push the field back up
	PowerBlast  Power = '/' // destroy everything near where it went off
	PowerRepair Power = '+' // one life back

	// Not spawnable. Earned from chain milestones, and named in the status row.
	PowerSlow   Power = ','
	PowerShield Power = '\''
)

// powerSuffixes is every power-up character, for trimming a suffix off a word.
//
// Three of the four are unshifted on a Latin layout, which is the whole reason
// the suffix can be typed at all. '+' is the exception and is allowed to be:
// REPAIR is the rarest of the four, and the one moment where the player is on
// low lives rather than mid-chain, so a beat spent reaching for shift costs
// nothing.
const powerSuffixes = ".-/+"

// Typeable reports whether a key can consume a letter. One source of truth,
// because the input layer and the word validator must agree: they did not, and
// the input layer silently dropped every punctuation suffix, which made power-ups
// impossible to finish for as long as they existed.
func Typeable(ch rune) bool {
	switch {
	case ch >= 'A' && ch <= 'Z':
		return true
	case ch >= '0' && ch <= '9':
		return true
	}
	for _, p := range powerSuffixes {
		if ch == p {
			return true
		}
	}
	return false
}

// Powers is the spawnable set, weakest first.
var Powers = []Power{
	PowerBlast, PowerFreeze, PowerRewind, PowerRepair,
}

// weight is how often a power-up turns up, relative to the others.
//
// Rarity tracks strength. A common power-up is one you can plan around, so the
// mild ones need to appear often enough to become part of how the game is played;
// a strong one is a moment, and a moment that happens every thirty seconds is not
// a moment. REPAIR hands back a life and is the rarest thing in the game.
//
// Tunable per power, because this is exactly the kind of number that is wrong
// until it has been played with.
func (p Power) weight(t *Tuning) float64 {
	switch p {
	case PowerBlast:
		return t.WeightBlast
	case PowerFreeze:
		return t.WeightFreeze
	case PowerRewind:
		return t.WeightRewind
	case PowerRepair:
		return t.WeightRepair
	}
	return 0
}

func (p Power) String() string {
	switch p {
	case PowerFreeze:
		return "FREEZE"
	case PowerRewind:
		return "REWIND"
	case PowerBlast:
		return "BLAST"
	case PowerRepair:
		return "REPAIR"
	case PowerSlow:
		return "SLOW"
	case PowerShield:
		return "SHIELD"
	default:
		return ""
	}
}

// Instant reports whether the effect happens once rather than running for a time.
func (p Power) Instant() bool {
	switch p {
	case PowerRewind, PowerRepair, PowerBlast:
		return true
	}
	return false
}

// powerLenRange is the length band power-up words are drawn from: the band the
// schedule would use a couple of levels from now, so a power-up is always a
// longer word than the ones around it.
//
// This is the whole risk-and-reward of the mechanic. A power-up on a short word
// would be a free gift, taken without thought. On a long one it is a commitment:
// more letters to type, more time exposed, and because chip acceleration scales
// with letters destroyed, a long word accelerates hard as you eat it. You are
// choosing to take on the most dangerous thing on the field in exchange for the
// effect.
func (s *State) powerLenRange() (minLen, maxLen int) {
	return s.lenRangeAt(s.LevelF() + s.Tune.PowerupLevelBonus)
}

// PowerLenRangeFor exposes the power-up length band for diagnostics.
func PowerLenRangeFor(s *State) (int, int) { return s.powerLenRange() }

// pickPowerup decides whether the next spawn carries a power-up, and which. It
// returns PowerNone most of the time.
func (s *State) pickPowerup() Power {
	if s.LevelF() < s.Tune.PowerupFromLevel {
		return PowerNone
	}
	// One at a time. Two active markers on the field is unreadable, and stacking
	// their effects raises questions nobody needs answered.
	for i := range s.Words {
		if s.Words[i].Power != PowerNone {
			return PowerNone
		}
	}
	if s.Tune.PowerupChance <= 0 {
		return PowerNone
	}
	if float64(s.Rng.Float32()) >= s.Tune.PowerupChance {
		return PowerNone
	}

	total := 0.0
	for _, p := range Powers {
		total += p.weight(s.Tune)
	}
	if total <= 0 {
		return PowerNone
	}
	roll := float64(s.Rng.Float32()) * total
	for _, p := range Powers {
		roll -= p.weight(s.Tune)
		if roll < 0 {
			return p
		}
	}
	return Powers[len(Powers)-1]
}

// applyPower fires an effect. Durations come from tuning so they can be tuned
// live; the instant ones act on the field immediately.
//
// at is where the triggering word was, which only BLAST needs. Returns the boxes
// of any words the effect destroyed, so the renderer can put explosions on them.
func (s *State) applyPower(p Power) []Rect {
	return s.applyPowerAt(p, Rect{})
}

func (s *State) applyPowerAt(p Power, at Rect) []Rect {
	switch p {
	case PowerFreeze:
		s.freezeUntil = s.Elapsed + float32(s.Tune.FreezeSeconds)
	case PowerSlow:
		// Not spawnable any more; this is the chain milestone at ChainSlowAt.
		//
		// Stacks rather than replacing. A flat halving was too strong: it turned a
		// minor helper into a mode change. Each grant shaves a slice off the field's
		// speed and the slices add up, so one is a nudge and several in a row are the
		// old effect earned rather than handed over.
		s.slowStack += float32(s.Tune.SlowPerPickup)
		if max := float32(s.Tune.SlowMax); s.slowStack > max {
			s.slowStack = max
		}
	case PowerShield:
		s.Shield = true
	case PowerRepair:
		if max := int(s.Tune.MaxLives); s.Lives < max {
			s.Lives++
		}
	case PowerRewind:
		up := int(s.Tune.RewindRows) * M.DotsY
		for i := range s.Words {
			s.Words[i].DotY -= up
			// A rewound word loses any collision kick, since whatever kicked it is
			// no longer beneath it.
			s.Words[i].Kick = 0
		}
	case PowerBlast:
		// The field-wipe, with a radius. It takes out whatever happened to be near
		// the word when it went off, which makes WHERE you finish it matter rather
		// than only THAT you finished it. That shape is the fun part, and it is why
		// this is the one that survived: a whole-field CLEAR was strictly less
		// interesting and had to be kept rare to stay a moment.
		//
		// No letter score. A button that pays out is a button pressed mindlessly,
		// which is the same reason the score section refuses to reward farming. The
		// streak survives, because the player earned this by finishing a word.
		return s.blast(at)
	}
	return nil
}

// blast destroys every word overlapping a box centred on at, and returns their
// letter boxes for the explosion.
func (s *State) blast(at Rect) []Rect {
	if at.W == 0 && at.H == 0 {
		return nil
	}
	cx := at.X + at.W/2
	cy := at.Y + at.H/2
	halfW := int(s.Tune.BlastCols) / 2
	halfH := int(s.Tune.BlastRows) / 2

	var boxes []Rect
	kept := s.Words[:0]
	for i := range s.Words {
		w := &s.Words[i]
		// Bombs are immune. Detonating one for the player would be the single
		// thing they have no defence against, and the bomb rules promise that
		// only their own keystrokes can set one off.
		if w.Bomb {
			kept = append(kept, *w)
			continue
		}
		x0, x1 := w.CellX(), w.CellX()+TextWidth(w.Text)-1
		y0, y1 := w.TopRow(), w.BottomRow()
		if x1 < cx-halfW || x0 > cx+halfW || y1 < cy-halfH || y0 > cy+halfH {
			kept = append(kept, *w)
			continue
		}
		boxes = append(boxes, w.LetterBoxes()...)
	}
	s.Words = kept
	return boxes
}

// Frozen reports whether movement is suspended.
func (s *State) Frozen() bool { return s.Elapsed < s.freezeUntil }

// Slowed reports whether any slow is in effect.
func (s *State) Slowed() bool { return s.slowStack > 0 }

// SlowFactor is how much the field is slowed, 0 to SlowMax. 0.2 means everything
// falls 20% slower.
func (s *State) SlowFactor() float64 { return float64(s.slowStack) }

// drainSlow bleeds the stack away at a constant rate, so each pickup is worth
// SlowSeconds of its own slice however many are stacked. Linear rather than
// exponential because the status row shows a countdown, and a countdown has to be
// honest about when the effect ends.
func (s *State) drainSlow(dt float32) {
	if s.slowStack <= 0 {
		s.slowStack = 0
		return
	}
	if s.Tune.SlowSeconds <= 0 {
		s.slowStack = 0
		return
	}
	rate := float32(s.Tune.SlowPerPickup / s.Tune.SlowSeconds)
	s.slowStack -= rate * dt
	if s.slowStack < 0.001 {
		s.slowStack = 0
	}
}

// SlowSecondsLeft is how long until the field is back to full speed.
func (s *State) SlowSecondsLeft() float32 {
	if s.slowStack <= 0 || s.Tune.SlowPerPickup <= 0 {
		return 0
	}
	return s.slowStack / float32(s.Tune.SlowPerPickup/s.Tune.SlowSeconds)
}

// FiredSeconds is how long an instant effect names itself in the status row.
// Long enough to read while still typing, short enough that it is naming what
// just happened rather than something from ten seconds ago.
const FiredSeconds = 2

// FiredPower is the instant effect the player just set off, if the flash is still
// up.
//
// Timed effects announce themselves by their countdown. The instant ones had no
// readout anywhere: REWIND, BLAST and REPAIR fired, changed the field, and never
// said their own name, so the only way to learn which mark did what was to press
// it and infer. That is the half of the legibility problem the symbols were being
// asked to carry on their own.
func (s *State) FiredPower() (Power, bool) {
	if s.firedPower == PowerNone || s.Elapsed >= s.firedUntil {
		return PowerNone, false
	}
	return s.firedPower, true
}

// noteFired raises the flash. Called where a completed word fires its effect,
// not from applyPower: a chain milestone grants the same effects and has its own
// banner, and two announcements of one event is one too many.
func (s *State) noteFired(p Power) {
	if !p.Instant() {
		return
	}
	s.firedPower = p
	s.firedUntil = s.Elapsed + FiredSeconds
}

// ActiveEffect is one running effect and its remaining time, for the status row.
type ActiveEffect struct {
	Power   Power
	Seconds float32
}

// ActiveEffects lists what is running. An effect whose end the player cannot see
// cannot be planned around.
func (s *State) ActiveEffects() []ActiveEffect {
	var out []ActiveEffect
	if s.Frozen() {
		out = append(out, ActiveEffect{PowerFreeze, s.freezeUntil - s.Elapsed})
	}
	if s.Slowed() {
		out = append(out, ActiveEffect{PowerSlow, s.SlowSecondsLeft()})
	}
	if s.Shield {
		out = append(out, ActiveEffect{PowerShield, 0})
	}
	return out
}
