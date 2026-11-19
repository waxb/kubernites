package soulstone

import (
	"fmt"
	"sort"
)

const (
	SizeParcel = iota
	SizeCrate
	SizeChest
	SizeContainer
	SizeClasses
)

const (
	KindParcel   = "parcel"
	KindLedger   = "ledger"
	KindCrate    = "crate"
	KindDispatch = "dispatch"
	KindSealed   = "sealed"
)

const (
	TradePort int32 = 8080
	DepotPort int32 = 80
)

type Depot struct {
	Kind       string
	Service    string
	Crew       string
	Image      string
	Port       int32
	TargetPort int32
}

var depots = []Depot{
	{Kind: KindParcel, Service: "parcels", Crew: "porters", Image: "porter:1.0", Port: DepotPort, TargetPort: TradePort},
	{Kind: KindLedger, Service: "ledgers", Crew: "clerks", Image: "clerk:1.0", Port: DepotPort, TargetPort: TradePort},
	{Kind: KindCrate, Service: "crates", Crew: "haulers", Image: "hauler:1.0", Port: DepotPort, TargetPort: TradePort},
	{Kind: KindDispatch, Service: "dispatches", Crew: "couriers", Image: "courier:1.0", Port: DepotPort, TargetPort: TradePort},
	{Kind: KindSealed, Service: "seals", Crew: "sealwrights", Image: "sealwright:1.0", Port: DepotPort, TargetPort: TradePort},
}

func clampSize(size int) int {
	return max(SizeParcel, min(size, SizeClasses-1))
}

func (p Profile) BoxCPU(size int) float64 {
	return p.BoxMilliS[clampSize(size)]
}

func (p Profile) BoxMemory(size int) float64 {
	return p.BoxMemMi[clampSize(size)]
}

func (p Profile) Serves(kind string) bool {
	for _, k := range p.Kinds {
		if k == kind {
			return true
		}
	}
	return false
}

func Trades() []Profile {
	out := []Profile{}
	for _, p := range Playable() {
		if p.Trade != "" && p.Tag == "1.0" {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Cost < out[j].Cost })
	return out
}

func Depots() []Depot {
	return append([]Depot(nil), depots...)
}

func findDepot(match func(Depot) bool) (Depot, bool) {
	for _, d := range depots {
		if match(d) {
			return d, true
		}
	}
	return Depot{}, false
}

func DepotFor(kind string) (Depot, bool) {
	return findDepot(func(d Depot) bool { return d.Kind == kind })
}

func DepotOfService(service string) (Depot, bool) {
	return findDepot(func(d Depot) bool { return d.Service == service })
}

func DepotOfCrew(crew string) (Depot, bool) {
	return findDepot(func(d Depot) bool { return d.Crew == crew })
}

var boxNouns = map[string][2]string{
	KindParcel: {"parcel", "parcels"}, KindLedger: {"ledger", "ledgers"}, KindCrate: {"crate", "crates"},
	KindDispatch: {"dispatch", "dispatches"}, KindSealed: {"sealed box", "sealed boxes"},
}

func (p Profile) ServedLine(n int, every float64) string {
	noun := [2]string{"box", "boxes"}
	if len(p.Kinds) > 0 {
		if known, ok := boxNouns[p.Kinds[0]]; ok {
			noun = known
		}
	}
	word := noun[1]
	if n == 1 {
		word = noun[0]
	}
	return fmt.Sprintf("carried %d %s in the last %gs", n, word, every)
}
