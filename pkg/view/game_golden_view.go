package view

type GoldenBoxView struct {
	ID    string  `json:"id"`
	Fades float64 `json:"fades"`
	Left  float64 `json:"left"`
}

type GoldenShipView struct {
	ID     string          `json:"id"`
	Harbor string          `json:"harbor"`
	Docked float64         `json:"docked"`
	Leaves float64         `json:"leaves"`
	Boxes  []GoldenBoxView `json:"boxes"`
}

type BoostView struct {
	Kind       string  `json:"kind"`
	Multiplier float64 `json:"multiplier"`
	Ends       float64 `json:"ends"`
	Left       float64 `json:"left"`
}

type GoldenView struct {
	Ship    *GoldenShipView `json:"ship,omitempty"`
	Boosts  []BoostView     `json:"boosts"`
	Claimed int             `json:"claimed"`
}
