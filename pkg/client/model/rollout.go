package model

import "github.com/waxb/kubernites/pkg/view"

type RolloutWatch struct {
	Rev    int64
	From   string
	Income float64
}

type Rollouts map[string]RolloutWatch

type RolloutDone struct {
	Crew  string
	Image string
	From  string
	Huts  int
	Gain  float64
}

func Newest(c CrewChain) (Foreman, bool) {
	for _, f := range c.Foremen {
		if f.Newest {
			return f, true
		}
	}
	return Foreman{}, false
}

func outgoing(c CrewChain) string {
	for _, f := range c.Foremen {
		if !f.Newest && (f.Desired > 0 || f.Current > 0) {
			return f.Image
		}
	}
	return ""
}

func DepotIncome(fv view.FreightView, service string) float64 {
	for _, k := range fv.Kinds {
		if k.Depot == service {
			return k.ServedRate * k.Value
		}
	}
	return 0
}

func (w Rollouts) Gain(c CrewChain, fv view.FreightView) float64 {
	f, ok := Newest(c)
	base, seen := w[c.ID()]
	if !ok || !seen || base.Rev != f.Revision {
		return 0
	}
	return max(0, DepotIncome(fv, c.Depot)-base.Income)
}

func (w Rollouts) Step(crews []CrewChain, fv view.FreightView) (Rollouts, []RolloutDone) {
	next, done := Rollouts{}, []RolloutDone{}
	for _, c := range crews {
		f, ok := Newest(c)
		if !ok {
			continue
		}
		base, seen := w[c.ID()]
		switch {
		case c.Rolling() && seen && base.Rev == f.Revision:
			next[c.ID()] = base
		case c.Rolling():
			next[c.ID()] = RolloutWatch{Rev: f.Revision, From: outgoing(c), Income: DepotIncome(fv, c.Depot)}
		case seen && base.Rev == f.Revision && base.From != f.Image:
			done = append(done, RolloutDone{Crew: c.Name, Image: f.Image, From: base.From, Huts: max(f.Desired, f.Current, f.Ready), Gain: w.Gain(c, fv)})
		}
	}
	return next, done
}
