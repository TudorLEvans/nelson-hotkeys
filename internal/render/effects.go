package render

import (
	"math"

	"github.com/gdamore/tcell/v2"
	"nelson/internal/engine"
)

// Effects holds the purely visual state: particles, flashes and shake. It lives
// in render rather than engine because none of it affects play, and the engine
// stays free of anything the game does not simulate.
type Effects struct {
	parts  []particle
	popups []popup
	rng    *engine.RNG

	flash   float32 // whole-field white flash, seconds remaining
	shake   float32 // screen shake, seconds remaining
	banner  string
	bannerT float32
}

// popup is a number that floats off a word as it dies.
//
// It exists because the score stopped being a letter count. The brief was one
// point per letter; a depth multiplier and a completion bonus were layered on top
// and neither was ever shown, so the total jumped on completion with nothing on
// screen to explain it. A number rising off the word is the whole explanation.
type popup struct {
	text   string
	x, y   float32
	life   float32
	colour tcell.Color
}

type particle struct {
	x, y   float32 // playfield-relative cells
	vx, vy float32 // cells per second
	life   float32
	max    float32
	colour tcell.Color
}

func NewEffects(seed uint64) *Effects {
	return &Effects{parts: make([]particle, 0, 512), rng: engine.NewRNG(seed)}
}

// multColour ties the explosion colour to the depth multiplier, so the feedback
// matches the size of the bet the player just took.
func multColour(mult int) tcell.Color {
	switch mult {
	case 5:
		return tcell.ColorWhite
	case 3:
		return tcell.ColorOrangeRed
	case 2:
		return tcell.ColorYellow
	default:
		return tcell.ColorAqua
	}
}

// Burst throws particles out of a destroyed letter's cell box. Count and speed
// scale with the multiplier: a x5 kill should look excessive.
func (e *Effects) Burst(r engine.Rect, mult int) {
	n := 8 + 3*mult
	w, h := r.W, r.H
	cx := float32(r.X) + float32(w)/2
	cy := float32(r.Y) + float32(h)/2
	col := multColour(mult)
	for i := 0; i < n; i++ {
		ang := e.rng.Range(0, 2*math.Pi)
		spd := e.rng.Range(6, 10+4*float32(mult))
		e.parts = append(e.parts, particle{
			x: cx + e.rng.Range(-float32(w)/2, float32(w)/2),
			y: cy + e.rng.Range(-float32(h)/2, float32(h)/2),
			// Halve vertical speed: cells are about twice as tall as they are
			// wide, so equal cell velocities look stretched vertically.
			vx:     spd * float32(math.Cos(float64(ang))),
			vy:     spd * float32(math.Sin(float64(ang))) * 0.5,
			life:   e.rng.Range(0.20, 0.35),
			max:    0.35,
			colour: col,
		})
	}
}

// Popup floats a number off a point, for a score the player would otherwise see
// only as the total changing.
func (e *Effects) Popup(text string, r engine.Rect, mult int) {
	e.popups = append(e.popups, popup{
		text:   text,
		x:      float32(r.X),
		y:      float32(r.Y),
		life:   1.1,
		colour: multColour(mult),
	})
}

// Detonate is the word-completion burst: every letter box, plus a flash.
func (e *Effects) Detonate(boxes []engine.Rect, mult, chain int) {
	// One burst per letter box, so a completion scatters across the whole word
	// rather than out of its middle. Boxes come from the word itself because the
	// font is proportional.
	for _, b := range boxes {
		e.Burst(b, mult)
	}
	e.flash = 0.05
	if chain >= 2 {
		e.banner = chainBanner(chain)
		e.bannerT = 0.9
	}
}

func chainBanner(chain int) string {
	switch {
	case chain >= 5:
		return "CHAIN x8"
	case chain == 4:
		return "CHAIN x5"
	case chain == 3:
		return "CHAIN x3"
	default:
		return "CHAIN x2"
	}
}

// Impact is a word reaching the floor.
func (e *Effects) Impact() {
	e.shake = 0.2
	e.flash = 0.06
}

func (e *Effects) Update(dt float32) {
	alive := e.parts[:0]
	for _, p := range e.parts {
		p.life -= dt
		if p.life <= 0 {
			continue
		}
		p.x += p.vx * dt
		p.y += p.vy * dt
		alive = append(alive, p)
	}
	e.parts = alive

	livePopups := e.popups[:0]
	for _, p := range e.popups {
		p.life -= dt
		if p.life <= 0 {
			continue
		}
		p.y -= 2.5 * dt // drift up out of the way of the field
		livePopups = append(livePopups, p)
	}
	e.popups = livePopups

	if e.flash > 0 {
		e.flash -= dt
	}
	if e.shake > 0 {
		e.shake -= dt
	}
	if e.bannerT > 0 {
		e.bannerT -= dt
		if e.bannerT <= 0 {
			e.banner = ""
		}
	}
}

// ShakeOffset is the whole-playfield displacement for this frame.
func (e *Effects) ShakeOffset(frame int) (int, int) {
	if e.shake <= 0 {
		return 0, 0
	}
	// Deterministic from the frame counter so a seeded replay shakes identically.
	switch frame % 4 {
	case 0:
		return 1, 0
	case 1:
		return -1, 1
	case 2:
		return 0, -1
	default:
		return -1, 0
	}
}

func (e *Effects) Flashing() bool { return e.flash > 0 }

func (e *Effects) draw(scr tcell.Screen, l Layout, dx, dy int) {
	for _, p := range e.parts {
		x, y := l.PlayX+int(p.x)+dx, l.PlayY+int(p.y)+dy
		if x < l.PlayX || x >= l.PlayX+l.PlayW || y < l.PlayY || y > l.PlayY+l.PlayH-1 {
			continue
		}
		// Decay through the ramp so a particle visibly cools rather than
		// blinking out.
		i := int((1 - p.life/p.max) * float32(len(Ramp)-1))
		if i < 0 {
			i = 0
		}
		if i >= len(Ramp) {
			i = len(Ramp) - 1
		}
		scr.SetContent(x, y, Ramp[i], nil, tcell.StyleDefault.Foreground(p.colour))
	}
}

func (e *Effects) drawPopups(scr tcell.Screen, l Layout, dx, dy int) {
	for _, p := range e.popups {
		y := l.PlayY + int(p.y) + dy
		if y < l.PlayY || y > l.PlayY+l.PlayH-1 {
			continue
		}
		st := tcell.StyleDefault.Foreground(p.colour).Bold(true)
		for i, ch := range p.text {
			x := l.PlayX + int(p.x) + i + dx
			if x >= l.PlayX && x < l.PlayX+l.PlayW {
				scr.SetContent(x, y, ch, nil, st)
			}
		}
	}
}

// Count is the live particle count, for the debug overlay.
func (e *Effects) Count() int { return len(e.parts) }
