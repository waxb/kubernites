package view

type TrialVerdict struct {
	Won      bool
	Lost     bool
	Good     bool
	Progress TrialText
	Why      TrialText
}
