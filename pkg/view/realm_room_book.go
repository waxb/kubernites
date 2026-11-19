package view

import (
	"math"
)

const OpenRoom = math.MaxInt32

type RoomBook struct {
	Keeps []KeepRoom `json:"keeps"`
	Owed  []OwedRoom `json:"owed"`
}

type KeepRoom struct {
	Name   string         `json:"name"`
	Free   ResourcePair   `json:"free"`
	Spare  ResourcePair   `json:"spare"`
	Seated map[string]int `json:"seated,omitempty"`
}

type OwedRoom struct {
	Crew  string       `json:"crew"`
	Count int          `json:"count"`
	Surge int          `json:"surge,omitempty"`
	Asks  ResourcePair `json:"asks"`
	Takes ResourcePair `json:"takes"`
}

type RoomFit struct {
	Seated int    `json:"seated"`
	Fed    int    `json:"fed"`
	Limit  string `json:"limit,omitempty"`
	Short  int    `json:"short,omitempty"`
}

func (f RoomFit) Safe() int { return max(0, min(f.Seated, f.Fed)) }

type Work struct {
	Busy  int          `json:"busy"`
	Takes ResourcePair `json:"takes"`
	Idle  ResourcePair `json:"idle"`
}

func AllBusy(takes ResourcePair) Work {
	return Work{Busy: OpenRoom, Takes: takes, Idle: takes}
}

func (b RoomBook) Fit(crew string, asks, takes ResourcePair) RoomFit {
	return b.FitWork(crew, asks, AllBusy(takes))
}

func (b RoomBook) FitWork(crew string, asks ResourcePair, w Work) RoomFit {
	keeps, short := b.place(crew, true)
	fit := fitIn(keeps, asks, w)
	fit.Short = short
	return fit
}

func (b RoomBook) Repledged(crew string, count int, old, asks, current ResourcePair, w Work) RoomFit {
	keeps := b.placed(crew, false)
	for i, k := range keeps {
		n := int64(k.Seated[crew])
		keeps[i].Free = AddPair(k.Free, scalePair(old, n))
		keeps[i].Spare = AddPair(k.Spare, scalePair(current, n))
	}
	fit := fitWith(keeps, asks, func(i int) ResourcePair {
		if i < count {
			return current
		}
		return w.Take(i - count)
	})
	return RoomFit{Seated: fit.Seated - count, Fed: fit.Fed - count, Limit: fit.Limit}
}

func (w Work) Take(i int) ResourcePair {
	if i < w.Busy {
		return w.Takes
	}
	return w.Idle
}

func (b RoomBook) FreeAfterOwed() ResourcePair {
	out := ResourcePair{}
	for _, k := range b.placed("", true) {
		out = AddPair(out, ResourcePair{CPU: max(0, k.Free.CPU), Memory: max(0, k.Free.Memory)})
	}
	return out
}

func (b RoomBook) OwedTo(crew string) int {
	for _, o := range b.Owed {
		if o.Crew == crew {
			return o.Count
		}
	}
	return 0
}

func (b RoomBook) placed(crew string, own bool) []KeepRoom {
	keeps, _ := b.place(crew, own)
	return keeps
}

func (b RoomBook) place(crew string, own bool) ([]KeepRoom, int) {
	keeps := make([]KeepRoom, len(b.Keeps))
	copy(keeps, b.Keeps)
	short := 0
	for _, o := range b.Owed {
		if o.Crew == crew && !own {
			continue
		}
		for n := range o.Count {
			i, ok := Roomiest(keeps, o.Asks)
			if !ok {
				if o.Crew == crew && n < o.Count-o.Surge {
					short++
				}
				continue
			}
			keeps[i].Free = SubPair(keeps[i].Free, o.Asks)
			keeps[i].Spare = SubPair(keeps[i].Spare, o.Takes)
		}
	}
	return keeps, short
}

func Roomiest(keeps []KeepRoom, asks ResourcePair) (int, bool) {
	best, found := 0, false
	for i, k := range keeps {
		if k.Free.CPU < asks.CPU || k.Free.Memory < asks.Memory {
			continue
		}
		if !found || k.Free.CPU > keeps[best].Free.CPU {
			best, found = i, true
		}
	}
	return best, found
}

const seatCap = 100

func fitIn(keeps []KeepRoom, asks ResourcePair, w Work) RoomFit {
	return fitWith(keeps, asks, w.Take)
}

func fitWith(keeps []KeepRoom, asks ResourcePair, takeOf func(int) ResourcePair) RoomFit {
	if asks.CPU <= 0 && asks.Memory <= 0 {
		return RoomFit{Seated: OpenRoom, Fed: OpenRoom, Limit: FlowCPU}
	}
	out := RoomFit{Limit: FlowCPU}
	placed, food, housing := 0, 0, 0
	for _, k := range keeps {
		seats := min(RoomFor(k.Free, asks), seatCap)
		out.Seated += seats
		spare := k.Spare
		for range seats {
			take := takeOf(placed)
			placed++
			switch {
			case spare.CPU < take.CPU:
				food++
			case spare.Memory < take.Memory:
				housing++
			default:
				out.Fed++
				spare = SubPair(spare, take)
			}
		}
	}
	if housing > food {
		out.Limit = FlowMemory
	}
	return out
}

func AddPair(a, b ResourcePair) ResourcePair {
	return ResourcePair{CPU: a.CPU + b.CPU, Memory: a.Memory + b.Memory}
}

func SubPair(a, b ResourcePair) ResourcePair {
	return ResourcePair{CPU: a.CPU - b.CPU, Memory: a.Memory - b.Memory}
}

func scalePair(a ResourcePair, n int64) ResourcePair {
	return ResourcePair{CPU: a.CPU * n, Memory: a.Memory * n}
}
