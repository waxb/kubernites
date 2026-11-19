package sound

import (
	"encoding/binary"
	"io"
	"strings"
)

const (
	frameBytes     = 4
	phraseSteps    = 32
	jumpChanceMin  = 0.15
	jumpChanceMax  = 0.5
	jumpChanceStep = 0.07
	jumpFadeSecs   = 0.02
)

type phraseWalk struct {
	phraseBytes int
	twins       [][]int
	fadeFrames  int
	seed        uint32
}

func walkFor(s song, rate int) phraseWalk {
	return phraseWalk{
		phraseBytes: stepFrames(s.bpm, rate) * phraseSteps * frameBytes,
		twins:       phraseGraph(s),
		fadeFrames:  int(float64(rate) * jumpFadeSecs),
		seed:        s.seed,
	}
}

func phraseGraph(s song) [][]int {
	if s.steps <= 0 || s.steps%phraseSteps != 0 {
		return nil
	}
	n := s.steps / phraseSteps
	sig := make([]string, n)
	for p := range n {
		sig[p] = bassSignature(s, p)
	}
	out := make([][]int, n)
	for p := range n {
		for q := range n {
			if p != q && sig[p] != "" && sig[p] == sig[q] {
				out[p] = append(out[p], q)
			}
		}
	}
	return out
}

func bassSignature(s song, phrase int) string {
	tokens := []string{}
	for _, pt := range s.parts {
		if pt.inst.kind != kindBass {
			continue
		}
		fields := strings.Fields(pt.line)
		if len(fields) == 0 || s.steps%len(fields) != 0 {
			return ""
		}
		for st := phrase * phraseSteps; st < (phrase+1)*phraseSteps; st++ {
			tokens = append(tokens, fields[st%len(fields)])
		}
	}
	return strings.Join(tokens, " ")
}

type jukebox struct {
	pcm        []byte
	walk       phraseWalk
	phrases    int
	pos        int
	boundaries int
	chance     float64
	fadeFrom   int
	fadeLeft   int
	rng        *rng
}

func newJukebox(pcm []byte, w phraseWalk) *jukebox {
	j := &jukebox{pcm: pcm, walk: w, chance: jumpChanceMin, rng: newRNG(w.seed)}
	if w.phraseBytes > 0 && len(pcm)%w.phraseBytes == 0 && len(w.twins) == len(pcm)/w.phraseBytes {
		j.phrases = len(pcm) / w.phraseBytes
	}
	return j
}

func (j *jukebox) Read(p []byte) (int, error) {
	if len(j.pcm) < frameBytes {
		return 0, errEmptyPCM
	}
	if len(p) < frameBytes {
		return 0, io.ErrShortBuffer
	}
	n := 0
	for ; n+frameBytes <= len(p); n += frameBytes {
		if j.phrases > 0 && j.fadeLeft == 0 && j.pos%j.walk.phraseBytes == 0 {
			j.turn()
		}
		j.writeFrame(p[n : n+frameBytes])
	}
	return n, nil
}

func (j *jukebox) turn() {
	lapDone := j.boundaries >= j.phrases
	j.boundaries++
	if !lapDone {
		return
	}
	next := j.pos / j.walk.phraseBytes
	prev := (next - 1 + j.phrases) % j.phrases
	cands := []int{}
	for _, q := range j.walk.twins[next] {
		if q != prev {
			cands = append(cands, q)
		}
	}
	if len(cands) == 0 || j.rng.unit() >= j.chance {
		j.chance = min(jumpChanceMax, j.chance+jumpChanceStep)
		return
	}
	target := cands[int(j.rng.next()%uint32(len(cands)))]
	j.fadeFrom, j.fadeLeft = j.pos, j.walk.fadeFrames
	j.pos, j.chance = target*j.walk.phraseBytes, jumpChanceMin
}

func (j *jukebox) writeFrame(dst []byte) {
	if j.fadeLeft > 0 {
		w := float64(j.walk.fadeFrames-j.fadeLeft) / float64(j.walk.fadeFrames)
		for c := 0; c < frameBytes; c += 2 {
			from := float64(int16(binary.LittleEndian.Uint16(j.pcm[j.fadeFrom+c:])))
			to := float64(int16(binary.LittleEndian.Uint16(j.pcm[j.pos+c:])))
			binary.LittleEndian.PutUint16(dst[c:], uint16(int16(from*(1-w)+to*w)))
		}
		j.fadeFrom, j.fadeLeft = j.advance(j.fadeFrom), j.fadeLeft-1
	} else {
		copy(dst, j.pcm[j.pos:j.pos+frameBytes])
	}
	j.pos = j.advance(j.pos)
}

func (j *jukebox) advance(off int) int {
	return (off + frameBytes) % len(j.pcm)
}
