package view

type NamesBridge struct {
	Story string
	K8s   string
}

var Content = struct {
	AgeTitle           func(age string) string
	StageTheme         func(age string) string
	AllIntros          func() []IntroView
	IntroFor           func(target string) (string, bool)
	ObjectiveConcept   func(id string) string
	RuneReachable      func(rune string) bool
	ChronicleKinds     func() []string
	FeudalObjectiveIDs func() []string
	DarkObjectiveIDs   func() []string
	KeepCrew           func(fix *CommandOffer, crews []CrewView, kindCrews []string) string
	PlotUpkeep         func(kind string, cpu, memoryGiB int, price float64) float64
	SeatVerdictText    func(msg string) (string, bool)
	Pages              func() []Page
	ForSubject         func(kind, id string) (Page, bool)
	Search             func(query string) []Page
	BridgeFor          func(page string) (Bridge, bool)
	HasBridge          func(page string) bool
	BridgeMatches      func(page, command string) bool
	BridgeMatchesWords func(page string, words []string) bool
	Chapters           func() []TutorialChapter
	Trials             func() []Trial
	NamesBridges       func() []NamesBridge
}{
	AgeTitle:           func(string) string { return "" },
	StageTheme:         func(string) string { return "" },
	AllIntros:          func() []IntroView { return nil },
	IntroFor:           func(string) (string, bool) { return "", false },
	ObjectiveConcept:   func(string) string { return "" },
	RuneReachable:      func(string) bool { return false },
	ChronicleKinds:     func() []string { return nil },
	FeudalObjectiveIDs: func() []string { return nil },
	DarkObjectiveIDs:   func() []string { return nil },
	KeepCrew:           func(*CommandOffer, []CrewView, []string) string { return "" },
	PlotUpkeep:         func(string, int, int, float64) float64 { return 0 },
	SeatVerdictText:    func(string) (string, bool) { return "", false },
	Pages:              func() []Page { return nil },
	ForSubject:         func(string, string) (Page, bool) { return Page{}, false },
	Search:             func(string) []Page { return nil },
	BridgeFor:          func(string) (Bridge, bool) { return Bridge{}, false },
	HasBridge:          func(string) bool { return false },
	BridgeMatches:      func(string, string) bool { return false },
	BridgeMatchesWords: func(string, []string) bool { return false },
	Chapters:           func() []TutorialChapter { return nil },
	Trials:             func() []Trial { return nil },
	NamesBridges:       func() []NamesBridge { return nil },
}
