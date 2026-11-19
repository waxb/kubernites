package view

type StepLevel int

const BreakIt = "BREAK IT"

const FollowTheOrder = "FOLLOW THE ORDER"

const (
	StepShow StepLevel = iota
	StepTell
	StepAsk
)

func (l StepLevel) String() string {
	switch l {
	case StepTell:
		return "tell"
	case StepAsk:
		return "ask"
	}
	return "show"
}

type StepProgress struct {
	Met  bool
	Line string
}

type StepLesson struct {
	Praise string
	Fact   string
}

type DebriefMyth struct {
	Belief string
	Truth  string
	Saw    string
}

type DebriefChoice struct {
	Text  string
	Right bool
	Why   string
}

type DebriefQuestion struct {
	Ask     string
	Choices []DebriefChoice
}

type Debrief struct {
	Sentence string
	Myth     DebriefMyth
	Check    []DebriefQuestion
	Verify   []string
	Kinds    []string
	Rule     string
	Doc      string
	Real     string
	Skips    string
	Pain     string
}
