package hud

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	OpenLand     = 999
	noScheduleFx = ":NoSchedule"
	sealedName   = "sealed boxes"
)

type ReplicaFacts struct {
	Income      float64
	CPU         int64
	Memory      int64
	UpkeepShare float64
}

type keepLand struct {
	Name   string
	CPU    [2]int64
	Mem    [2]int64
	Upkeep float64
}

type flowFacts struct {
	Name     string
	Arriving float64
	Carried  float64
	Lost     float64
	Waiting  float64
	Known    bool
}

type CrewFacts struct {
	Per       ReplicaFacts
	Fits      int
	Fed       int
	Seats     int
	Short     int
	Limit     string
	Takes     view.ResourcePair
	Free      view.ResourcePair
	Booked    bool
	Legacy    bool
	Pending   int
	Keeps     []keepLand
	Flow      flowFacts
	Offer     view.CrewKeepOffer
	Offered   bool
	Use       useFacts
	Peak      view.UsePeakView
	Binding   view.ResourcePair
	Squire    view.ResourcePair
	CharterAt int
}

func (k CrewFacts) pledge() view.ResourcePair {
	return view.ResourcePair{CPU: k.Per.CPU - k.Squire.CPU, Memory: k.Per.Memory - k.Squire.Memory}
}

func CrewFactsOf(f Frame, c model.CrewChain) CrewFacts {
	land := landOf(f)
	eff := EffectiveResources(c.Resources, f)
	out := CrewFacts{Keeps: land, Flow: crewFlow(f.State.Freight, c.Name), Use: crewUse(c, f), Peak: crewPeakOf(f, c),
		Binding: view.ResourcePair{CPU: eff.CPULimit, Memory: eff.MemLimit * mebi}}
	book := f.State.Room
	out.Booked = len(book.Keeps) > 0
	if out.Booked {
		out.Free = book.FreeAfterOwed()
	}
	if cv, ok := crewByName(f, c.Name); ok {
		p := cv.PerReplica
		out.Per = ReplicaFacts{Income: p.Income, CPU: p.CPU, Memory: p.Memory, UpkeepShare: p.UpkeepShare}
		out.Fits, out.Pending, out.Takes = cv.Fits, cv.Pending, cv.Takes
		out.Seats, out.Short, out.Limit = seatsOf(cv)
		out.Legacy = cv.Room == (view.RoomFit{})
		out.Fed = out.Fits
		if !out.Legacy {
			out.Fed = min(out.Seats, max(cv.Room.Safe(), 0))
		}
		out.Offer, out.Offered = offerOf(cv)
		out.Squire = view.ResourcePair{CPU: p.SquireCPU, Memory: p.SquireMem}
		out.CharterAt = cv.CharterAt
	} else {
		out.Per = ReplicaFacts{CPU: c.Resources.CPURequest, Memory: c.Resources.MemRequest * mebi}
		out.Pending = pendingOf(f, c)
		out.Fits, out.Seats, out.Legacy = fitsOn(land, out.Per), fitsOn(land, out.Per), true
		out.Fed = out.Fits
		if out.Booked {
			asks := view.ResourcePair{CPU: out.Per.CPU, Memory: out.Per.Memory}
			fit := book.Fit(c.Name, asks, asks)
			out.Fits, out.Seats, out.Short, out.Legacy = fit.Safe(), max(fit.Seated, 0), fit.Short, false
			out.Fed = out.Fits
		}
	}
	if out.Per.CPU <= 0 && out.Per.Memory <= 0 {
		out.Fits, out.Seats, out.Fed = OpenLand, OpenLand, OpenLand
	}
	return out
}

func seatsOf(cv view.CrewView) (int, int, string) {
	if cv.Room == (view.RoomFit{}) {
		return cv.Fits, 0, ""
	}
	seats := max(cv.Fits, min(max(cv.Room.Seated, 0), MaxCrewSize-cv.Replicas))
	if cv.Fits >= seats {
		return seats, cv.Room.Short, ""
	}
	return seats, cv.Room.Short, cv.Room.Limit
}

func offerOf(cv view.CrewView) (view.CrewKeepOffer, bool) {
	if cv.KeepOffer == nil {
		return view.CrewKeepOffer{}, false
	}
	return *cv.KeepOffer, true
}

func crewFlow(fv view.FreightView, crew string) flowFacts {
	for _, k := range fv.Kinds {
		if k.CrewName == crew && k.Open {
			return flowFacts{Name: k.Name, Arriving: k.ArrivalRate, Carried: k.ServedRate, Lost: k.SpoiledRate, Waiting: k.Waiting, Known: true}
		}
	}
	if s := fv.Seals; s.CrewName == crew && s.ArrivalRate > 0 {
		return flowFacts{Name: sealedName, Arriving: s.ArrivalRate, Carried: s.ServedRate, Lost: s.SpoiledRate, Waiting: s.Waiting, Known: true}
	}
	return flowFacts{}
}

func landOf(f Frame) []keepLand {
	if len(f.State.Keeps) == 0 {
		return metricLand(f)
	}
	out := []keepLand{}
	for _, k := range f.State.Keeps {
		if k.Open {
			out = append(out, keepLand{Name: k.Name, CPU: [2]int64{k.Requested.CPU, k.Allocatable.CPU},
				Mem: [2]int64{k.Requested.Memory, k.Allocatable.Memory}, Upkeep: k.Upkeep})
		}
	}
	return out
}

func metricLand(f Frame) []keepLand {
	out := []keepLand{}
	for _, name := range slices.Sorted(maps.Keys(f.Metrics.Nodes)) {
		if n, ok := f.Nodes[name]; ok && !Schedulable(n) {
			continue
		}
		m := f.Metrics.Nodes[name]
		out = append(out, keepLand{Name: name, CPU: [2]int64{m.CPURequestedMilli, m.CPUAllocatableMilli},
			Mem: [2]int64{m.MemoryRequestedBytes, m.MemoryAllocatableBytes}})
	}
	return out
}

func Schedulable(n model.NodeView) bool {
	if n.Unschedulable {
		return false
	}
	for _, t := range n.TaintSpecs {
		if strings.HasSuffix(t, noScheduleFx) {
			return false
		}
	}
	return true
}

func fitsOn(land []keepLand, per ReplicaFacts) int {
	if per.CPU <= 0 && per.Memory <= 0 {
		return OpenLand
	}
	total := 0
	for _, k := range land {
		total += min(RoomFor(k.CPU, per.CPU), RoomFor(k.Mem, per.Memory))
	}
	return total
}

func RoomFor(use [2]int64, each int64) int {
	if each <= 0 {
		return OpenLand
	}
	return int(max(use[1]-use[0], 0) / each)
}

func pendingOf(f Frame, c model.CrewChain) int {
	n := 0
	for _, p := range crewPods(c, f) {
		if p.State == model.HearthPending {
			n++
		}
	}
	return n
}

func (k CrewFacts) LandTotals() ([2]int64, [2]int64) {
	var cpu, mem [2]int64
	for _, l := range k.Keeps {
		cpu[0], cpu[1] = cpu[0]+l.CPU[0], cpu[1]+l.CPU[1]
		mem[0], mem[1] = mem[0]+l.Mem[0], mem[1]+l.Mem[1]
	}
	return cpu, mem
}

func (k CrewFacts) Bounded() bool {
	return k.Seats < OpenLand
}

func BuyKeepAction(c model.CrewChain, o view.KeepOffer) view.Action {
	return view.Action{Action: "buyKeep", KeepType: o.KeepType, For: c.Name}
}

func buyKeepFor(c model.CrewChain, f Frame) []Intent {
	cv, ok := crewByName(f, c.Name)
	if !ok || cv.KeepOffer == nil {
		return nil
	}
	return []Intent{ActFrom(BuyKeepAction(c, cv.KeepOffer.KeepOffer), FromCrewCard)}
}

func (k CrewFacts) requests() bool {
	return k.Per.CPU > 0 || k.Per.Memory > 0
}

func (k CrewFacts) tooBig() bool {
	return k.Offered && k.requests() && k.Offer.Instance != "" && k.Offer.Seats == 0
}

func shortCost(limit string) string {
	if limit == view.FlowMemory {
		return ui.Bridge("CROWDED", "MEMORY")
	}
	return ui.Bridge("HUNGRY", "CPU")
}

func shortTip(k CrewFacts) string {
	if k.Limit == view.FlowMemory {
		return fmt.Sprintf(ui.Bridge("The Keeps would have room for %d more by their pledges, but each takes about %s: more would crowd a Keep, and its Reeve banishes workers.",
			"The scheduler would place %d more by requests, but each uses about %s: more would put a Node under memory pressure, and the kubelet evicts Pods."),
			k.Seats, ui.Bridge(housingOf(float64(k.Takes.Memory)), MemQuantity(k.Takes.Memory)))
	}
	fed := ui.Bridge("no more", "no more")
	if k.Fed > 0 {
		fed = fmt.Sprintf(ui.Bridge("only %d more", "only %d more"), k.Fed)
	}
	if k.Use.Pods == 0 {
		return fmt.Sprintf(ui.Bridge("The Keeps would have room for %d more by their pledges, but at what the huts really eat the Keeps can feed %s: more would go hungry.",
			"The scheduler would place %d more by requests, but at the CPU the Pods really use the Nodes have spare CPU for %s: more would be CPU-throttled."), k.Seats, fed)
	}
	each := k.Use.CPU / float64(k.Use.Pods)
	return fmt.Sprintf(ui.Bridge("The Keeps would have room for %d more by their pledges, but your huts eat about %s each now (TOP HUTS shows it), and the Keeps can feed %s: more would go hungry.",
		"The scheduler would place %d more by requests, but the Pods use about %s each now (kubectl top shows it), and the Nodes have spare CPU for %s: more would be CPU-throttled."),
		k.Seats, ui.Bridge(foodOf(each), fmt.Sprintf("%.0fm CPU", math.Round(each))), fed)
}
