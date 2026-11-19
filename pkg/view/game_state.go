package view

const AgeDark = "dark"

const AgeFeudal = "feudal"

const ChronicleWisdom = 15.0

const FluencyWisdom = 3.0

type Proclamation struct {
	T       float64 `json:"t"`
	Kind    string  `json:"kind"`
	Title   string  `json:"title,omitempty"`
	Text    string  `json:"text"`
	Gain    float64 `json:"gain,omitempty"`
	Chest   float64 `json:"chest,omitempty"`
	Subject string  `json:"subject,omitempty"`
	Gone    bool    `json:"gone,omitempty"`
	Major   bool    `json:"major,omitempty"`
	Routine bool    `json:"routine,omitempty"`
}
