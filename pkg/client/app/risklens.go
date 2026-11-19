package app

import (
	"fmt"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

type LensSources struct {
	Site    view.Site
	Metrics view.MetricsSnapshot
	Caesar  model.Caesar
}

func Share(part, whole float64) float64 {
	if whole <= 0 {
		return 0
	}
	return part / whole
}

func LensOf(mode string, src LensSources) world.Lens {
	l := world.Lens{Mode: mode, Heat: map[string]float64{}, Labels: map[string]string{}}
	if mode == world.LensCodex {
		codexLens(l, src)
		return l
	}
	for _, s := range src.Site.Structures {
		if s.Kind != view.StructKeep || s.Object.Name == "" {
			continue
		}
		if heat, label, ok := keepHeat(mode, s.Object.Name, src); ok {
			l.Heat[s.ID], l.Labels[s.ID] = heat, label
		}
	}
	return l
}

func keepHeat(mode, node string, src LensSources) (float64, string, bool) {
	n, ok := src.Metrics.Nodes[node]
	alloc := float64(n.CPUAllocatableMilli)
	switch mode {
	case world.LensLabor:
		if src.Metrics.Augury {
			v := Share(n.CPUMilli, alloc)
			return v, fmt.Sprintf("FOOD %d%%", int(v*100+.5)), ok && alloc > 0
		}
		v := Share(float64(n.CPURequestedMilli), alloc)
		return v, fmt.Sprintf("PLEDGED %d%%", int(v*100+.5)), ok && alloc > 0
	case world.LensUnhoused:
		free := max(alloc-float64(n.CPURequestedMilli), 0) / 1000
		return Share(float64(n.CPURequestedMilli), alloc), "ROOM FOR " + strings.ToUpper(view.Rations(free*view.RationsPerCPU)), ok && alloc > 0
	case world.LensMidden:
		m, found := src.Caesar.Middens[node]
		if m.Pressure {
			return 1, ui.Bridge("CELLAR WARD", "DISK PRESSURE"), found
		}
		return m.Fill(), fmt.Sprintf("GROUND %d%%", int(m.Fill()*100+.5)), found
	}
	return 0, "", false
}

func codexLens(l world.Lens, src LensSources) {
	c := src.Caesar.Codex
	s, ok := world.FirstOfKind(src.Site, view.StructCodex, false)
	if !ok || !c.Known() {
		return
	}
	label := fmt.Sprintf("CODEX %d%%", int(c.Fill()*100+.5))
	heat := c.Fill()
	if c.Alarm {
		label, heat = "CODEX FULL", 1
	}
	l.Heat[s.ID], l.Labels[s.ID] = heat, label
}
