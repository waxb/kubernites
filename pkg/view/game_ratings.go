package view

type RatingFacts struct {
	DayPhase       float64
	DayLength      float64
	Hearths        int
	PledgedHearths int
	LaborUsed      float64
	LaborPledged   float64
	OpenKeeps      int
	IdleKeeps      int
	Levies         int
	HoundLevies    int
	KeepSpread     float64
	ZoneSpread     float64
	SpreadKeeps    int
	SpreadZones    int
	RavenedMusters int
	CodexBytes     float64
	CodexQuota     float64
}

type RatingsView struct {
	Reliability int `json:"reliability"`
	Efficiency  int `json:"efficiency"`
	Resilience  int `json:"resilience"`
	Favor       int `json:"favor"`
}

func (r RatingsView) Average() float64 {
	return float64(r.Reliability+r.Efficiency+r.Resilience+r.Favor) / 4
}

const ratingsBonusMax = 0.2

func ReforgeBonus(r RatingsView) float64 {
	return ratingsBonusMax * r.Average() / 100
}
