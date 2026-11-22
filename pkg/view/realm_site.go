package view

import (
	"errors"
)

const MaxSiteSide = 1024

type TileKind uint8

const (
	TileWater TileKind = iota
	TileShallows
	TileSand
	TileGrass
	TileMeadow
	TileForest
	TileRock
	TileRoad
	TilePlaza
	TileForum
)

const StateConstruction = "construction"

const StateReady = "ready"

const StateDark = "dark"

const StateRuined = "ruined"

const StateJoining = "joining"

const StructGate = "gate"

const StructTreasury = "treasury"

const StructKeep = "keep"

const StructCodex = "codex"

const StructChancery = "chancery"

const StructQuartermaster = "quartermaster"

const StructStewards = "stewards-hall"

const StructHerald = "herald"

const StructWayfinder = "wayfinder"

const StructGuildhall = "guildhall"

const StructObservatory = "observatory"

const StructArchive = "archive"

const StructWarHorn = "warhorn"

type ObjectRef struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
	Resource  string `json:"resource,omitempty"`
}

type Structure struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Variant   string    `json:"variant,omitempty"`
	X         int       `json:"x"`
	Y         int       `json:"y"`
	W         int       `json:"w"`
	H         int       `json:"h"`
	State     string    `json:"state"`
	Progress  float64   `json:"progress"`
	Label     string    `json:"label"`
	Object    ObjectRef `json:"object"`
	Plot      int       `json:"plot,omitempty"`
	Territory int       `json:"territory,omitempty"`
	Class     string    `json:"class,omitempty"`
	CPU       int       `json:"cpu,omitempty"`
	MemoryGiB int       `json:"memoryGiB,omitempty"`
	Instance  string    `json:"instance,omitempty"`
	Paid      float64   `json:"paid,omitempty"`
}

type Island struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Zone   string `json:"zone"`
	Owned  bool   `json:"owned"`
	Center Point  `json:"center"`
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Haven struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Quay     Point   `json:"quay"`
	Out      Point   `json:"out"`
	Pier     int     `json:"pier"`
	Sea      Point   `json:"sea"`
	Open     bool    `json:"open"`
	OpenedAt float64 `json:"openedAt,omitempty"`
	Tier     uint8   `json:"tier"`
	TierAt   float64 `json:"tierAt,omitempty"`
	Share    float64 `json:"share"`
	Level    int     `json:"level,omitempty"`
	Moored   int     `json:"berths,omitempty"`
}

type Site struct {
	Gate       Point       `json:"gate"`
	Plaza      Point       `json:"plaza"`
	TowerSpot  Point       `json:"towerSpot"`
	Width      int         `json:"width"`
	Height     int         `json:"height"`
	Tiles      []TileKind  `json:"tiles"`
	Elevation  []uint8     `json:"elevation"`
	Structures []Structure `json:"structures"`
	Islands    []Island    `json:"islands,omitempty"`
	Region     []uint8     `json:"region,omitempty"`
	Power      []uint8     `json:"power,omitempty"`
	Wear       []uint16    `json:"wear,omitempty"`
	Havens     []Haven     `json:"havens,omitempty"`
	Spots      []Spot      `json:"spots,omitempty"`
	Forum      *Forum      `json:"forum,omitempty"`
	NoHarbor   bool        `json:"noHarbor,omitempty"`
	Plots      []Point     `json:"plots,omitempty"`
	Roads      *RoadCache  `json:"-"`
	Land       *LandCache  `json:"-"`
}

func (s Site) Tile(x, y int) TileKind {
	if x < 0 || y < 0 || x >= s.Width || y >= s.Height {
		return TileWater
	}
	return s.Tiles[y*s.Width+x]
}

func (s Site) ElevationAt(x, y int) uint8 {
	if x < 0 || y < 0 || x >= s.Width || y >= s.Height || len(s.Elevation) == 0 {
		return 0
	}
	return s.Elevation[y*s.Width+x]
}

func (s Site) Clone() Site {
	next := s
	next.Tiles = append([]TileKind(nil), s.Tiles...)
	next.Elevation = append([]uint8(nil), s.Elevation...)
	next.Structures = append([]Structure(nil), s.Structures...)
	next.Islands = append([]Island(nil), s.Islands...)
	next.Region = append([]uint8(nil), s.Region...)
	next.Power = append([]uint8(nil), s.Power...)
	next.Wear = append([]uint16(nil), s.Wear...)
	next.Havens = append([]Haven(nil), s.Havens...)
	next.Spots = append([]Spot(nil), s.Spots...)
	next.Roads, next.Land = nil, nil
	return next
}

func (s Site) Index(x, y int) (int, bool) {
	if x < 0 || y < 0 || x >= s.Width || y >= s.Height {
		return 0, false
	}
	return y*s.Width + x, true
}

func (s Site) IslandAt(x, y int) (Island, bool) {
	i, ok := s.Index(x, y)
	if !ok || i >= len(s.Region) || s.Region[i] == 0 || int(s.Region[i]) > len(s.Islands) {
		return Island{}, false
	}
	return s.Islands[s.Region[i]-1], true
}

func (s Site) PowerAt(x, y int) uint8 {
	if i, ok := s.Index(x, y); ok && i < len(s.Power) {
		return s.Power[i]
	}
	return 0
}

func (s Site) WearAt(x, y int) uint16 {
	if i, ok := s.Index(x, y); ok && i < len(s.Wear) {
		return s.Wear[i]
	}
	return 0
}

func (s Site) RoadAt(x, y int) int {
	if s.Tile(x, y) == TileRoad {
		return 3
	}
	return RoadLevel(s.WearAt(x, y))
}

func (s Site) Restored() (Site, error) {
	n := s.Width * s.Height
	if s.Width <= 0 || s.Height <= 0 || s.Width > MaxSiteSide || s.Height > MaxSiteSide || len(s.Tiles) != n {
		return Site{}, errors.New("save has no valid site map")
	}
	next := s.Clone()
	if len(next.Elevation) != n {
		next.Elevation = make([]uint8, n)
	}
	if len(next.Islands) == 0 || len(next.Region) != n {
		next.Islands, next.Region = legacyIslands(next)
	}
	next.Power = nil
	if len(next.Wear) != n {
		next.Wear = nil
	}
	return next, nil
}
