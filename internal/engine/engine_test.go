package engine

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestEnginePurity enforces the architectural rule the whole test strategy rests
// on: the engine has no terminal dependency, so the game is testable headless.
// A convention would drift; this fails the build instead.
func TestEnginePurity(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Skipf("go list unavailable: %v", err)
	}
	for _, banned := range []string{"gdamore/tcell", "rivo/uniseg"} {
		if strings.Contains(string(out), banned) {
			t.Errorf("engine depends on %s; keep terminal concerns in render", banned)
		}
	}
}

// TestRNGStability pins the generator's output. --daily and --replay promise
// that a seed reproduces a run, and a refactor here would silently change every
// recorded game without breaking anything visibly.
func TestRNGStability(t *testing.T) {
	r := NewRNG(12345)
	got := make([]uint32, 8)
	for i := range got {
		got[i] = r.Uint32()
	}
	// Derived independently from the xorshift64* definition, not copied from a
	// run of this code, so the test verifies the algorithm rather than recording
	// whatever it happens to do.
	want := []uint32{
		2555902770, 3234773579, 328846939, 3161420795,
		513335584, 904356694, 4293856061, 2283851398,
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sequence changed at %d: got %v, want %v\n"+
				"If this is a deliberate PRNG change, every recorded replay is invalidated.",
				i, got, want)
		}
	}
}

func TestRNGZeroSeedNotStuck(t *testing.T) {
	r := NewRNG(0)
	a, b := r.Uint32(), r.Uint32()
	if a == 0 || a == b {
		t.Errorf("zero seed produced a degenerate sequence: %d, %d", a, b)
	}
}

func TestRNGRanges(t *testing.T) {
	r := NewRNG(7)
	for i := 0; i < 10000; i++ {
		if n := r.Intn(5); n < 0 || n > 4 {
			t.Fatalf("Intn(5) = %d", n)
		}
		if f := r.Float32(); f < 0 || f > 1 {
			t.Fatalf("Float32() = %v", f)
		}
		if v := r.Range(2, 3); v < 2 || v > 3 {
			t.Fatalf("Range(2,3) = %v", v)
		}
	}
}

// TestMetricsGeometry pins the size of a word under each drawing style. This is
// the whole point of the dot abstraction: a proportional 6-letter word is 29x3 cells
// in blocks and 15x2 in braille, and nothing else in the game has to
// know which.
func TestMetricsGeometry(t *testing.T) {
	defer SetMetrics(M)

	SetMetrics(MetricsBlocks)
	if got := TextWidth("SOURCE"); got != 29 {
		t.Errorf("blocks: 6-letter word is %d cols, want 29", got)
	}
	if got := HeightCells(); got != 3 {
		t.Errorf("blocks: word is %d rows, want 3", got)
	}

	SetMetrics(MetricsBraille)
	if got := TextWidth("SOURCE"); got != 15 {
		t.Errorf("braille: 6-letter word is %d cols, want 15", got)
	}
	if got := HeightCells(); got != 2 {
		t.Errorf("braille: word is %d rows, want 2", got)
	}

	// Braille must be strictly smaller on both axes, or it is not worth having.
	SetMetrics(MetricsBlocks)
	bw, bh := TextWidth("SOURCE"), HeightCells()
	SetMetrics(MetricsBraille)
	if TextWidth("SOURCE") >= bw || HeightCells() >= bh {
		t.Error("braille is not smaller than blocks")
	}
}

// TestBrailleMotionIsFiner is the other half of why braille exists: four dots per
// cell vertically means the movement step is a quarter of a row rather than a
// half, so a word at the same speed moves twice as often.
func TestBrailleMotionIsFiner(t *testing.T) {
	defer SetMetrics(M)
	tune := DefaultTuning()

	SetMetrics(MetricsBlocks)
	blockSteps := BaseStepFor(1.2, tune)
	SetMetrics(MetricsBraille)
	brailleSteps := BaseStepFor(1.2, tune)

	if brailleSteps >= blockSteps {
		t.Errorf("braille steps every %d ticks, blocks every %d; braille should "+
			"move more often at the same speed", brailleSteps, blockSteps)
	}
}

// TestWordsFitSideBySide is the density the smaller drawing style was for.
func TestWordsFitSideBySide(t *testing.T) {
	defer SetMetrics(M)
	const innerCols = 78

	SetMetrics(MetricsBraille)
	per := (innerCols + 2) / (TextWidth("SOURCE") + 2)
	if per < 4 {
		t.Errorf("only %d six-letter words fit a braille row, want at least 4", per)
	}
	bands := 18 / (HeightCells() + 1)
	if bands < 6 {
		t.Errorf("only %d bands of braille words fit 18 rows, want at least 6", bands)
	}
}

func TestSubOffset(t *testing.T) {
	defer SetMetrics(M)
	SetMetrics(MetricsBraille)
	cases := []struct {
		dotY    int
		wantRow int
		wantSub int
	}{
		{0, 0, 0}, {1, 0, 1}, {2, 0, 2}, {3, 0, 3},
		{4, 1, 0}, {5, 1, 1}, {8, 2, 0},
	}
	for _, c := range cases {
		w := Word{Text: "A", DotY: c.dotY, Tier: TierNormal}
		if got := w.TopRow(); got != c.wantRow {
			t.Errorf("dotY=%d: TopRow = %d, want %d", c.dotY, got, c.wantRow)
		}
		if got := w.SubOffset(); got != c.wantSub {
			t.Errorf("dotY=%d: SubOffset = %d, want %d", c.dotY, got, c.wantSub)
		}
	}
}

// TestBottomRowMatchesRenderedHeight keeps the collision box honest. A word on a
// half-row offset covers one more row than its tier's nominal height, and if the
// box does not know that, words will overlap on screen while the physics thinks
// they are clear.
func TestBottomRowMatchesRenderedHeight(t *testing.T) {
	defer SetMetrics(M)
	SetMetrics(MetricsBraille)
	// A 6-dot word starting on a cell boundary covers 2 rows; pushed 3 dots into
	// a cell it spans 3. If the box did not know that, words would overlap on
	// screen while the physics thought they were clear.
	aligned := Word{Text: "A", DotY: 4, Tier: TierNormal}
	if got := aligned.HeightCells(); got != 2 {
		t.Errorf("aligned word covers %d rows, want 2", got)
	}
	if got := aligned.BottomRow(); got != 2 {
		t.Errorf("aligned word bottom row %d, want 2", got)
	}
	offset := Word{Text: "A", DotY: 7, Tier: TierNormal}
	if got := offset.HeightCells(); got != 3 {
		t.Errorf("offset word covers %d rows, want 3", got)
	}
}

// TestChipAcceleration is the game's central tension: eating a word speeds it
// up, so pre-chewing a cluster raises the danger you are setting up.
func TestChipAcceleration(t *testing.T) {
	tune := DefaultTuning()
	tune.ChipAccel = 0.12
	st := NewState(78, 18, tune, NewRNG(1), []string{"STRAWMAN"})
	w := Word{Text: "STRAWMAN", Tier: TierNormal}
	base := st.RowsPerSecond(&w)
	w.Typed = 5
	eaten := st.RowsPerSecond(&w)
	if eaten <= base {
		t.Fatalf("eating letters did not speed the word up: %.2f then %.2f", base, eaten)
	}
	// The spec's balance target: a fully pre-chewed word runs at 1.5-1.7x. The
	// tick quantisation means this lands on a step boundary rather than exactly.
	if r := eaten / base; r < 1.4 || r > 1.8 {
		t.Errorf("chipped speed is %.2fx baseline, want about 1.5-1.7x", r)
	}
}

func TestSpawnStaysInsideField(t *testing.T) {
	tune := DefaultTuning()
	st := NewState(78, 18, tune, NewRNG(99), []string{"RATIO", "WHATABOUT", "ANTIDISESTAB"})
	for i := 0; i < 2000; i++ {
		st.Words = st.Words[:0]
		ok := st.Spawn()
		if !ok {
			t.Fatal("spawn failed on an empty field")
		}
		w := st.Words[0]
		if w.CellX() < 0 || w.CellX()+TextWidth(w.Text) > st.PlayW {
			t.Fatalf("%q at col %d width %d overflows a %d-col field",
				w.Text, w.CellX(), TextWidth(w.Text), st.PlayW)
		}
		if w.BottomDot() > 0 {
			t.Fatalf("%q spawned already visible at dotY=%d", w.Text, w.DotY)
		}
	}
}

func TestStepAdvancesWord(t *testing.T) {
	tune := DefaultTuning()
	tune.FallSpeed = 6
	st := NewState(78, 18, tune, NewRNG(5), []string{"RATIO"})
	st.Spawn()
	start := st.Words[0].DotY
	for i := 0; i < 30; i++ {
		st.Step(1.0 / 30)
	}
	if st.Words[0].DotY <= start {
		t.Error("word did not fall")
	}
	if st.Elapsed <= 0 || st.Frame == 0 {
		t.Error("clock did not advance")
	}
}

func TestTuningLoadAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tuning.conf")
	write := func(s string) {
		if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("# comment\nfall_speed = 2.5\n\nchip_accel = 0.2\n")
	tune, err := LoadTuning(path)
	if err != nil {
		t.Fatal(err)
	}
	if tune.FallSpeed != 2.5 || tune.ChipAccel != 0.2 {
		t.Errorf("parsed %v / %v, want 2.5 / 0.2", tune.FallSpeed, tune.ChipAccel)
	}
	// Unset keys keep their defaults.
	if tune.Lives != DefaultTuning().Lives {
		t.Error("an absent key overwrote its default")
	}

	// A reload must be picked up.
	tune.mtime = tune.mtime.Add(-time.Hour)
	write("fall_speed = 9\n")
	reloaded, err := tune.Watch()
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded || tune.FallSpeed != 9 {
		t.Errorf("reload = %v, fall_speed = %v, want true / 9", reloaded, tune.FallSpeed)
	}
}

// TestTuningRejectsUnknownKey is deliberate: a silent typo in a tunables file is
// indistinguishable from bad game feel, which is the worst kind of bug to chase.
func TestTuningRejectsUnknownKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tuning.conf")
	if err := os.WriteFile(path, []byte("fall_speeed = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadTuning(path); err == nil {
		t.Fatal("misspelled key accepted")
	}
}

// TestTuningBadFileKeepsOldValues matters because reloads happen mid-game: a
// half-saved file must not leave the game in a mixed state.
func TestTuningBadFileKeepsOldValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tuning.conf")
	os.WriteFile(path, []byte("fall_speed = 3\nchip_accel = 0.5\n"), 0o644)
	tune, err := LoadTuning(path)
	if err != nil {
		t.Fatal(err)
	}
	tune.mtime = tune.mtime.Add(-time.Hour)
	os.WriteFile(path, []byte("fall_speed = 7\nnonsense\n"), 0o644)
	if _, err := tune.Watch(); err == nil {
		t.Fatal("malformed file accepted")
	}
	if tune.FallSpeed != 3 || tune.ChipAccel != 0.5 {
		t.Errorf("partial apply: fall_speed=%v chip_accel=%v, want 3 / 0.5",
			tune.FallSpeed, tune.ChipAccel)
	}
}

func TestTuningMissingFileIsFine(t *testing.T) {
	tune, err := LoadTuning(filepath.Join(t.TempDir(), "absent.conf"))
	if err != nil {
		t.Fatalf("missing file should not be an error: %v", err)
	}
	if tune.FallSpeed != DefaultTuning().FallSpeed {
		t.Error("defaults not applied")
	}
}

// TestNegativeDotsFloor pins the entry path. Positions above the field are
// negative, and Go's / truncates toward zero, so TopRow needs a real floor or
// entering words sit a row too low with no sub-cell smoothing.
func TestNegativeDotsFloor(t *testing.T) {
	defer SetMetrics(M)
	SetMetrics(MetricsBraille)
	cases := []struct {
		dotY    int
		wantRow int
		wantSub int
	}{
		{-8, -2, 0}, {-7, -2, 1}, {-5, -2, 3},
		{-4, -1, 0}, {-1, -1, 3}, {0, 0, 0},
	}
	for _, c := range cases {
		w := Word{Text: "A", DotY: c.dotY, Tier: TierNormal}
		if got := w.TopRow(); got != c.wantRow {
			t.Errorf("dotY=%d: TopRow = %d, want %d", c.dotY, got, c.wantRow)
		}
		if got := w.SubOffset(); got != c.wantSub {
			t.Errorf("dotY=%d: SubOffset = %d, want %d", c.dotY, got, c.wantSub)
		}
	}
}

// TestFieldMovesInLockstep is the fix for words appearing to animate
// independently. Every word advances on one shared clock, so words at the same
// speed always move on the same tick however their positions started.
func TestFieldMovesInLockstep(t *testing.T) {
	tune := DefaultTuning()
	st := NewState(78, 18, tune, NewRNG(1), []string{"RATIO"})
	st.MaxWordsCap = 0
	st.Words = append(st.Words,
		Word{Text: "RATIO", DotX: 0, DotY: 0, Tier: TierNormal},
		Word{Text: "PEDANT", DotX: 60, DotY: 7, Tier: TierNormal},
	)

	prev0, prev1 := st.Words[0].DotY, st.Words[1].DotY
	moves, apart := 0, 0
	for i := 0; i < 300 && len(st.Words) == 2; i++ {
		st.Step(1.0 / 30)
		if len(st.Words) != 2 {
			break // a word reached the floor; stop before indexing past it
		}
		m0 := st.Words[0].DotY != prev0
		m1 := st.Words[1].DotY != prev1
		if m0 || m1 {
			moves++
			if m0 != m1 {
				apart++
			}
		}
		prev0, prev1 = st.Words[0].DotY, st.Words[1].DotY
	}
	if moves == 0 {
		t.Fatal("nothing moved")
	}
	if apart != 0 {
		t.Errorf("%d of %d movement ticks moved only one word; same-speed words "+
			"must advance together", apart, moves)
	}
}

// TestTuningAcceptsTrailingComments: the file is edited live, and annotating a
// value on its own line is the natural way to record why it is what it is.
func TestTuningAcceptsTrailingComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tuning.conf")
	body := "fall_speed = 2.5   # because it felt slow\n" +
		"# a whole-line comment\n" +
		"chip_accel = 0.2#no space\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	tune, err := LoadTuning(path)
	if err != nil {
		t.Fatal(err)
	}
	if tune.FallSpeed != 2.5 {
		t.Errorf("fall_speed = %v, want 2.5", tune.FallSpeed)
	}
	if tune.ChipAccel != 0.2 {
		t.Errorf("chip_accel = %v, want 0.2", tune.ChipAccel)
	}
}
