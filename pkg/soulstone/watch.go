package soulstone

var tradeWarmup = map[string]float64{
	"porter":     3,
	"clerk":      8,
	"hauler":     5,
	"courier":    4,
	"sealwright": 6,
}

var classFreezeMTBF = map[string]float64{
	"2.0":     2400,
	"3.0":     1800,
	LatestTag: 1200,
}

func withWatch(p Profile) Profile {
	if p.System || p.Trade == "" {
		return p
	}
	next := p
	next.WarmupSeconds = tradeWarmup[p.Trade]
	if !p.Cursed {
		next.FreezeMTBFSeconds = classFreezeMTBF[p.Tag]
	}
	return next
}
