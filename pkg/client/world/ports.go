package world

import (
	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	StakeKind   = "stake"
	houseKind   = "house"
	landingKind = "landing"
	flatLevel   = 1
)

var portArtKinds = map[string]string{
	"quay": "deck", "pier-x": "pier-x", "pier-y": "pier-y", houseKind: houseKind, "sign": "sign", "bollard": "bollard",
	"crane": "crane", "warehouse": "warehouse", "lighthouse": "lighthouse", StakeKind: StakeKind, landingKind: landingKind,
}

var levelFree = map[string]bool{"sign": true, "bollard": true, StakeKind: true, landingKind: true}

var havenFallback = map[string]bool{"quay": true, "pier-x": true, "pier-y": true, "sign": true, "bollard": true, "crane": true, "warehouse": true, "lighthouse": true, "crates": true}

func PortLevel(h view.Haven) int {
	switch {
	case !h.Open:
		return 0
	case h.Level > 0:
		return min(h.Level, art.PortLevels)
	}
	return min(int(h.Tier)+1, art.PortLevels)
}

func PortPieceNames(prop HavenProp, level, age int) []string {
	out := make([]string, 0, 2)
	if kind, ok := portArtKinds[prop.Kind]; ok {
		if levelFree[prop.Kind] {
			level = flatLevel
		}
		out = append(out, art.PortLook{Kind: kind, Level: max(level, flatLevel), Age: age, Variant: prop.Variant}.Name())
	}
	if havenFallback[prop.Kind] {
		out = append(out, gfx.HavenName(prop.Kind, pieceVariant(prop.Kind, prop.Variant)))
	}
	return out
}

type pieceKey struct {
	kind                string
	variant, level, age int
}

func (h *Harbor) pieceNames(prop HavenProp, level, age int) []string {
	k := pieceKey{kind: prop.Kind, variant: prop.Variant, level: level, age: age}
	if names, ok := h.pieceMemo[k]; ok {
		return names
	}
	if h.pieceMemo == nil {
		h.pieceMemo = map[pieceKey][]string{}
	}
	names := PortPieceNames(prop, level, age)
	h.pieceMemo[k] = names
	return names
}

func stakeProps(p HavenPlan) []HavenProp {
	return []HavenProp{{Kind: StakeKind, At: p.Quay}}
}
