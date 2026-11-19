package app

import "time"

func stoodSeconds(since map[string]time.Time, founded time.Time) map[string]float64 {
	out := make(map[string]float64, len(since))
	for name, t := range since {
		out[name] = t.Sub(founded).Seconds()
	}
	return out
}

func (s *Session) trackStood() {
	s.stood = stoodSeconds(s.Cache.NodeStood(), s.State.FoundedAt)
}
