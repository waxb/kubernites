package hud

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	Mib        = 1 << 20
	millicores = 1000
)

func crewByName(f Frame, name string) (view.CrewView, bool) {
	for _, c := range f.State.Crews {
		if c.Name == name {
			return c, true
		}
	}
	return view.CrewView{}, false
}

func ReplicaPayoff(f Frame, crew string) (view.ReplicaCost, bool) {
	c, ok := crewByName(f, crew)
	if !ok {
		return view.ReplicaCost{}, false
	}
	return c.PerReplica, true
}

func GoldPerSecond(v float64) string {
	sign, amount := signedGold(v)
	return ui.Bridge(sign+amount+" GOLD/S", sign+"$"+amount+"/S")
}

func signedGold(v float64) (string, string) {
	amount := goldAmount(math.Abs(v))
	if v < 0 && amount != "0" {
		return "-", amount
	}
	return "+", amount
}

func goldAmount(v float64) string {
	switch {
	case v < 1:
		return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
	case v < 10:
		return strconv.FormatFloat(math.Round(v*10)/10, 'f', -1, 64)
	}
	return Compact(v)
}

func StepPayoff(f Frame, crew string, step int) string {
	per, ok := ReplicaPayoff(f, crew)
	if !ok || step == 0 {
		return ""
	}
	return GoldPerSecond(per.Income * float64(step))
}

func ScalePayoff(c model.CrewChain, f Frame, step int) string {
	return StepPayoff(f, c.Name, step)
}

func CPUAmount(milli int64) string {
	if milli%millicores == 0 {
		return fmt.Sprint(milli / millicores)
	}
	return fmt.Sprintf("%dm", milli)
}

func MemoryAmount(bytes int64) string {
	return fmt.Sprintf("%dMi", bytes/Mib)
}

func ScaleDetail(c model.CrewChain, f Frame, step int) string {
	per, ok := ReplicaPayoff(f, c.Name)
	if !ok || step == 0 {
		return ""
	}
	n := int64(max(step, -step))
	verb := ui.Bridge("reserves", "requests")
	if step < 0 {
		verb = ui.Bridge("frees", "releases")
	}
	return fmt.Sprintf("%s, %s %s CPU and %s memory (about %s of Keep upkeep)", GoldPerSecond(per.Income*float64(step)), verb,
		CPUAmount(per.CPU*n), MemoryAmount(per.Memory*n), GoldPerSecond(per.UpkeepShare*float64(step)))
}

func offerCapacity(o view.KeepOffer) (int64, int64, bool) {
	cpu, err := resource.ParseQuantity(o.CPU)
	if err != nil {
		return 0, 0, false
	}
	mem, err := resource.ParseQuantity(o.Memory)
	if err != nil {
		return 0, 0, false
	}
	return cpu.MilliValue(), mem.Value(), true
}

func payoffCrew(f Frame) (view.CrewView, bool) {
	kinds := make([]string, 0, len(f.State.Freight.Kinds))
	for _, k := range f.State.Freight.Kinds {
		kinds = append(kinds, k.CrewName)
	}
	return crewByName(f, view.Content.KeepCrew(EconomyOf(f.State).Fix, f.State.Crews, kinds))
}

func KeepPayoff(f Frame, o view.KeepOffer) string {
	cpu, mem, ok := offerCapacity(o)
	if !ok {
		return ""
	}
	return JoinParts(HandsGain(cpu), roomLine(f, cpu, mem, o.UpkeepPerSec))
}

func roomLine(f Frame, cpu, mem int64, upkeep float64) string {
	c, ok := payoffCrew(f)
	return crewRoomLine(c, ok, cpu, mem, upkeep)
}

func crewRoomLine(c view.CrewView, ok bool, cpu, mem int64, upkeep float64) string {
	if !ok || c.PerReplica.CPU <= 0 && c.PerReplica.Memory <= 0 {
		return UpkeepCost(upkeep)
	}
	asks := view.ResourcePair{CPU: c.PerReplica.CPU, Memory: c.PerReplica.Memory}
	return seatsFor(view.SeatsOnKeep(view.ResourcePair{CPU: cpu, Memory: mem}, asks, view.KeepTakes(asks, c.Takes)), c.Name, c.Pending, upkeep)
}

func offerPayoff(o view.CrewKeepOffer, crew string, waiting int, requests bool) string {
	cpu, _, ok := offerCapacity(o.KeepOffer)
	if !ok {
		return ""
	}
	if !requests {
		return JoinParts(HandsGain(cpu), UpkeepCost(o.UpkeepPerSec))
	}
	return JoinParts(HandsGain(cpu), seatsFor(o.Seats, crew, waiting, o.UpkeepPerSec))
}

func HandsGain(cpu int64) string {
	cores := CPUAmount(cpu)
	return ui.Bridge("+"+foodOf(float64(cpu)), "+"+cores+" CPU")
}

func JoinParts(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, ", ")
}

func UpkeepCost(upkeep float64) string {
	if upkeep < .005 {
		return ""
	}
	return ui.Bridge("upkeep up to "+Rate(upkeep)+" gold/s", "upkeep up to $"+Rate(upkeep)+"/s")
}

func SeatsLine(seats int, crew string, upkeep float64) string {
	return seatsFor(seats, crew, 0, upkeep)
}

func seatsFor(seats int, crew string, waiting int, upkeep float64) string {
	room := fmt.Sprintf("room for %d %s", seats, crew)
	if waiting > 0 && seats > 0 {
		room += fmt.Sprintf(ui.Bridge(" (the %d waiting move in first)", " (the %d Pending land first)"), min(waiting, seats))
	}
	return JoinParts(room, UpkeepCost(upkeep))
}

func landValue(o view.KeepOffer) float64 {
	if o.Value > 0 {
		return o.Value
	}
	return o.Cost
}

func PlotPayoff(f Frame, kind string, cores, memGiB int) string {
	return PlotPayoffFor(f, kind, "", cores, memGiB)
}

func PlotPayoffFor(f Frame, kind, crew string, cores, memGiB int) string {
	offer, ok := KeepOfferFor(f, kind)
	if !ok {
		return ""
	}
	c, ok := payoffCrew(f)
	if crew != "" {
		c, ok = crewByName(f, crew)
	}
	upkeep := 0.0
	if offer.UpkeepPerSec > 0 {
		upkeep = view.Content.PlotUpkeep(kind, cores, memGiB, landValue(offer))
	}
	line := crewRoomLine(c, ok, int64(cores)*millicores, int64(memGiB)<<30, upkeep)
	if line == "" {
		return ""
	}
	if offer.Net != "" && surveyedHere(offer, cores, memGiB) {
		line += ". " + offer.Net
	}
	return strings.ToUpper(line[:1]) + line[1:]
}

func surveyedHere(o view.KeepOffer, cores, memGiB int) bool {
	return o.CPU == strconv.Itoa(cores) && o.Memory == fmt.Sprintf("%dGi", memGiB)
}
