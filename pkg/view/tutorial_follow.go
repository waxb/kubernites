package view

type Hop struct {
	Who    string
	Line   string
	Target string
	Event  string
	RV     uint64
}

type HopQuestion struct {
	Hop     int
	Ask     string
	Choices []string
	Right   int
	Picked  int
}
