// NELSON: HERO OF THE KEYS. Words fall, you type them, letters explode.
//
// There is no lock and no selected word: any keypress takes a letter off
// whichever word wants it and is furthest down the screen. See SPEC.md 2.3.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"

	"nelson/internal/audio"
	"nelson/internal/engine"
	"nelson/internal/render"
	"nelson/internal/replay"
	"nelson/internal/stats"
	"nelson/internal/words"
)

const (
	fps      = 30
	dt       = float32(1) / fps
	tickDur  = time.Second / fps
	watchInt = 30 // frames between tuning.conf mtime checks
)

type opts struct {
	debug      bool
	seed       uint64
	speed      float64
	tuning     string
	pack       string
	oneWord    string
	font       string
	god        bool
	startLevel float64
	daily      bool
	record     string
	replayFile string
	noIntro    bool
	sound      bool
	music      bool
	// soundSet and musicSet record whether the flag was actually typed. Both
	// default on now, so the default alone cannot be allowed to speak: it would
	// overrule a player who turned audio off from the menu, every launch.
	soundSet bool
	musicSet bool
	volume   float64
}

func main() {
	var o opts
	flag.BoolVar(&o.debug, "debug", false, "show the tuning overlay")
	flag.Uint64Var(&o.seed, "seed", 0, "PRNG seed; 0 picks one from the clock")
	flag.Float64Var(&o.speed, "speed", 0, "override speed_scale")
	flag.StringVar(&o.tuning, "tuning", "tuning.conf", "path to the tunables file")
	flag.StringVar(&o.pack, "words", "english", "word pack: english (47k defined words) or argument, or a path to a .txt file")
	flag.StringVar(&o.oneWord, "word", "", "force this single word, for font work")
	flag.BoolVar(&o.god, "god", false, "never lose a life, for looking at the render")
	flag.Float64Var(&o.startLevel, "level", 0, "start at this level")
	flag.BoolVar(&o.daily, "daily", false, "seed from today's date, so everyone gets the same run")
	flag.StringVar(&o.record, "record", "", "write a replay of this run to a file")
	flag.StringVar(&o.replayFile, "replay", "", "play back a recorded run")
	flag.BoolVar(&o.noIntro, "no-intro", false, "skip the opening sequence")
	flag.BoolVar(&o.sound, "sound", true, "sound effects; -sound=false for silence")
	flag.BoolVar(&o.music, "music", true, "background music; -music=false for silence")
	flag.Float64Var(&o.volume, "volume", 0.5, "audio volume, 0 to 1")
	flag.StringVar(&o.font, "font", "braille", "drawing style: braille (half the size, 2x4 dots per cell) or blocks (larger, 2 dots per cell)")
	flag.Parse()
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "sound":
			o.soundSet = true
		case "music":
			o.musicSet = true
		}
	})

	if err := run(o); err != nil {
		fmt.Fprintln(os.Stderr, "nelson:", err)
		os.Exit(1)
	}
}

// scene is which screen the game is showing. Everything before this was a single
// mode; the menu and guide need somewhere to live that is not the play loop.
type scene int

const (
	sceneIntro scene = iota
	sceneMenu
	sceneStats
	sceneGuide
	sceneReady
	scenePlay
	sceneDefeat
)

// game is one session. The loop body lives on it as tick() so it can be driven
// by a test against a SimulationScreen instead of a terminal.
type game struct {
	scr  tcell.Screen
	tune *engine.Tuning
	st   *engine.State
	lay  render.Layout
	fx   *render.Effects
	dbg  *render.Debug

	termW, termH int
	quit         bool
	syncNeeded   bool

	scene     scene
	menuPick  int
	guidePage int
	daily     bool

	rec    *replay.Recording
	replay *replay.Recording

	audio    *audio.Player
	wantSFX  bool
	wantTune bool
	volume   float64

	introAt  float64
	defeatAt float64

	stats *stats.Stats
	// ranked is false when a flag has changed the run: god mode, a speed
	// override, a level skip or a forced word. Those are useful for development
	// and are not the same game, so they must not touch personal bests. A best
	// that anyone can produce with a flag is not a best.
	ranked bool
	// highScore is whether the last finished run beat the best, for the game over
	// screen.
	highScore bool
}

func newGame(scr tcell.Screen, tune *engine.Tuning, vocab []string, seed uint64, debug bool) *game {
	w, h := scr.Size()
	g := &game{scr: scr, tune: tune, termW: w, termH: h}
	if debug {
		g.dbg = &render.Debug{}
	}
	g.fx = render.NewEffects(seed)
	g.scene = sceneMenu
	g.stats = &stats.Stats{Daily: map[string]int{}}
	g.ranked = true
	g.lay = render.ComputeLayout(w, h, tune)
	g.st = engine.NewState(g.lay.PlayW, g.lay.PlayH, tune, engine.NewRNG(seed), vocab)
	g.st.MaxWordsCap = g.lay.MaxConcurrent()
	// No spawn here. The spawner owns every spawn, and its timer starts at zero so
	// the first word arrives on the first tick. Seeding one by hand as well is what
	// made every run open with two words at once.
	return g
}

func (g *game) handleEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventKey:
		if e.Key() == tcell.KeyCtrlC {
			g.quit = true
			return
		}
		if e.Key() == tcell.KeyCtrlL {
			g.syncNeeded = true
			return
		}
		switch g.scene {
		case sceneIntro:
			// Any key skips, from the first frame. Nobody should be held in a
			// cutscene they have already decided against.
			g.scene = sceneMenu
			return
		case sceneDefeat:
			g.scene = scenePlay
			g.defeatAt = 0
			return
		case sceneMenu:
			g.menuKey(e)
			return
		case sceneStats:
			if e.Key() == tcell.KeyEscape || e.Key() == tcell.KeyEnter ||
				(e.Key() == tcell.KeyRune && e.Rune() == ' ') {
				g.scene = sceneMenu
			}
			return
		case sceneGuide:
			g.guideKey(e)
			return
		case sceneReady:
			g.readyKey(e)
			return
		}

		switch e.Key() {
		case tcell.KeyEscape:
			// Back to the menu rather than out of the program: a run just ended or
			// is in progress, and quitting the whole game is a bigger step than a
			// player pressing escape usually means.
			g.toMenu()
			return
		case tcell.KeyRune:
		default:
			return
		}

		ch := e.Rune()
		if g.dbg != nil {
			g.dbg.KeysSeen++
			g.dbg.LastKey = string(ch)
		}
		g.typeKey(ch)

	case *tcell.EventResize:
		g.resize()
	}
}

func (g *game) resize() {
	g.termW, g.termH = g.scr.Size()
	prev := g.lay
	g.lay = render.ComputeLayout(g.termW, g.termH, g.tune)
	// Only restart the field when the playable area actually changed. A resize
	// that does not alter the capped playfield must not throw away the run.
	if g.lay.TooSmall != prev.TooSmall || g.lay.PlayW != prev.PlayW || g.lay.PlayH != prev.PlayH {
		g.st.PlayW, g.st.PlayH = g.lay.PlayW, g.lay.PlayH
		g.st.MaxWordsCap = g.lay.MaxConcurrent()
		g.st.Words = g.st.Words[:0]
	}
	g.syncNeeded = true
}

// typeKey is the whole typing path, shared by the keyboard and by a replay.
func (g *game) typeKey(ch rune) {
	// Instant restart. Every keypress of friction between death and the next
	// attempt costs attempts, so SPACE goes straight into a new run with no menu
	// in the way.
	if g.st.GameOver {
		if ch == ' ' {
			g.highScore = false
			g.st.Reset()
		}
		return
	}

	// Normalise to uppercase so caps lock and shift both work.
	if ch >= 'a' && ch <= 'z' {
		ch -= 32
	}
	if !engine.Typeable(ch) {
		return
	}
	if g.rec != nil {
		g.rec.Record(g.st.Frame, ch)
	}

	hit := g.st.Type(byte(ch))
	if !hit.Ok {
		g.sfx(audio.EffectTypo, 1)
		return
	}
	if hit.Completed {
		// The completion bonus is most of the jump in the total, so it gets a
		// number floating off the word rather than arriving unexplained.
		g.fx.Popup(fmt.Sprintf("+%d", hit.Points), hit.Cell, hit.Mult)
		g.fx.Detonate(hit.Boxes, hit.Mult, hit.Chain)
		switch {
		case hit.Bomb:
			g.sfx(audio.EffectBomb, hit.Mult)
		case hit.Power != engine.PowerNone:
			g.sfx(audio.EffectPowerup, hit.Mult)
		case hit.Chain >= 2:
			g.sfx(audio.EffectChain, hit.Mult)
		default:
			g.sfx(audio.EffectWord, hit.Mult)
		}
		if hit.Bomb {
			g.fx.Impact()
		}
		if len(hit.Cleared) > 0 {
			g.fx.Detonate(hit.Cleared, hit.Mult, 0)
		}
		if hit.Reward != nil && len(hit.Reward.Boxes) > 0 {
			g.fx.Detonate(hit.Reward.Boxes, hit.Mult, 0)
		}
	} else {
		g.fx.Burst(hit.Cell, hit.Mult)
		g.sfx(audio.EffectLetter, hit.Mult)
	}
}

// sfx plays an effect, if effects are on.
func (g *game) sfx(e audio.Effect, mult int) {
	if g.wantSFX {
		g.audio.Play(e, mult)
	}
}

// playMusic sets the background track, if music is on. Safe to call every frame:
// the player ignores a request for whatever is already running.
func (g *game) playMusic(t audio.Tune) {
	if g.wantTune {
		g.audio.Music(t)
	}
}

func (g *game) toMenu() {
	g.scene = sceneMenu
	g.menuPick = 0
	g.st.Reset()
}

func (g *game) menuKey(e *tcell.EventKey) {
	switch e.Key() {
	case tcell.KeyEscape:
		g.quit = true
	case tcell.KeyUp:
		g.menuPick = (g.menuPick + render.MenuCount - 1) % render.MenuCount
	case tcell.KeyDown:
		g.menuPick = (g.menuPick + 1) % render.MenuCount
	case tcell.KeyEnter:
		g.choose()
	case tcell.KeyRune:
		// Letters must not select anything. The player is one keypress away from a
		// mode where every letter does something, and a menu that also reacts to
		// letters teaches the wrong reflex on the very first screen. J and K are
		// the exception because they are movement, not selection.
		switch e.Rune() {
		case 'k', 'K':
			g.menuPick = (g.menuPick + render.MenuCount - 1) % render.MenuCount
		case 'j', 'J':
			g.menuPick = (g.menuPick + 1) % render.MenuCount
		case ' ':
			g.choose()
		}
	}
}

func (g *game) choose() {
	switch g.menuPick {
	case render.MenuPlay:
		g.st.Reset()
		g.scene = sceneReady
	case render.MenuGuide:
		g.guidePage = 0
		g.scene = sceneGuide
	case render.MenuStats:
		g.scene = sceneStats
	case render.MenuSound:
		g.setAudio(!g.wantSFX, g.wantTune)
	case render.MenuMusic:
		g.setAudio(g.wantSFX, !g.wantTune)
	case render.MenuQuit:
		g.quit = true
	}
}

// applyAudio turns sound and music on or off for this run.
//
// The player is built on first use rather than at startup, because rendering
// every sound costs about fifty milliseconds and somebody who runs with
// -sound=false -music=false should never pay it.
func (g *game) applyAudio(sound, music bool) {
	if (sound || music) && g.audio == nil {
		g.audio = audio.New(g.volume)
		if g.audio == nil {
			// Nothing here can play a WAV. Leave both off rather than showing a
			// switch that does nothing.
			return
		}
	}
	g.wantSFX, g.wantTune = sound, music
	if !music {
		g.audio.StopMusic()
	}
}

// setAudio is applyAudio plus remembering the answer, for the menu switches. The
// flags do not go through here: a flag is for one run, and a run should not
// quietly rewrite the setting the player will get next time.
func (g *game) setAudio(sound, music bool) {
	g.applyAudio(sound, music)
	g.stats.Sound, g.stats.Music = g.wantSFX, g.wantTune
	g.stats.AudioChosen = true
	_ = g.stats.Save()
}

func (g *game) guideKey(e *tcell.EventKey) {
	switch e.Key() {
	case tcell.KeyEscape:
		g.scene = sceneMenu
	case tcell.KeyLeft:
		if g.guidePage > 0 {
			g.guidePage--
		}
	case tcell.KeyRight:
		g.nextGuidePage()
	case tcell.KeyEnter:
		g.nextGuidePage()
	case tcell.KeyRune:
		if e.Rune() == ' ' {
			g.nextGuidePage()
		}
	}
}

func (g *game) nextGuidePage() {
	if g.guidePage < render.GuidePages()-1 {
		g.guidePage++
		return
	}
	g.scene = sceneMenu
}

// readyKey starts the run on ANY key, not a specific one: the player is about to
// be in a mode where every key does something.
func (g *game) readyKey(e *tcell.EventKey) {
	if e.Key() == tcell.KeyEscape {
		g.toMenu()
		return
	}
	g.scene = scenePlay
}

// tick advances one fixed timestep and draws it.
func (g *game) tick() {
	start := time.Now()

	// A replay feeds the recorded keys in at the frame they were pressed. It works
	// only because the simulation takes a constant dt and owns its own PRNG, so a
	// run is a pure function of seed, tuning and keys-by-frame.
	if g.replay != nil && g.scene == scenePlay {
		for _, ch := range g.replay.At(g.st.Frame) {
			g.typeKey(ch)
		}
	}

	if g.st.Frame%watchInt == 0 {
		reloaded, err := g.tune.Watch()
		if g.dbg != nil {
			if err != nil {
				g.dbg.TuneErr = err.Error()
			} else if reloaded {
				g.dbg.TuneErr = ""
				g.dbg.Reloads++
			}
		}
		if reloaded {
			// A tuning edit can change the screen contract, so re-resolve it.
			g.lay = render.ComputeLayout(g.termW, g.termH, g.tune)
			g.st.PlayW, g.st.PlayH = g.lay.PlayW, g.lay.PlayH
			g.st.MaxWordsCap = g.lay.MaxConcurrent()
		}
	}

	switch {
	case g.lay.TooSmall:
		g.st.Frame++
		render.DrawTooSmall(g.scr, g.lay)
	case g.scene == sceneIntro:
		g.playMusic(audio.TuneIntro)
		g.introAt += float64(dt)
		if !render.DrawIntro(g.scr, g.lay, g.introAt, g.st.Frame) {
			g.scene = sceneMenu
		}
		g.st.Frame++
	case g.scene == sceneDefeat:
		g.playMusic(audio.TuneDefeat)
		g.defeatAt += float64(dt)
		if !render.DrawDefeat(g.scr, g.lay, g.defeatAt, g.st.Killer) {
			g.scene = scenePlay
		}
		g.st.Frame++
	case g.scene == sceneMenu:
		g.playMusic(audio.TuneMenu)
		render.DrawMenu(g.scr, g.lay, g.menuPick,
			render.MenuLabels(g.wantSFX, g.wantTune), g.bestLine())
	case g.scene == sceneStats:
		render.DrawStats(g.scr, g.lay, g.stats.Detail(), g.leaderboard())
	case g.scene == sceneGuide:
		render.DrawGuide(g.scr, g.lay, g.guidePage)
	case g.scene == sceneReady:
		render.DrawReady(g.scr, g.lay)
	default:
		before := g.st.Level()
		res := g.st.Step(dt)
		if res.LostLife || res.Shielded {
			g.fx.Impact()
			g.sfx(audio.EffectLife, 1)
		}
		if res.Swarm > 0 {
			g.sfx(audio.EffectSwarm, 1)
		}
		if g.st.Level() > before {
			g.sfx(audio.EffectLevelUp, 1)
		}
		// Later levels get the harder tune. Music is set every frame and the
		// player ignores a request for what is already running.
		if g.st.LevelF() >= 6 {
			g.playMusic(audio.TunePlayLate)
		} else {
			g.playMusic(audio.TunePlayEarly)
		}
		if res.GameOver {
			g.sfx(audio.EffectGameOver, 1)
			g.finish()
			// The Trafalgar sequence sits between death and the score screen, so
			// it is friction by construction. It is kept short and always
			// skippable; the retry path from the score screen is unchanged.
			g.scene = sceneDefeat
			g.defeatAt = 0
		}
		g.fx.Update(dt)
		render.ComposeWithHint(g.scr, g.st, g.lay, g.fx, g.dbg, g.highScore)
	}

	if g.syncNeeded {
		g.scr.Sync()
		g.syncNeeded = false
	} else {
		g.scr.Show()
	}

	if g.dbg != nil {
		// Reported one frame late, which is the price of drawing the number
		// inside the frame it describes.
		ms := float64(time.Since(start).Nanoseconds()) / 1e6
		g.dbg.FrameMS = ms
		if ms > g.dbg.PeakMS {
			g.dbg.PeakMS = ms
		}
	}
}

// leaderboard is the top runs, shown on the stats screen.
func (g *game) leaderboard() []render.Leader {
	var out []render.Leader
	// Five rather than three: the menu used to carry this and had room for one,
	// and the stats screen has rows to spare.
	for _, e := range g.stats.Top(5) {
		out = append(out, render.Leader{Score: e.Score, Seconds: e.Seconds})
	}
	return out
}

// bestLine is the note on the last row of the menu.
func (g *game) bestLine() string {
	line := g.stats.Summary()
	if !g.ranked {
		if line == "" {
			return "unranked: flags are set"
		}
		return line + "   (unranked: flags are set)"
	}
	return line
}

// finish folds a completed run into the stats and saves.
func (g *game) finish() {
	if !g.ranked {
		return
	}
	run := stats.Run{
		Score:   g.st.Score,
		Seconds: float64(g.st.Elapsed),
		Chain:   g.st.BestChain,
		Streak:  g.st.BestStreak,
		Words:   g.st.WordsDestroyed,
		Letters: g.st.Hits,
		Keys:    g.st.Keys,
	}
	if g.daily {
		run.DailySeed, run.DailyStamp = true, stats.Today()
	}
	g.highScore = g.stats.Record(run)
	// A failed save must never interrupt play. The score screen is already up and
	// the player is about to press space.
	_ = g.stats.Save()
}

func run(o opts) error {
	// East Asian ambiguous width. Every block character the game draws (█ ▀ ▄ ▓
	// ▒ · ▔) is Unicode Ambiguous, which means two cells wide when a CJK width
	// setting is on. tcell turns that on from RUNEWIDTH_EASTASIAN, so a user
	// with it set would see every word render double-width, the playfield
	// geometry go wrong and the half-row motion break. Force it off. This is a
	// Unicode property, not a Go one, and would have shipped in any language.
	uniseg.EastAsianAmbiguousWidth = 1

	// Drawing style, chosen before any State exists because it decides how many
	// dots fit in a cell and therefore how wide and tall every word is.
	m, ok := engine.MetricsByName(o.font)
	if !ok {
		return fmt.Errorf("unknown --font %q: want braille or blocks", o.font)
	}
	engine.SetMetrics(m)

	tune, err := engine.LoadTuning(o.tuning)
	if err != nil {
		return err
	}
	if o.speed > 0 {
		tune.SpeedScale = o.speed
	}

	// Parse the definition table off the frame path, so the first cleared word does
	// not pay for it.
	words.Preload()

	vocab, err := words.Load(o.pack)
	if err != nil {
		return err
	}
	if o.oneWord != "" {
		vocab = []string{o.oneWord}
	}

	var rp *replay.Recording
	if o.replayFile != "" {
		var err error
		rp, err = replay.Load(o.replayFile)
		if err != nil {
			return err
		}
		// Everything the run depended on comes from the file, or it is not the
		// same run.
		o.seed, o.font, o.pack = rp.Seed, rp.Font, rp.Pack
	}

	seed := o.seed
	if o.daily {
		// Same sequence for everyone today, so scores are comparable and there is
		// something to argue about.
		y, m, d := time.Now().Date()
		seed = uint64(y)*10000 + uint64(m)*100 + uint64(d)
	}
	if seed == 0 {
		seed = uint64(time.Now().UnixNano())
	}

	scr, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := scr.Init(); err != nil {
		return err
	}
	// Restore the terminal before anything else can print. A panic with the
	// screen still live leaves the user with a wrecked shell and no visible
	// error, which is the worst failure mode a TUI has.
	var gameRef *game
	defer func() {
		scr.Fini()
		if g := gameRef; g != nil && g.rec != nil {
			if err := g.rec.Save(o.record); err != nil {
				fmt.Fprintln(os.Stderr, "nelson: saving replay:", err)
			} else {
				fmt.Fprintln(os.Stderr, "nelson: replay written to", o.record)
			}
		}
		if r := recover(); r != nil {
			panic(r)
		}
	}()
	scr.HideCursor()
	scr.Clear()

	// PollEvent blocks, so it runs in its own goroutine feeding a buffered
	// channel that the loop drains without blocking.
	events := make(chan tcell.Event, 64)
	go func() {
		for {
			ev := scr.PollEvent()
			if ev == nil {
				return
			}
			events <- ev
		}
	}()

	g := newGame(scr, tune, vocab, seed, o.debug)
	gameRef = g
	g.volume = o.volume
	// The remembered setting is the answer unless the flag was typed, in which
	// case it wins for this run. Merging them with an OR would mean a default of
	// on could never be turned off.
	wantSound, wantMusic := g.stats.Sound, g.stats.Music
	if o.soundSet {
		wantSound = o.sound
	}
	if o.musicSet {
		wantMusic = o.music
	}
	if wantSound || wantMusic {
		g.applyAudio(wantSound, wantMusic)
		if g.audio == nil {
			fmt.Fprintln(os.Stderr,
				"nelson: no afplay, paplay or aplay found; running silent")
		}
	}
	defer g.audio.Close()
	if !o.noIntro && rp == nil {
		g.scene = sceneIntro
	}
	if o.record != "" {
		g.rec = replay.NewRecording(seed, o.font, o.pack)
	}
	if rp != nil {
		g.replay = rp
		// A replay is somebody else's run being reproduced, not yours.
		g.ranked = false
		g.scene = scenePlay
	}
	g.stats = stats.Load()
	g.daily = o.daily
	g.ranked = !o.god && o.speed == 0 && o.startLevel == 0 && o.oneWord == ""
	g.st.God = o.god
	if o.startLevel > 1 {
		// Jump the clock forward rather than faking a level, so everything derived
		// from it stays consistent.
		g.st.Elapsed = float32(engine.TimeForLevel(tune, o.startLevel-1))
	}

	next := time.Now()
	for {
		// Drain every pending event. A fast typist landing three keys inside one
		// 33 ms frame must get all three, so this never stops early.
		for draining := true; draining; {
			select {
			case ev := <-events:
				g.handleEvent(ev)
			default:
				draining = false
			}
		}
		if g.quit {
			return nil
		}

		g.tick()

		// Fixed timestep. Overrunning frames are dropped rather than
		// accumulated: a typing game needs no catch-up physics, and a constant
		// dt is what makes a seeded run reproducible.
		next = next.Add(tickDur)
		if d := time.Until(next); d > 0 {
			time.Sleep(d)
		} else if d < -4*tickDur {
			next = time.Now()
		}
	}
}
