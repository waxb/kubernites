package view

import (
	"strings"
)

const DepotPrefix = "depot-"

func DepotStructure(kind string) string {
	return DepotPrefix + kind
}

func IsDepotStructure(s Structure) bool {
	return strings.HasPrefix(s.Kind, DepotPrefix)
}

func DepotDef(kind string) BuildDef {
	return BuildDef{Kind: DepotStructure(kind), W: 1, H: 2, Placement: PlaceForum}
}
