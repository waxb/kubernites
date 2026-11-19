package sound

import (
	"encoding/binary"
	"io"
	"slices"
	"testing"
)

const testPhraseFrames = 64

func markerPCM(phrases int) []byte {
	out := make([]byte, phrases*testPhraseFrames*frameBytes)
	for p := range phrases {
		for f := range testPhraseFrames {
			at := (p*testPhraseFrames + f) * frameBytes
			v := uint16(int16((p + 1) * 1000))
			binary.LittleEndian.PutUint16(out[at:], v)
			binary.LittleEndian.PutUint16(out[at+2:], v)
		}
	}
	return out
}

func testWalk(twins [][]int) phraseWalk {
	return phraseWalk{phraseBytes: testPhraseFrames * frameBytes, twins: twins, fadeFrames: 8, seed: 7}
}

func readFrames(t *testing.T, r io.Reader, frames int) []int16 {
	t.Helper()
	buf := make([]byte, frames*frameBytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		t.Fatalf("read: %v", err)
	}
	out := make([]int16, frames)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(buf[i*frameBytes:]))
	}
	return out
}

func phraseOf(v int16) int { return int(v)/1000 - 1 }

func TestPhraseGraphLinksPhrasesWithTheSameBass(t *testing.T) {
	x, y, z := times("G2 -- D3 --", 8), times("E2 -- B2 --", 8), times("C2 -- G2 --", 8)
	s := song{bpm: 88, steps: 128, parts: []part{
		{inst: instBass, line: seq(x, y, x, z)},
		{inst: instLute, line: seq(times("G4 A4 B4 D5", 8), times("E4 -- -- --", 24))},
	}}
	got := phraseGraph(s)
	want := [][]int{{2}, nil, {0}, nil}
	for i := range want {
		if !slices.Equal(got[i], want[i]) {
			t.Fatalf("twins = %v, want %v", got, want)
		}
	}
	if len(phraseGraph(song{steps: 100, parts: []part{{inst: instBass, line: "G2"}}})) != 0 {
		t.Fatal("a song that is not whole phrases gets no graph")
	}
}

func TestTheDarkThemeHasTwinsOverTheSameBass(t *testing.T) {
	s := darkSong()
	twins := phraseGraph(s)
	pairs := 0
	for p, list := range twins {
		for _, q := range list {
			pairs++
			if bassSignature(s, p) != bassSignature(s, q) {
				t.Fatalf("phrases %d and %d are linked over different bass lines", p, q)
			}
		}
	}
	if pairs < 10 {
		t.Fatalf("the dark theme has only %d twin links", pairs)
	}
}

func TestTheFirstLapPlaysStraight(t *testing.T) {
	pcm := markerPCM(4)
	r := newJukebox(pcm, testWalk([][]int{{2}, nil, {0}, nil}))
	got := readFrames(t, r, 4*testPhraseFrames)
	for i, v := range got {
		if want := int16((i/testPhraseFrames + 1) * 1000); v != want {
			t.Fatalf("frame %d = %d, want %d on the first lap", i, v, want)
		}
	}
}

func TestNoTwinsMeansAPlainLoop(t *testing.T) {
	pcm := markerPCM(3)
	r := newJukebox(pcm, testWalk(make([][]int, 3)))
	got := readFrames(t, r, 10*3*testPhraseFrames)
	for i, v := range got {
		if want := int16(((i/testPhraseFrames)%3 + 1) * 1000); v != want {
			t.Fatalf("frame %d = %d, want %d", i, v, want)
		}
	}
}

func TestJumpsLandOnTwinsAndNeverReplayThePhraseJustPlayed(t *testing.T) {
	twins := [][]int{{2, 4}, nil, {0, 4}, nil, {0, 2}, nil}
	r := newJukebox(markerPCM(6), testWalk(twins))
	readFrames(t, r, 6*testPhraseFrames)
	prev, jumps := 5, 0
	for range 2000 {
		got := readFrames(t, r, testPhraseFrames)
		cur := phraseOf(got[testPhraseFrames/2])
		natural := (prev + 1) % 6
		if cur != natural {
			jumps++
			if !slices.Contains(twins[natural], cur) || cur == prev {
				t.Fatalf("jumped from %d to %d; the natural next was %d with twins %v", prev, cur, natural, twins[natural])
			}
		}
		prev = cur
	}
	if jumps < 100 {
		t.Fatalf("only %d jumps in 2000 phrases", jumps)
	}
}

func TestAJumpCrossfadesFromTheNaturalNextIntoTheTwin(t *testing.T) {
	twins := [][]int{{2, 4}, nil, {0, 4}, nil, {0, 2}, nil}
	walk := testWalk(twins)
	r := newJukebox(markerPCM(6), walk)
	readFrames(t, r, 6*testPhraseFrames)
	prev, checked := 5, 0
	for range 3000 {
		got := readFrames(t, r, testPhraseFrames)
		cur, natural := phraseOf(got[testPhraseFrames/2]), (prev+1)%6
		if cur != natural {
			from, to := int16((natural+1)*1000), int16((cur+1)*1000)
			if got[0] != from || got[walk.fadeFrames] != to {
				t.Fatalf("jump %d to %d: fade starts at %d and ends at %d, want %d and %d", natural, cur, got[0], got[walk.fadeFrames], from, to)
			}
			for f := 1; f < walk.fadeFrames; f++ {
				if lo, hi := min(from, to), max(from, to); got[f] <= lo || got[f] >= hi {
					t.Fatalf("fade frame %d = %d, not between %d and %d", f, got[f], lo, hi)
				}
			}
			checked++
		}
		prev = cur
	}
	if checked == 0 {
		t.Fatal("no jump to check")
	}
}

func TestTheWalkReachesEveryPhraseAndNeverStalls(t *testing.T) {
	s := darkSong()
	twins := phraseGraph(s)
	r := newJukebox(markerPCM(len(twins)), testWalk(twins))
	seen := map[int]bool{}
	for range 20000 {
		got := readFrames(t, r, testPhraseFrames)
		seen[phraseOf(got[testPhraseFrames/2])] = true
	}
	if len(seen) != len(twins) {
		t.Fatalf("the walk reached %d of %d phrases", len(seen), len(twins))
	}
}

func TestTheManagerPlaysMusicThroughItsPhraseWalk(t *testing.T) {
	out, clk := &fakeOutput{}, newFakeClock()
	m := newManager(out, newBank(minRate), clk.now)
	m.PlayMusic(MusicDark)
	walk := out.loopAt(0).walk
	want := walkFor(darkSong(), minRate)
	if walk.phraseBytes != want.phraseBytes || walk.fadeFrames != want.fadeFrames || len(walk.twins) != 16 {
		t.Fatalf("walk = %+v", walk)
	}
}

func TestStagingsThemeIsLongAndWalks(t *testing.T) {
	s := feudalSong()
	if s.steps != 512 {
		t.Fatalf("staging's theme has %d steps, want 512", s.steps)
	}
	links := 0
	for _, list := range phraseGraph(s) {
		links += len(list)
	}
	if links < 10 {
		t.Fatalf("staging's theme has only %d twin links", links)
	}
}

func TestEveryThemeWalks(t *testing.T) {
	for _, m := range AllMusic() {
		links := 0
		for _, list := range phraseGraph(songBook[m]()) {
			links += len(list)
		}
		if links < 4 {
			t.Errorf("%s has only %d twin links: the jukebox cannot walk it", m, links)
		}
	}
}
