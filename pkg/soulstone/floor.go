package soulstone

type FloorHabits struct {
	LogKiPerPilgrim   float64
	HoardMiPerPilgrim float64
	ShipsJournals     bool
}

var floorHabits = map[string]FloorHabits{
	"porter":     {LogKiPerPilgrim: 96},
	"clerk":      {LogKiPerPilgrim: 32, HoardMiPerPilgrim: 6},
	"hauler":     {LogKiPerPilgrim: 64},
	"courier":    {LogKiPerPilgrim: 48},
	"sealwright": {LogKiPerPilgrim: 64},
	"raven":      {ShipsJournals: true},
}

var habitsByImage = indexHabits()

func indexHabits() map[string]FloorHabits {
	out := make(map[string]FloorHabits, len(catalog))
	for _, p := range catalog {
		out[p.Image] = floorHabits[withParts(p).Repository]
	}
	return out
}

func FloorHabitsOf(image string) FloorHabits {
	if h, ok := habitsByImage[image]; ok {
		return h
	}
	p, ok := Lookup(image)
	if !ok {
		return FloorHabits{}
	}
	return floorHabits[p.Repository]
}
