# MIDI sources

Reference MIDI for the tunes in `internal/audio/music.go`, which were originally
transcribed from memory. Four of the five now come from the files here instead.

The music is all out of copyright. The MIDI files are third-party sequences,
so they are reference material, not something to ship: nothing reads them at
runtime. `generate.py` here turns one into note data you paste into `music.go`.

The four tunes currently taken from these files were extracted with:

    python3 assets/midi/generate.py assets/midi/handel-zadok-the-priest.mid \
        --track 8 --grid 16 --limit 40
    python3 assets/midi/generate.py assets/midi/handel-la-rejouissance.mid \
        --track 4 --grid 16 --limit 32
    python3 assets/midi/generate.py assets/midi/purcell-didos-lament.mid \
        --track 3 --voice bottom --flats --transpose 9 --grid 8 --limit 20

Dido's ground then had `Gb3` corrected to `F#3` by hand: a MIDI file records
pitch, not notation, so the script cannot pick the right spelling. The
`--transpose 9` is because that sequence is in B flat minor. The Abdelazer
Rondeau predates the script and came out through
`abdelazer-rondeau-melody.txt`. Heart of Oak has no MIDI here and is still a
transcription from memory.

Downloaded 5 September 2026.

## Caveats

- The Tempest pieces (`purcell-tempest-*`) are now usually attributed to John
  Weldon, not Purcell.
- `purcell-lillibullero.mid` is a tune Purcell arranged rather than wrote.
- `purcell-trumpet-tune.mid` is unlabelled beyond "Trumpet Tune". The famous
  Trumpet Voluntary in D is by Jeremiah Clarke, so check which one this is
  before crediting it.
- Sequencer quality varies. The Mutopia files are LilyPond output from typeset
  scores and are the most reliable. The MidiWorld and ClassicalMIDI files are
  hobbyist sequences, some with added drum kits and octave doublings.
- IMSLP has more (the full Fairy Queen set, Music for a While, the five
  Funeral of Queen Mary movements) but now sits behind a captcha, so those
  need downloading by hand from
  https://imslp.org/wiki/Category:Purcell,_Henry

## Purcell

| File | Source | URL |
| --- | --- | --- |
| `purcell-didos-lament.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music2/2465dido.mid |
| `purcell-fairest-isle.mid` | Mutopia | https://www.mutopiaproject.org/ftp/PurcellH/FairestIsle/FairestIsle.mid |
| `purcell-fairy-queen-while-you-here-do-sleeping-lie.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music4/4684tlwyhdsn.mid |
| `purcell-funeral-queen-mary.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music2/2654mary.mid |
| `purcell-golden-sonata-1.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music3/3793goldn1.mid |
| `purcell-golden-sonata-2.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music3/3792goldn2.mid |
| `purcell-golden-sonata-3.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music3/3791goldn3.mid |
| `purcell-golden-sonata-4.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music3/3790goldn4.mid |
| `purcell-hornpipe-e-minor-harpsichord.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music2/2488tmpchnem.mid |
| `purcell-hornpipe-e-minor.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music2/2489tmpchnsh.mid |
| `purcell-jehova-quam-multi.mid` | Mutopia | https://www.mutopiaproject.org/ftp/PurcellH/Z135/JehovaQuamMulti/JehovaQuamMulti.mid |
| `purcell-lillibullero.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music2/2413purcellero.mid |
| `purcell-minuet-a-minor.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music2/2414purcellminuet.mid |
| `purcell-tempest-ariel-pipe-and-tabor.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music3/3069hpcat2.mid |
| `purcell-tempest-stephanos-songs.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music3/3070hpsteph.mid |
| `purcell-trumpet-tune.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music2/2363trumptune.mid |
| `purcell-what-power-art-thou.mid` | Mutopia | https://www.mutopiaproject.org/ftp/PurcellH/Z628/WhatPowerArtThou/WhatPowerArtThou.mid |
| `purcell-what-shall-i-do.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music3/3071purhwsid.mid |

## Handel

| File | Source | URL |
| --- | --- | --- |
| `handel-air-hwv471.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music1/2271hwv471.mid |
| `handel-arrival-queen-of-sheba.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music2/sheba.mid |
| `handel-aylesford-01-overture.mid` | Mutopia | https://www.mutopiaproject.org/ftp/HandelGF/Aylesford/01-overture/01-overture.mid |
| `handel-aylesford-02-entree.mid` | Mutopia | https://www.mutopiaproject.org/ftp/HandelGF/Aylesford/02-entree/02-entree.mid |
| `handel-aylesford-03-gavotte.mid` | Mutopia | https://www.mutopiaproject.org/ftp/HandelGF/Aylesford/03-gavotte/03-gavotte.mid |
| `handel-aylesford-04-toccata.mid` | Mutopia | https://www.mutopiaproject.org/ftp/HandelGF/Aylesford/04-toccata/04-toccata.mid |
| `handel-aylesford-05-fuga.mid` | Mutopia | https://www.mutopiaproject.org/ftp/HandelGF/Aylesford/05-fuga/05-fuga.mid |
| `handel-aylesford-06-impertinence.mid` | Mutopia | https://www.mutopiaproject.org/ftp/HandelGF/Aylesford/06-impertinence/06-impertinence.mid |
| `handel-aylesford-07-concerto.mid` | Mutopia | https://www.mutopiaproject.org/ftp/HandelGF/Aylesford/07-concerto/07-concerto.mid |
| `handel-aylesford-08-preludio.mid` | Mutopia | https://www.mutopiaproject.org/ftp/HandelGF/Aylesford/08-preludio/08-preludio.mid |
| `handel-aylesford-09-menuet-i.mid` | Mutopia | https://www.mutopiaproject.org/ftp/HandelGF/Aylesford/09-menueti/09-menueti.mid |
| `handel-aylesford-10-menuet-ii.mid` | Mutopia | https://www.mutopiaproject.org/ftp/HandelGF/Aylesford/10-menuetii/10-menuetii.mid |
| `handel-chaconne-g-major.mid` | MidiWorld | https://www.midiworld.com/midis/other/handel/chaconng.mid |
| `handel-chaconne-hwv435-1.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music4/4282hwv435n1.mid |
| `handel-chaconne-hwv435-2.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music4/4281hwv435n2.mid |
| `handel-chaconne-hwv435-3.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music4/4280hwv435n3.mid |
| `handel-concerto-grosso-op6-no6.mid` | MidiWorld | https://www.midiworld.com/midis/other/handel/op6n06gm.mid |
| `handel-hallelujah.mid` | MidiWorld | https://www.midiworld.com/midis/other/handel/hallujah.mid |
| `handel-harmonious-blacksmith.mid` | MidiWorld | https://www.midiworld.com/midis/other/handel/blksmith.mid |
| `handel-la-rejouissance.mid` | MidiWorld | https://www.midiworld.com/midis/other/handel/rejou.mid |
| `handel-lascia-chio-pianga.mid` | MidiWorld | https://www.midiworld.com/midis/other/handel/lascia.mid |
| `handel-let-the-bright-seraphim.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music3/3109letthebrightseraphim.mid |
| `handel-march-for-trumpets.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music1/2241trumpetmarch.mid |
| `handel-ombra-mai-fu.mid` | MidiWorld | https://www.midiworld.com/midis/other/handel/largo.mid |
| `handel-organ-concerto-in-g.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/224Handel.mid |
| `handel-royal-fireworks.mid` | MidiWorld | https://www.midiworld.com/midis/other/handel/gp_fwork.mid |
| `handel-sarabande-d-minor.mid` | MidiWorld | https://www.midiworld.com/midis/other/handel/sarabnde.mid |
| `handel-suite-b-flat-hwv434-1.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music4/4285hwv434n1.mid |
| `handel-suite-b-flat-hwv434-2.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music4/4284hwv434n2.mid |
| `handel-suite-b-flat-hwv434-3.mid` | ClassicalMIDI | https://www.classicalmidi.co.uk/music4/4283hwv434n3.mid |
| `handel-water-music-hornpipe-1.mid` | MidiWorld | https://www.midiworld.com/midis/other/handel/hpipe1.mid |
| `handel-water-music-hornpipe-2.mid` | MidiWorld | https://www.midiworld.com/midis/other/handel/hpipe2.mid |
| `handel-water-music-suite.mid` | MidiWorld | https://www.midiworld.com/midis/other/handel/wtrmusic.mid |
| `handel-zadok-the-priest.mid` | MidiWorld | https://www.midiworld.com/midis/other/handel/zadok1.mid |
