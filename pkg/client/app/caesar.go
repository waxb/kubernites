package app

import (
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func CaesarOf(st view.GameState, m view.MetricsSnapshot) model.Caesar {
	r := st.Ratings
	c := model.Caesar{
		Counsel: st.Counsel,
		Ratings: model.Ratings{Reliability: float64(r.Reliability), Efficiency: float64(r.Efficiency), Resilience: float64(r.Resilience), Favor: float64(r.Favor)},
		Rated:   r != view.RatingsView{},
		Codex: model.Codex{SizeBytes: float64(st.Codex.SizeBytes), InUseBytes: float64(st.Codex.InUseBytes), QuotaBytes: float64(st.Codex.QuotaBytes),
			Alarm: st.Codex.Alarm, LastRebind: st.Codex.LastRebind},
		Middens: map[string]model.Midden{},
	}
	for _, d := range st.Decrees {
		c.Decrees = append(c.Decrees, model.Decree{ID: d.ID, Title: d.Title, Text: d.Text, Kind: d.Kind, State: d.State, Deadline: d.Deadline,
			Progress: decreeShare(d), Status: d.Progress, Reward: d.Reward.Favor, Gold: d.Reward.Gold, Wisdom: d.Reward.Wisdom, Asks: d.Tribute})
	}
	for node, n := range m.Nodes {
		if n.EphemeralCapacityBytes > 0 {
			c.Middens[node] = model.Midden{Node: node, UsedBytes: float64(n.EphemeralUsedBytes), CapacityBytes: float64(n.EphemeralCapacityBytes), Pressure: n.DiskPressure}
		}
	}
	return c
}

func decreeShare(d view.DecreeView) float64 {
	if d.ProgressTotal > 0 {
		return min(max(d.ProgressDone/d.ProgressTotal, 0), 1)
	}
	return model.ProgressOf(d.Progress)
}
