# Nelson: Hero of the Keys

A typing game for the terminal. Words fall from the top of the screen in large block letters. Type a word before it reaches the floor to destroy it. Each word that lands costs a life. You start with three.

The premise: Samuel Johnson has risen from Westminster Abbey to bury London in words, and you play the ghost of Nelson sent to stop him.

## How it works

There is no selected word. Every key you press goes to whichever word on screen wants that letter next. If several words want it, the lowest one gets it. You can type half of one word, switch to another, and come back.

- Letters score more the further a word has fallen: 1, 2, 3, or 5 points in the last two rows. The word's colour shows which band it is in.
- Finishing words within 1.5 seconds of each other builds a chain, which multiplies the completion bonus up to x20. Long chains also slow the field, block a landing, clear words, or give a life.
- Each letter you take off a word makes it fall 12% faster. Typos speed up the whole field for a few seconds.
- Red bracketed words are bombs. Do not finish them. They only receive a letter when no other word wants it, and they cost nothing if they land.
- Words ending in punctuation are power-ups. Type the mark to finish them: `.` freezes the field, `-` pushes words up, `/` destroys nearby words, `+` gives a life.
- Difficulty rises with time only. Two runs of the same length face the same schedule.

The menu has a five-page guide with the full rules. `SPEC.md` has the design in detail.

## Running it

Needs Go 1.25 or later and a terminal of at least 80x24. The intro pictures need 256 colours.

```sh
go run .
```

Or build a binary:

```sh
go build -o nelson .
./nelson
```

Sound uses `afplay` on macOS and `paplay` or `aplay` on Linux. Without one of those the game runs silent.

Controls: letters to type, arrow keys and Enter on the menu, Esc to go back, Space to restart from the game over screen.

### Flags

| Flag | Effect |
|---|---|
| `-words NAME` | Word pack: `english` (default), `argument`, or a path to a `.txt` file |
| `-daily` | Seed from today's date so everyone gets the same run |
| `-seed N` | Fixed random seed |
| `-no-intro` | Skip the opening sequence |
| `-sound=false`, `-music=false` | Turn audio off |
| `-volume X` | Volume from 0 to 1 |
| `-font blocks` | Larger block letters instead of the default braille |
| `-record FILE`, `-replay FILE` | Record or play back a run |
| `-level N`, `-speed X`, `-god`, `-word TEXT` | Testing aids. Runs using these are not saved to your bests |
| `-tuning PATH` | Tunables file, reloaded while the game runs (default `tuning.conf`) |
| `-debug` | Show the tuning overlay |

Personal bests are saved to `~/.nelson/stats.json`.

## Tests

```sh
go test ./...
```

## Licence notes

The English word list includes definitions from WordNet; see `LICENSE-WORDNET`. Image sources are listed in `assets/art/README.md` and MIDI sources in `assets/midi/SOURCES.md`.
