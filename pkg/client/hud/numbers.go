package hud

import "github.com/waxb/kubernites/pkg/client/model"

type CrewNumbers struct {
	Crew      string
	Seats     int
	Fed       int
	ResSeats  int
	ResFed    int
	Pledge    int64
	Uses      int64
	UseButton int64
	Binding   int64
	Request   int64
	Limit     int64
	Measured  bool
	Booked    bool
	Raw       model.Resources
	Listed    bool
}

func CrewPanelNumbers(f Frame) []CrewNumbers {
	out := []CrewNumbers{}
	for _, c := range f.Crews {
		if c.Bare || c.Horn != nil {
			continue
		}
		out = append(out, crewNumbers(f, c))
	}
	return out
}

func crewNumbers(f Frame, c model.CrewChain) CrewNumbers {
	k := CrewFactsOf(f, c)
	eff := EffectiveResources(c.Resources, f)
	food, _ := FieldNamed("cpu-req")
	n := CrewNumbers{Crew: c.Name, Seats: k.Seats, Fed: k.Fed, Pledge: k.pledge().CPU, Uses: k.Peak.CPU, UseButton: food.measured(k.Peak),
		Binding: k.Binding.CPU, Request: eff.CPURequest, Limit: eff.CPULimit, Measured: k.Peak.Known(), Booked: !k.Legacy, Raw: c.Resources}
	_, n.Listed = crewByName(f, c.Name)
	if fit, _, ok := repledgeFit(f, c, c.Resources); ok {
		n.ResSeats, n.ResFed = fit.Seated, fit.Fed
	}
	return n
}
