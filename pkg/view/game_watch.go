package view

const WatchPage = "watch"

const FrozenFirst = "frozen-felt"

const EagerWatchmanKind = "eagerwatchman"

const WatchButton = "crew:probes"

const FrozenFeelFor = 30.0

const FrozenToFeel = FrozenFeelFor

type WatchFacts struct {
	WarmHeld       float64
	WarmHut        string
	FrozenHut      string
	FrozenFor      float64
	FrozenRelit    string
	EagerKills     int
	Belled         bool
	BellCommand    string
	Unwatched      string
	UnwatchedShort string
	UnwatchedWaits int
	UnwatchedDips  bool
	UnwatchedPlan  CrewRoomPlan
}
