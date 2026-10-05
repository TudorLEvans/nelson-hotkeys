// Package stats keeps personal bests between runs.
//
// A score with nothing to beat is just a number. This is what closes the "one
// more go" loop, and it is why the file matters more than its size suggests.
package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Entry is one finished run, kept for the leaderboard.
type Entry struct {
	Score   int     `json:"score"`
	Seconds float64 `json:"seconds"`
	Words   int     `json:"words"`
	Letters int     `json:"letters"`
	Chain   int     `json:"chain"`
	Date    string  `json:"date"`
}

// KeepScores is how many runs the table holds. The menu shows the top three; the
// rest are there so the stats screen has something to say and so a good run is
// not lost the moment a better one lands.
const KeepScores = 10

// Stats is everything remembered between sessions.
type Stats struct {
	Games          int            `json:"games"`
	Scores         []Entry        `json:"scores,omitempty"`
	BestSeconds    float64        `json:"best_seconds"`
	BestChain      int            `json:"best_chain"`
	BestStreak     int            `json:"best_streak"`
	WordsDestroyed int            `json:"words_destroyed"`
	Letters        int            `json:"letters"`
	Keys           int            `json:"keys"`
	Daily          map[string]int `json:"daily,omitempty"`

	// Sound and Music are remembered between sessions, because they are reached
	// from the menu rather than from a flag. Both are on for a fresh install: the
	// game is better with them and a player who disagrees is one keystroke from
	// the switch, whereas audio nobody knows about goes unheard forever. The menu
	// carries the switch and the file carries the answer, so turning it off sticks.
	Sound bool `json:"sound"`
	Music bool `json:"music"`

	// AudioChosen is set once the player has actually worked the menu switches.
	// Without it a stored false cannot be told apart from a false that only ever
	// came from an old default, so a file written before audio defaulted on would
	// pin it off forever.
	AudioChosen bool `json:"audio_chosen"`
}

// BestScore is the top of the table.
func (s *Stats) BestScore() int {
	if len(s.Scores) == 0 {
		return 0
	}
	return s.Scores[0].Score
}

// Top returns up to n leaderboard entries.
func (s *Stats) Top(n int) []Entry {
	if n > len(s.Scores) {
		n = len(s.Scores)
	}
	return s.Scores[:n]
}

// Accuracy is lifetime accuracy across every recorded run.
func (s *Stats) Accuracy() float64 {
	if s.Keys == 0 {
		return 0
	}
	return 100 * float64(s.Letters) / float64(s.Keys)
}

// Run is one finished game, offered to Record.
type Run struct {
	Score      int
	Seconds    float64
	Chain      int
	Streak     int
	Words      int
	Letters    int
	Keys       int
	DailySeed  bool
	DailyStamp string // YYYY-MM-DD, only when DailySeed
}

// Path is where the file lives.
func Path() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".nelson", "stats.json")
}

// legacyPath is where bests were kept before the rename.
func legacyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".keyboardwarrior", "stats.json")
}

// Fresh is what a player who has never run the game gets: no history, audio on.
func Fresh() *Stats {
	return &Stats{Daily: map[string]int{}, Sound: true, Music: true}
}

// Load reads the file. A missing, unreadable or corrupt file is not an error: it
// returns empty stats and the game carries on.
//
// This is deliberate. Personal bests are a nicety and the game is the point, so
// nothing here may ever stop someone playing. A parse failure means somebody hand
// edited the file or a disk filled mid-write, and the right response to both is a
// fresh start rather than a refusal to launch.
func Load() *Stats {
	s := Fresh()
	path := Path()
	if path == "" {
		return s
	}
	data, err := os.ReadFile(path)
	if err != nil {
		// The game was called Keyboard Warrior and kept its bests in
		// ~/.keyboardwarrior. Read from there until the first save writes the new path.
		if data, err = os.ReadFile(legacyPath()); err != nil {
			return s
		}
	}
	var loaded Stats
	if err := json.Unmarshal(data, &loaded); err != nil {
		return s
	}
	if loaded.Daily == nil {
		loaded.Daily = map[string]int{}
	}
	if !loaded.AudioChosen {
		loaded.Sound, loaded.Music = s.Sound, s.Music
	}
	return &loaded
}

// Record folds a finished run in and reports whether it was a new high score, so
// the game over screen can say so.
//
// It used to report every best the run beat - time, chain, streak - and the
// screen listed them. Only the score is worth interrupting for; the rest were
// noise on the one line a player actually reads. The bests are still tracked,
// because the stats screen shows them, they are just not announced.
func (s *Stats) Record(r Run) bool {
	s.Games++
	s.WordsDestroyed += r.Words
	s.Letters += r.Letters
	s.Keys += r.Keys

	highScore := r.Score > s.BestScore()
	s.insert(Entry{
		Score: r.Score, Seconds: r.Seconds, Words: r.Words,
		Letters: r.Letters, Chain: r.Chain, Date: Today(),
	})
	if r.Seconds > s.BestSeconds {
		s.BestSeconds = r.Seconds
	}
	if r.Chain > s.BestChain {
		s.BestChain = r.Chain
	}
	if r.Streak > s.BestStreak {
		s.BestStreak = r.Streak
	}
	if r.DailySeed && r.DailyStamp != "" {
		if s.Daily == nil {
			s.Daily = map[string]int{}
		}
		if r.Score > s.Daily[r.DailyStamp] {
			s.Daily[r.DailyStamp] = r.Score
		}
		s.trimDaily()
	}
	return highScore
}

// insert puts a run into the leaderboard, keeping it sorted and capped.
func (s *Stats) insert(e Entry) {
	s.Scores = append(s.Scores, e)
	sort.SliceStable(s.Scores, func(i, j int) bool {
		if s.Scores[i].Score != s.Scores[j].Score {
			return s.Scores[i].Score > s.Scores[j].Score
		}
		// Ties broken by survival, so two runs on the same score are not ordered
		// by which happened to be saved first.
		return s.Scores[i].Seconds > s.Scores[j].Seconds
	})
	if len(s.Scores) > KeepScores {
		s.Scores = s.Scores[:KeepScores]
	}
}

// trimDaily keeps the file small. Yesterday's daily score is a curiosity; last
// year's is landfill.
func (s *Stats) trimDaily() {
	const keep = 60
	if len(s.Daily) <= keep {
		return
	}
	dates := make([]string, 0, len(s.Daily))
	for d := range s.Daily {
		dates = append(dates, d)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	for _, d := range dates[keep:] {
		delete(s.Daily, d)
	}
}

// Save writes the file atomically: to a temporary file in the same directory,
// then rename over the target. A crash or a full disk part way through then
// leaves the previous file intact rather than a half-written one, which is the
// difference between losing one run and losing every best you have.
func (s *Stats) Save() error {
	path := Path()
	if path == "" {
		return fmt.Errorf("stats: no home directory")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".stats-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// Today is the stamp used for daily bests.
func Today() string { return time.Now().Format("2006-01-02") }

// Summary is the one line the menu shows under the leaderboard.
func (s *Stats) Summary() string {
	if s.Games == 0 {
		return ""
	}
	return fmt.Sprintf("%d games played", s.Games)
}

// Detail is the stats screen, as label and value pairs.
func (s *Stats) Detail() [][2]string {
	if s.Games == 0 {
		return [][2]string{{"nothing yet", "play a game"}}
	}
	return [][2]string{
		{"GAMES PLAYED", fmt.Sprintf("%d", s.Games)},
		{"BEST SCORE", fmt.Sprintf("%d", s.BestScore())},
		{"LONGEST RUN", Clock(s.BestSeconds)},
		{"BEST CHAIN", fmt.Sprintf("%d words", s.BestChain)},
		{"BEST STREAK", fmt.Sprintf("%d keys", s.BestStreak)},
		{"", ""},
		{"WORDS DESTROYED", fmt.Sprintf("%d", s.WordsDestroyed)},
		{"LETTERS TYPED", fmt.Sprintf("%d", s.Letters)},
		{"LIFETIME ACCURACY", fmt.Sprintf("%.1f%%", s.Accuracy())},
	}
}

// Clock formats seconds as mm:ss.
func Clock(sec float64) string {
	n := int(sec)
	return fmt.Sprintf("%02d:%02d", n/60, n%60)
}
