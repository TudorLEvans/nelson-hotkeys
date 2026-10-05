package render

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"nelson/internal/engine"
	"nelson/internal/words"
)

// Lives are drawn with these rather than with the glyph blocks, so glyph ink
// outside the playfield stays detectable.
const (
	LifeFull = '▮'
	LifeGone = '▯'
)

var (
	styBorder  = tcell.StyleDefault.Foreground(tcell.ColorGray)
	styHUD     = tcell.StyleDefault.Foreground(tcell.ColorWhite)
	styFloor   = tcell.StyleDefault.Foreground(tcell.ColorSilver)
	styBroken  = tcell.StyleDefault.Foreground(tcell.ColorDarkRed)
	styStatus  = tcell.StyleDefault.Foreground(tcell.ColorGray)
	styWarn    = tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)
	styDebug   = tcell.StyleDefault.Foreground(tcell.ColorFuchsia)
	styLife    = tcell.StyleDefault.Foreground(tcell.ColorRed)
	styGone    = tcell.StyleDefault.Foreground(tcell.ColorDarkSlateGray)
	styNext    = tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
	styMark    = tcell.StyleDefault.Foreground(tcell.ColorFuchsia).Bold(true)
	styTrail   = tcell.StyleDefault.Foreground(tcell.ColorDarkSlateGray)
	styChain   = tcell.StyleDefault.Foreground(tcell.ColorFuchsia).Bold(true)
	styLoaded  = tcell.StyleDefault.Foreground(tcell.ColorLime).Bold(true)
	styDanger  = tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true)
	styKick    = tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
	styPower   = tcell.StyleDefault.Foreground(tcell.ColorFuchsia).Bold(true)
	styDefWord = tcell.StyleDefault.Foreground(tcell.ColorAqua).Bold(true)
	styDefText = tcell.StyleDefault.Foreground(tcell.ColorSilver)
	styHeat    = tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true)
	styBomb    = tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true)
)

// depthStyle colours a word from the same multiplier the scorer uses, so the
// colour on screen IS the score readout rather than a decoration that can drift
// out of step with it.
func depthStyle(w *engine.Word, playH int) tcell.Style {
	return tcell.StyleDefault.Foreground(multColour(engine.DepthMultiplier(w.BottomRow(), playH)))
}

// DrawTooSmall is the resize card. It updates live as the terminal is dragged.
func DrawTooSmall(scr tcell.Screen, l Layout) {
	scr.Clear()
	putCentered(scr, l.TermW, l.TermH/2-1, fmt.Sprintf("RESIZE TO %dx%d", l.MinCols, l.MinRows), styWarn)
	putCentered(scr, l.TermW, l.TermH/2+1, fmt.Sprintf("currently %dx%d", l.TermW, l.TermH), styStatus)
}

// Compose draws one frame.
func Compose(scr tcell.Screen, st *engine.State, l Layout, fx *Effects, dbg *Debug) {
	ComposeWithHint(scr, st, l, fx, dbg, false)
}

// ComposeWithHint is Compose plus whether the last run set a high score.
func ComposeWithHint(scr tcell.Screen, st *engine.State, l Layout, fx *Effects, dbg *Debug, highScore bool) {
	scr.Clear()
	dx, dy := 0, 0
	if fx != nil {
		dx, dy = fx.ShakeOffset(st.Frame)
	}

	if st.GameOver {
		// Border only. The HUD and the floor belong to a run in progress, and
		// every figure on them is repeated by the summary a few rows below.
		drawFrame(scr, l)
		drawGameOver(scr, l, st, highScore)
	} else {
		drawChrome(scr, l, st, dx, dy)
		contested := contestedLetters(st.Words)
		// Trails first, in their own pass. Drawn per-word they would overwrite
		// the glyphs of words above, because a lower word is drawn later.
		for i := range st.Words {
			drawTrail(scr, l, st, &st.Words[i], dx, dy)
		}
		for i := range st.Words {
			drawWord(scr, l, st, &st.Words[i], i, st.PlayH, contested, dx, dy)
		}
		if fx != nil {
			fx.draw(scr, l, dx, dy)
			fx.drawPopups(scr, l, dx, dy)
			if fx.banner != "" {
				putCentered(scr, l.TermW, l.PlayY+l.PlayH/2, fx.banner, styChain)
			}
		}
		if lvl, ok := st.LevelUpBanner(); ok {
			putCentered(scr, l.TermW, l.PlayY, fmt.Sprintf("LEVEL %d", lvl), styHead)
		}
		if st.RewardBanner() != "" {
			putCentered(scr, l.TermW, l.PlayY+l.PlayH/2-1, st.RewardBanner(), styLoaded)
		}
		if letter, ok := st.SwarmBanner(); ok {
			putCentered(scr, l.TermW, l.PlayY+1,
				fmt.Sprintf("SWARM  %c", letter), styChain)
		}
	}
	if dbg != nil {
		dbg.draw(scr, l, st, fx)
	}
}

// contestedLetters finds, for each letter that more than one word is waiting on,
// which word would actually receive the keypress.
//
// This is the cue that replaces the lock. Without a lock the player is tracking
// several partial words and an invisible priority order, and has no way to know
// who gets the next key. Free fire is unfair without this.
func contestedLetters(words []engine.Word) map[int]bool {
	counts := make(map[byte]int, len(words))
	for i := range words {
		w := &words[i]
		// Bombs are excluded: they are last-resort targets, so they never win a
		// contest, and marking one would point the player at a word to avoid.
		if !w.Done() && !w.Bomb {
			counts[w.Text[w.Typed]]++
		}
	}
	marked := make(map[int]bool)
	for key, n := range counts {
		if n < 2 {
			continue
		}
		if i := engine.Resolve(words, key); i >= 0 && !words[i].Bomb {
			marked[i] = true
		}
	}
	return marked
}

// drawTrail marks a word that is moving faster than the baseline, one dim row
// per increment of excess speed. It is deliberately silent for a word at
// baseline speed: a cue drawn on every word carries no information and is just
// noise. What it shows is a word that has been chipped (chip acceleration) or
// that was kicked by the word above it, so overtakes and priority flips become
// visible before they happen rather than after.
func drawTrail(scr tcell.Screen, l Layout, st *engine.State, w *engine.Word, dx, dy int) {
	baseline := st.FallSpeedNow()
	if baseline <= 0 {
		return
	}
	excess := st.RowsPerSecond(w)/baseline - 1
	rows := int(excess * 4)
	if rows > 4 {
		rows = 4
	}
	for t := 1; t <= rows; t++ {
		y := l.PlayY + w.TopRow() - t + dy
		if y < l.PlayY || y > l.PlayY+l.PlayH-1 {
			continue
		}
		for i := w.Typed; i < len(w.Text); i++ {
			x := l.PlayX + w.CellX() + engine.FloorDivX(engine.LetterOffsetDots(w.Text, i)) + dx
			if x >= l.PlayX && x < l.PlayX+l.PlayW {
				scr.SetContent(x, y, '·', nil, styTrail)
			}
		}
	}
}

func drawWord(scr tcell.Screen, l Layout, st *engine.State, w *engine.Word, idx, playH int, contested map[int]bool, dx, dy int) {
	base := depthStyle(w, playH)
	switch {
	case w.Bomb:
		base = styBomb
	case w.Power != engine.PowerNone:
		base = styPower
	}

	// A word one key from done is chain fuel, and gets the loudest state in the
	// game. This is the change most likely to make chains land: a field with three
	// loaded words should LOOK like an opportunity, so a player fires three keys
	// and learns the whole strategy without being told it.
	loaded := w.Loaded()
	if loaded {
		base = styLoaded
	}

	// A word about to land pulses. The depth colour already says "worth more";
	// this says "about to cost you", which is a different message and the more
	// urgent one, so it overrides everything except a bomb.
	if !w.Bomb && st.InDanger(w) && (st.Frame/4)%2 == 0 {
		base = styDanger
	}

	// A word that just inherited another's speed flashes, along with the word that
	// kicked it. Momentum transfer is the one rule whose effect is felt seconds
	// after the collision that caused it, by which time the cause has left the
	// player's attention entirely.
	if w.KickFlash > 0 {
		base = styKick
	}

	// Width of the next untyped letter, for the highlight and the marker. The
	// font is proportional, so this is per-letter rather than a constant.
	nextCells := 0
	if !w.Done() {
		nextCells = engine.CellsForGlyph(rune(w.Text[w.Typed])) * w.Tier.Scale()
	}

	// Everything a word draws goes through one dot bitmap and one Pack call.
	//
	// That is not tidiness. Anything positioned by cell row instead ticks only
	// when the word crosses a whole cell, while the glyphs move every dot via the
	// sub-cell offset, so the two drift apart visibly. The bomb brackets were
	// drawn that way at first and slid against their own letters as the word fell.
	dots := engine.BitmapFor(w.Remaining(), w.Tier.Scale())
	dotY, dotX := w.DotY, w.DotX+engine.LetterOffsetDots(w.Text, w.Typed)*w.Tier.Scale()
	if w.Bomb {
		dots, dotY, dotX = bombFrame(w, dots)
	}

	topRow := engine.FloorDivY(dotY)
	sub := dotY - topRow*engine.M.DotsY
	grid := Pack(dots, sub)
	xOff := engine.FloorDivX(dotX)

	for gy, row := range grid {
		y := l.PlayY + topRow + gy + dy
		if y < l.PlayY || y > l.PlayY+l.PlayH-1 {
			continue
		}
		for gx, ch := range row {
			if ch == Blank {
				continue
			}
			x := l.PlayX + xOff + gx + dx
			if x < l.PlayX || x >= l.PlayX+l.PlayW {
				continue
			}
			// Next-letter highlight: the first untyped glyph of every word is
			// bright white while the rest takes its depth colour, so the whole
			// field's live front edge reads at a glance.
			//
			// Suppressed on bombs. The cue means "press this", and it must never
			// appear on a word the player should not press.
			st := base
			if !w.Bomb && !loaded && gx < nextCells {
				st = styNext
			}
			scr.SetContent(x, y, ch, nil, st)
		}
	}

	// A power-up word says what it does, above itself, with the mark to press.
	//
	// The mark alone was carrying the whole explanation, and a mark is a poor
	// teacher: nothing on screen ever connected '/' to BLAST, so the only way to
	// learn one was to type it and infer from what happened. Naming it on the word
	// means the set does not have to be memorised between runs, and it is why four
	// power-ups is enough to be worth having rather than a table to revise.
	//
	// Above rather than below, because below is where the loaded bar and the
	// contested marker go. Skipped when it would fall outside the playfield, which
	// only happens for the second or so a word spends entering from the top.
	if w.Power != engine.PowerNone {
		if y := l.PlayY + topRow - 1 + dy; y >= l.PlayY && y <= l.PlayY+l.PlayH-1 {
			label := fmt.Sprintf("%s %c", w.Power, rune(w.Power))
			x := l.PlayX + xOff + dx
			if n := l.PlayX + l.PlayW - len(label); x > n {
				x = n // shunt left rather than run off the right edge
			}
			if x < l.PlayX {
				x = l.PlayX
			}
			put(scr, x, y, label, styPower)
		}
	}

	// A loaded word carries a bar under it, so it reads as ready even in a
	// 16-colour terminal or to a colour-blind player.
	if loaded {
		y := l.PlayY + topRow + len(grid) + dy
		if y >= l.PlayY && y <= l.PlayY+l.PlayH-1 {
			for gx := 0; gx < nextCells; gx++ {
				x := l.PlayX + xOff + gx + dx
				if x >= l.PlayX && x < l.PlayX+l.PlayW {
					scr.SetContent(x, y, '━', nil, styLoaded)
				}
			}
		}
	}

	// Contested marker under the letter that will take the keypress.
	if contested[idx] {
		y := l.PlayY + topRow + len(grid) + dy
		if y >= l.PlayY && y <= l.PlayY+l.PlayH-1 {
			for gx := 0; gx < nextCells; gx++ {
				x := l.PlayX + xOff + gx + dx
				if x >= l.PlayX && x < l.PlayX+l.PlayW {
					scr.SetContent(x, y, '▔', nil, styMark)
				}
			}
		}
	}
}

// bombFrame wraps a bomb's glyphs in a hazard bracket, in dot space so the
// bracket is packed into cells by the same call that packs the letters and cannot
// drift against them.
//
// The bracket spans the word's whole original footprint, not just what is left,
// and the top edge doubles as a fuse: solid over the letters already eaten,
// dashed over the ones remaining. Accumulated danger is then readable without
// counting letters, and it is a shape rather than only a colour, so it survives a
// colour-blind player and a 16-colour terminal.
func bombFrame(w *engine.Word, glyphs [][]bool) ([][]bool, int, int) {
	scale := w.Tier.Scale()
	fullW := engine.TextWidthDots(w.Text) * scale
	eaten := engine.LetterOffsetDots(w.Text, w.Typed) * scale
	const pad = 1

	h := len(glyphs) + 2*pad
	out := make([][]bool, h)
	for i := range out {
		out[i] = make([]bool, fullW)
	}
	for y, row := range glyphs {
		for x, on := range row {
			if px := eaten + x; on && px < fullW {
				out[y+pad][px] = true
			}
		}
	}
	for x := 0; x < fullW; x++ {
		out[0][x] = x < eaten || x%2 == 0 // fuse: solid where burnt, dashed ahead
		out[h-1][x] = x%2 == 0
	}
	return out, w.DotY - pad, w.DotX
}

func drawChrome(scr tcell.Screen, l Layout, st *engine.State, dx, dy int) {
	right := l.X + l.W - 1
	bottom := l.Y + l.H - 1
	hline(scr, l.X, right, l.Y, '─', styBorder)
	hline(scr, l.X, right, l.Y+2, '─', styBorder)
	hline(scr, l.X, right, bottom, '─', styBorder)
	for y := l.Y + 1; y < bottom; y++ {
		scr.SetContent(l.X, y, '│', nil, styBorder)
		scr.SetContent(right, y, '│', nil, styBorder)
	}
	scr.SetContent(l.X, l.Y, '┌', nil, styBorder)
	scr.SetContent(right, l.Y, '┐', nil, styBorder)
	scr.SetContent(l.X, bottom, '└', nil, styBorder)
	scr.SetContent(right, bottom, '┘', nil, styBorder)

	// The floor is a wall that visibly loses chunks, so lives read from the
	// playfield without looking at the HUD.
	lost := int(st.Tune.Lives) - st.Lives
	span := l.PlayW
	for i := 0; i < span; i++ {
		st2 := styFloor
		ch := '═'
		if lost > 0 {
			// Break out evenly spaced sections, one more per life lost.
			seg := span / (int(st.Tune.Lives) + 1)
			if seg > 0 && (i/seg)%2 == 0 && i/seg < lost*2 {
				st2, ch = styBroken, '╌'
			}
		}
		scr.SetContent(l.PlayX+i, l.FloorY, ch, nil, st2)
	}

	// Lives as blocks rather than ♥: U+2665 is East Asian Ambiguous exactly like
	// the glyph characters, so it would double-width under a CJK setting and
	// shove the HUD sideways. Blocks also match the pixel aesthetic.
	total := int(st.Tune.Lives)
	x := l.X + 1
	x += put(scr, x, l.HUDY, fmt.Sprintf(" %-7d ", st.Score), styHUD)
	// Deliberately not █ or ░: those are glyph characters, and a test asserts no
	// glyph ink is ever drawn outside the playfield. Reusing them here would make
	// the HUD indistinguishable from an escaped word.
	x += put(scr, x, l.HUDY, strings.Repeat(string(LifeFull), st.Lives), styLife)
	x += put(scr, x, l.HUDY, strings.Repeat(string(LifeGone), max(0, total-st.Lives)), styGone)

	// Just the chain number while a chain is running. A previous version showed
	// the multiplier, the seconds left and the next tier all at once, which was
	// more information than anyone can read while typing.
	if st.ChainOpen() {
		x += put(scr, x, l.HUDY, fmt.Sprintf("   CHAIN %-3d", st.Chain), styChain)
	} else {
		x += put(scr, x, l.HUDY, fmt.Sprintf("   STREAK %-3d", st.Streak), styHUD)
	}

	// How many words are one key from done. The bluntest possible teaching aid:
	// a number that goes up as the player chips, next to a chain readout that
	// pays out when they fire.
	if n := st.LoadedCount(); n > 0 {
		x += put(scr, x, l.HUDY, fmt.Sprintf("  LOADED %d", n), styLoaded)
	}

	// Typo heat. A speed penalty the player cannot see is indistinguishable from
	// the game feeling janky, so it is always on screen while it lasts. The speed
	// trails on the words show the same thing from the other direction.
	if h := st.TypoHeat(); h > 0.02 {
		put(scr, x, l.HUDY, fmt.Sprintf(" TYPO +%2.0f%%", h*100), styHeat)
	}

	mm, ss := int(st.Elapsed)/60, int(st.Elapsed)%60
	// LETTERS is a plain count of correct keys, which is what the score used to
	// be before depth multipliers and completion bonuses were added to it. Showing
	// both means the score can be a game score without the letter count being lost.
	rightHUD := fmt.Sprintf("LETTERS %-4d ACC %3.0f%%  LVL %-2d %s %02d:%02d ",
		st.Hits, st.Accuracy(), st.Level()+1, levelBar(st.LevelProgress()), mm, ss)
	put(scr, right-len(rightHUD), l.HUDY, rightHUD, styHUD)

	drawStatus(scr, l, st)
}

// drawStatus fills the one row under the floor. It carries a running effect if
// there is one, otherwise the definition of the word just destroyed, otherwise
// nothing.
//
// It used to also carry the controls and a first-run tip. Both are gone: the row
// sits directly under the playfield in a game where the player is reading falling
// words, so anything permanently parked there is noise competing for the same
// attention. The definition is the one thing worth putting there, because it
// appears in response to something the player just did.
func drawStatus(scr tcell.Screen, l Layout, st *engine.State) {
	x := l.PlayX + 1
	drew := false

	// An instant effect names itself for a moment after it fires. A timed effect
	// announces itself with its countdown; REWIND, BLAST and REPAIR had no readout
	// anywhere, so they changed the field without ever saying what they were.
	if p, ok := st.FiredPower(); ok {
		x += put(scr, x, l.StatusY, p.String()+"   ", styPower)
		drew = true
	}
	for _, e := range st.ActiveEffects() {
		label := e.Power.String()
		if e.Seconds > 0 {
			label = fmt.Sprintf("%s %.1fs", label, e.Seconds)
		}
		x += put(scr, x, l.StatusY, label+"   ", styPower)
		drew = true
	}
	if drew {
		return
	}

	// The definition of the last word destroyed. This is the educational half of
	// the game: the word is defined at the moment it is cleared, which is a reward
	// for finishing rather than a hint about what to type.
	if st.LastCleared != "" && st.Elapsed-st.LastClearedAt < defineSeconds {
		if d, ok := words.Lookup(st.LastCleared); ok {
			text := fmt.Sprintf(" %s (%s) %s", st.LastCleared, d.POS, d.Text)
			if n := l.PlayW - 1; len(text) > n {
				text = text[:n-1] + "…"
			}
			x := put(scr, l.PlayX, l.StatusY, " "+st.LastCleared, styDefWord)
			put(scr, l.PlayX+x, l.StatusY,
				trimTo(fmt.Sprintf(" (%s) %s", d.POS, d.Text), l.PlayW-x-1), styDefText)
			return
		}
	}

}

// defineSeconds is how long a definition lingers. Long enough to read, short
// enough that it is showing the word you just cleared rather than an old one.
const defineSeconds = 6

func trimTo(s string, n int) string {
	if n <= 1 {
		return ""
	}
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}

// drawGameOver is the end of a run. It is a summary, not a dashboard: the word
// that killed you, the handful of numbers worth remembering, and the two keys
// that get you out. Everything else the player might want is on the stats
// screen, which exists for exactly that.
//
// The HUD and the floor are deliberately not drawn behind it. They carry the
// score, the letter count, the accuracy and the clock, all of which the summary
// repeats a few rows lower, so leaving them up meant every figure appeared twice
// on the noisiest screen in the game.
func drawGameOver(scr tcell.Screen, l Layout, st *engine.State, highScore bool) {
	rows := gameOverRows(st, highScore, l.PlayW)

	// The block is centred as a whole. Pinning it to a fixed row left twenty
	// blank rows under it on a tall terminal and clipped it on a short one.
	h := 0
	for _, r := range rows {
		h += r.height()
	}
	y := l.PlayY + (l.PlayH-h)/2
	if y < l.PlayY {
		y = l.PlayY
	}
	for _, r := range rows {
		r.draw(scr, l, y)
		y += r.height()
	}
}

// gameOverRow is one band of the summary: either a rasterized word, a centred
// line, or a blank spacer.
type gameOverRow struct {
	grid  [][]rune
	text  string
	pairs [][2]string
	style tcell.Style
	blank int
}

func (r gameOverRow) height() int {
	switch {
	case r.grid != nil:
		return len(r.grid)
	case r.pairs != nil:
		return len(r.pairs)
	case r.blank > 0:
		return r.blank
	}
	return 1
}

func (r gameOverRow) draw(scr tcell.Screen, l Layout, y int) {
	switch {
	case r.grid != nil:
		drawGrid(scr, l, r.grid, centreX(l, gridWidth(r.grid)), y, r.style)
	case r.pairs != nil:
		drawPairs(scr, l, y, r.pairs, r.style)
	case r.blank > 0:
	default:
		putMid(scr, l, y, r.text, r.style)
	}
}

// drawPairs lays out label/value rows as one centred block: labels flush left,
// values flush right, so the block is a rectangle and the digits line up by
// place value.
//
// The figures used to be three sentences of "LABEL 12  LABEL 553  LABEL 72",
// each centred on its own, so no two labels and no two numbers shared a column
// and the eye had to re-find the start of every value.
func drawPairs(scr tcell.Screen, l Layout, y int, pairs [][2]string, st tcell.Style) {
	const gutter = 6
	labelW, valueW := 0, 0
	for _, p := range pairs {
		labelW = max(labelW, len([]rune(p[0])))
		valueW = max(valueW, len([]rune(p[1])))
	}
	w := labelW + gutter + valueW
	x := centreX(l, w)
	for i, p := range pairs {
		put(scr, x, y+i, p[0], styStatus)
		put(scr, x+w-len([]rune(p[1])), y+i, p[1], st)
	}
}

// gameOverRows builds the summary top to bottom.
func gameOverRows(st *engine.State, highScore bool, playW int) []gameOverRow {
	var rows []gameOverRow

	if st.Killer != "" {
		if grid := killerGrid(st.Killer, playW); grid != nil {
			rows = append(rows, gameOverRow{grid: grid, style: styLife})
		} else {
			// Longer than the field even at the small tier. Plain text still
			// names the word, which is the whole point of showing it.
			rows = append(rows, gameOverRow{text: st.Killer, style: styLife})
		}
		rows = append(rows, gameOverRow{blank: 1})
	}
	// The score and nothing else. This line used to list every best the run beat
	// - time, chain, streak - which buried the one that matters in a row of ones
	// that do not. The others are still tracked and still on the stats screen.
	if highScore {
		rows = append(rows,
			gameOverRow{text: "NEW HIGH SCORE", style: styLoaded},
			gameOverRow{blank: 1})
	}

	// Six figures, down from eight. LETTERS and TYPOS were dropped: WORDS says
	// the same thing as LETTERS at the scale anyone remembers a run by, and TYPOS
	// is ACCURACY counted the other way round on the line above it.
	rows = append(rows, gameOverRow{style: styHUD, pairs: [][2]string{
		{"SCORE", fmt.Sprintf("%d", st.Score)},
		{"WORDS", fmt.Sprintf("%d", st.WordsDestroyed)},
		{"ACCURACY", fmt.Sprintf("%.0f%%", st.Accuracy())},
		{"BEST CHAIN", fmt.Sprintf("%d", st.BestChain)},
		{"BEST STREAK", fmt.Sprintf("%d", st.BestStreak)},
		{"SURVIVED", fmt.Sprintf("%02d:%02d", int(st.Elapsed)/60, int(st.Elapsed)%60)},
	}})

	rows = append(rows,
		gameOverRow{blank: 1},
		gameOverRow{text: "SPACE to go again      ESC for the menu", style: styWarn})
	return rows
}

// killerGrid rasterizes the word that ended the run at the largest tier that
// fits the playfield, or returns nil if even the normal tier is too wide.
//
// It used to be huge unconditionally, measured with engine.TextWidth, which
// reports the normal tier. A thirteen-letter word came out at twice the width
// the placement assumed: pushed left of centre and cut off against the right
// border.
func killerGrid(killer string, playW int) [][]rune {
	for _, tier := range []engine.Tier{engine.TierHuge, engine.TierNormal} {
		grid := Rasterize(killer, tier, 0)
		if playW <= 0 || gridWidth(grid) <= playW {
			return grid
		}
	}
	return nil
}

// Debug is the day-one tuning overlay. Without per-word speed and offset on
// screen there is no telling a font bug from a physics bug.
type Debug struct {
	FrameMS  float64
	PeakMS   float64
	Reloads  int
	KeysSeen int
	LastKey  string
	TuneErr  string
}

func (d *Debug) draw(scr tcell.Screen, l Layout, st *engine.State, fx *Effects) {
	parts := 0
	if fx != nil {
		parts = fx.Count()
	}
	lines := []string{
		fmt.Sprintf("frame %.3fms peak %.3fms parts %d", d.FrameMS, d.PeakMS, parts),
		fmt.Sprintf("term %dx%d play %dx%d font %s", l.TermW, l.TermH,
			l.PlayW, l.PlayH, engine.M.Name),
		fmt.Sprintf("lvl %.2f fall %.2f (eff %.2f, heat +%.0f%%) every %.2fs maxw %d/%d len %d-%d",
			st.LevelF()+1, st.FallSpeedNow(), st.EffectiveSpeedNow(), st.TypoHeat()*100,
			st.SpawnIntervalNow(), st.MaxWordsNow(), st.MaxWordsCap, lenLo(st), lenHi(st)),
		fmt.Sprintf("keys %d last %q reloads %d chip %.2f",
			d.KeysSeen, d.LastKey, d.Reloads, st.Tune.ChipAccel),
	}
	for i := range st.Words {
		w := &st.Words[i]
		lines = append(lines, fmt.Sprintf("%-13s +%d hy=%4d h=%d rows=%d-%d every=%d %.2fr/s x%d",
			w.Text, w.Typed, w.DotY, w.SubOffset(), w.TopRow(), w.BottomRow(),
			st.StepEvery(w), st.RowsPerSecond(w),
			engine.DepthMultiplier(w.BottomRow(), st.PlayH)))
	}
	if d.TuneErr != "" {
		lines = append(lines, "TUNING: "+d.TuneErr)
	}
	for i, s := range lines {
		if l.PlayY+i > l.PlayY+l.PlayH-1 {
			break
		}
		style := styDebug
		if d.TuneErr != "" && i == len(lines)-1 {
			style = styWarn
		}
		put(scr, l.PlayX+1, l.PlayY+i, s, style)
	}
}

// put writes s and returns how many cells it advanced, so callers can lay a row
// out left to right without recomputing offsets.
// levelBar shows progress through the current level. Progression is continuous,
// so a bare level number hides most of what is happening.
func levelBar(p float64) string {
	const w = 6
	n := int(p * w)
	if n > w {
		n = w
	}
	return "[" + strings.Repeat("=", n) + strings.Repeat(" ", w-n) + "]"
}

func lenLo(st *engine.State) int { lo, _ := st.WordLenRange(); return lo }
func lenHi(st *engine.State) int { _, hi := st.WordLenRange(); return hi }

func put(scr tcell.Screen, x, y int, s string, st tcell.Style) int {
	n := 0
	for _, ch := range s {
		scr.SetContent(x+n, y, ch, nil, st)
		n++
	}
	return n
}

func putCentered(scr tcell.Screen, w, y int, s string, st tcell.Style) {
	x := (w - len([]rune(s))) / 2
	if x < 0 {
		x = 0
	}
	put(scr, x, y, s, st)
}

// centreX is the left column that centres n cells inside the playfield.
//
// Screens centre on the playfield, not on the terminal. The two agree only when
// the box happens to be centred on an even number of spare columns; everywhere
// else centring on the terminal leaves text a cell off from the box drawn round
// it, and off from the title, which has always been placed this way.
func centreX(l Layout, n int) int {
	x := l.PlayX + (l.PlayW-n)/2
	if x < l.PlayX {
		x = l.PlayX
	}
	return x
}

// putMid centres one line in the playfield.
func putMid(scr tcell.Screen, l Layout, y int, s string, st tcell.Style) {
	put(scr, centreX(l, len([]rune(s))), y, s, st)
}

// drawGrid paints a rasterized word with its top-left at (x0, y0), skipping
// blanks and anything outside the playfield.
func drawGrid(scr tcell.Screen, l Layout, grid [][]rune, x0, y0 int, st tcell.Style) {
	for gy, row := range grid {
		y := y0 + gy
		if y < l.PlayY || y > l.PlayY+l.PlayH-1 {
			continue
		}
		for gx, ch := range row {
			x := x0 + gx
			if ch == Blank || x < l.PlayX || x >= l.PlayX+l.PlayW {
				continue
			}
			scr.SetContent(x, y, ch, nil, st)
		}
	}
}

// gridWidth is the drawn width of a rasterized word in cells. It is not
// engine.TextWidth: that measures the normal tier, so using it to place a huge
// word puts it half a word left of centre and runs it off the right edge.
func gridWidth(grid [][]rune) int {
	if len(grid) == 0 {
		return 0
	}
	return len(grid[0])
}

func hline(scr tcell.Screen, x0, x1, y int, ch rune, st tcell.Style) {
	for x := x0; x <= x1; x++ {
		scr.SetContent(x, y, ch, nil, st)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
