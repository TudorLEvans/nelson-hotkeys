package audio

import (
	"math"
	"os"
	"testing"
)

func TestPitchIsInTune(t *testing.T) {
	cases := map[string]float64{
		"A4": 440, "A3": 220, "A5": 880,
		"C4": 261.63, "C5": 523.25,
		"F#4": 369.99, "Bb3": 233.08,
	}
	for name, want := range cases {
		got := Pitch(name)
		if math.Abs(got-want) > 0.5 {
			t.Errorf("%s = %.2fHz, want %.2f", name, got, want)
		}
	}
}

// TestBadPitchIsARest, not a panic. This is a subsystem nobody has to enable, so
// a typo in a tune must never take the game down with it.
func TestBadPitchIsARest(t *testing.T) {
	for _, bad := range []string{"", "-", "H4", "C", "C99", "###", "4C", "C4x", "Cb"} {
		if got := Pitch(bad); got != 0 {
			t.Errorf("Pitch(%q) = %v, want a rest", bad, got)
		}
	}
}

func TestWAVIsWellFormed(t *testing.T) {
	pcm := Track{{Freq: 440, Ms: 100, Wave: Square}}.Render(1)
	data := WAV(pcm)

	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		t.Fatal("not a RIFF WAVE file")
	}
	if len(data) != 44+len(pcm)*2 {
		t.Errorf("file is %d bytes, header plus samples is %d", len(data), 44+len(pcm)*2)
	}
	// The declared sizes must match reality or players truncate or hang.
	declared := int(data[4]) | int(data[5])<<8 | int(data[6])<<16 | int(data[7])<<24
	if declared != len(data)-8 {
		t.Errorf("RIFF size says %d, file is %d", declared, len(data)-8)
	}
}

// TestNotesDoNotClick. Without an envelope every note starts and ends at full
// amplitude, which is the difference between chiptune and a fault.
func TestNotesDoNotClick(t *testing.T) {
	pcm := Track{{Freq: 440, Ms: 200, Wave: Square, Gain: 1}}.Render(1)
	if len(pcm) < 100 {
		t.Fatal("no samples")
	}
	if abs16(pcm[0]) > 2000 {
		t.Errorf("first sample is %d; the attack is missing", pcm[0])
	}
	if abs16(pcm[len(pcm)-1]) > 2000 {
		t.Errorf("last sample is %d; the decay is missing", pcm[len(pcm)-1])
	}
	// And it must actually get loud in between, or the envelope ate the note.
	var peak int16
	for _, s := range pcm {
		if abs16(s) > peak {
			peak = abs16(s)
		}
	}
	if peak < 20000 {
		t.Errorf("peak is only %d; the note never sounds", peak)
	}
}

func abs16(v int16) int16 {
	if v < 0 {
		return -v
	}
	return v
}

// TestEverySoundRenders: a missing case would be silence in play with nothing to
// say why.
func TestEverySoundRenders(t *testing.T) {
	for _, e := range Effects {
		track := Sound(e, 1)
		if len(track) == 0 {
			t.Errorf("%s produces no notes", e)
			continue
		}
		if d := track.Duration(); d < 20 || d > 1200 {
			t.Errorf("%s runs %dms; effects should be short and gameover brief", e, d)
		}
		if len(track.Render(1)) == 0 {
			t.Errorf("%s rendered to nothing", e)
		}
	}
	for _, tune := range Tunes {
		track := Music(tune)
		if len(track) == 0 {
			t.Errorf("%s produces no notes", tune)
			continue
		}
		if d := track.Duration(); d < 3000 {
			t.Errorf("%s runs only %dms; a loop that short is a stutter", tune, d)
		}
	}
}

// TestLetterBlipRisesWithTheMultiplier is the one sound carrying information
// rather than atmosphere: it makes the score dial audible.
func TestLetterBlipRisesWithTheMultiplier(t *testing.T) {
	var last float64
	for _, mult := range []int{1, 2, 3, 5} {
		track := Sound(EffectLetter, mult)
		if len(track) != 1 {
			t.Fatalf("multiplier %d produced %d notes", mult, len(track))
		}
		if track[0].Freq <= last {
			t.Errorf("multiplier %d sounds at %.0fHz, not above the previous %.0f",
				mult, track[0].Freq, last)
		}
		last = track[0].Freq
	}
}

// TestTypoIsQuiet. It is already punished by the streak and by the heat; a loud
// noise for every slip would make the game hostile.
func TestTypoIsQuiet(t *testing.T) {
	// Loudest note, not the first: several effects open with a deliberate rest.
	loudest := func(e Effect) float64 {
		var g float64
		for _, n := range Sound(e, 1) {
			if n.Gain > g {
				g = n.Gain
			}
		}
		return g
	}
	typo := loudest(EffectTypo)
	for _, e := range []Effect{EffectWord, EffectLife, EffectBomb, EffectPowerup} {
		if g := loudest(e); g <= typo {
			t.Errorf("%s peaks at gain %.2f, no louder than a typo at %.2f", e, g, typo)
		}
	}
}

// TestNilPlayerIsSafe is how audio stays genuinely optional: the game never has
// to ask whether sound is on.
func TestNilPlayerIsSafe(t *testing.T) {
	var p *Player
	p.Play(EffectLetter, 3)
	p.Music(TuneMenu)
	p.StopMusic()
	p.Close()
}

// TestPlayerCleansUpAfterItself. Leaving a player process running after the game
// exits would be the rudest possible bug, and leaving temp files is not far
// behind.
func TestPlayerCleansUpAfterItself(t *testing.T) {
	if !Available() {
		t.Skip("no system player here")
	}
	p := New(0.01)
	if p == nil {
		t.Skip("player could not start")
	}
	dir := p.dir
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("render directory missing: %v", err)
	}
	p.Close()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("temp directory %s survived Close", dir)
	}
	// Close twice must not panic; the game closes on a deferred call that can run
	// after an earlier one on a panic path.
	p.Close()
}

// TestPlayDoesNotBlockWhenSaturated: if the game is making noise faster than the
// machine can play it, dropping a blip is right and stalling a frame is not.
func TestPlayDoesNotBlockWhenSaturated(t *testing.T) {
	p := &Player{
		effects: map[Effect][]string{EffectLetter: {"/nonexistent.wav"}},
		queue:   make(chan string, 4),
	}
	for i := 0; i < 1000; i++ {
		p.Play(EffectLetter, 1) // no pump goroutine, so the queue fills and stays full
	}
}

// TestAudioIsSmallAndFast. The whole point of note data over recordings is that
// nothing ships in the binary and startup does not stall; both need pinning.
func TestAudioIsSmallAndFast(t *testing.T) {
	var bytes int
	for _, tune := range Tunes {
		bytes += len(WAV(Music(tune).Render(2)))
	}
	for _, e := range Effects {
		bytes += len(WAV(Sound(e, 1).Render(1)))
	}
	mb := float64(bytes) / (1024 * 1024)
	t.Logf("all audio renders to %.1f MB of WAV", mb)
	if mb > 6 {
		t.Errorf("%.1f MB of rendered audio; the temp directory should stay modest", mb)
	}
}

// TestDottedNotesLastHalfAsLongAgain covers the one bit of the tune notation
// that is not obvious from reading it. Without dots, a tune taken off a MIDI
// has to round every dotted note to something wrong.
func TestDottedNotesLastHalfAsLongAgain(t *testing.T) {
	track := parse(400, Square, "C4/4 C4/4. C4/8 C4/8. C4/2.")
	want := []int{400, 600, 200, 300, 1200}
	if len(track) != len(want) {
		t.Fatalf("parsed %d notes, want %d", len(track), len(want))
	}
	for i, ms := range want {
		if track[i].Ms != ms {
			t.Errorf("note %d runs %dms, want %dms", i, track[i].Ms, ms)
		}
	}
}
