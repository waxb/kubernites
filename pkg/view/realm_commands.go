package view

type CommandResult struct {
	Output  string `json:"output"`
	OK      bool   `json:"ok"`
	Summary string `json:"summary,omitempty"`
}
