package engine

import "fmt"

// Chains: the payoff for free fire, and the highest-skill thing in the game.
//
// Finish a word and a window opens. Finish another inside it and they chain, the
// window reopening each time. Because any key can hit any word, a player can
// leave several words sitting on their final letter and then detonate them in a
// burst, which is the only mechanic here that rewards reading the whole field
// rather than typing quickly.
//
// It was also, until this point, invisible. The window was not drawn, the loaded
// words were not marked, and the reward was a number that flashed and vanished.
// Someone who had played the game for an hour could not describe what a chain
// was, which is a design failure rather than a player failure: nothing on screen
// said that a word was one key from done, that a window was open, or how long
// was left. See Loaded, ChainSecondsLeft and the HUD.

// ChainMultiplier is what a completion is worth at a given chain position.
//
// It runs past the old ceiling of five. Capping there stopped the scale at
// exactly the moment the best players were doing the most impressive thing, and a
// ten-chain should be worth talking about.
func ChainMultiplier(chain int) int {
	switch {
	case chain >= 10:
		return 20
	case chain >= 7:
		return 12
	case chain >= 5:
		return 8
	case chain == 4:
		return 5
	case chain == 3:
		return 3
	case chain == 2:
		return 2
	default:
		return 1
	}
}

// Loaded reports whether a word is one key from being finished. A loaded word is
// chain fuel, and marking it is the single change most likely to make the
// mechanic land: a field with three loaded words should LOOK like an
// opportunity, so a player tries three keys and learns the whole strategy
// without being told it.
func (w *Word) Loaded() bool { return !w.Bomb && w.Typed == len(w.Text)-1 }

// LoadedCount is how many words are one key from done, for the HUD.
func (s *State) LoadedCount() int {
	n := 0
	for i := range s.Words {
		if s.Words[i].Loaded() {
			n++
		}
	}
	return n
}

// ChainOpen reports whether a completion right now would extend the chain.
func (s *State) ChainOpen() bool {
	return s.Chain > 0 && s.Elapsed <= s.chainDeadline
}

// ChainSecondsLeft is how long the window has to run. Drawing this is the second
// half of making chains legible: a window the player cannot see is a window they
// cannot aim at.
func (s *State) ChainSecondsLeft() float32 {
	if !s.ChainOpen() {
		return 0
	}
	return s.chainDeadline - s.Elapsed
}

// NextChainMultiplier is what the next completion would be worth, so the HUD can
// show what is being played for rather than only what has been won.
func (s *State) NextChainMultiplier() int {
	if !s.ChainOpen() {
		return ChainMultiplier(1)
	}
	return ChainMultiplier(s.Chain + 1)
}

// LevelUpBanner is the level to announce, if the schedule just advanced.
func (s *State) LevelUpBanner() (int, bool) {
	if s.Elapsed < s.levelUpUntil {
		return s.Level() + 1, true
	}
	return 0, false
}

// InDanger reports whether a word is close enough to the floor to warrant the
// alarm. It is deliberately tied to rows rather than to time: the player is
// reading positions, not doing arithmetic.
func (s *State) InDanger(w *Word) bool {
	return w.BottomRow() >= s.PlayH-int(s.Tune.DangerRows)
}

// RewardBanner is the milestone to announce, if one just fired.
func (s *State) RewardBanner() string {
	if s.Elapsed < s.rewardUntil {
		return s.rewardName
	}
	return ""
}

// ChainReward is a milestone effect earned by a deep chain.
type ChainReward struct {
	Chain int
	Name  string
	Boxes []Rect // words destroyed by the reward, for the explosion
}

// chainReward fires at the tiers that pay in survival rather than points.
//
// Past two or three minutes the field is fast enough that a score multiplier is a
// consolation prize: the problem is staying alive, and points do not help with
// that. So the deep tiers pay in the same currency power-ups do, except earned
// entirely by the player rather than handed over by a lucky spawn.
//
// It needs no gate on level, because it gates itself. Chains get harder to build
// as the game speeds up: less time to pre-chew, less time inside the window, and
// chip acceleration punishing every loaded word left sitting. The deep tiers
// become rarer exactly as they become more valuable.
func (s *State) chainReward(chain int) *ChainReward {
	r := s.chainRewardFor(chain)
	if r != nil {
		s.rewardName = fmt.Sprintf("CHAIN %d  %s", chain, r.Name)
		s.rewardUntil = s.Elapsed + 1.6
	}
	return r
}

func (s *State) chainRewardFor(chain int) *ChainReward {
	switch chain {
	case int(s.Tune.ChainSlowAt):
		// SLOW and SHIELD are no longer spawnable, but the effects live on here.
		// Routed through applyPower so the stack and its cap have one definition.
		s.applyPower(PowerSlow)
		return &ChainReward{Chain: chain, Name: "BREATHING ROOM"}

	case int(s.Tune.ChainShieldAt):
		s.applyPower(PowerShield)
		return &ChainReward{Chain: chain, Name: "SHIELD"}

	case int(s.Tune.ChainSweepAt):
		// Everything in the lower half goes, and only the lower half. A chain should
		// not hand out more than BLAST, which is the field-wipe a player can find.
		cut := s.PlayH / 2
		var boxes []Rect
		kept := s.Words[:0]
		for i := range s.Words {
			w := &s.Words[i]
			if w.Bomb || w.BottomRow() < cut {
				kept = append(kept, *w)
				continue
			}
			boxes = append(boxes, w.LetterBoxes()...)
		}
		s.Words = kept
		return &ChainReward{Chain: chain, Name: "SWEEP", Boxes: boxes}

	case int(s.Tune.ChainLifeAt):
		if max := int(s.Tune.MaxLives); s.Lives < max {
			s.Lives++
			return &ChainReward{Chain: chain, Name: "EXTRA LIFE"}
		}
		return &ChainReward{Chain: chain, Name: "LIVES FULL"}
	}
	return nil
}
