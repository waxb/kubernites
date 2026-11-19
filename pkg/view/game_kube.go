package view

const EndingTitle = "THE DEMO ENDS HERE"

const EndingSubtitle = "You have walked every finished step of the Feudal Age."

const ExperimentalNote = "Past the demo's end: this step plays, but it is unfinished and not balanced yet."

type SaidCommand struct {
	Command string  `json:"command"`
	At      float64 `json:"at"`
	Typed   bool    `json:"typed,omitempty"`
	YAML    string  `json:"yaml,omitempty"`
}

type BridgeMark struct {
	First   float64 `json:"first"`
	Presses int     `json:"presses"`
}

type EndingView struct {
	At     float64  `json:"at"`
	Read   bool     `json:"read"`
	Built  []string `json:"built"`
	Goal   string   `json:"goal"`
	PlayOn string   `json:"playOn"`
	Close  string   `json:"close"`
}

type KubeView struct {
	Bridged map[string]BridgeMark `json:"bridged,omitempty"`
	Said    []SaidCommand         `json:"said,omitempty"`
	Ending  *EndingView           `json:"ending,omitempty"`
}
