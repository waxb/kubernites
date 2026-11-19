package view

import (
	"math"
	"strconv"
)

const CalamityNoDepot = "no-depot"

const CalamityNoCrew = "no-crew"

const CalamitySpoiling = "spoiling"

type RemedyView struct {
	Label   string    `json:"label"`
	Command string    `json:"command,omitempty"`
	Action  *Action   `json:"action,omitempty"`
	Blocked string    `json:"blocked,omitempty"`
	Unwired bool      `json:"-"`
	Order   string    `json:"-"`
	Target  ObjectRef `json:"-"`
}

type CalamityView struct {
	ID       string        `json:"id"`
	Kind     string        `json:"kind"`
	Object   string        `json:"object"`
	Cause    string        `json:"cause"`
	Why      string        `json:"why"`
	Remedies []RemedyView  `json:"remedies"`
	Invite   *Invitation   `json:"invite,omitempty"`
	Incident string        `json:"incident,omitempty"`
	First    bool          `json:"first,omitempty"`
	Struck   float64       `json:"struck,omitempty"`
	Mended   string        `json:"mended,omitempty"`
	Passing  bool          `json:"passing,omitempty"`
	Related  []RelatedView `json:"related,omitempty"`
	Held     bool          `json:"-"`
	Standing bool          `json:"-"`
	Quiet    bool          `json:"-"`
	Rated    *RatedCause   `json:"-"`
}

type RatedCause struct {
	Head, Tail string `json:"-"`
	Depot      int    `json:"-"`
}

func RateFigure(v float64) string {
	a := math.Abs(v)
	switch {
	case a < .005:
		return "0"
	case a < 1:
		return strconv.FormatFloat(v, 'f', 2, 64)
	case a < 10:
		return strconv.FormatFloat(v, 'f', 1, 64)
	}
	return Compact(v)
}
