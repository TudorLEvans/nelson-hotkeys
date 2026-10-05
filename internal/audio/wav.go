// Package audio synthesises the game's sound and hands it to the system player.
//
// Everything is generated from note data rather than shipped as recordings. A
// whole soundtrack is a few kilobytes of pitches against several megabytes of
// audio files, it is genuinely 8-bit rather than an imitation of it, and it means
// tempo and timbre are tunable the same way everything else in this game is.
//
// It plays by shelling out to afplay, paplay or aplay. That is a deliberate
// choice over a Go audio library: every one of those opens a device through cgo,
// and cgo costs the single-command five-platform build that decided the language.
// Measured, spawning a player costs 1-2ms typically and spikes to about 30ms
// under sustained fire, so it runs off the game loop rather than on it.
package audio

import (
	"encoding/binary"
	"math"
	"math/rand"
)

// Wave is a timbre. Three is enough: this is meant to sound like hardware that
// could only manage three.
type Wave int

const (
	Square Wave = iota
	Triangle
	Noise
)

const sampleRate = 22050

// Note is one sound. A zero frequency is a rest.
type Note struct {
	Freq float64
	Ms   int
	Wave Wave
	Gain float64 // 0 to 1; zero means the default
}

// Track is a sequence of notes played one after another.
type Track []Note

// samples renders a note to PCM.
func (n Note) samples(rng *rand.Rand) []int16 {
	count := sampleRate * n.Ms / 1000
	out := make([]int16, count)
	if n.Freq <= 0 || count == 0 {
		return out
	}
	gain := n.Gain
	if gain <= 0 {
		gain = 0.35
	}
	amp := gain * 32767

	period := float64(sampleRate) / n.Freq
	var noiseVal float64
	var noiseHold int

	for i := range out {
		var v float64
		switch n.Wave {
		case Square:
			if math.Mod(float64(i), period) < period/2 {
				v = 1
			} else {
				v = -1
			}
		case Triangle:
			p := math.Mod(float64(i), period) / period
			if p < 0.5 {
				v = 4*p - 1
			} else {
				v = 3 - 4*p
			}
		case Noise:
			if noiseHold <= 0 {
				noiseVal = rng.Float64()*2 - 1
				noiseHold = int(period / 2)
				if noiseHold < 1 {
					noiseHold = 1
				}
			}
			noiseHold--
			v = noiseVal
		}

		// A short attack and a longer decay. Without them every note begins and
		// ends with a click, which is the difference between chiptune and a fault.
		env := 1.0
		const attack = 40
		if i < attack {
			env = float64(i) / attack
		}
		if tail := count - i; tail < count/3 {
			env *= float64(tail) / float64(count/3)
		}
		out[i] = int16(v * amp * env)
	}
	return out
}

// Render turns a track into PCM.
func (t Track) Render(seed int64) []int16 {
	rng := rand.New(rand.NewSource(seed))
	var out []int16
	for _, n := range t {
		out = append(out, n.samples(rng)...)
	}
	return out
}

// Duration is how long a track runs.
func (t Track) Duration() int {
	ms := 0
	for _, n := range t {
		ms += n.Ms
	}
	return ms
}

// WAV wraps PCM in the smallest container every system player understands.
func WAV(pcm []int16) []byte {
	data := make([]byte, len(pcm)*2)
	for i, s := range pcm {
		binary.LittleEndian.PutUint16(data[i*2:], uint16(s))
	}

	var h []byte
	str := func(s string) { h = append(h, s...) }
	u32 := func(v uint32) {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], v)
		h = append(h, b[:]...)
	}
	u16 := func(v uint16) {
		var b [2]byte
		binary.LittleEndian.PutUint16(b[:], v)
		h = append(h, b[:]...)
	}

	str("RIFF")
	u32(uint32(36 + len(data)))
	str("WAVEfmt ")
	u32(16)
	u16(1) // PCM
	u16(1) // mono
	u32(sampleRate)
	u32(sampleRate * 2)
	u16(2)
	u16(16)
	str("data")
	u32(uint32(len(data)))
	return append(h, data...)
}

// Pitch converts a note name to a frequency: "C4", "F#5", "Bb3".
//
// Returns 0 for anything unrecognised, which renders as a rest. A wrong note is
// better than a panic in a subsystem nobody has to enable.
func Pitch(name string) float64 {
	if name == "" || name == "-" {
		return 0
	}
	semis := map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}
	base, ok := semis[name[0]]
	if !ok {
		return 0
	}
	i := 1
	for ; i < len(name); i++ {
		switch name[i] {
		case '#':
			base++
		case 'b':
			base--
		default:
			goto octave
		}
	}
octave:
	// The octave is exactly one digit and must end the name. Reading one digit and
	// ignoring the rest let "C99" parse as C9, which is the sort of leniency that
	// turns a typo in a tune into a mystery rather than an error.
	if i != len(name)-1 {
		return 0
	}
	oct := int(name[i] - '0')
	if oct < 0 || oct > 9 {
		return 0
	}
	// A4 = 440Hz, and MIDI note 69.
	midi := (oct+1)*12 + base
	return 440 * math.Pow(2, float64(midi-69)/12)
}
