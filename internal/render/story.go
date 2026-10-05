package render

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

var (
	styJohnson = tcell.StyleDefault.Foreground(tcell.ColorMediumPurple)
	styStone   = tcell.StyleDefault.Foreground(tcell.ColorGray)
	styNavy    = tcell.StyleDefault.Foreground(tcell.ColorSteelBlue)
)

// Johnson appears in the intro and nowhere else. He used to pace a band above
// the playfield during play; that band is gone, and its three rows went back to
// the field, which is why the maximum playable area is 34 rows again.
//
// johnsonTallFrames is the drawn figure, kept for the terminals the portrait
// cannot reach: below 256 colours the bitmap is a grey smudge, and this is not a
// degraded version of it but the only version that works there.
//
// He is drawn from dot bitmaps like the font, so he moves in sub-cell steps and
// works under either drawing style. Two frames on a slow cycle: a corpse that
// has been dead since 1784 should not stride. What identifies him is the
// full-bottomed wig, wider than the man and hanging past his shoulders, and the
// stick, which is one column and does more period work than any detail spent on
// a face this small.
var johnsonTallFrames = [][]string{
	{
		".....######.........", // wig crown
		"....########........",
		"...###....###.......", // wig at the sides, face between
		"...###.##.###.......", // eyes
		"...##########.......", // jaw
		"..############......", // the wig comes down onto the shoulders
		"..###########.###...", // coat; hand on the stick
		"..############..#...",
		"..############..#...", // stout, which he was
		".##############.#...", // skirts
		"...###....###...#...", // stockings
		"..####....####..#...", // buckled shoes
	},
	{
		".....######.........",
		"....########........",
		"...###....###.......",
		"...###.##.###.......",
		"...##########.......",
		"..############......",
		"..###########.###...",
		"..############..#...",
		"..############..#...",
		".##############.#...",
		"...###....###...#...",
		".####......####.#...",
	},
}

// johnsonDots turns a frame into the dot grid the packer wants.
func johnsonDots(frames [][]string, frame int) [][]bool {
	rows := frames[frame%len(frames)]
	out := make([][]bool, len(rows))
	for y, r := range rows {
		out[y] = make([]bool, len(r))
		for x := 0; x < len(r); x++ {
			out[y][x] = r[x] == '#'
		}
	}
	return out
}

// Story beats. The intro is a sequence of held tableaux rather than continuous
// motion: a terminal at 30 FPS could animate far more, but a few composed frames
// read as deliberate where continuous movement in text reads as a screensaver.
type beat struct {
	seconds float64
	draw    func(scr tcell.Screen, l Layout, t float64, tick int)
}

// The five beats. Four of them are plates: a picture with its words beside or
// under it, each tinted to its own subject rather than to one house palette, so
// the sequence moves stone, purple, stone, navy rather than sitting in one
// colour for fifteen seconds. The third is the exception on purpose - the words
// themselves are the picture there, and a run of five plates would flatten into
// a slideshow without it.
var introPlates = struct {
	abbey, johnson, lambeth, nelson introPlate
}{
	abbey: introPlate{
		art: &plateAbbey, title: "WESTMINSTER ABBEY", titleSty: styStone,
		lines: []string{"something is moving under the flagstones"},
	},
	johnson: introPlate{
		art: &plateJohnson, title: "DR SAMUEL JOHNSON", titleSty: styTitle,
		lines: []string{
			"lexicographer, deceased 1784",
			"has risen in horror at the decline of literacy",
		},
	},
	lambeth: introPlate{
		art: &plateLambeth, title: "THE MINISTRY OF DEFENCE", titleSty: styStone,
		lines: []string{
			"has consulted Lambeth Palace.",
			"There is one course of action left",
			"to a desperate nation.",
		},
	},
	nelson: introPlate{
		art: &plateNelson, title: "ADMIRAL LORD NELSON", titleSty: styNavy,
		lines: []string{"raised once more in defence of these islands"},
	},
}

var introBeats = []beat{
	{2.6, func(scr tcell.Screen, l Layout, t float64, tick int) {
		if !introPlates.abbey.draw(scr, l, t) {
			drawPlateText(scr, l, introPlates.abbey)
		}
	}},
	{3.4, func(scr tcell.Screen, l Layout, t float64, tick int) {
		if introPlates.johnson.draw(scr, l, t) {
			return
		}
		// His is the one beat with something to show when the picture cannot be
		// drawn: the figure the game used to pace above the playfield.
		drawPlateText(scr, l, introPlates.johnson)
		drawWalker(scr, l, l.PlayY+l.PlayH/2+2, t, tick)
	}},
	{3.4, func(scr tcell.Screen, l Layout, t float64, tick int) {
		mid := l.PlayY + l.PlayH/2
		putCentered(scr, l.TermW, mid-2, "Tired of life, he has grown weary of London", styBody)
		putCentered(scr, l.TermW, mid, "and means to bury it under an onslaught of verbiage", styBody)
		rainWords(scr, l, t, 10, l.PlayH-2)
	}},
	{3.6, func(scr tcell.Screen, l Layout, t float64, tick int) {
		if !introPlates.lambeth.draw(scr, l, t) {
			drawPlateText(scr, l, introPlates.lambeth)
		}
	}},
	{3.4, func(scr tcell.Screen, l Layout, t float64, tick int) {
		if !introPlates.nelson.draw(scr, l, t) {
			drawPlateText(scr, l, introPlates.nelson)
			drawColumn(scr, l, l.PlayY+l.PlayH/2+4, 1.0)
		}
	}},
}

// IntroSeconds is how long the whole sequence runs.
func IntroSeconds() float64 {
	var total float64
	for _, b := range introBeats {
		total += b.seconds
	}
	return total
}

// DrawIntro renders the sequence at a point in time. Returns false once it is
// finished.
func DrawIntro(scr tcell.Screen, l Layout, elapsed float64, tick int) bool {
	scr.Clear()
	drawFrame(scr, l)

	t := elapsed
	for _, b := range introBeats {
		if t < b.seconds {
			b.draw(scr, l, t, tick)
			putCentered(scr, l.TermW, l.PlayY+l.PlayH-1, "any key to skip", styUnpicked)
			return true
		}
		t -= b.seconds
	}
	return false
}

// introPlate is one beat's composition: a picture, a title, and body lines. The
// intro is a sequence of held tableaux rather than continuous motion, so each
// beat is a composed still and the only movement in it is the reveal.
type introPlate struct {
	art      *plateArt
	title    string
	titleSty tcell.Style
	lines    []string
}

// revealSeconds is how long a plate takes to come up out of the dark at the
// start of its beat. Long enough to read as a deliberate reveal, short enough
// that it is finished well inside the shortest beat.
const revealSeconds = 0.55

// plateGap is the space between a picture and the text next to or under it.
const plateGap = 3

// shade dims a palette index toward black. Scaling the index rather than
// blending the colours keeps every frame of the reveal inside the ramp, so it
// stays the same eight colours coming up rather than a wash of new ones.
func shade(i int, reveal float64) int {
	if reveal >= 1 {
		return i
	}
	if reveal <= 0 {
		return 0
	}
	return int(float64(i)*reveal + 0.5)
}

// fit picks the largest tier that leaves room for the text, or nil if none does
// and the beat has to fall back to words alone.
func (p introPlate) fit(scr tcell.Screen, l Layout) *plateTier {
	if scr.Colors() < 256 {
		return nil
	}
	textW, textH := p.textBox()
	for i := range p.art.Tiers {
		t := &p.art.Tiers[i]
		if p.art.Wide {
			// A band with its text underneath, so width is nearly free and the
			// rows have to cover both.
			if t.W <= l.PlayW && t.H+1+textH <= l.PlayH-1 {
				return t
			}
		} else if t.W+plateGap+textW <= l.PlayW && t.H <= l.PlayH-2 {
			return t
		}
	}
	return nil
}

func (p introPlate) textBox() (w, h int) {
	w = len(p.title)
	for _, line := range p.lines {
		if n := len(line); n > w {
			w = n
		}
	}
	return w, len(p.lines) + 2 // title, a blank row, then the body
}

// draw composes the plate, or reports that it could not and the caller should
// fall back to text.
func (p introPlate) draw(scr tcell.Screen, l Layout, t float64) bool {
	tier := p.fit(scr, l)
	if tier == nil {
		return false
	}
	reveal := t / revealSeconds
	if reveal > 1 {
		reveal = 1
	}
	textW, textH := p.textBox()

	var ax, ay, tx, ty int
	if p.art.Wide {
		ax = l.PlayX + (l.PlayW-tier.W)/2
		ay = l.PlayY + (l.PlayH-(tier.H+1+textH))/2
		tx = l.PlayX + (l.PlayW-textW)/2
		ty = ay + tier.H + 1
	} else {
		ax = l.PlayX + (l.PlayW-(tier.W+plateGap+textW))/2
		ay = l.PlayY + (l.PlayH-tier.H)/2
		tx = ax + tier.W + plateGap
		// Centred against the picture rather than the playfield, so the two read
		// as one plate instead of as two things that happen to be adjacent.
		ty = ay + (tier.H-textH)/2
	}

	drawArt(scr, l, p.art, *tier, ax, ay, reveal)
	put(scr, tx, ty, p.title, p.titleSty)
	for i, line := range p.lines {
		put(scr, tx, ty+2+i, line, styBody)
	}
	return true
}

// drawArt paints a tier with upper-half blocks: foreground is the top pixel of
// the cell, background the bottom, which doubles the vertical resolution for the
// price of owning the cell's background.
func drawArt(scr tcell.Screen, l Layout, art *plateArt, tier plateTier, x0, y0 int, reveal float64) {
	for cy := 0; cy < tier.H; cy++ {
		top, bottom := tier.Pixels[cy*2], tier.Pixels[cy*2+1]
		y := y0 + cy
		if y < l.PlayY || y >= l.PlayY+l.PlayH {
			continue
		}
		for cx := 0; cx < tier.W; cx++ {
			x := x0 + cx
			if x < l.PlayX || x >= l.PlayX+l.PlayW {
				continue
			}
			sty := tcell.StyleDefault.
				Foreground(art.Ramp[shade(int(top[cx]-'0'), reveal)]).
				Background(art.Ramp[shade(int(bottom[cx]-'0'), reveal)])
			scr.SetContent(x, y, '\u2580', nil, sty)
		}
	}
}

// drawPlateText is the fallback: the same words, centred, with no picture. It is
// what a terminal below 256 colours gets, and what any terminal gets if the
// smallest tier still will not fit.
func drawPlateText(scr tcell.Screen, l Layout, p introPlate) {
	_, textH := p.textBox()
	y := l.PlayY + (l.PlayH-textH)/2
	putCentered(scr, l.TermW, y, p.title, p.titleSty)
	for i, line := range p.lines {
		putCentered(scr, l.TermW, y+2+i, line, styBody)
	}
}

func drawWalker(scr tcell.Screen, l Layout, row int, t float64, tick int) {
	grid := Pack(johnsonDots(johnsonTallFrames, tick/8), 0)
	x0 := l.PlayX + 4 + int(t*6)
	for gy, r := range grid {
		y := row + gy
		if y < l.PlayY || y > l.PlayY+l.PlayH-1 {
			continue
		}
		for gx, ch := range r {
			if ch == Blank {
				continue
			}
			if x := x0 + gx; x >= l.PlayX && x < l.PlayX+l.PlayW {
				scr.SetContent(x, y, ch, nil, styJohnson)
			}
		}
	}
}

// rainWords drops a scatter of words down the field, deterministically from the
// time so the sequence looks the same every run.
// rainWords drops a scatter of words, deterministically from the time so the
// sequence looks the same every run.
//
// rows bounds it to the sky. Letting it fall over the skyline made both
// unreadable: two kinds of dense content in the same cells is not two things, it
// is noise.
func rainWords(scr tcell.Screen, l Layout, t float64, n, rows int) {
	if rows <= 0 {
		return
	}
	for i := 0; i < n; i++ {
		word := introRain[i%len(introRain)]
		x := l.PlayX + (i*23+3)%maxInt(1, l.PlayW-len(word))
		y := l.PlayY + int(t*6+float64(i*5))%rows
		for j, ch := range word {
			if x+j < l.PlayX+l.PlayW {
				scr.SetContent(x+j, y, ch, nil, styUnpicked)
			}
		}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var introRain = []string{
	"PEDANTRY", "VERBIAGE", "LEXICON", "GLOSSARY", "SOPHISTRY",
	"CIRCUMLOCUTION", "ETYMOLOGY", "PROLIXITY",
}

// drawColumn is Nelson's Column, rising or falling depending on how much of it is
// standing.
func drawColumn(scr tcell.Screen, l Layout, base int, standing float64) {
	const height = 7
	x := l.PlayX + l.PlayW/2
	n := int(float64(height) * standing)
	for i := 0; i < n; i++ {
		y := base - i
		if y < l.PlayY || y > l.PlayY+l.PlayH-1 {
			continue
		}
		ch := '║'
		if i == n-1 && standing >= 1 {
			ch = '▲'
		}
		scr.SetContent(x, y, ch, nil, styStone)
	}
	if base >= l.PlayY && base <= l.PlayY+l.PlayH-1 {
		for dx := -3; dx <= 3; dx++ {
			if cx := x + dx; cx >= l.PlayX && cx < l.PlayX+l.PlayW {
				scr.SetContent(cx, base, '▄', nil, styStone)
			}
		}
	}
}

// DefeatSeconds is the length of the Trafalgar sequence.
const DefeatSeconds = 2.6

// DrawDefeat is what happens when the third life goes, before the score screen.
//
// It must stay short and it must always be skippable. Every keypress of friction
// between death and the next attempt costs attempts, and this is friction by
// construction; the only defence is brevity and an exit.
func DrawDefeat(scr tcell.Screen, l Layout, elapsed float64, killer string) bool {
	if elapsed >= DefeatSeconds {
		return false
	}
	scr.Clear()
	drawFrame(scr, l)

	base := l.PlayY + l.PlayH - 2
	sky := l.PlayH - skylineRows - 2
	skyline(scr, l, base)

	switch {
	case elapsed < 0.5:
		// A beat of silence with the city as it stood.
		drawColumn(scr, l, base-1, 1)
	case elapsed < 1.5:
		rainWords(scr, l, elapsed-0.5, 10, sky)
		drawColumn(scr, l, base-1, 1-(elapsed-0.5)*1.1)
	default:
		rainWords(scr, l, elapsed-0.5, 10, sky)
		putCentered(scr, l.TermW, l.PlayY+sky/2, "KISS ME, HARDY", styNavy)
		if killer != "" {
			putCentered(scr, l.TermW, l.PlayY+sky/2+2,
				fmt.Sprintf("undone by %s", killer), styBody)
		}
	}
	putCentered(scr, l.TermW, l.PlayY+l.PlayH-1, "any key", styUnpicked)
	return true
}

// skylineRows is how much of the bottom of the field London occupies.
const skylineRows = 5

// building is a width and a height, in cells.
type building struct{ w, h int }

// london is a fixed profile, so it is the same city every time. Contiguous blocks
// of varying width rather than one column each: single columns read as a bar
// chart, not a city.
var london = []building{
	{6, 2}, {3, 4}, {5, 1}, {2, 5}, {7, 2}, {4, 3}, {3, 1}, {8, 4},
	{2, 2}, {5, 5}, {6, 1}, {3, 3}, {9, 2}, {4, 4}, {2, 1}, {7, 3},
	{5, 2}, {3, 5}, {6, 3}, {4, 1}, {8, 2}, {2, 4}, {5, 3}, {6, 2},
}

func skyline(scr tcell.Screen, l Layout, base int) {
	x := l.PlayX
	for _, b := range london {
		if x >= l.PlayX+l.PlayW {
			break
		}
		for dx := 0; dx < b.w && x+dx < l.PlayX+l.PlayW; dx++ {
			for dy := 0; dy < b.h; dy++ {
				y := base - dy
				if y < l.PlayY || y > l.PlayY+l.PlayH-1 {
					continue
				}
				scr.SetContent(x+dx, y, '█', nil, styStone)
			}
		}
		x += b.w
	}
}
