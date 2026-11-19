package talk

import "hash/fnv"

type Tone int

const (
	Cheerful Tone = iota
	Grumpy
	Anxious
	Proud
	Dreamy
	Joker
	tones
	AnyTone Tone = -1
)

var toneNames = [tones]string{"cheerful", "grumpy", "anxious", "proud", "dreamy", "joker"}

func (t Tone) String() string {
	if t < 0 || t >= tones {
		return "any"
	}
	return toneNames[t]
}

type Identity struct {
	Key      string
	Seed     uint32
	Familiar string
}

func Worker(namespace, pod, container string) Identity {
	key := namespace + "/" + pod + "/" + container
	return Identity{Key: key, Seed: hash(key)}
}

func Pet(namespace, pod, container, kind string) Identity {
	id := Worker(namespace, pod, container)
	id.Familiar = kind
	return id
}

func hash(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

func mix(seed, salt uint32) uint32 {
	x := seed ^ (salt * 0x9e3779b9)
	x ^= x >> 16
	x *= 0x7feb352d
	x ^= x >> 15
	x *= 0x846ca68b
	x ^= x >> 16
	return x
}

func unit(x uint32) float64 {
	return float64(x%10007) / 10006
}

func (id Identity) Tone() Tone {
	return Tone(mix(id.Seed, 1) % uint32(tones))
}

type Voice struct {
	Pitch  float64
	CPS    float64
	Seed   uint32
	Timbre string
}

const (
	TimbreVoice = ""
	TimbreChirp = "chirp"
	TimbreHum   = "hum"
	PitchLow    = 0.55
	PitchHigh   = 1.55
	CPSLow      = 26.0
	CPSHigh     = 34.0
)

var toneBands = [tones][2]float64{
	Cheerful: {1.1, 1.55}, Grumpy: {0.55, 0.85}, Anxious: {1.0, 1.4},
	Proud: {0.7, 1.05}, Dreamy: {0.8, 1.2}, Joker: {0.85, 1.45},
}

var petVoices = map[string]Voice{
	"raven": {Pitch: 1.5, CPS: 22, Timbre: TimbreChirp},
	"hound": {Pitch: 0.6, CPS: 18, Timbre: TimbreHum},
}

func (id Identity) Voice() Voice {
	if v, ok := petVoices[id.Familiar]; ok {
		v.Seed = id.Seed
		v.Pitch *= 0.95 + 0.1*unit(mix(id.Seed, 2))
		return v
	}
	band := toneBands[id.Tone()]
	pitch := band[0] + (band[1]-band[0])*unit(mix(id.Seed, 2))
	cps := CPSLow + (CPSHigh-CPSLow)*unit(mix(id.Seed, 3))
	return Voice{Pitch: pitch, CPS: cps, Seed: id.Seed}
}
