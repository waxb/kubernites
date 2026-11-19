package view

type ChronicleState struct {
	ID       string            `json:"id"`
	Kind     string            `json:"kind"`
	Order    []string          `json:"order"`
	Resolved bool              `json:"resolved"`
	Correct  bool              `json:"correct"`
	Ravens   bool              `json:"ravens,omitempty"`
	Incident string            `json:"incident,omitempty"`
	Known    bool              `json:"known,omitempty"`
	Choices  map[string]string `json:"choices,omitempty"`
}
