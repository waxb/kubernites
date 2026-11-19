package view

const BorgQuest = "borg-founding-dev"

const BorgIntro = "borg"

type QuestView struct {
	ID        string          `json:"id"`
	Title     string          `json:"title"`
	Giver     string          `json:"giver"`
	Intro     string          `json:"intro"`
	Parts     []QuestPartView `json:"parts"`
	Current   int             `json:"current"`
	Fulfilled bool            `json:"fulfilled"`
	Recap     string          `json:"recap,omitempty"`
	Closed    bool            `json:"closed,omitempty"`
}

type QuestPartView struct {
	Objective string `json:"objective"`
	Title     string `json:"title"`
	Purpose   string `json:"purpose"`
	Gained    string `json:"gained,omitempty"`
	Done      bool   `json:"done"`
	Steps     int    `json:"steps,omitempty"`
	StepsDone int    `json:"stepsDone,omitempty"`
}
