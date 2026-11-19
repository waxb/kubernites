package view

type ClassOffer struct {
	Image  string  `json:"image"`
	Tag    string  `json:"tag"`
	Price  float64 `json:"price"`
	Effect string  `json:"effect"`
	Gate   string  `json:"gate,omitempty"`
}

type RumourView struct {
	Image string `json:"image"`
	Tag   string `json:"tag"`
}

type PledgeView struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type TradeView struct {
	ID        string      `json:"id"`
	Kind      string      `json:"kind"`
	Open      bool        `json:"open"`
	ArrivedAt float64     `json:"arrivedAt"`
	Class     string      `json:"class"`
	Next      *ClassOffer `json:"next,omitempty"`
	Rumour    *RumourView `json:"rumour,omitempty"`
	Workers   int         `json:"workers"`
	Income    float64     `json:"income"`
	Share     float64     `json:"share"`
	PerWorker float64     `json:"perWorker"`
	Pledge    PledgeView  `json:"pledge"`
	LandTier  float64     `json:"landTier"`
}
