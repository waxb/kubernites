package view

const StallLift = 4

const tallStallLift = 5

var structureLift = map[string]int{
	"keep-tower": 10, "keep-stone": 8, "keep-great": 12, "keep-citadel": 15, "codex": 9, "chancery": 8, "quartermaster": 5,
	"stewards-hall": 6, "herald": tallStallLift, "wayfinder": 5, "guildhall": StallLift, "observatory": 7, "gate": 7, "treasury": 4,
	"archive": tallStallLift, "warhorn": 10, "shrine": 5, "signpost": 5, "foreign": 7, "harbor-gate": 7, "sea-chancery": 11,
}

func StructureLift(kind, variant string) int {
	if StallKind(kind) && !isKnownLift(kind) {
		return StallLift
	}
	if kind == StructKeep {
		if variant == "" {
			variant = "stone"
		}
		kind = "keep-" + variant
	}
	if l, ok := structureLift[kind]; ok {
		return l
	}
	return structureLift["keep-stone"]
}

func isKnownLift(kind string) bool {
	_, ok := structureLift[kind]
	return ok
}

func (a Rect) Columns() (int, int) {
	return a.X0 - a.Y1 - 1, a.X1 - a.Y0 + 1
}

func (a Rect) front() int {
	return a.X1 + a.Y1
}

func Hides(a Rect, liftA int, b Rect, liftB int) bool {
	alo, ahi := a.Columns()
	blo, bhi := b.Columns()
	if alo >= bhi || blo >= ahi {
		return false
	}
	if diff := b.front() - a.front(); diff >= 0 {
		return diff < liftB
	}
	return a.front()-b.front() < liftA
}
