package render

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"keyboardwarrior/internal/engine"
)

var (
	styTitle = tcell.StyleDefault.Foreground(tcell.ColorAqua).Bold(true)
	// styName is the game's name on the menu. Plain white, on its own.
	styName     = tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
	styPick     = tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
	styUnpicked = tcell.StyleDefault.Foreground(tcell.ColorGray)
	styBody     = tcell.StyleDefault.Foreground(tcell.ColorSilver)
	styKey      = tcell.StyleDefault.Foreground(tcell.ColorLime).Bold(true)
	styHead     = tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)

	// The signal is white, not the menu's aqua. Aqua is the game's chrome; a
	// hoist of flags reading a plain order is the one screen that should look
	// like it came from somewhere else.
	stySignal = tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
)

// Menu item identities, in order. Labels are built per frame because two of them
// carry state.
const (
	MenuPlay = iota
	MenuGuide
	MenuStats
	MenuSound
	MenuMusic
	MenuQuit
	MenuCount
)

// MenuLabels builds the visible menu for the current audio settings.
func MenuLabels(sound, music bool) []string {
	return []string{
		"PLAY",
		"HOW TO PLAY",
		"STATS",
		"SOUND    " + onOff(sound),
		"MUSIC    " + onOff(music),
		"QUIT",
	}
}

func onOff(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}

// The game's name, in two pieces. The block font needs about five cells a
// letter, so the whole of it on one line wants 120 against the 78 an 80-column
// terminal has. The name is set in blocks and the rest reads as a tagline under
// it; the colon exists only to join the two on one line, and is dropped here.
const (
	gameName    = "NELSON"
	gameTagline = "HERO OF THE KEYS"
)

// titleTrailer is what the title block costs under the name itself: a blank row,
// the tagline, and a blank row before whatever comes next. The name is glyph ink
// and the tagline is plain text, so with no row between them the tagline reads as
// the bottom of the letters rather than as a line of its own.
const titleTrailer = 3

// titleGrid rasterizes the name at the largest tier that still leaves room for
// what has to go under it, and reports how many rows the whole title block takes
// including the tagline and the blank row beneath.
//
// The name is twice the size at the huge tier and three rows taller. Those three
// rows come out of the monument, so on a short terminal the choice is a big name
// or a column with Nelson on it, and the column wins: the menu is inscribed on
// it and has nowhere else to go.
func titleGrid(l Layout, avail, need int) ([][]rune, int) {
	for _, tier := range []engine.Tier{engine.TierHuge, engine.TierNormal} {
		grid := Rasterize(gameName, tier, 0)
		rows := len(grid) + titleTrailer
		if gridWidth(grid) <= l.PlayW && avail-rows >= need {
			return grid, rows
		}
	}
	grid := Rasterize(gameName, engine.TierNormal, 0)
	return grid, len(grid) + titleTrailer
}

// drawTitle puts the game's name above the monument and returns the first free
// row under it.
func drawTitle(scr tcell.Screen, l Layout, top int, grid [][]rune) int {
	drawGrid(scr, l, grid, centreX(l, gridWidth(grid)), top, styName)

	// The tagline is letter-spaced to bring it near the width of the name above
	// it. Plain text at one cell a letter under a name at five reads as a caption
	// that has fallen off something rather than as part of the title, and the
	// gap has to widen again when the name doubles.
	gap := 1
	if len(grid) > engine.HeightCells() {
		gap = 2
	}
	putMid(scr, l, top+len(grid)+1, spaced(gameTagline, gap), styBody)
	return top + len(grid) + titleTrailer
}

// spaced sets s out with gap cells between its letters.
func spaced(s string, gap int) string {
	var b strings.Builder
	for i, ch := range s {
		if i > 0 {
			b.WriteString(strings.Repeat(" ", gap))
		}
		b.WriteRune(ch)
	}
	return b.String()
}

// Leader is one row of the menu's leaderboard.
//
// No date. The board is a ranking, and when a run happened says nothing about
// where it places; it was a third column of noise next to the two that matter.
// The date is still kept per entry in the saved stats.
type Leader struct {
	Score   int
	Seconds float64
}

// DrawMenu is the opening screen. It is drawn inside the same bordered playfield
// as the game, so nothing jumps when a run starts.
//
// The menu is inscribed on the monument's pedestal rather than floating beside
// it. That is what decides the layout: the pedestal is as wide as the widest
// label needs, the rest of the column is measured out from the pedestal, and the
// whole monument is centred under the title.
//
// There is no leaderboard here any more. It never fitted at 80x24, it cost the
// column its shaft everywhere else, and the stats screen exists to show exactly
// that table. A menu that is a menu and a monument reads better than either
// with a scoreboard bolted under it.
func DrawMenu(scr tcell.Screen, l Layout, selected int, items []string, note string) {
	scr.Clear()
	drawFrame(scr, l)

	last := l.PlayY + l.PlayH - 1
	if note != "" {
		last-- // the note sits on the bottom row, on its own
	}
	avail := last - l.PlayY + 1

	// The monument needs 256 colours to be stone rather than a grey smear. On a
	// terminal without them, or one too narrow to stand it up, the menu falls back
	// to the plain centred list it was before.
	stone := scr.Colors() >= 256 && monumentW(items) <= l.PlayW

	// The title is sized against whatever has to fit under it, so the name grows
	// only into rows nothing else needs.
	need := len(items)
	if stone {
		need = capitalCells() + pedestalCells(len(items)) + 1 // a cell of shaft
	}
	grid, titleRows := titleGrid(l, avail, need)

	statue, shaft, ok := nelsonPlan(avail-titleRows, len(items))
	ok = ok && stone

	body := titleRows
	if ok {
		body += monumentCells(statue, shaft, len(items))
	} else {
		body += len(items)
	}

	// Everything above the note is centred as one stack.
	y := l.PlayY + (avail-body)/2
	if y < l.PlayY {
		y = l.PlayY
	}
	y = drawTitle(scr, l, y, grid)
	if ok {
		drawMonument(scr, l, y, statue, shaft, items, selected)
	} else {
		drawItems(scr, l, y, selected, items)
	}

	if note != "" {
		putMid(scr, l, l.PlayY+l.PlayH-1, note, styUnpicked)
	}
	put(scr, l.PlayX, l.StatusY,
		" up/down or j/k to move   enter to choose   esc to quit", styStatus)
}

// markerW is the gutter the selection arrow lives in, left of the labels.
const markerW = 2

// drawItems draws the menu as a left-aligned block that is itself centred. It is
// the fallback for a terminal that cannot draw the monument.
//
// Centring each label on its own put PLAY, HOW TO PLAY and SOUND ON at three
// different left edges, so the list wandered from row to row and the selection
// arrow moved with it. One block, one left edge, one column for the arrow.
func drawItems(scr tcell.Screen, l Layout, y, selected int, items []string) {
	x := centreX(l, menuTextW(items))
	for i, item := range items {
		style := styUnpicked
		if i == selected {
			style = styPick
			put(scr, x, y+i, ">", styKey)
		}
		put(scr, x+markerW, y+i, item, style)
	}
}

// Seconds2 formats a leaderboard entry's duration.
func (e Leader) Seconds2() string {
	n := int(e.Seconds)
	return fmt.Sprintf("%02d:%02d", n/60, n%60)
}

// DrawStats is the detailed figures and the leaderboard. Both used to be split
// between here and the menu, which meant the menu carried a table it had no room
// for and this screen showed lifetime totals with no runs against them.
func DrawStats(scr tcell.Screen, l Layout, rows [][2]string, top []Leader) {
	scr.Clear()
	drawFrame(scr, l)
	put(scr, l.PlayX+2, l.PlayY, "STATS", styHead)

	last := l.PlayY + l.PlayH - 1
	y := l.PlayY + 2
	for _, r := range rows {
		if y > last {
			break
		}
		if r[0] != "" {
			put(scr, l.PlayX+4, y, r[0], styBody)
			put(scr, l.PlayX+4+26, y, r[1], styPick)
		}
		y++
	}

	// The runs themselves, under the totals. Columns are fixed width so the
	// scores line up by place value rather than by where the rank happens to end.
	if len(top) > 0 && y+1 < last {
		y++
		put(scr, l.PlayX+2, y, "TOP RUNS", styHead)
		y++
		for i, e := range top {
			if y > last {
				break
			}
			style := styBody
			if i == 0 {
				style = styKey
			}
			put(scr, l.PlayX+4, y,
				fmt.Sprintf("%d.%9d%9s", i+1, e.Score, e.Seconds2()), style)
			y++
		}
	}
	put(scr, l.PlayX, l.StatusY, " esc to go back", styStatus)
}

// guidePage is one screen of the guide.
type guidePage struct {
	title string
	rows  []guideRow
}

// guideRow is one line of a page: a section heading, a blank, a line of prose,
// or a key in the left column with its explanation beside it.
//
// The key column is what makes this a reference rather than an essay. A player
// who opens HOW TO PLAY halfway through a session wants to find one thing, and
// they find it by scanning a column of marks and multipliers, not by reading
// four paragraphs to see whether the answer is in them.
type guideRow struct {
	head bool        // a section heading
	key  string      // left column: a key, a mark, a multiplier, a colour
	sty  tcell.Style // the key's own colour, so a x5 row is drawn x5's colour
	text string
}

func sect(s string) guideRow  { return guideRow{head: true, text: s} }
func prose(s string) guideRow { return guideRow{text: s} }
func spacer() guideRow        { return guideRow{} }

func entry(key, text string) guideRow {
	return guideRow{key: key, sty: styKey, text: text}
}

// depthRow is a scoring row drawn in the colour the field actually uses for
// that multiplier, so the table is a key to the screen rather than a list of
// numbers to hold beside it.
func depthRow(key string, c tcell.Color, text string) guideRow {
	return guideRow{key: key, sty: tcell.StyleDefault.Foreground(c).Bold(true), text: text}
}

// The guide is paged rather than scrolled: it does not fit one 80x24 screen, and
// scrolling text in a terminal game is worse than pages.
//
// It is a manual, in the order a player needs it: what the keys do, what things
// are worth, how to score properly, what the marks mean, and what the colours on
// the field are telling you. The story is the intro's job and is not repeated
// here beyond the one line that says who you are and what is falling.
//
// Plain text, not the block font. The block font is for words you must destroy
// under time pressure; a page of it is unreadable.
var guidePages = []guidePage{
	{"HOW TO PLAY", []guideRow{
		prose("Dr Johnson's zombie is burying London in words. You are Nelson."),
		prose("Type the words before they reach the floor. You get three lives."),
		spacer(),
		sect("CONTROLS"),
		entry("A-Z 0-9", "take a letter off a word"),
		entry(". - / +", "power-up marks, typed like any other letter"),
		entry("ESC", "back to the menu, or quit from the menu"),
		entry("SPACE", "start the next run, from the game over screen"),
		spacer(),
		sect("YOU DO NOT PICK A TARGET"),
		prose("A key takes a letter off the LOWEST word that wants it, so words"),
		prose("sharing a letter compete for your keys."),
		spacer(),
		prose("    SOURCE        SPITE        STRAWMAN"),
		prose("    press S three times and all three lose their S, lowest first."),
	}},
	{"SCORING", []guideRow{
		prose("A letter is worth the depth it died at. Finishing a word pays"),
		prose("its length, times that depth, times the chain."),
		spacer(),
		sect("A WORD'S COLOUR IS ITS MULTIPLIER"),
		depthRow("x1", tcell.ColorAqua, "the top third of the field"),
		depthRow("x2", tcell.ColorYellow, "the middle third"),
		depthRow("x3", tcell.ColorOrangeRed, "the lower third"),
		depthRow("x5", tcell.ColorWhite, "the last two rows above the floor"),
		spacer(),
		sect("FOR EXAMPLE"),
		prose("SOURCE finished on the bottom row, third word in a chain:"),
		prose("6 letters x 5 depth x 3 chain = 90 for the word, on top of the"),
		prose("5 a letter it paid on the way down."),
	}},
	{"CHAINS", []guideRow{
		prose("Finish a word, then finish another within 1.5 seconds, and they"),
		prose("chain. The window reopens with each word. A typo breaks it."),
		spacer(),
		entry("x2", "2 words in a row"),
		entry("x3", "3 words"),
		entry("x5", "4 words"),
		entry("x8", "5 or 6 words"),
		entry("x12", "7 to 9 words"),
		entry("x20", "10 or more"),
		spacer(),
		prose("Past four, a chain pays in survival as well: a breather at four,"),
		prose("a shield at six, the lower field swept at eight, a life at ten."),
		spacer(),
		prose("The tactic: leave several words on their last letter, where they"),
		prose("turn green and the top bar counts them, then fire those letters"),
		prose("back to back."),
	}},
	{"POWER-UPS AND BOMBS", []guideRow{
		prose("A word ending in a punctuation mark is a power-up. Type the mark"),
		prose("as its last letter to set it off."),
		spacer(),
		entry(".", "FREEZE   everything stops"),
		entry("-", "REWIND   the field is pushed back up"),
		entry("/", "BLAST    every word near it is destroyed"),
		entry("+", "REPAIR   one life back"),
		spacer(),
		prose("They ride the longest words, so taking one is a commitment."),
		prose("Missing one costs nothing."),
		spacer(),
		sect("BOMBS"),
		prose("Red, in brackets, and finishing one costs a life. A bomb is only"),
		prose("ever offered a letter no other word wants, so you cannot be"),
		prose("forced into one. Leave it alone and it lands harmlessly."),
	}},
	{"READING THE FIELD", []guideRow{
		prose("The field only ever gets faster."),
		spacer(),
		prose("Every letter you take makes that word fall faster, so a half-eaten"),
		prose("long word is the most dangerous thing on screen."),
		prose("A typo heats the whole field for a few seconds."),
		prose("A word that catches another passes its speed down to it."),
		spacer(),
		sect("COLOURS"),
		entry("GREEN", "one key from done, underlined: chain fuel"),
		entry("RED", "in brackets, a bomb: never finish it"),
		entry("PULSING", "about to land"),
		entry("MAGENTA", "a power-up, named above the word"),
		spacer(),
		prose("The bright letter at the front of a word is the one your next key"),
		prose("takes. A mark under a letter means another word wants that key too."),
	}},
}

// GuidePages is how many pages the guide has.
func GuidePages() int { return len(guidePages) }

// Guide layout, playfield-relative. Headings sit a little left of the rows they
// cover, so a page reads as a table with sections rather than as a wall of text
// starting at one margin.
const (
	guideHeadX  = 2
	guideItemX  = 4
	guideKeyGap = 2
)

// guideTextX is where the right-hand column starts, measured across every page
// rather than per page, so turning a page does not shift the text sideways.
var guideTextX = guideItemX + widestGuideKey() + guideKeyGap

func widestGuideKey() int {
	w := 0
	for _, p := range guidePages {
		for _, r := range p.rows {
			if !r.head {
				w = max(w, len([]rune(r.key)))
			}
		}
	}
	return w
}

// x is where the row starts, playfield-relative.
func (r guideRow) x() int {
	switch {
	case r.head:
		return guideHeadX
	case r.key == "":
		return guideItemX
	}
	return guideTextX
}

// width is how many columns the row draws into, for the fit check.
func (r guideRow) width() int { return r.x() + len([]rune(r.text)) }

// DrawGuide renders one page.
func DrawGuide(scr tcell.Screen, l Layout, page int) {
	scr.Clear()
	drawFrame(scr, l)
	if page < 0 || page >= len(guidePages) {
		return
	}
	p := guidePages[page]

	put(scr, l.PlayX+guideHeadX, l.PlayY, p.title, styHead)
	for i, r := range p.rows {
		y := l.PlayY + 2 + i
		if y > l.PlayY+l.PlayH-1 {
			break
		}
		style := styBody
		if r.head {
			style = styHead
		}
		if r.key != "" {
			put(scr, l.PlayX+guideItemX, y, r.key, r.sty)
		}
		put(scr, l.PlayX+r.x(), y, r.text, style)
	}

	nav := fmt.Sprintf(" page %d of %d   left and right to turn   esc to go back",
		page+1, len(guidePages))
	put(scr, l.PlayX, l.StatusY, nav, styStatus)
}

// readyTextRows is what the signal costs: the two lines of the order, a blank,
// and the line telling the player to press a key.
const readyTextRows = 4

// shipCells is the ship's drawn height. Two pixel rows to a cell, as with the
// monument's art.
func shipCells() int { return len(shipArt) / 2 }

// shipGap is the sky between her waterline and the first line of the signal.
// Without it the words sit on the sea and read as part of the picture.
const shipGap = 1

// DrawReady is Nelson's signal, held between the menu and the first word.
//
// The ship is flown above the words rather than below them because the signal
// came off Victory's masts, and the eye should reach the order having just been
// told who is giving it.
//
// All one voice. The words were half block font and half plain text, which read
// as two messages stapled together rather than one signal being flown.
func DrawReady(scr tcell.Screen, l Layout) {
	scr.Clear()
	drawFrame(scr, l)

	// She needs 256 colours to be a ship rather than a grey smudge, and room to
	// float without pushing the order off the screen. Short of either, the signal
	// is flown on its own, which is what this screen was before her.
	ship := scr.Colors() >= 256 &&
		shipW <= l.PlayW &&
		shipCells()+shipGap+readyTextRows+2 <= l.PlayH

	body := readyTextRows
	if ship {
		body += shipCells() + shipGap
	}

	y := l.PlayY + (l.PlayH-body)/2
	if ship {
		drawSprite(scr, l, shipArt, shipW, shipRamp, centreX(l, shipW), y)
		y += shipCells() + shipGap
	}
	putCentered(scr, l.TermW, y, "ENGLAND EXPECTS", stySignal)
	putCentered(scr, l.TermW, y+1, "THAT EVERY MAN WILL DO HIS DUTY", stySignal)
	putCentered(scr, l.TermW, y+3, "press any key", styKey)
}

// drawFrame is the border and nothing else, so menus sit in the same box the game
// does and the transition into a run does not jump.
func drawFrame(scr tcell.Screen, l Layout) {
	right := l.X + l.W - 1
	bottom := l.Y + l.H - 1
	hline(scr, l.X, right, l.Y, '─', styBorder)
	hline(scr, l.X, right, bottom, '─', styBorder)
	for y := l.Y + 1; y < bottom; y++ {
		scr.SetContent(l.X, y, '│', nil, styBorder)
		scr.SetContent(right, y, '│', nil, styBorder)
	}
	scr.SetContent(l.X, l.Y, '┌', nil, styBorder)
	scr.SetContent(right, l.Y, '┐', nil, styBorder)
	scr.SetContent(l.X, bottom, '└', nil, styBorder)
	scr.SetContent(right, bottom, '┘', nil, styBorder)
}

// WrapCheck reports any guide row too wide for the playfield, for tests.
func WrapCheck(playW int) []string {
	var bad []string
	for _, p := range guidePages {
		if guideHeadX+len([]rune(p.title))+2 > playW {
			bad = append(bad, p.title)
		}
		for _, r := range p.rows {
			if r.width()+2 > playW {
				bad = append(bad, strings.TrimSpace(r.text))
			}
		}
	}
	return bad
}

// GuideRowsNeeded is the tallest page, for tests against the screen contract.
func GuideRowsNeeded() int {
	n := 0
	for _, p := range guidePages {
		if len(p.rows) > n {
			n = len(p.rows)
		}
	}
	return n + 2
}

// The monument is measured out from the menu, because the menu is inscribed on
// its pedestal. Everything else follows from how wide the widest label is.
const (
	panelPad    = 1 // stone between the inscription and the sunk panel's edge
	pedestalPad = 2 // stone between the panel and the pedestal's edge
	basePad     = 1 // how far the base steps out past the pedestal
)

// widestLabel is the longest any menu label can ever be, whatever the toggles
// are set to. OFF is a character longer than ON, so both off is the widest the
// menu ever gets.
var widestLabel = labelW(MenuLabels(false, false))

func labelW(items []string) int {
	w := 0
	for _, item := range items {
		w = max(w, len([]rune(item)))
	}
	return w
}

// menuTextW is the inscription itself: the arrow gutter plus the widest label.
//
// It is measured against the longest form the labels can take rather than the
// ones on screen, because every stone in the monument is sized from this number
// and SOUND ON is a cell narrower than SOUND OFF. Following the current labels
// made the whole column widen by two cells and shift by one the moment a toggle
// was pressed, which read as the screen twitching rather than as a setting
// changing. A shorter label now leaves a cell of blank panel on its right, which
// is what an inscription does anyway.
func menuTextW(items []string) int {
	return markerW + max(labelW(items), widestLabel)
}

func panelW(items []string) int    { return menuTextW(items) + 2*panelPad }
func pedestalW(items []string) int { return panelW(items) + 2*pedestalPad }
func monumentW(items []string) int { return pedestalW(items) + 2*basePad }

// statueCells and capitalCells come from the art. Two pixel rows to a cell.
func statueCells() int  { return len(nelsonStatue) / 2 }
func capitalCells() int { return len(nelsonCapital) / 2 }

// pedestalCells is the cornice, the inscription and the base.
//
// There is no sill under the panel. It looked better with one, and it cost the
// statue: a spare row buys a cell of shaft or a cell of Nelson, and at 100x30 a
// single row was the difference between a monument with a man on it and a
// monument without.
func pedestalCells(items int) int { return items + 2 }

// maxShaftCells stops the column becoming a flagpole on a tall terminal. A
// monument is allowed to be shorter than the room it stands in, and the rows it
// does not take go to the leaderboard under it.
const maxShaftCells = 5

// monumentCells is the drawn height of the whole thing.
func monumentCells(statue bool, shaft, items int) int {
	n := capitalCells() + shaft + pedestalCells(items)
	if statue {
		n += statueCells()
	}
	return n
}

// nelsonPlan fits the monument to the rows available. The pedestal is fixed by
// the menu and the capital by the art; what is left goes to the statue first and
// to the shaft after that.
//
// The statue is the piece that drops. At 80x24 there are eighteen playable rows,
// and a capital, a six-line inscription and a base already want thirteen of
// them: a figure eight cells tall does not fit beside a title as well, whatever
// order the rows are handed out in. Above about thirty rows he is back.
func nelsonPlan(rows, items int) (statue bool, shaft int, ok bool) {
	fixed := capitalCells() + pedestalCells(items)
	if rows <= fixed {
		return false, 0, false // no shaft at all, and the capital would float
	}
	spare := rows - fixed
	if spare >= statueCells() {
		statue, spare = true, spare-statueCells()
	}
	return statue, min(spare, maxShaftCells), true
}

// drawMonument stands the column under the title with the menu cut into its
// pedestal, and returns the row below the base.
func drawMonument(scr tcell.Screen, l Layout, y int, statue bool, shaft int, items []string, selected int) int {
	x := centreX(l, monumentW(items))
	pedestal := x + basePad

	// The art is a fixed width and narrower than the pedestal, so it is centred
	// on the pedestal rather than on the monument's overall footprint.
	art := pedestal + (pedestalW(items)-nelsonW)/2

	if statue {
		y = drawStone(scr, l, nelsonStatue, art, y)
	}
	y = drawStone(scr, l, nelsonCapital, art, y)
	for i := 0; i < shaft; i++ {
		y = drawStone(scr, l, []string{nelsonShaft, nelsonShaft}, art, y)
	}
	return drawPedestal(scr, l, pedestal, y, items, selected)
}

// drawPedestal is the block the menu is inscribed on: a cornice, a panel sunk
// into the face carrying the items, a sill, and a base that steps out past all
// of it.
//
// It is drawn rather than generated because its width comes from the menu and
// its height from how many items there are, neither of which the art knows. The
// panel is one flat dark tone rather than shaded like the rest of the stone:
// text over a gradient is unreadable, and a sunk panel is what a monument does
// with an inscription anyway.
func drawPedestal(scr tcell.Screen, l Layout, x, y int, items []string, selected int) int {
	pw, panel := pedestalW(items), panelW(items)
	sunk := tcell.StyleDefault.Background(nelsonRamp[1])

	stoneRow(scr, l, x, y, pw, 5) // cornice, catching the light
	y++

	for i, item := range items {
		stoneRow(scr, l, x, y, pw, 3)
		fill(scr, l, x+pedestalPad, y, panel, sunk)

		style := sunk.Foreground(tcell.ColorSilver)
		if i == selected {
			style = sunk.Foreground(tcell.ColorWhite).Bold(true)
			put(scr, x+pedestalPad+panelPad, y, ">", sunk.Foreground(tcell.ColorLime).Bold(true))
		}
		put(scr, x+pedestalPad+panelPad+markerW, y, item, style)
		y++
	}

	stoneRow(scr, l, x-basePad, y, monumentW(items), 2) // base
	return y + 1
}

// stoneRow paints one band of pedestal, lit from the left the way the generated
// art is, so drawn stone and bitmap stone are the same material.
func stoneRow(scr tcell.Screen, l Layout, x, y, w, top int) {
	const lo, soft = 1, 3
	span := max(w-1, soft)
	for i := 0; i < w; i++ {
		t := float64(top) - float64(top-lo)*float64(i)/float64(span)
		n := int(t + 0.5)
		if n < 0 {
			n = 0
		}
		if n >= len(nelsonRamp) {
			n = len(nelsonRamp) - 1
		}
		fill(scr, l, x+i, y, 1, tcell.StyleDefault.Background(nelsonRamp[n]))
	}
}

// fill paints w cells of solid colour, clipped to the playfield.
func fill(scr tcell.Screen, l Layout, x, y, w int, st tcell.Style) {
	if y < l.PlayY || y > l.PlayY+l.PlayH-1 {
		return
	}
	for i := 0; i < w; i++ {
		if c := x + i; c >= l.PlayX && c < l.PlayX+l.PlayW {
			scr.SetContent(c, y, ' ', nil, st)
		}
	}
}

// drawStone paints a piece of the monument with its top-left at (x0, y0) and
// returns the row below it.
func drawStone(scr tcell.Screen, l Layout, rows []string, x0, y0 int) int {
	return drawSprite(scr, l, rows, nelsonW, nelsonRamp, x0, y0)
}

// drawSprite paints one of the hand-drawn bitmaps: the monument and the ship,
// both one digit per pixel over an eight-tone ramp, two pixel rows to a cell.
//
// It is not drawArt. The plates own every cell they cover, because they are
// photographs and a photograph has a background. These are silhouettes standing
// on a screen with other things on it, so a cell with one lit pixel keeps the
// terminal's own background and the shape has an edge rather than a black box
// around it.
func drawSprite(scr tcell.Screen, l Layout, rows []string, w int, ramp *[8]tcell.Color, x0, y0 int) int {
	for cy := 0; cy < len(rows)/2; cy++ {
		top, bottom := rows[cy*2], rows[cy*2+1]
		y := y0 + cy
		if y < l.PlayY || y >= l.PlayY+l.PlayH {
			continue
		}
		for cx := 0; cx < w; cx++ {
			x := x0 + cx
			if x < l.PlayX || x >= l.PlayX+l.PlayW {
				continue
			}
			t, b := top[cx], bottom[cx]
			switch {
			case t == '.' && b == '.':
				continue
			case b == '.':
				scr.SetContent(x, y, '▀', nil,
					tcell.StyleDefault.Foreground(ramp[t-'0']))
			case t == '.':
				scr.SetContent(x, y, '▄', nil,
					tcell.StyleDefault.Foreground(ramp[b-'0']))
			default:
				scr.SetContent(x, y, '▀', nil, tcell.StyleDefault.
					Foreground(ramp[t-'0']).
					Background(ramp[b-'0']))
			}
		}
	}
	return y0 + len(rows)/2
}
