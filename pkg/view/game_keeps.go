package view

const KeepTypeTower = "tower"

const KeepTypeStone = "stone-keep"

const KeepTypeGreat = "great-keep"

const KeepTypeCitadel = "citadel-keep"

const TowerTerritory = 9

const TowerClaim = 9

const StoneKeepBase = 100.0

type KeepType struct {
	Kind         string
	KeepType     string
	Variant      string
	Name         string
	Size         string
	CPU          string
	Memory       string
	MinPower     uint8
	Territory    int
	Claim        int
	Yield        float64
	MaxCPU       int
	MaxMemoryGiB int
	Rune         string
	Base         float64 `json:"-"`
	Growth       float64 `json:"-"`
}

var KeepTypes = []KeepType{
	{Kind: "keep", KeepType: KeepTypeStone, Variant: "stone", Name: "Stone Keep", Size: "xlarge", CPU: "4", Memory: "8Gi", Territory: 5, Claim: 4, Yield: 1, MaxCPU: 8, MaxMemoryGiB: 16,
		Base: StoneKeepBase, Growth: 1.7},
	{Kind: "great-keep", KeepType: KeepTypeGreat, Variant: "great", Name: "Great Keep", Size: "2xlarge", CPU: "8", Memory: "16Gi", Territory: 7, Claim: 6, Yield: 1, MaxCPU: 16, MaxMemoryGiB: 32,
		Rune: "great-keeps", Base: 4000, Growth: 1.8},
	{Kind: "citadel-keep", KeepType: KeepTypeCitadel, Variant: "citadel", Name: "Citadel Keep", Size: "4xlarge", CPU: "16", Memory: "32Gi", Territory: 9, Claim: 8, Yield: 1, MaxCPU: 32, MaxMemoryGiB: 64,
		Rune: "citadel-keeps", Base: 40000, Growth: 2},
}

var towerType = KeepType{Kind: "keep", KeepType: KeepTypeTower, Variant: "tower", Name: "Lone Tower", Size: "large", CPU: "2", Memory: "4Gi",
	Territory: TowerTerritory, Claim: TowerClaim, Yield: 1, MaxCPU: TowerHands, MaxMemoryGiB: 4}

func TowerType() KeepType {
	return towerType
}

func KeepTier(kind string, first bool) (KeepType, bool) {
	if kind == "keep" && first {
		return towerType, true
	}
	k, ok := KeepTypeOf(kind)
	if !ok || k.Kind != kind {
		return KeepType{}, false
	}
	return k, true
}

func KeepTypeOf(name string) (KeepType, bool) {
	for _, k := range KeepTypes {
		if k.Kind == name || k.KeepType == name || k.Variant == name {
			return k, true
		}
	}
	return KeepType{}, false
}
