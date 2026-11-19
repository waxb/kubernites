package view

type AgeStep struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Theme   string   `json:"theme"`
	Goal    string   `json:"goal"`
	Asks    string   `json:"asks,omitempty"`
	Unlocks []string `json:"unlocks"`
	Status  string   `json:"status"`
}

type RoadView struct {
	Ages         []AgeStep `json:"ages"`
	Current      string    `json:"current"`
	ReforgeKeeps []string  `json:"reforgeKeeps"`
	ReforgeLoses []string  `json:"reforgeLoses"`
}

const AgeCastle = "castle"

const AgeEmpire = "empire"
