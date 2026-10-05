package render

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"nelson/internal/engine"
)

func menuScreen(t *testing.T, w, h int) (tcell.SimulationScreen, Layout) {
	t.Helper()
	prev := engine.M
	engine.SetMetrics(engine.MetricsBlocks)
	t.Cleanup(func() { engine.SetMetrics(prev) })

	scr := tcell.NewSimulationScreen("UTF-8")
	if err := scr.Init(); err != nil {
		t.Fatal(err)
	}
	scr.SetSize(w, h)
	t.Cleanup(scr.Fini)
	return scr, ComputeLayout(w, h, engine.DefaultTuning())
}

func rowText(scr tcell.SimulationScreen, y int) string {
	cells, w, _ := scr.GetContents()
	var b strings.Builder
	for x := 0; x < w; x++ {
		r := cells[y*w+x].Runes
		if len(r) == 0 || r[0] == 0 {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r[0])
	}
	return b.String()
}

// TestMenuStaysInsideItsBorder. The whole menu is budgeted against 18 playable
// rows at the minimum size, and an overflowing stack printed its note across the
// bottom border.
func TestMenuStaysInsideItsBorder(t *testing.T) {
	scr, l := menuScreen(t, 80, 24)
	DrawMenu(scr, l, 0, MenuLabels(false, false), "3 games played")
	scr.Show()

	bottom := rowText(scr, l.Y+l.H-1)
	for _, ch := range bottom {
		if ch != '─' && ch != '└' && ch != '┘' && ch != ' ' {
			t.Fatalf("the bottom border row carries content: %q", strings.TrimSpace(bottom))
		}
	}
	if !strings.Contains(rowText(scr, l.PlayY+l.PlayH-1), "games played") {
		t.Error("the note is not on the last playable row")
	}
}

// TestMenuSurvivesAnEmptyLeaderboard, which is what a fresh install looks like.
func TestMenuSurvivesAnEmptyLeaderboard(t *testing.T) {
	scr, l := menuScreen(t, 80, 24)
	DrawMenu(scr, l, 0, MenuLabels(false, false), "")
	scr.Show()
	for _, item := range MenuLabels(false, false) {
		var found bool
		for y := l.PlayY; y <= l.PlayY+l.PlayH-1; y++ {
			if strings.Contains(rowText(scr, y), item) {
				found = true
			}
		}
		if !found {
			t.Errorf("%q is missing from a fresh menu", item)
		}
	}
}

// TestReadyIsOneVoice. The signal was half block font and half plain text, which
// read as two messages stapled together rather than one being flown.
//
// The check is on the rows from the signal down, not on the whole screen: the
// ship above it is drawn in half blocks, and the rule was always about the words
// rather than about the screen having no picture on it.
func TestReadyIsOneVoice(t *testing.T) {
	scr, l := menuScreen(t, 80, 24)
	DrawReady(scr, l)
	scr.Show()

	signal := -1
	for y := l.PlayY; y <= l.PlayY+l.PlayH-1; y++ {
		if strings.Contains(rowText(scr, y), "ENGLAND EXPECTS") {
			signal = y
			break
		}
	}
	if signal < 0 {
		t.Fatal("the signal is not plain text")
	}
	if !strings.Contains(rowText(scr, signal+1), "THAT EVERY MAN WILL DO HIS DUTY") {
		t.Error("the second line of the signal is not plain text under the first")
	}

	for y := signal; y <= l.PlayY+l.PlayH-1; y++ {
		for _, block := range []rune{Full, Upper, Lower} {
			if strings.ContainsRune(rowText(scr, y), block) {
				t.Errorf("row %d of the signal draws block glyph %q alongside its text",
					y, block)
			}
		}
	}
}

// TestReadySignalIsWhite. The menu is aqua; the signal is not. It is the one
// screen that should not look like the game's own chrome.
func TestReadySignalIsWhite(t *testing.T) {
	scr, l := menuScreen(t, 80, 24)
	DrawReady(scr, l)
	scr.Show()

	cells, w, _ := scr.GetContents()
	for y := l.PlayY; y <= l.PlayY+l.PlayH-1; y++ {
		x := columnOf(scr, y, "ENGLAND EXPECTS")
		if x < 0 {
			continue
		}
		if fg, _, _ := cells[y*w+x].Style.Decompose(); fg != tcell.ColorWhite {
			t.Errorf("the signal is drawn in %v, want white", fg)
		}
		return
	}
	t.Fatal("the signal is missing")
}

// TestReadyFliesTheShip. The screen was two lines of text in the middle of an
// empty box. She is the picture on it, and she is above the words because the
// signal came off her masts.
func TestReadyFliesTheShip(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}} {
		scr, l := menuScreen(t, size[0], size[1])
		DrawReady(scr, l)
		scr.Show()

		sea := map[tcell.Color]bool{}
		for _, c := range shipRamp {
			sea[c] = true
		}
		cells, w, _ := scr.GetContents()

		top, bottom, lead, trail := -1, -1, l.PlayW, l.PlayW
		for y := l.PlayY; y <= l.PlayY+l.PlayH-1; y++ {
			first, last := -1, -1
			for x := l.PlayX; x < l.PlayX+l.PlayW; x++ {
				fg, bg, _ := cells[y*w+x].Style.Decompose()
				if !sea[fg] && !sea[bg] {
					continue
				}
				if first < 0 {
					first = x
				}
				last = x
			}
			if first < 0 {
				continue
			}
			if top < 0 {
				top = y
			}
			bottom = y
			lead = min(lead, first-l.PlayX)
			trail = min(trail, l.PlayX+l.PlayW-1-last)
		}

		if top < 0 {
			t.Fatalf("%dx%d: the ship was not drawn", size[0], size[1])
		}
		if got := bottom - top + 1; got != len(shipArt)/2 {
			t.Errorf("%dx%d: the ship is %d rows, want %d",
				size[0], size[1], got, len(shipArt)/2)
		}
		if d := lead - trail; d < -1 || d > 1 {
			t.Errorf("%dx%d: ship margins are %d left and %d right",
				size[0], size[1], lead, trail)
		}

		signal := -1
		for y := l.PlayY; y <= l.PlayY+l.PlayH-1; y++ {
			if strings.Contains(rowText(scr, y), "ENGLAND EXPECTS") {
				signal = y
				break
			}
		}
		if signal <= bottom {
			t.Errorf("%dx%d: the signal is on row %d, not under the ship at %d",
				size[0], size[1], signal, bottom)
		}
	}
}

func TestStatsScreenFits(t *testing.T) {
	scr, l := menuScreen(t, 80, 24)
	rows := [][2]string{
		{"GAMES PLAYED", "3"}, {"BEST SCORE", "1284"}, {"LONGEST RUN", "04:31"},
		{"BEST CHAIN", "6 words"}, {"BEST STREAK", "47 keys"}, {"", ""},
		{"WORDS DESTROYED", "72"}, {"LETTERS TYPED", "550"}, {"LIFETIME ACCURACY", "90.6%"},
	}
	DrawStats(scr, l, rows, nil)
	scr.Show()
	bottom := rowText(scr, l.Y+l.H-1)
	for _, ch := range bottom {
		if ch != '─' && ch != '└' && ch != '┘' && ch != ' ' {
			t.Fatalf("the stats screen overflowed onto the border: %q", strings.TrimSpace(bottom))
		}
	}
}

// leftEdge is the column of the first non-space cell on a row, or -1. Columns,
// not byte offsets: the block characters the game draws with are three bytes
// each, so indexing a row as a string puts every measurement to the right of
// one out by two per glyph.
func leftEdge(scr tcell.SimulationScreen, y int) int {
	for x, ch := range []rune(rowText(scr, y)) {
		if ch != ' ' && ch != '│' {
			return x
		}
	}
	return -1
}

// columnOf is where s starts on row y, in cells, or -1.
func columnOf(scr tcell.SimulationScreen, y int, s string) int {
	row := []rune(rowText(scr, y))
	want := []rune(s)
	for x := 0; x+len(want) <= len(row); x++ {
		if string(row[x:x+len(want)]) == s {
			return x
		}
	}
	return -1
}

// TestMenuItemsShareALeftEdge. Every item used to be centred on its own, so
// PLAY, HOW TO PLAY and SOUND ON started in three different columns and the list
// wandered from row to row.
func TestMenuItemsShareALeftEdge(t *testing.T) {
	scr, l := menuScreen(t, 80, 24)
	items := MenuLabels(true, false)
	DrawMenu(scr, l, 0, items, "")
	scr.Show()

	// One row per item, taken in order, so PLAY is not found inside HOW TO PLAY.
	edges := map[int]int{}
	y := l.PlayY
	for _, item := range items {
		for ; y <= l.PlayY+l.PlayH-1; y++ {
			if x := columnOf(scr, y, item); x >= 0 {
				edges[x]++
				break
			}
		}
	}
	if len(edges) == 0 {
		t.Fatal("no menu items were drawn")
	}
	if len(edges) != 1 {
		t.Errorf("menu labels start in %d different columns (%v), want one", len(edges), edges)
	}
}

// TestMenuMarkerHasItsOwnColumn: the arrow sits in a gutter left of the labels,
// so selecting an item does not shift the item sideways.
func TestMenuMarkerHasItsOwnColumn(t *testing.T) {
	scr, l := menuScreen(t, 80, 24)
	items := MenuLabels(true, false)

	col := func(selected int) int {
		DrawMenu(scr, l, selected, items, "")
		scr.Show()
		for y := l.PlayY; y <= l.PlayY+l.PlayH-1; y++ {
			if x := columnOf(scr, y, items[1]); x >= 0 {
				return x
			}
		}
		t.Fatalf("%q is missing from the menu", items[1])
		return -1
	}
	if a, b := col(0), col(1); a != b {
		t.Errorf("HOW TO PLAY sits at column %d unselected and %d selected", a, b)
	}
}

// TestMenuIsVerticallyBalanced on a terminal with room to spare. The stack used
// to be pinned to the top of the playfield, which left twenty blank rows under
// it at 120x40.
func TestMenuIsVerticallyBalanced(t *testing.T) {
	scr, l := menuScreen(t, 120, 40)
	DrawMenu(scr, l, 0, MenuLabels(true, false), "3 games played")
	scr.Show()

	// Measured over the middle third of the playfield. Everything in the stack is
	// centred and so crosses it; the margins hold decoration that runs the full
	// height and would otherwise report every row as occupied.
	//
	// A row counts as occupied if it carries ink OR stone. The pedestal's cornice
	// and base are painted as coloured cells with no character in them, so reading
	// runes alone reported the bottom of the monument as empty space and measured
	// the stack as a row shorter than it is.
	stone := map[tcell.Color]bool{}
	for _, c := range nelsonRamp {
		stone[c] = true
	}
	cells, w, _ := scr.GetContents()
	bare := func(y int) bool {
		row := []rune(rowText(scr, y))
		for x := l.PlayX + l.PlayW/3; x < l.PlayX+2*l.PlayW/3; x++ {
			if row[x] != ' ' {
				return false
			}
			if _, bg, _ := cells[y*w+x].Style.Decompose(); stone[bg] {
				return false
			}
		}
		return true
	}

	// The note owns the bottom row on its own, so balance is measured above it.
	last := l.PlayY + l.PlayH - 2
	above, below := 0, 0
	for y := l.PlayY; y <= last && bare(y); y++ {
		above++
	}
	for y := last; y >= l.PlayY && bare(y); y-- {
		below++
	}
	if d := above - below; d < -1 || d > 1 {
		t.Errorf("%d blank rows above the menu and %d below it", above, below)
	}
}

// TestHugeTierIsTwiceTheWidth pins the measurement that placement used to get
// wrong. engine.TextWidth reports the normal tier, so using it to centre a word
// rasterized at the huge one puts it half a word left of centre.
func TestHugeTierIsTwiceTheWidth(t *testing.T) {
	scr, _ := menuScreen(t, 80, 24)
	_ = scr
	for _, word := range []string{"PEDANT", "KEYBOARD"} {
		normal := gridWidth(Rasterize(word, engine.TierNormal, 0))
		huge := gridWidth(Rasterize(word, engine.TierHuge, 0))
		if normal != engine.TextWidth(word) {
			t.Errorf("%s: engine.TextWidth says %d, the drawn grid is %d",
				word, engine.TextWidth(word), normal)
		}
		if huge != 2*normal {
			t.Errorf("%s: the huge grid is %d cells, want twice the normal %d",
				word, huge, normal)
		}
	}
}

// nelsonInk reports the columns the monument painted on a row.
func nelsonInk(scr tcell.SimulationScreen, l Layout, y int) []int {
	cells, w, _ := scr.GetContents()
	stone := map[tcell.Color]bool{}
	for _, c := range nelsonRamp {
		stone[c] = true
	}
	var out []int
	for x := l.PlayX; x < l.PlayX+l.PlayW; x++ {
		fg, bg, _ := cells[y*w+x].Style.Decompose()
		// Either way round: the art is drawn as coloured block characters, the
		// pedestal as coloured cells the inscription is written over.
		if stone[fg] || stone[bg] {
			out = append(out, x)
		}
	}
	return out
}

// TestNelsonIsCentred. The monument was in the left margin, which left the menu
// beside it and the right half of the screen empty. It is the centrepiece now,
// so it stands on the middle of the playfield.
func TestNelsonIsCentred(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}} {
		scr, l := menuScreen(t, size[0], size[1])
		items := MenuLabels(true, false)
		DrawMenu(scr, l, 0, items, "")
		scr.Show()

		lead, trail, rows := l.PlayW, l.PlayW, 0
		for y := l.PlayY; y <= l.PlayY+l.PlayH-1; y++ {
			ink := nelsonInk(scr, l, y)
			if len(ink) == 0 {
				continue
			}
			rows++
			lead = min(lead, ink[0]-l.PlayX)
			trail = min(trail, l.PlayX+l.PlayW-1-ink[len(ink)-1])
		}
		if rows == 0 {
			t.Fatalf("%dx%d: the monument was not drawn at all", size[0], size[1])
		}
		if d := lead - trail; d < -1 || d > 1 {
			t.Errorf("%dx%d: monument margins are %d left and %d right",
				size[0], size[1], lead, trail)
		}
	}
}

// TestMenuIsInscribedOnThePedestal: every label sits inside the monument's
// footprint, on the sunk panel, rather than floating beside it.
func TestMenuIsInscribedOnThePedestal(t *testing.T) {
	scr, l := menuScreen(t, 80, 24)
	items := MenuLabels(true, false)
	DrawMenu(scr, l, 0, items, "")
	scr.Show()

	cells, w, _ := scr.GetContents()
	stone := map[tcell.Color]bool{}
	for _, c := range nelsonRamp {
		stone[c] = true
	}
	y := l.PlayY
	for _, item := range items {
		x := -1
		for ; y <= l.PlayY+l.PlayH-1; y++ {
			if x = columnOf(scr, y, item); x >= 0 {
				break
			}
		}
		if x < 0 {
			t.Fatalf("%q is missing from the menu", item)
		}
		for i := 0; i < len([]rune(item)); i++ {
			if _, bg, _ := cells[y*w+x+i].Style.Decompose(); !stone[bg] {
				t.Errorf("%q is not on the pedestal: column %d has no stone behind it",
					item, x+i)
				break
			}
		}
		// And the stone runs past both ends of the label.
		ink := nelsonInk(scr, l, y)
		if ink[0] >= x || ink[len(ink)-1] <= x+len([]rune(item))-1 {
			t.Errorf("%q reaches the edge of the pedestal", item)
		}
	}
}

// TestNelsonGrowsUpwardNotDownward. Letting the shaft take every row available
// put the statue in the top corner of a 40-row terminal with twenty cells of
// bare stone under it, which reads as a flagpole rather than a monument.
func TestNelsonGrowsUpwardNotDownward(t *testing.T) {
	tall, short := 0, 0
	for _, tc := range []struct {
		w, h int
		n    *int
	}{{80, 24, &short}, {120, 40, &tall}} {
		scr, l := menuScreen(t, tc.w, tc.h)
		DrawMenu(scr, l, 0, MenuLabels(true, false), "")
		scr.Show()
		for y := l.PlayY; y <= l.PlayY+l.PlayH-1; y++ {
			if len(nelsonInk(scr, l, y)) > 0 {
				*tc.n++
			}
		}
	}
	if short == 0 || tall == 0 {
		t.Fatalf("monument missing: %d rows at 80x24, %d at 120x40", short, tall)
	}
	if tall <= short {
		t.Errorf("the monument is %d rows at 80x24 and %d at 120x40; it should have "+
			"more shaft where there is more room", short, tall)
	}
	if tall >= 34 {
		t.Errorf("the monument fills all %d rows of a 120x40 playfield; it is capped "+
			"so it does not read as a flagpole", tall)
	}
}

// TestNelsonHasNoBackground. The portrait owns every cell it covers, which is
// right for a framed plate in the intro and wrong here: the monument stands on
// the menu with nothing behind it, so a bitmap that painted its own background
// would be a black rectangle rather than a column.
func TestNelsonHasNoBackground(t *testing.T) {
	scr, l := menuScreen(t, 80, 24)
	DrawMenu(scr, l, 0, MenuLabels(true, false), "")
	scr.Show()

	cells, w, _ := scr.GetContents()
	transparent := 0
	for y := l.PlayY; y <= l.PlayY+l.PlayH-1; y++ {
		for _, x := range nelsonInk(scr, l, y) {
			_, bg, _ := cells[y*w+x].Style.Decompose()
			if bg == tcell.ColorDefault {
				transparent++
			}
		}
	}
	if transparent == 0 {
		t.Error("every stone cell painted a background; the monument has no silhouette")
	}
}

// TestNelsonArtIsWellFormed guards the generated data against a hand edit.
// The pieces are stacked in pixel rows and packed two to a cell, so an odd
// count in any of them shears everything below it by half a cell.
func TestNelsonArtIsWellFormed(t *testing.T) {
	parts := map[string][]string{
		"statue":   nelsonStatue,
		"capital":  nelsonCapital,
		"pedestal": nelsonPedestal,
		"shaft":    {nelsonShaft},
	}
	for name, rows := range parts {
		if name != "shaft" && len(rows)%2 != 0 {
			t.Errorf("%s is %d pixel rows; cells are two rows tall", name, len(rows))
		}
		for i, r := range rows {
			if len([]rune(r)) != nelsonW {
				t.Errorf("%s row %d is %d wide, want %d", name, i, len([]rune(r)), nelsonW)
			}
			for _, ch := range r {
				if ch == '.' {
					continue
				}
				if ch < '0' || ch >= '0'+rune(len(nelsonRamp)) {
					t.Errorf("%s row %d: %q is not '.' or a palette index", name, i, ch)
				}
			}
		}
	}
}

// TestTitleIsTheGameName. The name is set in the block font and the rest of it
// reads as a tagline under, because the whole string in blocks wants about 120
// cells against the 78 an 80-column terminal has.
func TestTitleIsTheGameName(t *testing.T) {
	scr, l := menuScreen(t, 80, 24)
	DrawMenu(scr, l, 0, MenuLabels(true, false), "")
	scr.Show()

	var joined strings.Builder
	for y := l.PlayY; y <= l.PlayY+l.PlayH-1; y++ {
		joined.WriteString(rowText(scr, y))
	}
	if !strings.Contains(joined.String(), spaced(gameTagline, 1)) {
		t.Errorf("%q is missing from the menu", gameTagline)
	}

	// The name itself is glyph ink, so it is checked by extent rather than text.
	// At 80x24 there is no room to set it at the huge tier and still stand the
	// column up, so it is the normal one.
	grid := Rasterize(gameName, engine.TierNormal, 0)
	top := l.PlayY
	for top <= l.PlayY+l.PlayH-1 && leftEdge(scr, top) < 0 {
		top++
	}
	want := centreX(l, gridWidth(grid))
	if got := leftEdge(scr, top); got != want {
		t.Errorf("the name starts at column %d, want %d", got, want)
	}
}

// TestNelsonPlanSpendsRowsOnTheStatueFirst. The pedestal is fixed by the menu
// and the capital by the art; the argument is over what the rest buys. A cell of
// shaft is worth less than the man on top, so the statue is paid for first.
func TestNelsonPlanSpendsRowsOnTheStatueFirst(t *testing.T) {
	const items = 6
	fixed := capitalCells() + pedestalCells(items)

	if _, _, ok := nelsonPlan(fixed, items); ok {
		t.Error("the monument was drawn with no shaft under the capital")
	}
	statue, shaft, ok := nelsonPlan(fixed+1, items)
	if !ok || statue || shaft != 1 {
		t.Errorf("one spare row gave statue=%v shaft=%d, want a cell of shaft", statue, shaft)
	}
	statue, shaft, ok = nelsonPlan(fixed+statueCells(), items)
	if !ok || !statue || shaft != 0 {
		t.Errorf("exactly enough for the statue gave statue=%v shaft=%d", statue, shaft)
	}
	_, shaft, _ = nelsonPlan(fixed+statueCells()+100, items)
	if shaft != maxShaftCells {
		t.Errorf("shaft is %d cells on an enormous terminal, want it capped at %d",
			shaft, maxShaftCells)
	}
}

// TestMonumentWidthDoesNotFollowTheToggles. SOUND OFF is a character longer than
// SOUND ON, and the monument is measured out from the widest label, so the
// column used to grow two cells and slide one sideways the moment a toggle was
// pressed. The stone is furniture; it does not move because a setting changed.
func TestMonumentWidthDoesNotFollowTheToggles(t *testing.T) {
	type extent struct{ left, right int }
	got := map[[2]bool]extent{}

	for _, sound := range []bool{true, false} {
		for _, music := range []bool{true, false} {
			scr, l := menuScreen(t, 80, 24)
			DrawMenu(scr, l, 0, MenuLabels(sound, music), "")
			scr.Show()

			e := extent{left: l.PlayW, right: -1}
			rows := 0
			for y := l.PlayY; y <= l.PlayY+l.PlayH-1; y++ {
				ink := nelsonInk(scr, l, y)
				if len(ink) == 0 {
					continue
				}
				rows++
				e.left = min(e.left, ink[0])
				e.right = max(e.right, ink[len(ink)-1])
			}
			if rows == 0 {
				t.Fatalf("sound=%v music=%v: the monument was not drawn", sound, music)
			}
			got[[2]bool{sound, music}] = e
		}
	}

	want := got[[2]bool{true, true}]
	for k, e := range got {
		if e != want {
			t.Errorf("sound=%v music=%v: monument spans columns %d..%d, want %d..%d",
				k[0], k[1], e.left, e.right, want.left, want.right)
		}
	}
}

// TestLeaderboardIsOnTheStatsScreenOnly. It used to be under the menu, where it
// never fitted at 80x24 and cost the monument its shaft everywhere else.
func TestLeaderboardIsOnTheStatsScreenOnly(t *testing.T) {
	top := []Leader{{Score: 1284, Seconds: 271}, {Score: 940, Seconds: 198}}

	scr, l := menuScreen(t, 120, 40)
	DrawMenu(scr, l, 0, MenuLabels(true, false), "3 games played")
	scr.Show()
	for y := l.PlayY; y <= l.PlayY+l.PlayH-1; y++ {
		if row := rowText(scr, y); strings.Contains(row, "TOP RUNS") || strings.Contains(row, "1284") {
			t.Errorf("the menu still carries the leaderboard: %q", strings.TrimSpace(row))
		}
	}

	scr2, l2 := menuScreen(t, 80, 24)
	DrawStats(scr2, l2, [][2]string{{"GAMES PLAYED", "3"}, {"BEST SCORE", "1284"}}, top)
	scr2.Show()
	var joined strings.Builder
	for y := l2.PlayY; y <= l2.PlayY+l2.PlayH-1; y++ {
		joined.WriteString(rowText(scr2, y) + "\n")
	}
	all := joined.String()
	if !strings.Contains(all, "TOP RUNS") {
		t.Error("the stats screen has no leaderboard")
	}
	for _, want := range []string{"1284", "940", "04:31", "03:18"} {
		if !strings.Contains(all, want) {
			t.Errorf("%q is missing from the stats screen", want)
		}
	}
}

// TestStatsLeaderboardColumnsLineUp: rank, score and time each in one column.
func TestStatsLeaderboardColumnsLineUp(t *testing.T) {
	scr, l := menuScreen(t, 80, 24)
	DrawStats(scr, l, [][2]string{{"GAMES PLAYED", "12"}}, []Leader{
		{Score: 1284, Seconds: 271},
		{Score: 940, Seconds: 198},
		{Score: 61, Seconds: 15},
	})
	scr.Show()

	ends := map[int]bool{}
	for _, score := range []string{"1284", "940", "61"} {
		found := false
		for y := l.PlayY; y <= l.PlayY+l.PlayH-1; y++ {
			if x := columnOf(scr, y, score+" "); x >= 0 {
				ends[x+len([]rune(score))] = true
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%q is missing from the leaderboard", score)
		}
	}
	if len(ends) != 1 {
		t.Errorf("scores end in %d columns (%v), want one", len(ends), ends)
	}
}
