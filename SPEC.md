# KEYBOARD WARRIOR

A terminal typing game. Words fall from the top of the screen in big block capitals. You destroy a word by typing it, letter by letter, each one exploding as it goes. Miss a word and it hits the floor and costs a life. Three lives, then it is over.

The design has one idea underneath it: **you may type any letter of any word on screen at any time.** Everything else in this document exists to make that idea pay off.

## 1. Player experience

One session:

1. Title screen. Big block logo, high score, "PRESS ANY KEY".
2. A word drifts in from the top. You type its first letter. The letter detonates.
3. A second word arrives. You leave the first half-eaten, take two letters off the second, come back.
4. A cluster builds. You shave the front off all four words, then finish them in a burst. The chain counter climbs x2 x3 x5 and the score jumps by a few hundred.
5. You let a word fall to the last two rows for the x5 depth bonus, kill it there, and the explosion fills a third of the screen.
6. A RUNNER catches a DRIFTER you were ignoring. Both slam down together. The floor cracks, the screen shakes, a life is gone.
7. Third life. Game over, with the word that killed you rendered in the biggest font in the game. SPACE, and you are already playing again.

## 2. Rules of play

### 2.1 How to play

Words fall from the top of the screen in large block capitals. Type a word's letters in order, left to right. Each letter explodes as you type it. Type the last letter and the whole word detonates and pays a bonus.

A word that reaches the floor at the bottom of the screen costs you a life. You start with three. At zero the run ends.

Words are A to Z with no spaces, so case does not matter and there is nothing to press between words. A few words carry a punctuation mark on the end, which you type like any other character; those are power-ups, and they are described in 2.9.

Typing is the only input during play. There is nothing to select and nothing to aim.

### 2.2 There is no selected word

Most typing games make you pick a word and finish it before starting another. This one does not. Every key you press is offered to the whole field at once.

When you press a key, the game finds every word on screen whose next untyped letter is that key.

- If no word wants the letter, the press is a miss.
- If one word wants it, that word loses the letter.
- If several want it, the word furthest down the screen loses the letter.

So three words on screen starting with S means pressing S three times takes the S off all three, lowest first. You can type half of one word, switch to another, and come back later. Nothing is lost by switching, and a half-typed word stays exactly where it is on screen: the letters you destroyed leave gaps rather than the rest of the word sliding along.

When two words are level, the leftmost wins. If they are still level, the older one wins.

The screen shows you where a key will go. The next letter of every word is drawn bright white, and when several words are waiting on the same letter, a marker sits under the one that will receive the next press.

A miss costs you your streak and makes the whole field a little faster (2.6). It does not cost a life and it does not lock your keyboard. You are never punished for typing faster than the field.

### 2.3 Lives, the floor and landings

You start with three lives and can hold at most five.

A word that touches the floor is removed and costs one life. If two words land in the same instant, that is two lives.

A landing also does three things:

- Every other word within 3 rows of the floor is destroyed. You score nothing for them.
- Spawning stops for 1.5 seconds.
- Your streak and your chain reset to zero.

Those exist so that being overwhelmed is survivable. Without them, one bad moment ends the run before you can react.

The floor is a wall, and it loses a chunk at every life you lose, so you can read your remaining lives without looking at the score bar.

Power-up words and bombs never cost a life when they land. Only ordinary words can hurt you.

### 2.4 Score

Every letter you destroy scores points. How many depends on how far the word has fallen.

| Where the word is | Each letter is worth |
|---|---|
| Top third of the screen | 1 |
| Middle third | 2 |
| Bottom third | 3 |
| Last two rows | 5 |

The word's colour tells you which band it is in: cyan, yellow, orange, then white.

That is the central decision in the game. Killing a word early is safe and cheap. Letting it fall and killing it near the floor is worth five times as much and might cost a life.

Finishing a word pays a completion bonus on top:

    bonus = number of letters  x  the depth multiplier where it died  x  the chain multiplier

The chain multiplier is 1 unless a chain is running. Chains are 2.5.

Two counters sit in the score bar and are easy to confuse:

- **STREAK** counts correct keys in a row. It resets on a miss.
- **CHAIN** counts words finished in quick succession. It is explained below.

Also tracked and shown, but not part of the score: words per minute, accuracy, longest chain, words destroyed, and time survived.

### 2.5 Chains

Finish a word and a 1.5 second window opens. Finish another word inside that window and the two are chained, and the window reopens. Keep finishing words and the chain keeps climbing.

The chain multiplies the completion bonus:

| Words finished in a row | Multiplier |
|---|---|
| 2nd | x2 |
| 3rd | x3 |
| 4th | x5 |
| 5th and 6th | x8 |
| 7th, 8th, 9th | x12 |
| 10th and beyond | x20 |

You need a streak of at least 5 correct keys for a completion to extend a chain. Mashing does not chain.

**How to actually build one.** Because any key hits any word, you can bring several words down to their last letter without finishing them, then fire off those last letters one after another. Five words waiting on one letter each is five completions in under a second. This is where the points are. It rewards reading the whole field rather than typing quickly.

A word that is one key from done is drawn in its own bright state. When you see three of those on screen at once, that is a chain waiting to happen.

The catch is in 2.6: every letter you take off a word makes that word fall faster. Pre-chewing five words means five words accelerating while you set up.

Once a chain is running, the screen shows the time left and what the next completion is worth, like `CHAIN x3 · 0.9s · next x5`.

**Chain rewards.** Deep chains pay in survival rather than points, because past a few minutes staying alive matters more than the score. Each fires once as you pass through it:

| Chain | Reward |
|---|---|
| 4 | BREATHING ROOM. The whole field slows by 20% for 8 seconds. Repeat grants stack up to 50%. |
| 6 | SHIELD. The next landing is free. |
| 8 | SWEEP. Every ordinary word in the lower half of the screen is destroyed. |
| 10 | EXTRA LIFE, up to the cap of five. |

### 2.6 What makes the field faster

Four things. Three of them are your doing.

**The clock.** The difficulty schedule raises the fall speed as the run goes on. It applies to every word on screen, including ones already falling. See 2.7.

**Chip acceleration.** A word speeds up by 12% for every letter you take off it. Take four letters off a word and it falls 48% faster than an untouched one. This is the price of setting up a chain, and it is the reason free fire is a gamble rather than a free option.

**Typos.** Every miss adds 4% to the speed of the whole field. It decays with a half-life of 3 seconds and never exceeds 35%. Clean typing traces the schedule exactly; sloppy typing runs above it. A steady 95% accuracy sits around 3% over, 80% accuracy around 14% over.

**Collisions.** Two words are never allowed to overlap, because two block words on the same cells cannot be read. So when a word catches the word below it, the lower word inherits the faster speed and keeps it. Both words flash when this happens.

That last rule has a consequence worth knowing. Speeding up a word by chipping it can kick the word underneath into gear, and a slow word you had ten seconds on becomes a fast one. Words that catch each other fall as a locked pair.

### 2.7 How the game gets harder

Difficulty runs on the clock and on nothing else. It does not respond to how well you are doing. Two runs of the same length are the same run, so times and scores compare.

Level 1 lasts 8 seconds and each level after it is a little longer. Level 5 arrives at about a minute, level 10 at about three minutes.

As the level rises:

- Fall speed climbs from 1.2 rows per second, steeply at first and then flattening, to a ceiling of 4.5.
- The gap between spawns shrinks from 3.5 seconds, by 12% per level, to a floor of 0.6 seconds.
- One more word is allowed on screen every two levels, up to what the screen can hold.
- Words get longer, from 5 letters up to 12.
- Word choice is biased toward awkward words: same-finger repeats, same-hand runs, reaches off the home row. Late words are hard to type as well as long.

Speed and spawn gap change every frame rather than jumping when the level number changes. The level shown in the score bar is a readout, not a step.

At the default ceiling of 4.5 rows per second, the schedule stops biting after about twelve minutes, so a strong player can survive until they slip. Setting `max_fall_speed = 0` removes the ceiling and makes the run finite.

### 2.8 Bombs

A bomb is a word you must **not** finish. It is drawn in red on a red wash, bracketed above and below, and it never shows the white next-letter highlight that other words show.

**A bomb never takes priority.** A keypress reaches a bomb only when no ordinary word on the field wants that letter. There is no arrangement of words that can force you to feed a bomb, so a bomb can only ever receive a letter that nothing else wanted, which is what a typo looks like.

The rules:

- A bomb that reaches the floor costs nothing and disappears. Ignoring a bomb is always allowed and usually correct.
- Finishing a bomb costs a life and resets your streak and chain. Nothing is cleared and no shockwave fires, because nothing landed.
- At most one bomb is on screen at a time.
- Bombs appear from level 4, on about one spawn in twelve, and are 5 to 7 letters long.
- A bomb never spawns sharing a first letter with a word already on screen.
- Letters you have fed the bomb show as lit fuse segments above it, so you can see how close it is without counting.

In practice a bomb on screen is a readout of how accurate you are being. Type cleanly and it lands harmlessly.

### 2.9 Power-up words

Some words carry a punctuation mark on the end: `FREEZE.`, `REWIND-`, `BLAST/`, `REPAIR+`. You type the mark like any other key, and finishing the word fires the effect. The name of the effect is printed above the word, so there is nothing to memorise.

Punctuation appears only on power-up words, so a mark can never be stolen by an ordinary word.

Power-ups start appearing from level 3, and they always ride a longer word than their neighbours. That is the trade: more letters to type, more time exposed, and chip acceleration biting harder. A power-up word that reaches the floor costs nothing, so taking one is a choice rather than an obligation.

The four effects and their rules are in section 6.

### 2.10 Breaks and swarms

Two things interrupt the steady rise.

**Breath.** After every 10 words you destroy, spawning stops for 2 seconds.

**Swarm.** From level 4, about every 90 seconds, five short words drop at once, all starting with the same letter. Press that letter five times and you take the head off all five, lowest first. Swarms are where free fire pays off hardest.

### 2.11 The clock

The score bar shows elapsed time since the first spawn as `MM:SS`. It stops at game over and freezes while paused.

### 2.12 How words are drawn, and how many fit

Every word is drawn in a 5x6 pixel block font, in the style of 8-bit machines and LED matrix displays.

One glyph pixel is one **half** cell, not a full one. A terminal cell is about twice as tall as it is wide, so a full-cell pixel stretches every letter 2:1 and the blocks merge into slabs. Half cells are square, so the pixel grid stays visible and the letters keep their shape. Six pixel rows land on exactly 3 terminal rows.

| | Columns per letter | Rows | Used for |
|---|---|---|---|
| Normal | 6 (5 plus 1 of kerning) | 3 | every falling word |
| Huge | 6 | 6 | the title, and the word that ended your run |

A word of n letters is `6n - 1` columns wide in both sizes. A 12-letter word is 71 columns, and the playfield is at least 78, so length is never the constraint. Height is. The number of words allowed on screen is `min(6, playable_rows / 4)`: four at the 80x24 minimum, six on a large terminal.

Words falling move in half rows rather than whole ones. A slow word moving a full row at a time sits still for over a second and then jumps, which reads as a slideshow.

### 2.13 Not built yet

Two things in this section are design intent rather than shipped behaviour:

- **Word archetypes.** The plan is that each spawn draws a speed and length pair: DRIFTER at 0.5x speed and long, NORMAL at 1.0x, RUNNER at 1.8x and short, in a 25/55/20 split. Long-and-slow against short-and-fast guarantees overtakes, which is what gives the collision rule something to do. Nothing in the engine reads it today: every word falls at the schedule speed, and per-word differences come only from chipping and collisions.
- **Spawn fairness gate.** Nothing currently stops the schedule spawning a word that cannot be typed in the time it has. A hard level and an impossible word are different things.

## 3. Front end: story, menu and audio

Everything before and after the falling words. It comes after the mechanics are right, not before, but the game has an identity now and the identity is worth building.

### 3.1 The premise

> Dr Samuel Johnson's zombified corpse has risen from Westminster Abbey, in horror at the decline of literacy.
>
> Tired of life, he has grown weary of London, and means to bury it under an onslaught of verbiage.
>
> The boffins at the Ministry of Defence have consulted Lambeth Palace. There is only one course of action left to a desperate nation: raise the ghost of Admiral Lord Nelson, to stand once more in defence of these islands, and of the ancient and noble rights of the traders of the City of London.

Draft copy, to be argued with. It is doing three jobs: it explains why words are falling, it makes the player Nelson rather than a cursor, and it sets a register (mock-heroic, faintly absurd, straight-faced) that everything else can be written in.

Three details are load-bearing and should survive rewriting. **"Tired of life, weary of London"** inverts Johnson's own line, which is the joke that makes him the villain rather than a random monster. **The lexicographer is the enemy** and words are his weapon, so the mechanic and the fiction are the same thing. **The MoD consulting Lambeth Palace** is the tonal keystone: bureaucratic absurdity delivered deadpan.

### 3.2 The opening sequence

An animated title card sequence, roughly 20 seconds, drawn in the block font already in the game.

Broadly: cut between a handful of composed frames rather than animating continuously. A terminal at 30 FPS can do far more, but a sequence of held tableaux with a few moving elements reads as deliberate, whereas continuous motion in text reads as a screensaver. Suggested beats:

1. Westminster Abbey in silhouette, a grave, a hand. Title in TierHuge.
2. Johnson is named, and his plate appears.
3. Words begin falling behind him, a few at first.
4. The MoD and Lambeth Palace lines, deadpan, as plain text.
5. Nelson's column rises, and the sequence hands over to the menu (3.6).

**Skipping is not an afterthought.** Any key skips to the menu, from the very first frame, and the skip hint is on screen throughout.

It runs once per launch, before the menu, and that is the whole of the rule. A player who dies and takes the menu's PLAY again does not see it a second time, so there is nothing to remember and nothing to persist. An earlier draft proposed recording that the intro had been seen and skipping it thereafter, which was solving a problem that does not arise: it would have made first-launch behaviour depend on what was on disk, which is worse for demoing and testing than seeing a skippable title card once a session.

`--no-intro` exists for development, where the game is launched fifty times a day. That is a developer's problem and a flag is the right size of fix for it.

Implementation: a slice of `{duration, draw func}` steps run by the same fixed-timestep loop as the game, against the same cell buffer. No new machinery, and the intro gets shake, particles and the block font for free.

### 3.3 The intro plates

Four of the five intro beats are plates: a picture with its words beside or under it. The fifth is the exception on purpose.

**Sources.** Westminster Abbey (Bluck aquatint, Yale, CC0), Samuel Johnson (Reynolds, Tate, PD), Lambeth Palace (Wale drawing, Yale, CC0), Nelson (RMG, PD). Provenance and rights are tabulated in `assets/art/README.md`.

**Composition.** Each cell is an upper-half block: foreground is the top pixel, background the bottom, which doubles the vertical resolution for the price of owning the cell's background. Wide art - anything past 1.6:1, which in practice means the two building views - is drawn as a band with its words underneath. The rest sits beside its words, centred against the picture rather than the playfield so the two read as one plate instead of two adjacent things.

Every image ships at three sizes and the intro takes the largest that fits. Sizing for the 80x24 minimum and leaving it there is what made the first portrait look coarse on a terminal with room to spare. Width is set from the crop's aspect, never chosen: a cell is two pixel rows tall and character cells are roughly 2:1, so a pixel is roughly square, and getting this wrong is what made the first portrait look stretched.

**Colour is per subject, not per game.** Stone for the Abbey, purple for Johnson, stone for Lambeth, navy for Nelson, so the sequence moves through three tints rather than sitting in one for fifteen seconds. Each ramp is eight steps that hold their hue the whole way up; interpolating linearly to white greys out the midtones and the picture stops matching anything else on screen.

**The buildings are inverted.** Both prints are dark stone against a bright sky, which on a dark terminal fills the frame with lit cells and reads as a grey wall. Inverted, the stone glows against a night sky - which is also the right hour for a corpse getting out from under the flagstones.

**The reveal is the only motion.** Each plate comes up out of the dark over 0.55s at the start of its beat, by scaling the palette index toward zero rather than blending colours, so every frame of it stays inside the same eight tones. The intro is a sequence of held tableaux; a reveal that settles and holds keeps that character, where continuous movement in text reads as a screensaver.

**Beat three has no picture, deliberately.** It is the one where the words rain down the field, and the words are the picture there. Five plates in a row would flatten the sequence into a slideshow.

**Eight tones sets a colour requirement.** Four merged the wig into the face on the portrait, and the four ANSI greys could not separate them at any threshold - luminance alone cannot do it, the hue is doing the work. Six was enough at the smallest size but banded across the face at the largest. So the plates need 256 colours. Below that, and on any terminal where even the smallest tier will not fit, the beat falls back to its words alone; that is not a degraded plate but the only thing that works there. Johnson's beat is the one with something else to show, and keeps the drawn figure that used to pace above the playfield.

**Johnson no longer appears during play.** He paced a band above the field, which cost three rows and took the maximum playable area from 34 down to 31. Trying a real picture there settled it: three rows at portrait aspect is five cells by three, and each cell holds two pixels vertically, so that is a whole man in five pixels by six - a smudge, strictly worse than a drawn silhouette. The band could never carry an image, so the rows went back to the field and he moved to the beat with room to see him.

**Nothing image-shaped ships in the binary.** `assets/art/generate.py` emits one digit per pixel indexing an eight-colour ramp, about 28 KB of Go source across four images at three sizes each. The game decodes no images at runtime and links no image packages, for the same reason the music is note data rather than recordings.

### 3.4 "England expects"

Between the menu and the first word, hold Nelson's signal, flown under a
first-rate ship of the line:

```
                    (a three-masted ship, bow to the right)

                          ENGLAND EXPECTS
                  THAT EVERY MAN WILL DO HIS DUTY

                            press any key
```

The signal is white rather than the menu's aqua, because aqua is the game's own
chrome and this is the one screen that should read as coming from somewhere
else.

The ship goes above the words: the signal was flown from Victory's masts, so the
eye should reach the order having just been told who is giving it. She is 60
cells by 10, drawn from `assets/ship/generate.py` in the same encoding the
monument uses, and she indexes `rampNavy`, the blues the Nelson plate is drawn
in. A terminal under 256 colours, or one without room for her and the order both,
gets the signal on its own.

This is not only flavour, it fixes a real problem. The clock currently starts the instant the game does, so a player who is not ready loses seconds of a timed run before their hands are on the keys. A ready-gate makes the start of every run identical, which the difficulty schedule already promises.

Any key starts. Not a specific key, because the player is about to be in a mode where every key does something.

### 3.5 Defeat: Trafalgar

When the third life goes, before the score screen:

1. A beat of silence, the field frozen where it stood.
2. Words rain down over a London skyline in silhouette, Nelson's Column among them. The rain is bounded to the sky and never falls across the buildings: two kinds of dense content in the same cells is not two things, it is noise. The skyline is contiguous blocks of varying width, because one column each reads as a bar chart rather than a city.
3. The column falls.
4. `KISS ME, HARDY` in TierHuge.
5. The score screen.

Two seconds, three at the outside. **It must be skippable by any key, and the key that skips it must not also be the key that restarts**, or a player mashing space will blow through their own stats without reading them. Skip goes to the score screen; space from there restarts as it does now.

This directly trades against the instant-restart rule, which says every keypress between death and the next attempt costs attempts. The resolution is that the animation is short, always skippable, and the retry path from the score screen is unchanged at one key. Worth watching in playtesting: if people skip it every time, it is too long.


The game currently drops the player straight into a falling word with no explanation. That is fine while two people are testing it and useless for anyone else: free fire, chip acceleration, chains, bombs and the power-ups are not discoverable by dying.

### 3.6 The menu

Shown on launch, in the same bordered playfield as the game so nothing jumps when it starts.

```
   ▟▙          KEYBOARD WARRIOR          <- title, TierHuge
  ▟██▙
   ██           ▸ PLAY
   ██             HOW TO PLAY
  ▟██▙            QUIT
 ▟████▙
   ██           best 12840   04:31   68 wpm   19 games
   ██
 ▟████▙
```

The best line is empty on a fresh install and says so when the run would be unranked.

**Nelson's Column stands in the left margin.** It is a half-block bitmap like the story plates (3.3), drawn from `assets/monument/generate.py`, and the only one of them with no source photograph: the portrait in `assets/art` is the man, head and shoulders, which at fourteen cells is a smudge, and what the menu wants is the monument. So the shape is a silhouette and the generator derives the shading from it.

Three rules make it decoration rather than furniture:

- **It costs the menu no rows and no columns.** The eighteen playable rows at 80x24 are budgeted exactly, so anything above or below the stack would push the leaderboard off the screen; the margin was the only space already doing nothing. It is drawn only when its width plus a gap fits beside the title.
- **The shaft stretches, and is capped.** The art is three pieces (statue, capital, pedestal) with one repeating pixel row of shaft between them, so one bitmap is not squat at 80x24 and stranded at the top of a 40-row terminal. Letting it take every available row put the statue in the top corner with twenty cells of bare stone under it, which reads as a flagpole: a monument is allowed to be shorter than the room it stands in.
- **It has a silhouette, not a background.** The plates own every cell they cover, which is right for a framed portrait and wrong here: a cell with one lit pixel keeps the terminal's own background, or the column would be a black rectangle on the menu.

Below 256 colours the menu simply goes without it. Unlike the Johnson portrait, which carries a story beat and so has a drawn fallback, this is decoration and the menu reads fine bare.

- **Arrow keys or J/K move, ENTER or SPACE selects.** Typing letters must not select anything: the player is about to be in a mode where every letter does something, and a menu that also reacts to letters teaches the wrong reflex on the very first screen.
- **ENTER on PLAY starts immediately.** No countdown. The first word takes about ten seconds to arrive anyway, which is the countdown.
- **ESC quits from the menu, and returns to it from the guide.**
- The game over screen gains a third option beside SPACE to retry: ESC goes back to the menu rather than quitting outright.

### 3.7 The guide

Paged, because it does not fit one 80x24 screen and scrolling text in a terminal game is worse than pages. SPACE or the arrow keys page, ESC returns. Five pages.

**Laid out as a reference, not as prose.** Every page is a left column of keys, marks, multipliers or colours with the explanation beside it, under section headings. The column is measured across all five pages rather than per page, so turning a page does not shift the text sideways, and a player who opens the guide to check one thing scans a column instead of reading four paragraphs to find out whether the answer is in them.

**Page 1, HOW TO PLAY.** One line of premise, so skipping the intro costs nothing. Then the loop: type the words before they reach the floor, three lives, one lost per landing. A CONTROLS table — letters and digits, the four power-up marks, ESC, and SPACE to start the next run from the game over screen. Then the one rule that is not obvious and must be said explicitly: **there is no selected word.** Any key takes a letter off whichever word wants it and is furthest down. Shown as a diagram rather than a sentence: three words all starting with S, and what three presses of S do.

**Page 2, SCORING.** A letter is worth the depth it died at; finishing a word pays its length times that depth times the chain. The depth table is drawn with each multiplier in the colour the field itself uses for it, so the page is a key to the screen rather than four numbers to hold beside it. Then a worked example, because the formula is three multiplications and nobody does those under pressure: a six-letter word finished on the bottom row as the third of a chain.

**Page 3, CHAINS.** The 1.5 second window, the multiplier table out to x20, and the survival rewards at four, six, eight and ten. Then the tactic said plainly, because it is the whole skill of the game: **chip several words down to their last letter, then fire those letters back to back.**

**Page 4, POWER-UPS AND BOMBS.** The suffix table (four marks, 6.2), and the two rules that make them safe to learn by playing:

- A power-up that hits the floor costs nothing. Take it or leave it.
- A bomb never takes priority: it only ever receives a letter that no safe word wants. You cannot be forced into one, and ignoring one is free.

**Page 5, READING THE FIELD.** What speeds the field up — chip acceleration, typo heat, momentum transfer — and then what the colours are telling you: green for a loaded word, red brackets for a bomb, pulsing for about to land, magenta for a power-up, and the bright leading letter for what the next key takes. This page exists because all of that information is already on the screen and none of it is labelled there.

### 3.8 What the guide is for

Two jobs, and they are not the same job.

**The new player's job.** Within a minute, know the three things you cannot work out by dying: that there is no selected word, that chaining is where the points are, and that bombs cannot trap you. Pages 1, 3 and 4 carry those, in that order, and each says its rule outright rather than leaving it to be inferred.

**The returning player's job.** Check one fact and get back to the game: what does `-` do, what is a seven-chain worth, why is that word green. That is a lookup, and it is the reason the pages are tables with a key column. An essay answers the first question and fails the second, which is what the guide used to be.

A first-run hint is the cheaper half of the first job: on a player's first ever game, show one line in the status row for the first fifteen seconds saying that any letter hits the lowest word wanting it. Most players will never open the guide.

### 3.9 Audio

The one part of this with an architectural cost, so the options are set out honestly.

**The constraint.** The binary is built `CGO_ENABLED=0`, which is what makes five-platform cross-compilation a single command (8.9). Every Go audio library that opens a device directly (`oto`, and therefore `beep`) needs cgo on macOS for CoreAudio and on Linux for ALSA. Adopting one means giving up trivial cross-compilation and per-platform build complexity. That is a real price for sound effects.

**Implemented by shelling out, off by default.** The latency question was measured before committing: spawning `afplay` costs 1-2ms typically and spikes to about 30ms under sustained fire. Thirty is a whole frame, so effects are queued to their own goroutine rather than spawned on the loop that draws them. What was *not* measured is audible latency, the gap between `Start()` returning and the speaker moving, which needs loopback capture rather than a Go timer; if that turns out too slow, the fallback is still cgo and the cost is still the one-command build.

- **Music:** one long-lived subprocess playing a looping file (`afplay` on macOS, `paplay` or `aplay` on Linux). Started once, killed on exit, restarted on track change. Latency does not matter for music.
- **Sound effects:** latency does, and it is the open question. Spawning a process per effect costs roughly 10 to 30 ms plus process churn, and a letter explosion needs to land under about 50 ms to feel connected to the keypress. **Measure this before committing.** If per-effect spawning is too slow or too noisy, the fallback is a small helper process kept alive and fed filenames on stdin.
- If neither is good enough, the honest answer is that sound effects need cgo, and that is a decision to take deliberately rather than drift into.

**Generated, not embedded.** Confirmed in practice: the whole soundtrack and every effect render to 2.2 MB of WAV in a temp directory at startup, from a few hundred bytes of note data in the source. Startup cost is 47ms, paid once. Nothing ships in the binary, which stayed at 4.1 MB.

**Generate the audio rather than embedding recordings.** Store note data (pitch, duration, channel) and synthesise square, triangle and noise waves into PCM at startup, writing WAVs to a temp directory for the player process. Note data for a whole soundtrack is a few kilobytes against several megabytes of audio files, it is genuinely 8-bit rather than an imitation of it, and it means tempo and instrumentation are tunable in the same live-reloaded way as everything else.

**The music.** All safely out of copyright, and synthesising from note data creates no performance rights question either:

| Piece | Where |
|---|---|
| Purcell, *Rondeau* from Abdelazer | Menu |
| Handel, *Zadok the Priest* | The intro's Nelson beat |
| Handel, *Music for the Royal Fireworks* (La Réjouissance) | In play, later levels |
| *Heart of Oak* | In play, early levels |
| Purcell, *Dido's Lament* | Defeat |

Four of the five tunes are no longer transcriptions. Each is a line taken from a reference MIDI in `assets/midi/` by `assets/midi/generate.py`, at that file's own crotchet: the Rondeau's twelve-bar refrain at 326ms, Zadok's opening arpeggios at 833ms, the Rejouissance oboe line at 550ms, and Dido's ground bass at 1017ms. Rests come through as `R`, which `Pitch` already returns 0 for because `R` is not a note name. A trailing dot on the denominator is a dotted note, `D4/4.`, which the MIDI extraction needs and which rounding would otherwise lose.

The extractor is the same shape as `assets/art/generate.py`: it prints Go source to paste in, rather than being wired into the build. It flattens a chordal track to one line, quantises to a grid, and folds away the short gaps sequencers leave between detached notes. It cannot decide enharmonic spelling, since a MIDI file records pitch and not notation, so Dido's ground came out spelled `Gb3` and was corrected to `F#3` by hand. That sequence is also in B flat minor, presumably to suit a singer, so it was transposed back to G minor.

*Heart of Oak* is still from memory rather than from a score: recognisable rather than authoritative, and worth correcting against the printed music the same way. No free MIDI of it turned up. The pieces are centuries out of copyright and synthesising from note data raises no performance right, so only accuracy is at stake. The MIDI files themselves are third-party sequences and reference material only; nothing reads them at runtime, and `assets/midi/SOURCES.md` records where each came from.

Dido's Lament under the Trafalgar sequence is the joke worth keeping: it is the most funereal thing in English music and it is about a queen dying while a fleet leaves.

**Sound effects.**

| Event | Character |
|---|---|
| Letter destroyed | Short blip, pitch rising with the depth multiplier |
| Word completed | Chord, rising with the chain tier |
| Chain tier reached | Distinct arpeggio per tier |
| Typo | Dull thud, quiet: it is already punished twice |
| Life lost | Cannon |
| Power-up collected | One motif per effect, so it is identifiable without reading |
| Bomb completed | Explosion, and the only genuinely unpleasant sound in the game |
| SWARM | Drum roll |
| Level up | Bosun's whistle |

Pitching the letter blip to the depth multiplier is the one that earns its place: it makes the score dial audible, so a player learns that deeper is worth more without reading a table.

**All of it on by default**, switchable from the menu and from `-sound=false` and `-music=false`, with volume in `tuning.conf`. It was off by default at first, on the grounds that a terminal game which makes noise unasked is one people close. That was the wrong call: the audio carries information the screen does not, most players never found the switch, and the fix for someone who wants quiet is one keystroke on the menu, remembered from then on.

## 4. Screen contract

- **Minimum 80x24.** Below it the game draws a live-updating "RESIZE TO 80x24" card and resumes when the terminal is big enough.
- **Maximum playfield 120x40.** Bigger terminals get a centred bordered playfield with the excess dark. Capping is the point: difficulty must be comparable across displays.
- Playfield size is fixed at game start. A mid-game resize pauses and asks for the size back.

Layout for a playfield of W x H:

```
+----------------------------------------------------------+  row 0    border
| SCORE 12840  x3  CHAIN x5   WPM 68   ███░ LVL 7   01:53  |  row 1    HUD
+----------------------------------------------------------+  row 2    border
|                                                          |
|   falling words                                          |  rows 3..H-4
|                                                          |
+####==========####==============================+ damaged     row H-3   floor
|  FREEZE 2.4s                                             |  row H-2  status
+----------------------------------------------------------+  row H-1  border
```

Playable rows are `H - 6`: 18 at minimum size, 34 at maximum.

**Colour.** 256-colour, with 16-colour and monochrome fallbacks. Words colour by depth zone, which doubles as the multiplier readout: cyan x1, yellow x2, orange x3, white-hot x5. `NO_COLOR` and `--no-color` respected.

## 5. Rendering

### 5.1 Frame model

Fixed timestep at 30 FPS. Each tick: drain input, step state, compose a cell buffer, diff against the previous buffer, write only changed cells. Full redraws only on resize or scene change.

Measured cost of the worst case: 0.290 ms of a 33.3 ms budget. See 7.1.

### 5.2 Block font

- One 5x6 pixel glyph table covering A-Z, 0-9, `!?+*~` and space. See 2.1 for why there is one and not three.
- Rasterising goes through a half-row bitmap: expand each pixel row to the tier's half-row count, shift by the half-row offset, then pack pairs of half rows into `█ ▀ ▄` or blank. One code path serves square pixels, the huge title tier, and smooth motion.
- 1 blank column of kerning. Word width is `6n - 1` in both tiers.
- Every drawn character (`█ ▀ ▄ ▓ ▒ ░ · ▔`) is Unicode East Asian **Ambiguous** and measures two cells under a CJK width setting. Startup forces it to one; see 7.4. The HUD lives indicator deliberately uses `▮ ▯` instead of blocks so that "no glyph ink outside the playfield" stays a testable property.

### 5.3 Explosions

- A typed letter's glyph box becomes a particle burst: 8-14 particles with position, velocity and a 200-350 ms lifetime, decaying through the ramp `█ ▓ ▒ ░ · `.
- Particle count and radius scale with the depth multiplier. A x5 kill should be genuinely excessive.
- Word completion bursts across the whole word footprint plus a one-frame white flash. Chained completions add a shockwave ring per chain tier.
- **Destroying a letter never reflows the word.** Remaining glyphs stay exactly where they were, so eye and fingers do not re-track. A half-eaten word reads as chewed from the left, which is the correct mental model.

### 5.4 Juice

- Screen shake on life lost: whole playfield offset 1-2 cells for 6 frames.
- Floor damage that persists for the run.
- Chain counter that grows a tier and colour at each threshold.
- Level-up banner wiping across the middle for 1 s while play continues.
- Pulse on any word one letter from death.
- Momentum transfer flashes both words for 2 frames so the cause is visible.
- **Sound, on by default, off with `-sound=false` or from the menu.** Silence hurts a game built on impact. Terminal bell is unusable, but a long-lived `afplay`/`paplay` subprocess handles sparse events (word completion, life lost, level up, SWARM) fine. Not per-letter; the process churn would show.

### 5.5 Making free fire readable

Without a lock the player tracks up to six partial words and an invisible priority order. These three cues carry that load. They are requirements, not polish, and they ship in M1. Free fire is unfair without them.

**Next-letter highlight.** The first untyped glyph of every word renders bright white while the rest of the word takes its depth colour. The whole field's live front edge, visible at a glance.

**Contested-letter marker.** When two or more words share a next letter, the one that will receive the keypress gets a marker under its highlighted glyph. This is the cue that replaces the lock. The player never guesses who receives a key.

**Speed trail.** Each word drags dim block characters above it, length proportional to current speed: 1 row for a DRIFTER, 4 for a RUNNER, growing as the word is chipped. Overtakes and priority flips become visible a second before they happen, and chip acceleration becomes something you can watch rather than something that happens to you.

## 6. Power-ups

### 6.1 Suffixes are typed, not signage

An earlier draft made power-up markers **display only**, on the reasoning that `!` costs a shift and reaching for shift mid-chain breaks the flow the scoring is built to reward. That reasoning was right about `!` and wrong about punctuation in general: **most punctuation is unshifted on a Latin layout.** `.` `-` `/` are each one ordinary keystroke.

So the marker is typed after all. `FREEZE.` is seven keystrokes with no reach, and finishing it feels like landing something rather than collecting a token.

Two properties fall out for free:

- **Punctuation appears only on power-up words**, so pressing `.` can never be stolen by an ordinary word. Under free fire that means a power-up suffix is unambiguous by construction.
- The suffix sits at the **end**, so a power-up must be earned by completing the word. A prefix would force the player to commit before they had read it, and a mistyped prefix would waste it.

Layout caveat: which marks are unshifted varies (`.` is shifted on AZERTY, `/` on QWERTZ). The game matches on the resulting rune, so any layout that can produce the character works.

### 6.2 The set

Nine of these shipped first. Nine was too many, and the count was not the whole problem: six of the marks said nothing about what they did. Nobody looks at `[` and thinks "strips a letter off every word", so the player was memorising a table between runs instead of reading the field, and the two most obscure effects were also the two hardest to see happen.

Cut to four, chosen so the mark can be guessed at:

| Suffix | Name | Effect | Why this mark |
|---|---|---|---|
| `.` | FREEZE | Everything stops for 3 s | A full stop stops things |
| `-` | REWIND | Every word pushed up 4 rows | Minus, moving back |
| `/` | BLAST | Every word near where it went off destroyed, no letter score, combo kept | Slash through what is in reach |
| `+` | REPAIR | +1 life, capped at 5 | Plus one |

What went, and why:

- **STEADY** (`;`, chip acceleration off) — the effect was the absence of an effect. Chip acceleration not happening is not something a player can see happening, so the most mechanically interesting power-up in the set was also the one nobody could tell had fired.
- **AUTOCHIP** (`[`) — ate a letter off every word and sped them all up. The double edge was the design and the problem: a reward that cannot be told apart from a penalty teaches nothing.
- **CLEAR** (`/`) — BLAST does the same job with a radius, and the radius makes *where* the word is finished matter. Two field-wipes was one too many, so the rarer and less interesting one went and BLAST took its key.
- **SLOW** (`,`) — the same axis as FREEZE, and `,` is near enough identical to `.` at block-glyph scale that misreading one is a wrong keypress, which is how a bomb gets fed.
- **SHIELD** (`'`) — the same axis as REPAIR, and it fired silently.

SLOW and SHIELD survive as **chain milestones** (2.5), which is where they belonged: both are quiet effects that suit being earned rather than found.

**`+` is the one shifted mark**, against the rule in 6.1, and is allowed to be. REPAIR is the rarest of the four and the only one the player reaches for while on low lives rather than mid-chain, so a beat spent on shift costs nothing. The alternative was `=`, which is unshifted and means nothing.

**BLAST grants no letter score** for the same reason the score section refuses to reward farming: a button that pays out is a button you press mindlessly.

### 6.3 Weird ones worth trying

Ideas that use mechanics already in the game rather than bolting on new ones. Not committed, in rough order of promise:

- **VOWEL BOMB** — destroys every vowel on the field. Sounds purely good, but chip acceleration means it also speeds up every word it touches. A gift that makes the next ten seconds harder is more interesting than a gift.
- ~~**AUTOCHIP**~~ — built, shipped, cut. See 6.2: the double edge made it unreadable as a reward.
- **ROLLBACK** — winds the difficulty schedule back two levels for 30 s. The schedule is a clock (2.7), so this is the only thing in the game that can touch it, which makes it feel like cheating in a good way.
- **REVEAL** — shows the next three words queued to spawn. An information power-up rather than a mechanical one, and it fits a game that wants to teach vocabulary.
- **GRAVITY** — for 6 s, completing a word pulls every other word up by a row. Turns a chain attempt into a way of buying survival, coupling the two systems.
- **MIRROR** — for 5 s, targeting priority inverts to the *highest* word. Attacks the one rule the player has internalised. Probably infuriating; worth one playtest to find out.

### 6.4 Rules

- One power-up word on screen at a time, and never simultaneously with a bomb (2.8). Both are exceptions to ordinary play and two at once is unreadable.
- From level 3, roughly one spawn in eight.
- A power-up word that reaches the floor **costs nothing**. It lands harmlessly and disappears, exactly like a bomb (2.8).

  This is the rule that makes taking one a decision. If missing a power-up cost a life, the player would be obliged to take every one, and since power-ups ride the longest words on the field that is a punishment dressed as a reward. Free to ignore, it becomes a genuine choice: spend the time and the exposure, or let it go. Across the whole game the principle is the same, that **special words are optional** and only ordinary words can hurt you.
- Effects do not stack with themselves. A second FREEZE refreshes the timer rather than adding to it.
- Active effects show as named countdowns in the status row, because an effect the player cannot see the end of cannot be planned around.
- **Instant effects name themselves too**, for two seconds after firing. REWIND, BLAST and REPAIR have no duration and so had no countdown, which meant they changed the field and never said what they were: the only way to learn a mark was to press it and infer from the wreckage.
- **A power-up word carries its name above it**, with the mark to press: `BLAST /`. Nothing on screen previously connected a mark to an effect, which left the marks carrying the whole explanation on their own. With the name on the word there is nothing to memorise between runs, and that is what makes four a set worth having rather than a table to revise.
- Rendered in magenta with the suffix pulsing, so loot reads as loot at a glance.

**Tuning knobs:** `powerup_from_level`, `powerup_chance`, and one duration per effect.

## 7. Words

The word list decides more of the fun than any mechanic in this document, and a random dictionary dump wastes it.

**Theme it.** The game is called KEYBOARD WARRIOR. Ship the internet-argument pack: ACTUALLY, SOURCE, CITATION, STRAWMAN, PEDANT, DOWNVOTE, TOUCHE, RATIO, CIRCULAR, ANECDOTE, WHATABOUT, ERRATUM. The joke does the work of a lot of art, and it makes the game specifically this game rather than generically a typing game. Packs are plain text files, one per line, selectable with `--words NAME`.

**Weight by typing difficulty.** *(done)* Each word is scored on same-finger repeats, same-hand runs and reaches off the home row, and selection biases toward awkward words as the run goes on.

Length is a crude proxy and wrong in both directions: MINIMUM is eleven letters and rolls off the hand, POLKA is five and asks one finger to do two jobs in a row. Weighting by awkwardness raises difficulty without making words longer, so the late game feels different rather than merely more. Measured on the real list, the extremes are SLEIGH and ELFISH at one end against RECEDE and UNHEEDED at the other, which is the right answer.

Selection samples a few candidates and keeps the closest match to the target rather than taking the hardest available, or the late game would be a short rotation of the same tongue-twisters.

**It assumes QWERTY**, so the scores are simply wrong on Dvorak or Colemak. `typing_difficulty = 0` turns the bias off and restores uniform selection.

## 8. Technical approach

### 8.1 Language: Go, `tcell/v2`

A shippable binary is a requirement, so this is an architectural decision, not an M5 concern. All three candidates were built and measured on the same worst-case frame (120x40 buffer, 6 words of 11 letters at 5x5 glyphs, 400 particles) through each library's real cell-buffer and diffing path.

| | Python 3.13 + curses | Rust + ratatui | **Go + tcell** |
|---|---|---|---|
| Frame p50 / p99 | 0.68 / 0.77 ms | 0.091 / 0.128 ms | 0.290 / 0.561 ms |
| Share of 33.3 ms budget | 2.0% | 0.27% | 0.87% |
| Rebuild after editing a constant | 0 s | 0.26-0.43 s | **0.07 s** |
| Cold build | n/a | 11.6 s | **3.3 s** |
| Stripped binary | 10-20 MB (PyInstaller) | 429 KB | 1.9 MB |
| Five platforms from this laptop | impossible | needs Docker (`cross`) or `cargo-zigbuild` | **13 s, one command, no extra toolchain** |
| GC | n/a | none | 17 cycles per 20 s of play, 1.6 ms total |

None of the three is near the frame budget, so frame time decides nothing. The trade is iteration speed against distribution, and Go wins both:

**Distribution.** `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build` produces a statically linked 1.9 MB binary in about 3 seconds. Five targets (darwin/arm64, darwin/amd64, linux/amd64, linux/arm64, windows/amd64) built from this machine in 13 seconds total, no Docker, no cross toolchain, no CI matrix required. Rust needs `cross` or `cargo-zigbuild` for the same thing. Since a binary is the stated requirement, this is the axis that matters and it is not close.

**Iteration.** 0.07 s rebuilds, 4-6x faster than Rust's incremental and effectively instant. With hot-reloaded tunables (7.4), most tuning needs no rebuild at all.

Rust's remaining wins are 1.5 MB of disk and 0.2 ms of frame time. Neither is worth the cross-compilation friction.

**GC was measured, not assumed:** 17 cycles across 600 frames with 1.612 ms of total pause, so about 0.09 ms per cycle. The worst single frame in the run, GC pause included, was 0.709 ms against a 33.3 ms budget. Keep it that way by reusing slices in place (filter particles with `parts[:0]`) rather than allocating per frame.

**Use `tcell` directly.** Not `bubbletea` or `lipgloss`: they are built for forms, lists and string-diffed views, and this is a 30 FPS particle renderer. `tcell` is a cell buffer with diffing, key events, resize handling and terminfo, which is exactly the surface this game needs.

**The one real cost of Go here** is the absence of sum types. Scenes, power-up variants and word archetypes all want tagged unions and will instead be a typed-constant `kind` field plus a `switch`. That is a mild annoyance at this scale, not a design problem, but it means the compiler will not check exhaustiveness. Add a default case that panics in every such switch so a missing variant fails loudly in testing rather than silently in play.

### 8.2 Package layout

```
keyboardwarrior/
  go.mod
  tuning.conf                 # every tunable, hot-reloaded
  main.go                     # flags, terminal setup, panic recovery, teardown
  internal/
    app/                      # scenes: title, play, pause, gameover
    engine/                   # no tcell import anywhere below here
      state.go                # State, Word; Step(*State, dt, []Key)
      targeting.go            # the free fire rule
      physics.go              # falling, chip acceleration, momentum transfer
      spawner.go              # archetypes, fairness gate, pressure, waves, SWARM
      scoring.go              # depth, chain, combo, wpm
      powerups.go             # v2
      rng.go                  # xorshift64*, deterministic and version-stable
      tuning.go               # tunables struct, parser, mtime watch
    render/
      frame.go                # Compose(*State) into the tcell screen
      blockfont.go            # three glyph tiers, half-block offsets, measurement
      effects.go              # particles, flashes, banners, trails
      hud.go                  # score, clock, wpm, floor damage
      cues.go                 # 4.5, the three readability cues
    words/                    # //go:embed *.txt
  testdata/
```

`internal/engine` must not import `tcell`. Enforce it with a test rather than a convention:

```go
out, _ := exec.Command("go", "list", "-deps", "./internal/engine").Output()
if strings.Contains(string(out), "gdamore/tcell") { t.Fatal("engine depends on tcell") }
```

### 8.3 Loop

```go
const dt = float32(1) / 30
target := time.Second / 30
next := time.Now()
for {
    keys := drainInput(scr)          // PollEvent via a goroutine + buffered channel, drained non-blocking
    engine.Step(&state, dt, keys)    // dt is a const, never measured
    render.Compose(scr, &state)
    scr.Show()                       // tcell diffs and writes only changed cells
    next = next.Add(target)
    if d := time.Until(next); d > 0 { time.Sleep(d) }
}
```

Fixed dt, not measured dt: determinism makes runs reproducible, testable and comparable under `--daily`. Overrunning frames are dropped rather than accumulated; a typing game needs no catch-up physics.

`tcell.PollEvent` blocks, so run it in one goroutine feeding a buffered channel, and drain that channel non-blocking each tick. Every key must be processed in order, because a fast typist landing three keys inside 33 ms must get all three. Size the channel at 64 and never drop from it.

### 8.4 Details worth getting right early

- **East Asian ambiguous width. This is a real bug, found by measurement.** `█ ▀ ▄ ▓ ▒ · ▔` are all Unicode East Asian *Ambiguous*, which means `go-runewidth` reports them as width **2** when `EastAsianWidth` is on, as it is under `LANG=ja_JP.UTF-8`, `zh_CN` or `ko_KR`. Every word would render double-width, the playfield geometry would be wrong and the half-block motion in 2.12 would break. Force `runewidth.DefaultCondition.EastAsianWidth = false` at startup regardless of locale, and add a test asserting all seven glyphs measure 1. Only `░` (U+2591) is unambiguous, and it is too light to build words from. This is a Unicode property, not a Go one; it would have shipped in any language.
- **Panic recovery, before anything else.** A panic while the screen is live leaves the user with a wrecked shell and no visible error. `defer func(){ scr.Fini(); if r := recover(); r != nil { panic(r) } }()` in main, so the terminal is restored before the trace prints. Same teardown on normal exit and on Ctrl-C.
- **Hot-reloaded tunables.** `tuning.conf` holds every constant in this document: `CHIP_ACCEL`, chain tiers, depth multipliers, archetype shares, pressure bands, particle counts. Poll its mtime every 30 frames and reparse on change, so tuning needs no rebuild at all. About 40 `key = value` float lines, so hand-parse it in ~30 lines with `bufio` and `strconv`.
- **Own the PRNG.** A six-line xorshift64\*, seeded explicitly. Not `math/rand`: `--daily` and `--replay` need identical word sequences across versions and platforms, and the standard library gives no cross-version sequence guarantee.
- **Keys.** `tcell.KeyRune` uppercased, so caps lock and shift both work. Ctrl-C quits, Esc pauses, Ctrl-L forces a full redraw.
- **Instant restart.** SPACE on the game over screen starts the next run in under 100 ms, no menu, stats visible behind the first spawn. Every keypress of friction between death and the next attempt costs attempts.
- **Resize.** `*tcell.EventResize` arrives through the same channel; below minimum, switch scene.
- **Allocation discipline.** Reuse the particle slice in place and keep the glyph tables as fixed-size arrays. The GC measurement in 7.1 holds only if the frame path does not allocate.
- **Colour.** 256-colour confirmed available on the target terminals. Degrade to 16 and then to none on `NO_COLOR` or a dumb `TERM`.
- **Embed the word lists** with `//go:embed`, so one file is the whole game. `--words FILE` still reads from disk for custom packs.

### 8.5 Balance targets

State the intent, not just the constants, so tuning has something to aim at:

- A fully pre-chewed field of five words should run at roughly 1.5-1.7x baseline speed. That prices `CHIP_ACCEL`.
- A five-word chain should pay roughly 2.5x what clearing the same five words sequentially pays. That prices the chain tiers against the risk.
- A x5 depth kill should pay roughly 3x a x1 kill of the same word, and feel like a bad idea about a third of the time.
- Level 1 must give a beginner 15 seconds of reaction time on the first word.
- A 90 WPM player should never see an empty screen after level 3.

### 8.6 Testing

`internal/engine` has no terminal dependency and does no I/O. `Step(*State, dt, []Key)` makes the whole game headless-testable, and `go test ./...` runs it in milliseconds.

- Unit: keypress resolution, the full case table. No candidate, one, several ordered by row, exact ties, last-letter, six words sharing a letter under six presses.
- Unit: momentum transfer. Boxes touching, boxes nested, three words in a stack, transfer while the middle word is destroyed mid-tick.
- Unit: chip acceleration compounding, chain windows at the 1.5 s boundary, depth multiplier zone edges.
- Unit: the fairness gate never admits an untypeable word at any level.
- Fuzz (`go test -fuzz`) and table-driven randomised runs: 10,000 ticks with random keys. Lives never negative, score monotonic, no word outside the playfield, no two word boxes ever overlapping, typed prefix never exceeds length, one key never consumes two letters.
- Golden files under `testdata/`: fixed states rendered with `tcell.NewSimulationScreen`, dumped to a text grid and compared. `SimulationScreen` makes the whole renderer testable without a terminal, which is the main reason to prefer tcell's own buffer over a hand-rolled one.
- Unit: **every glyph measures width 1** under both `EastAsianWidth` settings after startup forces it off. This is the 7.4 bug; it needs a test, not a comment.
- Unit: the PRNG produces a fixed known sequence from a fixed seed. Guards `--daily` and `--replay` against a refactor silently changing every run.
- Unit: `go list -deps ./internal/engine` contains no `tcell`.
- `--seed N` and `--replay FILE` make every bug reproducible.

### 8.7 Tuning harness, day one

All present: `--level N` (moves the clock rather than faking a level, so everything derived from it stays consistent), `--speed X`, `--god`, `--words NAME`, `--font braille|blocks` (braille by default), `--word TEXT`, `--seed N`, `--daily` (seeded from the date, so scores are comparable and colleagues have something to argue about), `--tuning PATH`, `--debug` (frame time, level as a real number, effective speed and typo heat, per-word speed and depth multiplier, live particle count, tuning reload errors).

Still to add: `--replay FILE`.

### 8.8 Milestones

**Done.**

- **M0** terminal setup with panic recovery, forced non-ambiguous glyph width, the size gate, the proportional pixel font, two drawing styles, sub-cell motion, `tuning.conf` hot-reload.
- **M1** free fire, per-letter explosions, lives, floor damage, the impact shockwave, game over with instant restart, and the readability cues from 5.5.
- **M2** chip acceleration, collision with momentum transfer, lockstep motion on a shared clock, non-overlapping spawn placement, HUD with clock and WPM.
- **M3** depth and chain scoring, the particle and shake pass, typo heat, breath pauses and SWARM.
- **M4** power-ups on typed punctuation suffixes with weighted rarity (nine at first, cut to four; see 6.2), bombs with last-resort targeting, the continuous clock-driven difficulty schedule, the 47k defined-word English pack and the definition panel, and the day-one flags.

**Next, in the order that adds the most.**

- **M5 (done)** chain legibility and chain rewards: loaded words marked, the window drawn with its current and next value, COMBO renamed to STREAK, tiers extended to x20, and the four survival-paying milestones.
- **M6 (done)** the menu, the five-page guide, the "England expects" ready gate and the first-run hint. Escape now goes back rather than out: to the menu from a run, out only from the menu itself.
- **M7 (done)** personal bests in `~/.keyboardwarrior/stats.json`, shown on the menu and called out on the score screen. The call-out is one line, `NEW HIGH SCORE`, and only for the score: it used to list every best a run beat - time, chain, streak - which buried the one that matters among ones that do not. The rest are still tracked and still on the stats screen. The menu's board carries rank, score and time; the date each run was set was dropped, since when a run happened says nothing about where it places. Written atomically; a missing or corrupt file is never fatal. Runs made with `--god`, `--speed`, `--level` or `--word` are unranked and never touch it. Closes the "one more go" loop: a score with nothing to beat is just a number.
- **M8 (done)** typing-difficulty weighting, `--record` and `--replay`, the level-up banner, a pulse on words about to land, and a flash on both words when one kicks another.
- **M9 (done)** the intro sequence in five beats, Johnson in his own band above the field, the Trafalgar defeat sequence. The ready gate landed early with M6. The band was later removed and its rows returned to the playfield; Johnson is a portrait plate in the intro now, see 3.3.
- **M10 (done)** audio: five tunes and ten effects synthesised from note data, played through `afplay`, `paplay` or `aplay`, on by default and switchable from the menu or with `-sound=false` and `-music=false`. A machine with no player runs silent rather than failing.
- **M11** release engineering (8.9), README with a recorded GIF.

Every milestone ends at something playable.

### 8.9 Shipping the binary

Five targets, all built on one machine in about 13 seconds:

```sh
for t in darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64; do
  CGO_ENABLED=0 GOOS=${t%%/*} GOARCH=${t##*/} \
    go build -ldflags="-s -w" -o dist/keyboardwarrior-${t%%/*}-${t##*/}
done
```

`CGO_ENABLED=0` is what makes the Linux builds statically linked, so there is no glibc version to worry about on the user's box. `-ldflags="-s -w"` produced the 1.9 MB figure.

Use `goreleaser` for the actual releases. One config file builds every target, writes checksums, cuts the GitHub Release and generates the Homebrew tap formula. It also handles macOS notarisation if a Developer ID ever exists.

**The macOS reality, stated plainly.** An unsigned binary downloaded from a browser is quarantined by Gatekeeper and refuses to run, with a message that explains nothing. Three ways out, in preference order:

1. A Homebrew tap. `brew install` does not quarantine, it is one command, and it is what people expect from a terminal tool. `goreleaser` generates the formula.
2. `go install github.com/.../keyboardwarrior@latest`, for anyone with Go.
3. A direct download plus `xattr -d com.apple.quarantine ./keyboardwarrior` in the README.

Signing and notarising needs an Apple Developer ID at 99 USD a year. Worth it only if this goes to strangers who will not use Homebrew. Linux and Windows have no equivalent problem.
