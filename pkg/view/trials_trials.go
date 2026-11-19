package view

import (
	"fmt"
)

type Medal int

const (
	MedalNone Medal = iota
	MedalBronze
	MedalSilver
	MedalGold
)

var medalNames = map[Medal]string{MedalGold: "GOLD", MedalSilver: "SILVER", MedalBronze: "BRONZE"}

func (m Medal) Name() string { return medalNames[m] }

func MedalNamed(name string) Medal {
	for m, n := range medalNames {
		if n == name {
			return m
		}
	}
	return MedalNone
}

type TrialTimes struct {
	Gold, Silver, Bronze float64
}

func (t TrialTimes) Medal(seconds float64) Medal {
	switch {
	case seconds <= t.Gold:
		return MedalGold
	case seconds <= t.Silver:
		return MedalSilver
	case seconds <= t.Bronze:
		return MedalBronze
	}
	return MedalNone
}

func (t TrialTimes) Limit(m Medal) float64 {
	switch m {
	case MedalGold:
		return t.Gold
	case MedalSilver:
		return t.Silver
	case MedalBronze:
		return t.Bronze
	}
	return 0
}

type TrialText struct {
	Story string
	K8s   string
}

func (t TrialText) In(names bool) string {
	if names && t.K8s != "" {
		return t.K8s
	}
	return t.Story
}

type TrialKube struct {
	Commands []string
	YAML     []string
	Rule     string
	Doc      string
}

func TrialClock(seconds float64) string {
	s := max(int(seconds), 0)
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
