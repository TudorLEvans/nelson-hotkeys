package audio

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// Player renders the game's sound to files once and plays them by shelling out.
//
// Every method is safe to call on a nil Player and does nothing, which is how
// audio stays genuinely optional: the game never has to ask whether sound is on.
type Player struct {
	bin     string
	dir     string
	volume  string
	effects map[Effect][]string // one file per multiplier variant
	tunes   map[Tune]string

	queue chan string

	mu      sync.Mutex
	music   *exec.Cmd
	current Tune
	closed  bool
}

// players in preference order. afplay on macOS, the PulseAudio and ALSA players
// on Linux.
var players = []string{"afplay", "paplay", "aplay"}

// Available reports whether anything on this machine can play a WAV.
func Available() bool {
	for _, p := range players {
		if _, err := exec.LookPath(p); err == nil {
			return true
		}
	}
	return false
}

// New builds every sound up front and returns a Player, or nil if audio cannot
// work here. A missing player, an unwritable temp directory or a failed render
// all produce nil rather than an error: this is a garnish, and nothing about it
// may stop the game starting.
func New(volume float64) *Player {
	var bin string
	for _, p := range players {
		if path, err := exec.LookPath(p); err == nil {
			bin = path
			break
		}
	}
	if bin == "" {
		return nil
	}
	dir, err := os.MkdirTemp("", "nelson-audio")
	if err != nil {
		return nil
	}

	p := &Player{
		bin:     bin,
		dir:     dir,
		effects: map[Effect][]string{},
		tunes:   map[Tune]string{},
		queue:   make(chan string, 64),
	}
	if filepath.Base(bin) == "afplay" && volume > 0 && volume != 1 {
		p.volume = fmt.Sprintf("%.2f", volume)
	}

	for _, e := range Effects {
		// The letter blip has a variant per depth multiplier; everything else has
		// one file that ignores the argument.
		for _, mult := range []int{1, 2, 3, 5} {
			track := Sound(e, mult)
			if track == nil {
				continue
			}
			name := filepath.Join(dir, fmt.Sprintf("%s-%d.wav", e, mult))
			if err := os.WriteFile(name, WAV(track.Render(1)), 0o644); err != nil {
				p.Close()
				return nil
			}
			p.effects[e] = append(p.effects[e], name)
			if e != EffectLetter {
				break // the rest do not vary
			}
		}
	}
	for _, t := range Tunes {
		track := Music(t)
		if track == nil {
			continue
		}
		name := filepath.Join(dir, string(t)+".wav")
		if err := os.WriteFile(name, WAV(track.Render(2)), 0o644); err != nil {
			p.Close()
			return nil
		}
		p.tunes[t] = name
	}

	// Effects are spawned from their own goroutine. Measured, a spawn costs one to
	// two milliseconds and spikes to about thirty under sustained fire; thirty is
	// a whole frame, so it must not happen on the loop that draws them.
	go p.pump()
	return p
}

func (p *Player) pump() {
	for file := range p.queue {
		cmd := p.command(file)
		if err := cmd.Start(); err != nil {
			continue // a sound that will not play is not worth a message
		}
		go cmd.Wait()
	}
}

func (p *Player) command(file string) *exec.Cmd {
	if p.volume != "" {
		return exec.Command(p.bin, "-v", p.volume, file)
	}
	return exec.Command(p.bin, file)
}

// Play makes an effect sound. Never blocks: if the queue is full the game is
// making noise faster than the machine can play it, and the right answer is to
// drop this one rather than stall a frame.
func (p *Player) Play(e Effect, mult int) {
	if p == nil {
		return
	}
	files := p.effects[e]
	if len(files) == 0 {
		return
	}
	file := files[0]
	if e == EffectLetter {
		switch mult {
		case 2:
			file = files[min(1, len(files)-1)]
		case 3:
			file = files[min(2, len(files)-1)]
		case 5:
			file = files[min(3, len(files)-1)]
		}
	}
	select {
	case p.queue <- file:
	default:
	}
}

// Music starts a tune looping, replacing whatever was playing. Passing the tune
// already running does nothing, so callers can set it every frame.
func (p *Player) Music(t Tune) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.current == t {
		return
	}
	p.stopMusicLocked()

	file, ok := p.tunes[t]
	if !ok {
		return
	}
	p.current = t
	go p.loop(t, file)
}

// loop replays a tune until something else is asked for. None of the system
// players loop on their own, so the loop is a respawn.
func (p *Player) loop(t Tune, file string) {
	for {
		p.mu.Lock()
		if p.closed || p.current != t {
			p.mu.Unlock()
			return
		}
		cmd := p.command(file)
		if err := cmd.Start(); err != nil {
			p.mu.Unlock()
			return
		}
		p.music = cmd
		p.mu.Unlock()

		_ = cmd.Wait()
	}
}

// StopMusic silences the background track.
func (p *Player) StopMusic() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopMusicLocked()
}

func (p *Player) stopMusicLocked() {
	p.current = ""
	if p.music != nil && p.music.Process != nil {
		_ = p.music.Process.Kill()
		p.music = nil
	}
}

// Close stops everything and removes the rendered files. Leaving a player process
// running after the game exits would be the rudest possible bug.
func (p *Player) Close() {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.stopMusicLocked()
	p.mu.Unlock()

	close(p.queue)
	if p.dir != "" {
		_ = os.RemoveAll(p.dir)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
