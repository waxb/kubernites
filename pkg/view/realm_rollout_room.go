package view

type RolloutRoomView struct {
	Crew  string       `json:"crew"`
	Surge int32        `json:"surge"`
	Room  int          `json:"room"`
	Waits int          `json:"waits,omitempty"`
	Need  ResourcePair `json:"need"`
	Free  ResourcePair `json:"free"`
	Fits  bool         `json:"fits"`
	Text  string       `json:"text,omitempty"`
	Line  string       `json:"line,omitempty"`
	Dip   bool         `json:"dip,omitempty"`
	Short int          `json:"short,omitempty"`
	Huts  int          `json:"huts,omitempty"`
	Fit   int          `json:"fit,omitempty"`
	Bound string       `json:"bound,omitempty"`
	RoomPlan
}
