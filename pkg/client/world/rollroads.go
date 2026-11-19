package world

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	rollDimAlpha = 90
	rollLitAlpha = 200
)

var rollDim = color.RGBA{150, 150, 160, 255}

type rollRoad struct {
	pod   string
	from  Vec
	to    Vec
	ready bool
}

func selectedDepot(sc Scene) (view.Structure, model.CrewChain, bool) {
	for _, c := range sc.Crews {
		if c.Depot == "" {
			continue
		}
		for _, s := range sc.Site.Structures {
			if s.Object.Kind != "Service" || s.Object.Name != c.Depot {
				continue
			}
			if sc.Selected == s.ID || sc.Selected == c.Namespace+"/"+c.Name {
				return s, c, true
			}
		}
	}
	return view.Structure{}, model.CrewChain{}, false
}

func rollRoads(sc Scene, homes map[string]Vec) []rollRoad {
	stall, crew, ok := selectedDepot(sc)
	if !ok {
		return nil
	}
	shut := indexWatch(sc.Watch).huts
	porch := Porch(stall)
	out := []rollRoad{}
	for _, p := range sc.Pods {
		home, housed := homes[p.Name]
		if p.Crew != crew.Name || p.Namespace != crew.Namespace || !housed || p.Terminating {
			continue
		}
		out = append(out, rollRoad{pod: p.Name, from: porch, to: hutDoor(home), ready: p.Ready && !shut[p.Name].Shut})
	}
	return out
}

func (r *Renderer) drawRollRoads(dst *ebiten.Image, cam render.Camera, sc Scene) {
	var lit, dim vector.Path
	roads := rollRoads(sc, r.Homes)
	for _, road := range roads[:min(len(roads), maxFanLines)] {
		fx, fy := ScreenPointOf(cam, sc.Site, road.from)
		tx, ty := ScreenPointOf(cam, sc.Site, road.to)
		if road.ready {
			dashes(&lit, fx, fy, tx, ty, 0)
			continue
		}
		dashes(&dim, fx, fy, tx, ty, fanDash)
	}
	width := float32(max(cam.Scale(), 1) * fanWidth)
	strokeTinted(dst, &dim, width, ui.Tint(rollDim, rollDimAlpha))
	strokeTinted(dst, &lit, width, ui.Tint(fanColor, rollLitAlpha))
}
