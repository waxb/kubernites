package view

import (
	"encoding/json"
	"fmt"
)

const ScoresVersion = 1

type TrialScore struct {
	Seconds float64 `json:"seconds,omitempty"`
	Medal   string  `json:"medal,omitempty"`
	Won     bool    `json:"won,omitempty"`
	Tries   int     `json:"tries"`
}

type TrialScores struct {
	Version int                   `json:"version"`
	Best    map[string]TrialScore `json:"best"`
}

func NewScores() TrialScores {
	return TrialScores{Version: ScoresVersion, Best: map[string]TrialScore{}}
}

func (s TrialScores) Marshal() ([]byte, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("write trial scores: %w", err)
	}
	return data, nil
}

func (s TrialScores) Of(id string) (TrialScore, bool) {
	sc, ok := s.Best[id]
	return sc, ok
}

func (sc TrialScore) Earned() Medal { return MedalNamed(sc.Medal) }

func better(next TrialResult, best TrialScore) bool {
	switch {
	case !next.Won:
		return false
	case !best.Won:
		return true
	}
	return next.Seconds < best.Seconds
}

func (s TrialScores) Record(r TrialResult) (TrialScores, bool) {
	out := NewScores()
	for id, sc := range s.Best {
		out.Best[id] = sc
	}
	best := out.Best[r.Trial]
	improved := better(r, best)
	next := best
	next.Tries++
	if improved {
		next.Seconds, next.Medal, next.Won = r.Seconds, r.Medal.Name(), true
	}
	out.Best[r.Trial] = next
	return out, improved
}
