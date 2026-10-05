package audio

import (
	"strconv"
	"strings"
)

// Tunes are written in a compact notation so they can be read and edited as
// music rather than as data: "D4/4 E4/8 F4/8" is a crotchet D, then two quavers,
// at whatever the track's beat length is.
//
// A caveat worth stating plainly: Heart of Oak is a transcription from memory
// of a very familiar melody, not from a score. It is recognisable rather than
// authoritative, and anyone with the printed music should correct it. The other
// four came out of the reference MIDI in assets/midi via the extractor there,
// which also explains the tempos: each is the source file's own crotchet. R is
// a rest, which Pitch already returns 0 for because R is not a note name.
//
// The pieces themselves are all long out of copyright, and synthesising from
// note data creates no performance right either, so only accuracy is at stake.
func parse(beatMs int, wave Wave, s string) Track {
	var t Track
	for _, tok := range strings.Fields(s) {
		name, denom, ok := strings.Cut(tok, "/")
		div, dotted := 4, false
		if ok {
			// A trailing dot is the usual musical one: half as long again, so
			// "D4/4." is a dotted crotchet. Without it a transcription from a
			// MIDI has to round every dotted note to something wrong.
			denom, dotted = strings.CutSuffix(denom, ".")
			if n, err := strconv.Atoi(denom); err == nil && n > 0 {
				div = n
			}
		}
		ms := beatMs * 4 / div
		if dotted {
			ms = ms * 3 / 2
		}
		t = append(t, Note{
			Freq: Pitch(name),
			Ms:   ms,
			Wave: wave,
		})
	}
	return t
}

// Tune names the music the game can play.
type Tune string

const (
	// TuneMenu is Purcell's Rondeau from Abdelazer, the tune every English
	// schoolchild knows without knowing why.
	TuneMenu Tune = "menu"
	// TuneIntro is Zadok the Priest: Handel wrote it for a coronation, which is
	// roughly the register in which the Ministry of Defence raises a dead admiral.
	TuneIntro Tune = "intro"
	// TunePlayEarly is Heart of Oak, the march of the Royal Navy.
	TunePlayEarly Tune = "play-early"
	// TunePlayLate is La Rejouissance from the Fireworks music, for when the field
	// has stopped being funny.
	TunePlayLate Tune = "play-late"
	// TuneDefeat is Dido's Lament. The most funereal thing in English music, and
	// it is about a queen dying while a fleet leaves. It could not be more apt.
	TuneDefeat Tune = "defeat"
)

// Music returns a tune as a track.
func Music(t Tune) Track {
	switch t {
	case TuneMenu:
		// The rondeau's refrain, the twelve bars that come round again every time
		// an episode finishes. 326ms to the crotchet is the MIDI's own tempo.
		return parse(326, Square,
			"D4/2 F4/2 A4/2 D5/4 E5/8 F5/8 G5/8 F5/8 "+
				"E5/8 D5/8 C#5/2 A5/4 D5/8 F5/8 A5/8 F5/8 "+
				"D5/8 R/8 Bb5/2 G5/4 C5/8 E5/8 G5/8 E5/8 "+
				"C5/8 R/8 A5/2 F5/4 Bb4/8 D5/8 F5/8 D5/8 "+
				"Bb4/8 R/8 G5/2 E5/4 A4/8 C#5/8 E5/8 C#5/8 "+
				"A4/8 R/8 F5/2 E5/8 F5/8 E5/8 D5/8 C#5/8 "+
				"R/8 F5/8 R/8 E5/8 F5/8 E5/8 D5/8 A4/8 "+
				"R/8 D5/8 R/8 C#5/8 D5/8 E5/8 D5/8 D5/2")

	case TuneIntro:
		// Rising arpeggios stacking upward, which is the whole architecture of
		// the opening. Taken from the MIDI in assets/midi, at its own 833ms
		// crotchet: the harmony climbs D, E minor, A while the figure itself
		// never changes.
		return parse(833, Triangle,
			"D4/16 F#4/16 A4/16 D5/16 D4/16 F#4/16 A4/16 D5/16 "+
				"D4/16 F#4/16 A4/16 D5/16 D4/16 F#4/16 A4/16 D5/16 "+
				"E4/16 G4/16 B4/16 E5/16 E4/16 G4/16 B4/16 E5/16 "+
				"E4/16 G4/16 B4/16 E5/16 E4/16 G4/16 B4/16 E5/16 "+
				"E4/16 A4/16 C5/16 E5/16 E4/16 A4/16 C5/16 E5/16")

	case TunePlayEarly:
		return parse(420, Square,
			"G4/4 G4/8 A4/8 B4/4 B4/4 "+
				"C5/8 B4/8 A4/8 G4/8 A4/2 "+
				"D5/4 D5/8 C5/8 B4/4 A4/4 "+
				"G4/8 A4/8 B4/8 G4/8 D4/2")

	case TunePlayLate:
		// The oboe line from the MIDI in assets/midi, at its own 550ms
		// crotchet.
		return parse(550, Square,
			"A4/8 D5/8 D5/8 D5/8 D5/16 A4/16 D5/16 E5/16 "+
				"F#5/16 E5/16 D5/8 D5/16 E5/16 F#5/4 F#5/8 F#5/8 "+
				"F#5/16 G5/16 A5/16 G5/16 F#5/8 F#5/16 G5/16 A5/8 "+
				"A5/8 A5/8 A5/8 A5/4 R/8 D6/8 A5/8 F#5/16")

	case TuneDefeat:
		// The descending chromatic ground bass, which is the part of the piece
		// everybody actually remembers, twice round. From the MIDI in
		// assets/midi at its own 1017ms crotchet, transposed back to G minor:
		// that sequence is in B flat minor, presumably to suit a singer.
		return parse(1017, Triangle,
			"G3/4 F#3/2 F3/4 E3/2 Eb3/4 D3/2 Bb2/4 C3/4 D3/2 G2/2 "+
				"G3/4 F#3/2 F3/4 E3/2 Eb3/4 D3/2 Bb2/4 C3/4 D3/2 G2/2")
	}
	return nil
}

// Effect names a sound the game makes in response to something the player did.
type Effect string

const (
	EffectLetter   Effect = "letter"
	EffectWord     Effect = "word"
	EffectChain    Effect = "chain"
	EffectTypo     Effect = "typo"
	EffectLife     Effect = "life"
	EffectPowerup  Effect = "powerup"
	EffectBomb     Effect = "bomb"
	EffectSwarm    Effect = "swarm"
	EffectLevelUp  Effect = "levelup"
	EffectGameOver Effect = "gameover"
)

// Sound builds an effect. The letter blip takes a depth multiplier and rises with
// it, which is the one sound here that carries information rather than
// atmosphere: it makes the score dial audible, so a player learns that deeper
// pays more without reading a table.
func Sound(e Effect, mult int) Track {
	switch e {
	case EffectLetter:
		base := map[int]string{1: "C5", 2: "E5", 3: "G5", 5: "C6"}[mult]
		if base == "" {
			base = "C5"
		}
		return Track{{Freq: Pitch(base), Ms: 40, Wave: Square, Gain: 0.22}}

	case EffectWord:
		return Track{
			{Freq: Pitch("C5"), Ms: 45, Wave: Square, Gain: 0.28},
			{Freq: Pitch("E5"), Ms: 45, Wave: Square, Gain: 0.28},
			{Freq: Pitch("G5"), Ms: 70, Wave: Square, Gain: 0.28},
		}

	case EffectChain:
		return Track{
			{Freq: Pitch("G5"), Ms: 45, Wave: Triangle, Gain: 0.3},
			{Freq: Pitch("B5"), Ms: 45, Wave: Triangle, Gain: 0.3},
			{Freq: Pitch("D6"), Ms: 45, Wave: Triangle, Gain: 0.3},
			{Freq: Pitch("G6"), Ms: 90, Wave: Triangle, Gain: 0.3},
		}

	case EffectTypo:
		// Quiet on purpose. A typo is already punished twice, by the streak and by
		// the heat, and a loud noise for every slip would make the game hostile.
		return Track{{Freq: Pitch("F3"), Ms: 55, Wave: Triangle, Gain: 0.14}}

	case EffectLife:
		return Track{
			{Freq: 0, Ms: 10},
			{Freq: 90, Ms: 260, Wave: Noise, Gain: 0.45},
			{Freq: 60, Ms: 220, Wave: Noise, Gain: 0.3},
		}

	case EffectPowerup:
		return Track{
			{Freq: Pitch("C5"), Ms: 45, Wave: Triangle, Gain: 0.3},
			{Freq: Pitch("G5"), Ms: 45, Wave: Triangle, Gain: 0.3},
			{Freq: Pitch("C6"), Ms: 45, Wave: Triangle, Gain: 0.3},
			{Freq: Pitch("E6"), Ms: 110, Wave: Triangle, Gain: 0.3},
		}

	case EffectBomb:
		// The only genuinely unpleasant sound in the game, which is the point.
		return Track{
			{Freq: 200, Ms: 90, Wave: Noise, Gain: 0.5},
			{Freq: 120, Ms: 160, Wave: Noise, Gain: 0.5},
			{Freq: 70, Ms: 320, Wave: Noise, Gain: 0.4},
		}

	case EffectSwarm:
		return Track{
			{Freq: 300, Ms: 60, Wave: Noise, Gain: 0.3},
			{Freq: 300, Ms: 60, Wave: Noise, Gain: 0.35},
			{Freq: 300, Ms: 60, Wave: Noise, Gain: 0.4},
			{Freq: Pitch("D5"), Ms: 140, Wave: Square, Gain: 0.3},
		}

	case EffectLevelUp:
		// A bosun's whistle: up, hold, up again.
		return Track{
			{Freq: Pitch("D6"), Ms: 70, Wave: Triangle, Gain: 0.25},
			{Freq: Pitch("A6"), Ms: 130, Wave: Triangle, Gain: 0.25},
			{Freq: Pitch("D6"), Ms: 70, Wave: Triangle, Gain: 0.25},
			{Freq: Pitch("A6"), Ms: 180, Wave: Triangle, Gain: 0.25},
		}

	case EffectGameOver:
		return Track{
			{Freq: Pitch("D4"), Ms: 200, Wave: Triangle, Gain: 0.3},
			{Freq: Pitch("A3"), Ms: 200, Wave: Triangle, Gain: 0.3},
			{Freq: Pitch("F3"), Ms: 200, Wave: Triangle, Gain: 0.3},
			{Freq: Pitch("D3"), Ms: 500, Wave: Triangle, Gain: 0.3},
		}
	}
	return nil
}

// Effects lists every effect, for building them all up front.
var Effects = []Effect{
	EffectLetter, EffectWord, EffectChain, EffectTypo, EffectLife,
	EffectPowerup, EffectBomb, EffectSwarm, EffectLevelUp, EffectGameOver,
}

// Tunes lists every tune.
var Tunes = []Tune{TuneMenu, TuneIntro, TunePlayEarly, TunePlayLate, TuneDefeat}
