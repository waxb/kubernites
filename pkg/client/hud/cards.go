package hud

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	barLineW = 60
	barMinW  = 24
)

const joiningText = "JOINING: NOT READY YET"

var darkTexts = map[string]string{
	view.StructKeep:    "DARK: THE NODE IS NOT READY",
	view.StructWarHorn: "DARK: THE HORN IS SILENT",
}

func DarkText(kind string) string {
	if text, ok := darkTexts[kind]; ok {
		return text
	}
	return "DARK"
}

func KindLines(s view.Structure, f Frame) []Line {
	switch s.Kind {
	case view.StructArchive:
		return withServiceType(s, DepotLines(s, f))
	case view.StructWarHorn:
		return HornLines(s, f.Horns)
	case view.StructGate:
		return GateRouteLines(f)
	case view.StructGuildhall:
		return withServiceType(s, append(DepotLines(s, f), GuildhallLines(s, f)...))
	case view.StructKeep:
		if f.Live {
			return LiveKeepLines(s, f)
		}
		return taughtLines(Card{Lines: append(KeepSupplyLines(s, f), KeepLines(s)...)}, f).Lines
	case view.StructVault:
		return VaultLines(s, f)
	case view.StructCodex:
		if !f.Revealed(CodexReveal) {
			return nil
		}
		return CodexLines(f.Caesar.Codex)
	}
	if strings.HasPrefix(s.Kind, view.DepotStructure("")) {
		return withServiceType(s, append(DepotLines(s, f), GuildhallLines(s, f)...))
	}
	return nil
}

type depotFlow struct {
	name, crew               string
	crewN                    int
	arrival, served, waiting float64
}

func depotFlowOf(fv view.FreightView, service string) (depotFlow, bool) {
	for _, k := range fv.Kinds {
		if k.Depot == service {
			return depotFlow{name: k.Name, crew: k.CrewName, crewN: k.Crew, arrival: k.ArrivalRate, served: k.ServedRate, waiting: k.Waiting}, true
		}
	}
	if s := fv.Seals; s.Depot == service && service != "" {
		return depotFlow{name: "sealed boxes", crew: s.CrewName, crewN: s.Crew, arrival: s.ArrivalRate, served: s.ServedRate, waiting: s.Waiting}, true
	}
	return depotFlow{}, false
}

func DepotLines(s view.Structure, f Frame) []Line {
	d, ok := depotFlowOf(f.State.Freight, s.Object.Name)
	if !ok {
		return nil
	}
	share := CarriedShare(d.arrival, d.served, d.waiting)
	out := []Line{{Text: "CARRIED", Clr: ui.Palette.Text, Bar: true, Value: share, Fill: ShareColor(share),
		Suffix: CarriedSuffix(d.served, d.arrival)}}
	if d.crewN == 0 {
		return append(out, Line{Text: fmt.Sprintf("No %s answer: %s pile up on the quay", d.crew, d.name), Clr: ui.Palette.RedText})
	}
	waiting := int(d.waiting + .5)
	carry := fmt.Sprintf("%d %s carry", d.crewN, d.crew)
	if n := rollOutsiders(f, s.Object.Name, d.crew); n > 0 {
		carry += fmt.Sprintf(" (%d outside the crew)", n)
	}
	out = append(out, Line{Text: fmt.Sprintf("%s, %s waiting", carry, plural(waiting, singularOf(d.name), d.name)), Clr: ui.Palette.Text})
	if c, ok := CrewOf(f.State.Crews, s.Object.Name, d.crew); ok && SeasonLine(c) != "" {
		out = append(out, Line{Text: SeasonLine(c), Clr: ui.Palette.GoldBright})
	}
	return out
}

func rollOutsiders(f Frame, service, crew string) int {
	if !slices.ContainsFunc(f.Crews, func(c model.CrewChain) bool { return c.Name == crew && !c.Bare }) {
		return 0
	}
	pods := make(map[string]model.PodView, len(f.Pods))
	for _, p := range f.Pods {
		pods[p.Namespace+"/"+p.Name] = p
	}
	n := 0
	for _, r := range f.Rolls {
		if r.Service != service {
			continue
		}
		for _, m := range r.Members {
			if p, ok := pods[r.Namespace+"/"+m.Name]; ok && m.Ready && !strings.HasPrefix(p.Owner, "ReplicaSet/") {
				n++
			}
		}
	}
	return n
}

func singularOf(name string) string {
	for _, suffix := range []string{"ches", "shes", "xes"} {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, "es")
		}
	}
	return strings.TrimSuffix(name, "s")
}

func hornFor(s view.Structure, horns []model.Horn) (model.Horn, bool) {
	for _, h := range horns {
		if h.Name == s.Object.Name {
			return h, true
		}
	}
	return model.Horn{}, false
}

func HornLines(s view.Structure, horns []model.Horn) []Line {
	h, ok := hornFor(s, horns)
	if !ok {
		return []Line{{Text: "No War Horn is written for this crew.", Clr: ui.Palette.TextDim}}
	}
	out := []Line{{Text: fmt.Sprintf("%d -> %d huts, food %d%% of target %d%%", h.Current, h.Desired, h.Utilization, h.Goal), Clr: ui.Palette.Text}}
	switch {
	case !h.Sounding:
		out = append(out, Line{Text: h.Message, Clr: ui.Palette.RedText})
	case h.Current >= h.Max:
		out = append(out, Line{Text: fmt.Sprintf("at max (%d): add Keeps or raise max", h.Max), Clr: ui.Palette.GoldBright})
	}
	return out
}

func (h *HUD) drawBarLine(dst *ebiten.Image, l Line, x, y, w int) {
	h.Font.Draw(dst, l.Text, x, y, 1, l.Clr)
	bx, bw := x+h.Font.Span(l.Text, 1), barWidth(l.Text, l.Suffix, w)
	h.drawSplitBar(dst, ui.Rect{X: bx, Y: y + 2, W: bw, H: 5}, l)
	h.Font.Draw(dst, ui.Clip(l.Suffix, BarRoom(l.Text, l.Suffix, x, w)), bx+bw+4, y, 1, ui.Palette.Cream)
}

func barWidth(label, suffix string, w int) int {
	free := w - ui.Font{}.Span(label, 1) - 4 - ui.Font{}.Width(suffix, 1)
	return min(barLineW, max(free, barMinW))
}

func BarRoom(label, suffix string, x, w int) int {
	bx := x + ui.Font{}.Span(label, 1)
	return ui.MaxChars(x+w-bx-barWidth(label, suffix, w)-4, 1)
}

func CarriedSuffix(served, arrival float64) string {
	return Rate(served) + "/" + Rate(arrival)
}
