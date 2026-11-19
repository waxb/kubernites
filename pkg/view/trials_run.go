package view

type TrialStatus int

const (
	TrialBriefing TrialStatus = iota
	TrialRunning
	TrialWon
	TrialLost
)

type TrialResult struct {
	Trial   string
	Seconds float64
	Medal   Medal
	Won     bool
}
