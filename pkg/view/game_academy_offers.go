package view

type CrewRoll struct {
	Line  string
	Waits int
	Plan  CrewRoomPlan
}

type CrewRoomPlan struct {
	Need      string
	To        int
	Floor     int
	Shrink    int
	Horned    bool
	Surgeless string
	Starves   int
	Rising    bool
	Reserve   bool
	Settling  bool
}
