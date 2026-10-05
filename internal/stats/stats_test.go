package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withHome points Path at a temporary directory, so tests never touch the real
// stats file.
func withHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	return dir
}

func TestRecordTracksEachBestSeparately(t *testing.T) {
	s := &Stats{Daily: map[string]int{}}

	if !s.Record(Run{Score: 100, Seconds: 30, Chain: 3, Streak: 12, Words: 8, Letters: 50, Keys: 55}) {
		t.Error("the first ever run was not a high score")
	}

	// A better score but a worse everything else is still a high score, and must
	// not drag the unrelated bests down with it.
	if !s.Record(Run{Score: 200, Seconds: 10, Chain: 1, Streak: 2}) {
		t.Error("a better score was not reported as a high score")
	}
	if s.BestSeconds != 30 || s.BestChain != 3 {
		t.Error("a better score overwrote unrelated bests")
	}

	// A worse score is not, even though it changes nothing else either.
	if s.Record(Run{Score: 1}) {
		t.Error("a worse run reported a high score")
	}

	// The other bests are still tracked, they are just not announced: a run that
	// beats one of them without beating the score is not a high score.
	if s.Record(Run{Score: 2, Seconds: 99, Chain: 9, Streak: 99}) {
		t.Error("beating time, chain and streak reported a high score")
	}
	if s.BestSeconds != 99 || s.BestChain != 9 || s.BestStreak != 99 {
		t.Error("the unannounced bests stopped being tracked")
	}

	if s.Games != 4 {
		t.Errorf("games %d, want 4", s.Games)
	}
	if s.Letters != 50 {
		t.Errorf("letters %d, want 50", s.Letters)
	}
}

func TestRoundTrip(t *testing.T) {
	withHome(t)
	s := &Stats{Daily: map[string]int{}}
	s.Record(Run{Score: 1284, Seconds: 271, Chain: 6, Streak: 47, Words: 31, Letters: 240, Keys: 260})
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	got := Load()
	if got.BestScore() != 1284 || got.BestChain != 6 || got.Games != 1 {
		t.Errorf("loaded %+v", got)
	}
	if got.Summary() == "" {
		t.Error("no summary after a recorded game")
	}
}

// TestCorruptFileIsNotFatal. Personal bests are a nicety and the game is the
// point: nothing here may ever stop somebody playing.
func TestCorruptFileIsNotFatal(t *testing.T) {
	home := withHome(t)
	path := filepath.Join(home, ".nelson", "stats.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"", "{", "not json at all", `{"best_score": "words"}`} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		got := Load()
		if got == nil {
			t.Fatalf("Load returned nil for %q", body)
		}
		if got.Daily == nil {
			t.Errorf("Daily map is nil for %q; callers would panic writing to it", body)
		}
		if got.BestScore() != 0 {
			t.Errorf("corrupt file %q produced a score of %d", body, got.BestScore())
		}
	}
}

func TestMissingFileIsNotFatal(t *testing.T) {
	withHome(t)
	got := Load()
	if got.Games != 0 || got.Summary() != "" {
		t.Errorf("a fresh install reported %+v", got)
	}
}

// TestSaveIsAtomic: a crash part way through a write must leave the previous file
// intact rather than a half-written one, which is the difference between losing
// one run and losing every best.
func TestSaveIsAtomic(t *testing.T) {
	home := withHome(t)
	dir := filepath.Join(home, ".nelson")

	first := &Stats{Daily: map[string]int{}}
	first.Record(Run{Score: 500})
	if err := first.Save(); err != nil {
		t.Fatal(err)
	}
	second := Load()
	second.Record(Run{Score: 900})
	if err := second.Save(); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".stats-") {
			t.Errorf("temporary file %q left behind", e.Name())
		}
	}
	if Load().BestScore() != 900 {
		t.Error("the second save did not land")
	}
}

func TestDailyBestsAreKeptAndTrimmed(t *testing.T) {
	s := &Stats{Daily: map[string]int{}}
	s.Record(Run{Score: 100, DailySeed: true, DailyStamp: "2026-01-01"})
	s.Record(Run{Score: 50, DailySeed: true, DailyStamp: "2026-01-01"})
	if got := s.Daily["2026-01-01"]; got != 100 {
		t.Errorf("daily best %d, want 100 (the worse run overwrote it)", got)
	}

	// Yesterday's daily score is a curiosity; last year's is landfill.
	for d := 1; d <= 90; d++ {
		s.Record(Run{Score: d, DailySeed: true, DailyStamp: fmt.Sprintf("2026-03-%02d", d%28+1)})
	}
	if len(s.Daily) > 60 {
		t.Errorf("%d daily entries kept, want at most 60", len(s.Daily))
	}
}

func TestNonDailyRunsDoNotTouchDaily(t *testing.T) {
	s := &Stats{Daily: map[string]int{}}
	s.Record(Run{Score: 999})
	if len(s.Daily) != 0 {
		t.Errorf("an ordinary run wrote a daily entry: %v", s.Daily)
	}
}

// TestLeaderboardKeepsTopRuns. A single best hides the shape of how someone is
// playing; three runs show whether the last one was a fluke.
func TestLeaderboardKeepsTopRuns(t *testing.T) {
	s := &Stats{Daily: map[string]int{}}
	for _, sc := range []int{300, 900, 100, 700, 500} {
		s.Record(Run{Score: sc, Seconds: float64(sc) / 10})
	}
	top := s.Top(3)
	if len(top) != 3 {
		t.Fatalf("%d entries, want 3", len(top))
	}
	want := []int{900, 700, 500}
	for i, e := range top {
		if e.Score != want[i] {
			t.Errorf("place %d is %d, want %d", i+1, e.Score, want[i])
		}
	}
	if s.BestScore() != 900 {
		t.Errorf("best %d, want 900", s.BestScore())
	}
}

func TestLeaderboardIsCapped(t *testing.T) {
	s := &Stats{Daily: map[string]int{}}
	for i := 0; i < KeepScores*3; i++ {
		s.Record(Run{Score: i})
	}
	if len(s.Scores) != KeepScores {
		t.Errorf("%d entries kept, want %d", len(s.Scores), KeepScores)
	}
	if s.Scores[0].Score != KeepScores*3-1 {
		t.Errorf("top entry is %d, want the highest score", s.Scores[0].Score)
	}
}

// TestTiesBreakOnSurvival, so two runs on the same score are not ordered by which
// happened to be saved first.
func TestTiesBreakOnSurvival(t *testing.T) {
	s := &Stats{Daily: map[string]int{}}
	s.Record(Run{Score: 500, Seconds: 30})
	s.Record(Run{Score: 500, Seconds: 90})
	if s.Scores[0].Seconds != 90 {
		t.Errorf("tie ordered by %.0fs first, want the longer run", s.Scores[0].Seconds)
	}
}

func TestLifetimeAccuracy(t *testing.T) {
	s := &Stats{Daily: map[string]int{}}
	if s.Accuracy() != 0 {
		t.Error("accuracy before any keys should be zero, not a divide by zero")
	}
	s.Record(Run{Letters: 90, Keys: 100})
	s.Record(Run{Letters: 10, Keys: 100})
	if got := s.Accuracy(); got < 49.9 || got > 50.1 {
		t.Errorf("lifetime accuracy %.1f%%, want 50", got)
	}
}

// TestWPMIsGone: it measured nothing anybody cared about in a game where the
// score already rewards speed through the depth multiplier.
func TestWPMIsGone(t *testing.T) {
	s := &Stats{Daily: map[string]int{}}
	s.Record(Run{Score: 100, Letters: 50, Keys: 55, Seconds: 60})
	for _, row := range s.Detail() {
		if strings.Contains(strings.ToUpper(row[0]), "WPM") {
			t.Errorf("the stats screen still shows %q", row[0])
		}
	}
}

// TestLegacyPathIsRead: bests saved under the old name must survive the rename.
func TestLegacyPathIsRead(t *testing.T) {
	home := withHome(t)
	old := &Stats{Daily: map[string]int{}}
	old.Record(Run{Score: 700})
	data, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".keyboardwarrior", "stats.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load().BestScore(); got != 700 {
		t.Errorf("best score from the legacy file = %d, want 700", got)
	}
}
