package view

const ButtonToken = "{button}"

const ViaOpen = "open"

const ViaAck = "ack"

type IntroView struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Purpose string `json:"purpose"`
	Lore    string `json:"lore,omitempty"`
	Try     string `json:"try"`
	Target  string `json:"target"`
	Via     string `json:"via"`
}
