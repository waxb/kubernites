package view

const StructPlaque = "plaque"

func PlaqueDef() BuildDef {
	return BuildDef{Kind: StructPlaque, W: 1, H: 2, Placement: PlaceForum}
}
