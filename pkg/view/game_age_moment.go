package view

type AgeMomentView struct {
	Age     string   `json:"age"`
	At      float64  `json:"at"`
	Read    bool     `json:"read"`
	Lines   []string `json:"lines"`
	Opened  []string `json:"opened"`
	Changed []string `json:"changed"`
}
