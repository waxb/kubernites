package soulstone

import (
	"sort"
	"strings"
)

const RegistryHost = "academy.kubernites.dev"

type Profile struct {
	Image               string
	Repository          string
	Tag                 string
	Name                string
	Profession          string
	Sprite              string
	Description         string
	Cost                float64
	CPUPerPilgrimMilliS float64
	ThreadMilli         float64
	IdleMilli           float64
	MemBaseMi           float64
	MemPerRpsMi         float64
	MTBFSeconds         float64
	PullSeconds         float64
	StartSeconds        float64
	SizeMi              float64
	Cursed              bool
	CurseSeconds        float64
	System              bool
	SystemCPUMilli      float64
	SystemMemMi         float64
	StartLines          []string
	CrashLine           string
	Familiar            bool
	Proxy               bool
	DutyLine            string
	Trade               string
	Kinds               []string
	Port                int32
	BoxMilliS           [SizeClasses]float64
	BoxMemMi            [SizeClasses]float64
	PledgeCPU           string
	PledgeMemory        string
	WarmupSeconds       float64
	FreezeMTBFSeconds   float64
}

func systemProfile(image, name, profession, sprite string, cpu, mem float64) Profile {
	return Profile{
		Image: image, Name: name, Profession: profession, Sprite: sprite, System: true, SystemCPUMilli: cpu, SystemMemMi: mem,
		PullSeconds: 0, StartSeconds: 1, SizeMi: 80,
		StartLines: []string{name + " awakens"},
	}
}

func Normalize(image string) string {
	trimmed := strings.TrimSpace(image)
	trimmed = strings.TrimPrefix(trimmed, RegistryHost+"/")
	if at := strings.Index(trimmed, "@"); at >= 0 {
		trimmed = trimmed[:at]
	}
	if alias, ok := aliases[trimmed]; ok {
		return alias
	}
	return trimmed
}

var index = indexCatalog()

func indexCatalog() map[string]Profile {
	out := make(map[string]Profile, len(catalog))
	for _, p := range catalog {
		out[p.Image] = withParts(p)
	}
	return out
}

func Lookup(image string) (Profile, bool) {
	p, ok := index[Normalize(image)]
	return p, ok
}

func IsLatestReference(image string) bool {
	trimmed := strings.TrimPrefix(strings.TrimSpace(image), RegistryHost+"/")
	if strings.Contains(trimmed, "@") {
		return false
	}
	last := trimmed[strings.LastIndex(trimmed, "/")+1:]
	return !strings.Contains(last, ":") || strings.HasSuffix(last, ":latest")
}

func Playable() []Profile {
	out := make([]Profile, 0, len(catalog))
	for _, p := range catalog {
		if !p.System {
			out = append(out, withParts(p))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Cost < out[j].Cost })
	return out
}

func withParts(p Profile) Profile {
	repo, tag := p.Image, "latest"
	if i := strings.LastIndex(p.Image, ":"); i > strings.LastIndex(p.Image, "/") {
		repo, tag = p.Image[:i], p.Image[i+1:]
	}
	next := p
	next.Repository = repo
	next.Tag = tag
	return withWatch(next)
}

func ProfessionOf(image string) (string, string) {
	if p, ok := Lookup(image); ok && p.Sprite != "" {
		return p.Profession, p.Sprite
	}
	return "Wanderer", "wanderer"
}
