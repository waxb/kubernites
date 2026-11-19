package view

import (
	"github.com/waxb/kubernites/pkg/soulstone"
)

type LossView struct {
	Spoiled  float64 `json:"spoiled"`
	NoDepot  float64 `json:"noDepot"`
	NoCrew   float64 `json:"noCrew"`
	Blocked  float64 `json:"blocked"`
	QuayFull float64 `json:"quayFull"`
}

type KindView struct {
	Kind        string                         `json:"kind"`
	Name        string                         `json:"name"`
	Opens       int                            `json:"opens"`
	Open        bool                           `json:"open"`
	Waiting     float64                        `json:"waiting"`
	Oldest      float64                        `json:"oldest"`
	Cohorts     []CohortView                   `json:"cohorts"`
	ArrivalRate float64                        `json:"arrivalRate"`
	ServedRate  float64                        `json:"servedRate"`
	SpoiledRate float64                        `json:"spoiledRate"`
	TTL         float64                        `json:"ttl"`
	CrownedTTL  float64                        `json:"crownedTtl"`
	SizeMix     [soulstone.SizeClasses]float64 `json:"sizeMix"`
	Sealed      float64                        `json:"sealed"`
	Priority    float64                        `json:"priority"`
	Freshness   float64                        `json:"freshness"`
	Value       float64                        `json:"value"`
	Depot       string                         `json:"depot"`
	DepotOpen   bool                           `json:"depotOpen"`
	Crew        int                            `json:"crew"`
	CrewName    string                         `json:"crewName"`
	Image       string                         `json:"image"`
	Door        bool                           `json:"door,omitempty"`
	Losses      LossView                       `json:"losses"`
	Ports       []string                       `json:"ports"`
	LostGold    float64                        `json:"lostGold"`
	Backlog     float64                        `json:"backlog"`
}

type SealsView struct {
	Depot       string   `json:"depot"`
	DepotOpen   bool     `json:"depotOpen"`
	Crew        int      `json:"crew"`
	CrewName    string   `json:"crewName"`
	Image       string   `json:"image"`
	Waiting     float64  `json:"waiting"`
	ArrivalRate float64  `json:"arrivalRate"`
	ServedRate  float64  `json:"servedRate"`
	SpoiledRate float64  `json:"spoiledRate"`
	Losses      LossView `json:"losses"`
}

type DockingView struct {
	Ships     uint64  `json:"ships"`
	Last      float64 `json:"last"`
	LastCargo float64 `json:"lastCargo"`
	Next      float64 `json:"next"`
}

type FreightView struct {
	HarborView
	Waiting float64     `json:"waiting"`
	QuayCap float64     `json:"quayCap"`
	Docking DockingView `json:"docking"`
	Kinds   []KindView  `json:"kinds"`
	Seals   SealsView   `json:"seals"`
	Losses  LossView    `json:"losses"`
}
