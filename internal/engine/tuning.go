package engine

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Tuning holds every constant the game plays with. It lives in a file so that
// tuning needs no rebuild: Watch reloads it when the file changes on disk.
//
// Adding a field means adding one line to bindings(). A key in the file with no
// binding is an error rather than a silent no-op, which is what stops typos
// from looking like bad game feel.
type Tuning struct {
	// Screen contract
	MinCols float64
	MinRows float64
	MaxCols float64
	MaxRows float64

	// Falling
	FallSpeed    float64 // rows per second, baseline
	ChipAccel    float64 // speed gain per letter removed
	SpeedScale   float64 // global multiplier, driven by --speed
	StepInterval float64 // seconds per global movement tick; see motion.go

	// Lives and pacing
	Lives         float64
	SpawnInterval float64 // seconds between spawns at level 0
	SpawnPause    float64 // spawn freeze after a word lands
	ShockwaveRows float64 // rows above the floor cleared by an impact
	DangerRows    float64 // rows above the floor at which a word starts pulsing

	// Difficulty schedule, driven by the clock alone. See difficulty.go.
	//
	// Level 1 lasts LevelBase seconds and each level after lasts LevelGrowth
	// longer, fractionally. Early levels are short because the bottom of the range
	// is trivial for anyone who can already type.
	LevelBase         float64
	LevelGrowth       float64
	SpeedGrowth       float64 // rows per second added at level 1
	SpeedCurve        float64 // exponent on level; below 1 means steep early, flat later
	SpeedLate         float64 // linear term, so the curve never goes truly flat
	MaxFallSpeed      float64 // ceiling on spawn speed
	IntervalFactor    float64 // spawn interval multiplier each step
	MinSpawnInterval  float64 // floor on the spawn interval
	StartWords        float64 // words allowed on screen at level 0
	StepsPerExtraWord float64 // steps between concurrency increases

	// Word length grows with the schedule, so the early game is short words and
	// the late game long ones.
	WordLenStart  float64 // longest word at level 1
	WordLenGrowth float64 // letters added to the ceiling at level 1
	WordLenCurve  float64 // exponent on level; below 1 means steep early, flat later
	WordLenMax    float64 // hard ceiling, bounded by what fits the playfield
	WordLenMin    float64 // hard floor
	WordLenSpread float64 // how far below the ceiling the band reaches

	// Typing difficulty. Length is a crude proxy for how hard a word is; this
	// biases selection toward awkward ones as the run goes on. Assumes QWERTY, so
	// set typing_difficulty to 0 on another layout. See typing.go.
	TypingDifficulty      float64 // 0 disables it entirely
	TypingDifficultyBy    float64 // level at which the bias reaches full strength
	TypingDifficultyScale float64 // difficulty score treated as "hardest"

	// Scoring
	ChainWindow    float64 // seconds between completions that still chain
	ChainMinStreak float64 // streak of correct keys needed before chains count

	// Chain milestones. Past these depths a chain pays in survival rather than
	// points, because once the field is fast enough a score multiplier is a
	// consolation prize. See chain.go.
	ChainSlowAt   float64
	ChainShieldAt float64
	ChainSweepAt  float64
	ChainLifeAt   float64

	// Typo heat. See heat.go.
	HeatPerMiss  float64 // speed added per miss, as a fraction
	HeatHalfLife float64 // seconds for the penalty to halve
	HeatMax      float64 // ceiling, so a masher is punished rather than executed

	// Rhythm. See spawner.go.
	BreatheEvery   float64 // words destroyed between pauses
	BreatheSeconds float64
	SwarmFromLevel float64
	SwarmEvery     float64 // seconds between swarms
	SwarmSize      float64

	// Power-up rarity. Weights are relative; rarity tracks strength. Four of them,
	// because nine symbols is more than anyone learns by playing; see powerup.go.
	WeightBlast  float64
	WeightFreeze float64
	WeightRewind float64
	WeightRepair float64

	// Bombs. See bomb.go.
	BombFromLevel float64
	BombChance    float64
	BombLenMin    float64
	BombLenMax    float64
	BombCostsLife float64 // lives lost on completing one

	// Power-ups. See powerup.go.
	MaxLives          float64
	PowerupFromLevel  float64 // no power-ups before this level
	PowerupChance     float64 // probability a given spawn carries one
	PowerupLevelBonus float64 // levels ahead the length band is drawn from
	FreezeSeconds     float64
	SlowPerPickup     float64 // speed removed per SLOW granted by a chain
	SlowMax           float64 // ceiling on the stack
	SlowSeconds       float64 // how long one grant's slice lasts
	RewindRows        float64
	BlastRows         float64 // height of the blast box, in rows
	BlastCols         float64 // width of the blast box, in columns

	path  string
	mtime time.Time
}

// DefaultTuning is the built-in baseline. The file on disk overrides it, so the
// game still runs with no file present.
func DefaultTuning() *Tuning {
	return &Tuning{
		MinCols:        80,
		MinRows:        24,
		MaxCols:        120,
		MaxRows:        40,
		FallSpeed:      1.2,
		ChipAccel:      0.12,
		SpeedScale:     1.0,
		StepInterval:   0.05,
		Lives:          3,
		SpawnInterval:  3.5,
		SpawnPause:     1.5,
		ShockwaveRows:  3,
		DangerRows:     4,
		ChainWindow:    1.5,
		ChainMinStreak: 5,
		ChainSlowAt:    4,
		ChainShieldAt:  6,
		ChainSweepAt:   8,
		ChainLifeAt:    10,

		HeatPerMiss:  0.04,
		HeatHalfLife: 3.0,
		HeatMax:      0.35,

		WeightBlast:  15,
		WeightFreeze: 14,
		WeightRewind: 12,
		WeightRepair: 5,

		BreatheEvery:   10,
		BreatheSeconds: 2,
		SwarmFromLevel: 4,
		SwarmEvery:     90,
		SwarmSize:      5,

		BombFromLevel: 4,
		BombChance:    0.08,
		BombLenMin:    5,
		BombLenMax:    7,
		BombCostsLife: 1,

		MaxLives:          5,
		PowerupFromLevel:  3,
		PowerupChance:     0.22,
		PowerupLevelBonus: 2,
		FreezeSeconds:     3,
		SlowPerPickup:     0.20,
		SlowMax:           0.50,
		SlowSeconds:       8,
		RewindRows:        4,
		BlastRows:         9,
		BlastCols:         40,

		LevelBase:         8,
		LevelGrowth:       0.25,
		SpeedGrowth:       0.55,
		SpeedCurve:        0.5,
		SpeedLate:         0.02,
		MaxFallSpeed:      4.5,
		IntervalFactor:    0.88,
		MinSpawnInterval:  0.6,
		StartWords:        2,
		StepsPerExtraWord: 2,

		WordLenStart:  5,
		WordLenGrowth: 2.2,
		WordLenCurve:  0.5,
		WordLenMax:    12,
		WordLenMin:    3,
		WordLenSpread: 3,

		TypingDifficulty:      1.0,
		TypingDifficultyBy:    12,
		TypingDifficultyScale: 1.6,
	}
}

func (t *Tuning) bindings() map[string]*float64 {
	return map[string]*float64{
		"min_cols":      &t.MinCols,
		"min_rows":      &t.MinRows,
		"max_cols":      &t.MaxCols,
		"max_rows":      &t.MaxRows,
		"fall_speed":    &t.FallSpeed,
		"chip_accel":    &t.ChipAccel,
		"speed_scale":   &t.SpeedScale,
		"step_interval": &t.StepInterval,

		"lives":          &t.Lives,
		"spawn_interval": &t.SpawnInterval,
		"spawn_pause":    &t.SpawnPause,
		"shockwave_rows": &t.ShockwaveRows,
		"danger_rows":    &t.DangerRows,

		"chain_window":     &t.ChainWindow,
		"chain_min_streak": &t.ChainMinStreak,
		"chain_slow_at":    &t.ChainSlowAt,
		"chain_shield_at":  &t.ChainShieldAt,
		"chain_sweep_at":   &t.ChainSweepAt,
		"chain_life_at":    &t.ChainLifeAt,

		"heat_per_miss":  &t.HeatPerMiss,
		"heat_half_life": &t.HeatHalfLife,
		"heat_max":       &t.HeatMax,

		"weight_blast":  &t.WeightBlast,
		"weight_freeze": &t.WeightFreeze,
		"weight_rewind": &t.WeightRewind,
		"weight_repair": &t.WeightRepair,

		"breathe_every":    &t.BreatheEvery,
		"breathe_seconds":  &t.BreatheSeconds,
		"swarm_from_level": &t.SwarmFromLevel,
		"swarm_every":      &t.SwarmEvery,
		"swarm_size":       &t.SwarmSize,

		"bomb_from_level": &t.BombFromLevel,
		"bomb_chance":     &t.BombChance,
		"bomb_len_min":    &t.BombLenMin,
		"bomb_len_max":    &t.BombLenMax,
		"bomb_costs_life": &t.BombCostsLife,

		"max_lives":           &t.MaxLives,
		"powerup_from_level":  &t.PowerupFromLevel,
		"powerup_chance":      &t.PowerupChance,
		"powerup_level_bonus": &t.PowerupLevelBonus,
		"freeze_seconds":      &t.FreezeSeconds,
		"slow_per_pickup":     &t.SlowPerPickup,
		"slow_max":            &t.SlowMax,
		"slow_seconds":        &t.SlowSeconds,
		"rewind_rows":         &t.RewindRows,
		"blast_rows":          &t.BlastRows,
		"blast_cols":          &t.BlastCols,

		"level_base":           &t.LevelBase,
		"level_growth":         &t.LevelGrowth,
		"speed_growth":         &t.SpeedGrowth,
		"speed_curve":          &t.SpeedCurve,
		"speed_late":           &t.SpeedLate,
		"max_fall_speed":       &t.MaxFallSpeed,
		"interval_factor":      &t.IntervalFactor,
		"min_spawn_interval":   &t.MinSpawnInterval,
		"start_words":          &t.StartWords,
		"steps_per_extra_word": &t.StepsPerExtraWord,

		"word_len_start":  &t.WordLenStart,
		"word_len_growth": &t.WordLenGrowth,
		"word_len_curve":  &t.WordLenCurve,
		"word_len_max":    &t.WordLenMax,
		"word_len_min":    &t.WordLenMin,
		"word_len_spread": &t.WordLenSpread,

		"typing_difficulty":       &t.TypingDifficulty,
		"typing_difficulty_by":    &t.TypingDifficultyBy,
		"typing_difficulty_scale": &t.TypingDifficultyScale,
	}
}

// Value reads one tunable by name. Used by tests to compare a parsed file
// against the built-in defaults without touching unexported bookkeeping fields.
func (t *Tuning) Value(key string) (float64, bool) {
	if p, ok := t.bindings()[key]; ok {
		return *p, true
	}
	return 0, false
}

// Keys returns every recognised tuning key. Used by tests.
func (t *Tuning) Keys() []string {
	b := t.bindings()
	out := make([]string, 0, len(b))
	for k := range b {
		out = append(out, k)
	}
	return out
}

// LoadTuning reads path over the defaults. A missing file is not an error; the
// defaults stand and Watch will pick the file up if it appears later.
func LoadTuning(path string) (*Tuning, error) {
	t := DefaultTuning()
	t.path = path
	if err := t.reload(); err != nil && !os.IsNotExist(err) {
		return t, err
	}
	return t, nil
}

// Watch reparses the file if its mtime changed. Call it every N frames, not
// every frame. Reports whether a reload happened. A parse error leaves the
// previous values in place and is returned so the caller can show it rather
// than crash mid-game.
func (t *Tuning) Watch() (bool, error) {
	if t.path == "" {
		return false, nil
	}
	fi, err := os.Stat(t.path)
	if err != nil {
		return false, nil
	}
	if !fi.ModTime().After(t.mtime) {
		return false, nil
	}
	if err := t.reload(); err != nil {
		return false, err
	}
	return true, nil
}

func (t *Tuning) reload() error {
	f, err := os.Open(t.path)
	if err != nil {
		return err
	}
	defer f.Close()
	if fi, err := f.Stat(); err == nil {
		t.mtime = fi.ModTime()
	}

	// Parse into a scratch copy so a bad file cannot leave a half-applied set
	// of values behind.
	scratch := *t
	binds := scratch.bindings()

	sc := bufio.NewScanner(f)
	for line := 1; sc.Scan(); line++ {
		s := strings.TrimSpace(sc.Text())
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		// A trailing comment is stripped, so a value can be annotated on its own
		// line. Values are numbers, so a '#' can never be part of one.
		if i := strings.IndexByte(s, '#'); i >= 0 {
			s = strings.TrimSpace(s[:i])
			if s == "" {
				continue
			}
		}
		key, val, ok := strings.Cut(s, "=")
		if !ok {
			return fmt.Errorf("%s:%d: expected key = value, got %q", t.path, line, s)
		}
		key = strings.TrimSpace(key)
		dst, known := binds[key]
		if !known {
			return fmt.Errorf("%s:%d: unknown tuning key %q", t.path, line, key)
		}
		n, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil {
			return fmt.Errorf("%s:%d: key %q: %v", t.path, line, key, err)
		}
		*dst = n
	}
	if err := sc.Err(); err != nil {
		return err
	}

	path, mtime := t.path, t.mtime
	*t = scratch
	t.path, t.mtime = path, mtime
	return nil
}
