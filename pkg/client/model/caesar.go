package model

import (
	"fmt"
	"math"
	"sort"
)

const (
	CodexLevels  = 4
	MiddenLevels = 4

	GaugeCalm   = 0
	GaugeWarn   = 1
	GaugeDanger = 2

	DecreeOffered = "offered"
	DecreeActive  = "active"
	DecreeMet     = "met"
	DecreeMissed  = "missed"
	DecreePaid    = "paid"
	DecreeTribute = "tribute"

	gaugeWarnAt   = .6
	gaugeDangerAt = .85
	codexLargeAt  = .5
	codexFloor    = .35
	middenFloor   = .05
	kibibyte      = 1 << 10
	mebibyte      = 1 << 20
	gibibyte      = 1 << 30
)

var (
	codexSteps  = []float64{.5, .65, .8}
	middenSteps = []float64{.35, .6, .85}
)

type Codex struct {
	SizeBytes  float64
	InUseBytes float64
	QuotaBytes float64
	Alarm      bool
	LastRebind float64
}

type Midden struct {
	Node          string
	UsedBytes     float64
	CapacityBytes float64
	Pressure      bool
}

type Ratings struct {
	Reliability float64
	Efficiency  float64
	Resilience  float64
	Favor       float64
}

type Decree struct {
	ID       string
	Title    string
	Text     string
	Kind     string
	State    string
	Deadline float64
	Progress float64
	Status   string
	Reward   float64
	Gold     float64
	Wisdom   float64
	Asks     float64
}

type Caesar struct {
	Counsel string
	Ratings Ratings
	Rated   bool
	Decrees []Decree
	Codex   Codex
	Middens map[string]Midden
}

func share(part, whole float64) float64 {
	if whole <= 0 {
		return 0
	}
	return min(max(part/whole, 0), 1)
}

func stepOf(fill float64, steps []float64) int {
	level := 0
	for _, s := range steps {
		if fill >= s {
			level++
		}
	}
	return level
}

func (c Codex) Known() bool { return c.QuotaBytes > 0 }

func (c Codex) Fill() float64 { return share(c.SizeBytes, c.QuotaBytes) }

func (c Codex) Level() int {
	fill := c.Fill()
	switch {
	case c.Alarm:
		return CodexLevels - 1
	case fill < codexFloor:
		return -1
	}
	return stepOf(fill, codexSteps)
}

func (c Codex) Large() bool { return c.Alarm || c.Fill() >= codexLargeAt }

func (c Codex) Reclaimable() float64 { return max(c.SizeBytes-c.InUseBytes, 0) }

func (m Midden) Fill() float64 { return share(m.UsedBytes, m.CapacityBytes) }

func (m Midden) Level() int {
	if m.Pressure {
		return MiddenLevels - 1
	}
	fill := m.Fill()
	if fill < middenFloor {
		return -1
	}
	return stepOf(fill, middenSteps)
}

func GaugeLevel(fill float64, alarm bool) int {
	switch {
	case alarm || fill >= gaugeDangerAt:
		return GaugeDanger
	case fill >= gaugeWarnAt:
		return GaugeWarn
	}
	return GaugeCalm
}

func (d Decree) Open() bool {
	return d.State != DecreeMet && d.State != DecreeMissed && d.State != DecreePaid
}

func (d Decree) Tribute() bool { return d.Kind == DecreeTribute }

func (d Decree) Left(now float64) float64 { return max(d.Deadline-now, 0) }

func (c Caesar) OpenDecrees() []Decree {
	out := []Decree{}
	for _, d := range c.Decrees {
		if d.Open() {
			out = append(out, d)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Deadline < out[j].Deadline })
	return out
}

func Countdown(seconds float64) string {
	total := int(math.Ceil(max(seconds, 0)))
	h, m, s := total/3600, total/60%60, total%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func Bytes(v float64) string {
	switch {
	case v >= 10*gibibyte:
		return fmt.Sprintf("%.0f GiB", v/gibibyte)
	case v >= gibibyte:
		return fmt.Sprintf("%.1f GiB", v/gibibyte)
	case v >= 10*mebibyte:
		return fmt.Sprintf("%.0f MiB", v/mebibyte)
	case v >= mebibyte:
		return fmt.Sprintf("%.1f MiB", v/mebibyte)
	}
	return fmt.Sprintf("%.0f KiB", v/kibibyte)
}

func ProgressOf(text string) float64 {
	var done, total float64
	if n, _ := fmt.Sscanf(text, "%g/%g", &done, &total); n == 2 && total > 0 {
		return share(done, total)
	}
	return 0
}
