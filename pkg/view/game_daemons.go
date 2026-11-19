package view

const DaemonsPage = "daemons"

const WardenPlacedFlag = "warden:placed"

type DaemonFacts struct {
	StrandedCouriers int
	StrandedKeeps    []string
	CourierKeep      string
	WardenCommand    string
	DaemonRaised     bool
}
