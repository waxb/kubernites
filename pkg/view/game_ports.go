package view

const GatePortID = "haven-0"

type PortState struct {
	ID       string  `json:"id"`
	Name     string  `json:"name,omitempty"`
	Open     bool    `json:"open,omitempty"`
	Level    int     `json:"level,omitempty"`
	Berths   int     `json:"berths,omitempty"`
	OpenedAt float64 `json:"openedAt,omitempty"`
	LevelAt  float64 `json:"levelAt,omitempty"`
}

func (p PortState) Title() string {
	if p.Name != "" {
		return p.Name
	}
	return p.ID
}
