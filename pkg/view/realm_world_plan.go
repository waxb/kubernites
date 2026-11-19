package view

import (
	"errors"
	"fmt"
)

const minWorldSide = 48

const minHomeRadius = 27

const minOuterRadius = 9

const maxIsles = 250

const liveJitter = 1

type IslePlan struct {
	Name   string
	Zone   string
	FX, FY float64
	Radius int
}

type WorldPlan struct {
	Width, Height int
	Isles         []IslePlan
}

func GenerateWorld(seed uint64, plan WorldPlan) (Site, error) {
	if len(plan.Isles) == 0 {
		return Site{}, errors.New("a world needs at least one isle")
	}
	if len(plan.Isles) > maxIsles {
		return Site{}, fmt.Errorf("a world holds at most %d isles, not %d", maxIsles, len(plan.Isles))
	}
	if plan.Width < minWorldSide || plan.Height < minWorldSide || plan.Width > MaxSiteSide || plan.Height > MaxSiteSide {
		return Site{}, fmt.Errorf("a world is between %d and %d tiles a side, not %dx%d", minWorldSide, MaxSiteSide, plan.Width, plan.Height)
	}
	wp := WorldLayout{Width: plan.Width, Height: plan.Height, noSpots: true, shaped: true}
	for i, is := range plan.Isles {
		slot := IsleSlot{Fx: is.FX, Fy: is.FY, Radius: max(is.Radius, minOuterRadius), Jitter: liveJitter, name: is.Name, zone: is.Zone}
		if i == 0 {
			slot.Radius = max(slot.Radius, minHomeRadius)
			wp.Home = slot
			continue
		}
		wp.Outer = append(wp.Outer, slot)
	}
	site, _ := GenerateLayout(seed, wp)
	return site, nil
}
