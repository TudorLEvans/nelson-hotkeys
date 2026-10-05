package engine

import "strings"

// Tier is kept only for the oversized title rendering. Gameplay has one size.
type Tier int

const (
	// TierNormal is every falling word: one glyph dot per drawing unit.
	TierNormal Tier = iota
	// TierHuge doubles each dot, for the title and the word that ends a run.
	TierHuge
)

// Scale is how many drawing units each glyph dot occupies.
func (t Tier) Scale() int {
	switch t {
	case TierNormal:
		return 1
	case TierHuge:
		return 2
	default:
		panic("engine: unknown Tier")
	}
}

func (t Tier) String() string {
	switch t {
	case TierNormal:
		return "NORMAL"
	case TierHuge:
		return "HUGE"
	default:
		panic("engine: unknown Tier")
	}
}

// DepthMultiplier scores a letter by how far down the screen it was destroyed.
// This is the dial that turns "letters typed" from a measure of how long you
// survived into a bet you choose to take. The renderer colours words from the
// same function, so the colour on screen is the multiplier readout.
func DepthMultiplier(bottomRow, playH int) int {
	if playH <= 0 {
		return 1
	}
	if bottomRow >= playH-2 {
		return 5
	}
	switch f := float64(bottomRow) / float64(playH); {
	case f >= 2.0/3:
		return 3
	case f >= 1.0/3:
		return 2
	default:
		return 1
	}
}

// Word is one falling target.
//
// Both coordinates are integer glyph DOTS, not cells and not floats. Dots are the
// finest position anything can be drawn at, so a float would carry precision that
// can never be shown, and it was the source of every word crossing cell
// boundaries at its own separate moment. See motion.go and metrics.go.
type Word struct {
	Text  string
	Typed int // letters destroyed, from the left
	DotX  int // left edge, playfield-relative, in dots
	DotY  int // top edge in dots; negative is above the field
	Tier  Tier
	Age   int // spawn sequence number; lower means older
	// Power is the suffix character if this is a power-up word, else PowerNone.
	// The suffix is part of Text, so targeting needs no special case.
	Power Power
	// Bomb marks a word that must NOT be finished. Bombs are last-resort targets
	// and land harmlessly; see bomb.go.
	Bomb bool
	// Swarm marks a word that arrived as part of a swarm, for the renderer.
	Swarm bool
	// KickFlash counts down frames after this word was sped up by a collision, so
	// the renderer can show the cause. Momentum transfer is the one rule whose
	// effect is felt several seconds later, well after the collision that caused
	// it has left the player's attention.
	KickFlash int
	// Kick is a tick interval inherited from a faster word above, or 0. It is a
	// floor on this word's speed, not a replacement: the schedule can still
	// overtake it. Speed otherwise comes from the schedule every tick, so a level
	// change accelerates the whole field including words already falling.
	Kick int
}

// Remaining is the still-typeable suffix. Typed is a byte index, which is
// correct because word packs are validated to single-byte renderable
// characters; see words.parse.
func (w *Word) Remaining() string { return w.Text[w.Typed:] }

// Done reports whether every letter has been destroyed.
func (w *Word) Done() bool { return w.Typed >= len(w.Text) }

// HeightDots is the word's vertical extent in dots.
func (w *Word) HeightDots() int { return GlyphHeightDots * w.Tier.Scale() }

// WidthDots is the word's horizontal extent in dots.
func (w *Word) WidthDots() int { return TextWidthDots(w.Text) * w.Tier.Scale() }

// BottomDot is the bottom edge in dots: the exact ordering key for targeting and
// the test for reaching the floor.
func (w *Word) BottomDot() int { return w.DotY + w.HeightDots() }

// TopRow is the first terminal row the word touches.
func (w *Word) TopRow() int { return floorDiv(w.DotY, M.DotsY) }

// SubOffset is which sub-row inside the cell the glyph starts on, 0 to DotsY-1.
// Integer arithmetic throughout, so the negative positions a word holds while
// entering the field need no special case; a truncating divide got them wrong and
// slid entering words a row too low.
func (w *Word) SubOffset() int { return w.DotY - M.DotsY*w.TopRow() }

// HeightCells is the rows the word covers, including the extra row a sub-cell
// offset can push it onto.
func (w *Word) HeightCells() int {
	return ceilDiv(w.SubOffset()+w.HeightDots(), M.DotsY)
}

// BottomRow is the last terminal row the word touches.
func (w *Word) BottomRow() int { return w.TopRow() + w.HeightCells() - 1 }

// CellX is the left column the word is drawn at.
func (w *Word) CellX() int { return floorDiv(w.DotX, M.DotsX) }

// Rect is a playfield-relative cell rectangle.
type Rect struct{ X, Y, W, H int }

// LetterBox is the cell rectangle of one letter, used to place its explosion.
// Letters keep their columns as the word is eaten, so this stays correct for the
// prefix that is already gone.
func (w *Word) LetterBox(i int) Rect {
	dotX := w.DotX + LetterOffsetDots(w.Text, i)*w.Tier.Scale()
	glyphW := GlyphFor(rune(w.Text[i])).W * w.Tier.Scale()
	return Rect{
		X: floorDiv(dotX, M.DotsX),
		Y: w.TopRow(),
		W: ceilDiv(glyphW, M.DotsX),
		H: w.HeightCells(),
	}
}

// LetterBoxes is every letter's rectangle, for a whole-word detonation.
func (w *Word) LetterBoxes() []Rect {
	out := make([]Rect, 0, len(w.Text))
	for i := range w.Text {
		out = append(out, w.LetterBox(i))
	}
	return out
}

// Hit describes what one keypress did, so the renderer can put an explosion in
// the right place without the engine knowing anything about the screen.
type Hit struct {
	Ok        bool // a letter was destroyed
	Word      int  // index into State.Words at the time of the press
	Letter    int  // which letter of the word
	Completed bool // that was the last letter
	Points    int
	Mult      int // depth multiplier applied
	Chain     int // chain position, if Completed

	Cell  Rect   // box of the letter just destroyed
	Boxes []Rect // every letter's box, set only when Completed
	Power Power  // effect fired, if any
	Bomb  bool   // a bomb was completed; this is a wound, not a reward
	// Cleared holds the letter boxes of any words the effect destroyed, so the
	// renderer can put explosions where they were.
	Cleared []Rect
	// Reward is a chain milestone effect, if this completion earned one.
	Reward *ChainReward
}

// StepResult reports what one tick did that the caller may want to react to.
type StepResult struct {
	Landed   []string // words that reached the floor
	Cleared  int      // words removed by the impact shockwave
	LostLife bool
	Shielded bool // a landing was absorbed by SHIELD
	// PowerMissed counts power-up words that reached the floor. They cost nothing,
	// but the renderer may want to acknowledge the lost chance.
	PowerMissed int
	// BombsDodged counts bombs that reached the floor without being completed,
	// which is the good outcome.
	BombsDodged int
	// Swarm is how many words a swarm just dropped, if one did.
	Swarm    int
	GameOver bool
}

// State is the whole game. It holds no terminal handle and does no I/O, so the
// entire game is testable headless.
type State struct {
	PlayW, PlayH int
	// MaxWordsCap is what the playfield can hold without overlap. The schedule
	// works up toward it; see MaxWordsNow.
	MaxWordsCap int

	Words   []Word
	Elapsed float32
	Frame   int

	Lives  int
	Score  int
	Streak int
	Shield bool

	freezeUntil float32
	slowStack   float32
	typoHeat    float32

	// The instant effect last fired, and how long its name stays in the status
	// row. See FiredPower.
	firedPower Power
	firedUntil float32

	Chain         int
	chainDeadline float32

	Keys           int
	Hits           int
	Misses         int
	BestStreak     int
	BestChain      int
	WordsDestroyed int
	BombsHit       int

	// God disables life loss, for looking at the render.
	God bool

	GameOver bool
	Killer   string // the word that ended the run

	// LastCleared is the most recently destroyed word, for the definition panel.
	// The word just destroyed is the right one to define: it is a reward for
	// finishing, and showing a falling word's meaning would both distract and hint
	// at which word to type.
	LastCleared   string
	LastClearedAt float32

	spawnTimer   float32
	spawnPause   float32
	breatheUntil float32
	lastSwarm    float32
	swarmUntil   float32

	// SwarmLetter is the shared opening letter of the most recent swarm.
	SwarmLetter byte

	rewardName  string
	rewardUntil float32

	lastLevel    int
	levelUpUntil float32
	nextAge      int
	stepAccum    float32
	stepTick     int

	// Word choice state; see vocab.go.
	pool             []string
	poolMin, poolMax int
	recent           []string
	recentAt         int

	Tune  *Tuning
	Rng   *RNG
	Vocab []string
}

// NewState builds a game for a playfield of the given inner size.
func NewState(playW, playH int, t *Tuning, r *RNG, vocab []string) *State {
	s := &State{
		PlayW: playW, PlayH: playH,
		MaxWordsCap: 4,
		Tune:        t, Rng: r, Vocab: vocab,
		Words: make([]Word, 0, 8),
	}
	s.Lives = int(t.Lives)
	return s
}

// Reset starts a fresh run, keeping the playfield and vocabulary. The game over
// screen restarts on one keypress and must not make the player wait.
func (s *State) Reset() {
	s.Words = s.Words[:0]
	s.Elapsed, s.Frame = 0, 0
	s.Lives = int(s.Tune.Lives)
	s.Score, s.Streak, s.Chain, s.chainDeadline = 0, 0, 0, 0
	s.Keys, s.Hits, s.Misses = 0, 0, 0
	s.BestStreak, s.BestChain, s.WordsDestroyed = 0, 0, 0
	s.BombsHit = 0
	s.GameOver, s.Killer = false, ""
	s.LastCleared, s.LastClearedAt = "", 0
	s.Shield = false
	s.freezeUntil = 0
	s.slowStack = 0
	s.firedPower, s.firedUntil = PowerNone, 0
	s.typoHeat = 0
	s.spawnTimer, s.spawnPause, s.nextAge = 0, 0, 0
	s.breatheUntil, s.lastSwarm, s.swarmUntil = 0, 0, 0
	s.SwarmLetter = 0
	s.rewardName, s.rewardUntil = "", 0
	s.lastLevel, s.levelUpUntil = 0, 0
	s.stepAccum, s.stepTick = 0, 0
	s.poolMin, s.poolMax = 0, 0
	s.pool = s.pool[:0]
	for i := range s.recent {
		s.recent[i] = ""
	}
	s.recentAt = 0
}

// Accuracy is correct keys over all keys, in percent.
func (s *State) Accuracy() float64 {
	if s.Keys == 0 {
		return 100
	}
	return 100 * float64(s.Hits) / float64(s.Keys)
}

// WPM is the conventional five-characters-per-word rate.
func (s *State) WPM() float64 {
	if s.Elapsed <= 0 {
		return 0
	}
	return float64(s.Hits) / 5 / (float64(s.Elapsed) / 60)
}

// Spawn places one word above the field, at a horizontal position that does not
// overlap anything already up there. Reports false when there was no room, in
// which case nothing is spawned: forcing it would put two words on the same
// cells, and two block-glyph words on the same cells are unreadable.
func (s *State) Spawn() bool {
	bomb := s.pickBomb()
	power := PowerNone
	if !bomb {
		power = s.pickPowerup()
	}

	lo, hi := s.WordLenRange()
	switch {
	case bomb:
		lo, hi = s.bombLenRange()
	case power != PowerNone:
		lo, hi = s.powerLenRange()
	}

	text := s.pickWord(lo, hi)
	if bomb && s.sharesFirstLetter(text) {
		// Try once more for a word that does not open on a live letter, then give
		// up and spawn it anyway: the priority rule already makes it fair.
		text = s.pickWord(lo, hi)
	}
	if power != PowerNone {
		text += string(rune(power))
	}
	x, ok := s.placeX(text)
	if !ok {
		return false
	}
	s.Words = append(s.Words, Word{
		Text:  text,
		Tier:  TierNormal,
		DotX:  x,
		DotY:  -GlyphHeightDots,
		Age:   s.nextAge,
		Power: power,
		Bomb:  bomb,
	})
	s.nextAge++
	return true
}

// Type resolves one keypress under the free fire rule.
func (s *State) Type(key byte) Hit {
	if s.GameOver {
		return Hit{}
	}
	s.Keys++

	i := Resolve(s.Words, key)
	if i < 0 {
		// No word on the field wants that letter. Streak resets and accuracy
		// takes it; nothing else. No input stun, because punishing a fast typist
		// for outrunning the field would fight the whole design.
		s.Misses++
		s.Streak = 0
		s.addTypoHeat()
		return Hit{}
	}

	w := &s.Words[i]
	letter := w.Typed
	mult := DepthMultiplier(w.BottomRow(), s.PlayH)

	s.Hits++
	s.Streak++
	if s.Streak > s.BestStreak {
		s.BestStreak = s.Streak
	}
	s.Score += mult
	w.Typed++

	h := Hit{Ok: true, Word: i, Letter: letter, Points: mult, Mult: mult}
	h.Cell = w.LetterBox(letter)

	if w.Done() {
		h.Completed = true
		h.Boxes = w.LetterBoxes()
		// Chain: consecutive completions inside the window. Gated on combo so
		// mashing cannot stumble into the payout.
		if s.Streak >= int(s.Tune.ChainMinStreak) && s.Elapsed <= s.chainDeadline {
			s.Chain++
		} else {
			s.Chain = 1
		}
		s.chainDeadline = s.Elapsed + float32(s.Tune.ChainWindow)
		if s.Chain > s.BestChain {
			s.BestChain = s.Chain
		}

		bonus := len(w.Text) * mult * ChainMultiplier(s.Chain)
		s.Score += bonus
		h.Points += bonus
		h.Chain = s.Chain
		s.WordsDestroyed++
		s.noteDestroyed()

		h.Power = w.Power
		h.Bomb = w.Bomb
		blastAt := w.LetterBox(0)
		s.LastCleared = strings.TrimRight(w.Text, powerSuffixes)
		s.LastClearedAt = s.Elapsed
		s.Words = append(s.Words[:i], s.Words[i+1:]...)

		// Everything that can touch the word list runs AFTER the completed word is
		// removed, and the ordering is load-bearing rather than tidy. Firing a
		// chain reward first let SWEEP filter the slice while the removal below
		// still held an index into it, which panicked with a slice bounds error the
		// moment a deep chain landed with anything in the lower half.
		if h.Bomb {
			s.BombsHit++
			s.detonate()
		}
		if h.Power != PowerNone {
			h.Cleared = s.applyPowerAt(h.Power, blastAt)
			s.noteFired(h.Power)
		}
		h.Reward = s.chainReward(h.Chain)
	}
	return h
}

// Step advances the world by a fixed dt. dt is a constant, never a measured
// frame time, so runs are reproducible from a seed.
func (s *State) Step(dt float32) StepResult {
	var res StepResult
	if s.GameOver {
		return res
	}
	s.Elapsed += dt
	s.Frame++
	s.decayTypoHeat(dt)
	s.drainSlow(dt)

	// The level is displayed but nothing in the simulation reads it, so a player
	// can cross a boundary without noticing. Announcing it is the only signal that
	// the schedule has moved.
	if lvl := s.Level(); lvl != s.lastLevel {
		if lvl > s.lastLevel {
			s.levelUpUntil = s.Elapsed + 1.2
		}
		s.lastLevel = lvl
	}

	for i := range s.Words {
		if s.Words[i].KickFlash > 0 {
			s.Words[i].KickFlash--
		}
	}

	if s.Elapsed > s.chainDeadline {
		s.Chain = 0
	}

	// One shared clock. Every word advances on it, so the field moves as a whole
	// rather than each word crossing row boundaries at its own moment.
	s.stepAccum += dt
	// Typo heat shortens the shared movement tick, so the whole field speeds up
	// smoothly and in lockstep. Applying it here rather than to each word's integer
	// step count avoids the rounding that would swallow a small penalty entirely.
	interval := float32(s.Tune.StepInterval / (s.heatFactor() * (1 - s.SlowFactor())))
	if interval <= 0 {
		interval = 1.0 / 30
	}
	// FREEZE suspends movement but not the clock: the schedule keeps advancing, so
	// the effect buys breathing room rather than pausing the run.
	if !s.Frozen() {
		for s.stepAccum >= interval {
			s.stepAccum -= interval
			s.stepTick++
			for i := range s.Words {
				if s.stepTick%s.StepEvery(&s.Words[i]) == 0 {
					s.Words[i].DotY++
				}
			}
			s.resolveCollisions()
		}
	} else {
		for s.stepAccum >= interval {
			s.stepAccum -= interval
		}
	}

	floorDots := s.PlayH * M.DotsY
	kept := s.Words[:0]
	for _, w := range s.Words {
		if w.BottomDot() >= floorDots {
			// A power-up that reaches the floor costs nothing. It is an
			// opportunity, not a threat: if ignoring one cost a life the player
			// would be forced to take every single one, and since power-ups ride
			// the longest words on the field that would be a punishment rather
			// than a temptation. Same principle as bombs, so the rule across the
			// whole game is that special words are optional.
			if w.Power != PowerNone || w.Bomb {
				// Special words are optional: both power-ups and bombs land
				// harmlessly. A bomb you ignored is a bomb you beat.
				if w.Bomb {
					res.BombsDodged++
				} else {
					res.PowerMissed++
				}
				continue
			}
			res.Landed = append(res.Landed, w.Text)
			continue
		}
		kept = append(kept, w)
	}
	s.Words = kept

	if len(res.Landed) > 0 {
		if s.God {
			res.Shielded = true
		} else if s.Shield {
			// A shield eats one landing whole, however many words arrived with it.
			s.Shield = false
			res.Shielded = true
		} else {
			res.LostLife = true
			s.Lives -= len(res.Landed)
		}
		s.Streak, s.Chain = 0, 0
		s.Killer = res.Landed[0]

		// Impact shockwave. A landing also clears everything near the floor and
		// pauses spawning. Without it, being overwhelmed means three words land
		// inside two seconds and the run ends before the player can react, which
		// reads as arbitrary rather than earned.
		cut := s.PlayH - int(s.Tune.ShockwaveRows)
		kept = s.Words[:0]
		for _, w := range s.Words {
			if w.BottomRow() >= cut {
				res.Cleared++
				continue
			}
			kept = append(kept, w)
		}
		s.Words = kept
		s.spawnPause = float32(s.Tune.SpawnPause)

		if s.Lives <= 0 {
			s.Lives = 0
			s.GameOver = true
			res.GameOver = true
			return res
		}
	}

	if s.spawnPause > 0 {
		s.spawnPause -= dt
		return res
	}
	if s.breathing() {
		return res
	}

	if s.dueSwarm() {
		if n := s.spawnSwarm(); n > 0 {
			res.Swarm = n
			s.spawnTimer = float32(s.SpawnIntervalNow())
			return res
		}
	}

	s.spawnTimer -= dt
	if s.spawnTimer <= 0 && len(s.Words) < s.MaxWordsNow() {
		if s.Spawn() {
			s.spawnTimer = float32(s.SpawnIntervalNow())
		} else {
			// No room up there. Try again shortly rather than dropping the beat.
			s.spawnTimer = float32(s.Tune.StepInterval) * 4
		}
	}
	return res
}
