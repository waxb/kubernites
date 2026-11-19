package view

type HarborTierDef struct {
	Tier         int
	Ship         string
	Ships        string
	Multiplier   float64
	Sealed       float64
	Crowned      float64
	Opens        string
	Listener     string
	ListenerPort int32
}

var HarborTiers = [MaxTier]HarborTierDef{
	{Tier: 1, Ship: "cog", Ships: "cogs", Multiplier: 1, Opens: "small parcels", Listener: "HTTP", ListenerPort: 80},
	{Tier: 2, Ship: "caravel", Ships: "caravels", Multiplier: 1, Crowned: 0.1, Opens: "ledgers, medium boxes and crowned boxes", Listener: "TCP", ListenerPort: 5432},
	{Tier: 3, Ship: "galleon", Ships: "galleons", Multiplier: 1, Crowned: 0.15, Opens: "dispatches and large boxes", Listener: "HTTPS", ListenerPort: 443},
	{Tier: 4, Ship: "carrack", Ships: "carracks", Multiplier: 1, Sealed: 0.3, Crowned: 0.2, Opens: "sealed boxes and huge boxes", Listener: "TLS", ListenerPort: 8443},
}

type HarborView struct {
	Tier             int     `json:"tier"`
	Ship             string  `json:"ship"`
	Multiplier       float64 `json:"multiplier"`
	UpkeepMultiplier float64 `json:"upkeepMultiplier"`
	Fresh            float64 `json:"fresh"`
	TierProgress     float64 `json:"tierProgress"`
	NextTier         float64 `json:"nextTier"`
	NextShip         string  `json:"nextShip,omitempty"`
	NextOpens        string  `json:"nextOpens,omitempty"`
	NextPort         string  `json:"nextPort,omitempty"`
	NextPortName     string  `json:"nextPortName,omitempty"`
	NextLevel        int     `json:"nextLevel,omitempty"`
	NextWait         string  `json:"nextWait,omitempty"`
	NextWhy          string  `json:"nextWhy,omitempty"`
	TierAt           float64 `json:"tierAt"`
	FreshPerSec      float64 `json:"freshPerSec"`
	ValuePerSec      float64 `json:"valuePerSec"`
}

func HarborTierOf(tier int) HarborTierDef {
	return HarborTiers[max(1, min(MaxTier, tier))-1]
}
