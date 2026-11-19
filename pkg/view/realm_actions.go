package view

type Action struct {
	Action    string    `json:"action"`
	Soulstone string    `json:"soulstone,omitempty"`
	ID        string    `json:"id,omitempty"`
	Image     string    `json:"image,omitempty"`
	KeepType  string    `json:"keepType,omitempty"`
	Speed     float64   `json:"speed,omitempty"`
	Answer    string    `json:"answer,omitempty"`
	Name      string    `json:"name,omitempty"`
	Replicas  int       `json:"replicas,omitempty"`
	Rank      string    `json:"rank,omitempty"`
	Kind      string    `json:"kind,omitempty"`
	X         int       `json:"x,omitempty"`
	Y         int       `json:"y,omitempty"`
	At        float64   `json:"at,omitempty"`
	Crew      CrewOrder `json:"crew,omitzero"`
	For       string    `json:"for,omitempty"`
	Fix       bool      `json:"fix,omitempty"`
	Keep      string    `json:"keep,omitempty"`
	Isle      string    `json:"isle,omitempty"`
	Shelves   int       `json:"shelves,omitempty"`
	Class     string    `json:"class,omitempty"`
	From      string    `json:"from,omitempty"`
}

const MaxReplicas = 100
