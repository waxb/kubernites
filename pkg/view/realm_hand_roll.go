package view

const ActEnroll = "enrollHut"

const ActStrike = "strikeEntry"

const ActTearUpRoll = "tearUpRoll"

const ActLookFor = "lookFor"

type RollEntry struct {
	Hut     string `json:"hut"`
	Address string `json:"address"`
	Live    bool   `json:"live"`
}

type RollView struct {
	Hand     bool        `json:"hand,omitempty"`
	LooksFor bool        `json:"looksFor,omitempty"`
	Bare     bool        `json:"bare,omitempty"`
	Depot    string      `json:"depot,omitempty"`
	Entries  []RollEntry `json:"entries,omitempty"`
	Idle     []RollEntry `json:"idle,omitempty"`
	Listed   []string    `json:"listed,omitempty"`
	Lost     float64     `json:"lost,omitempty"`
	Hired    int         `json:"hired,omitempty"`
}
