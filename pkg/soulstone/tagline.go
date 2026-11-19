package soulstone

const TaglineChars = 26

var taglines = map[string]string{
	"porter":     "Carries the parcels",
	"clerk":      "Writes ledgers, needs room",
	"hauler":     "Hauls crates and chests",
	"courier":    "Batch riders: dispatches",
	"sealwright": "Carries sealed boxes",
	"raven":      "A familiar: ships the logs",
	"hound":      "A familiar: leads boxes",
}

func Tagline(image string) string {
	p, ok := Lookup(image)
	if !ok {
		return ""
	}
	return taglines[p.Repository]
}

func Repositories() []string {
	out := []string{}
	seen := map[string]bool{}
	for _, p := range Playable() {
		if !seen[p.Repository] {
			seen[p.Repository] = true
			out = append(out, p.Repository)
		}
	}
	return out
}
