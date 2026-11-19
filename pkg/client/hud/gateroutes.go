package hud

import (
	"fmt"
	"image/color"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func GateRouteLines(f Frame) []Line {
	routes := f.State.Routes
	if len(routes) == 0 {
		return []Line{{Text: ui.Bridge("No road yet: open a depot and the Gate sorts its boxes to it.", "No routes yet: open a depot (a Service) to attach one."), Clr: ui.Palette.TextDim}}
	}
	out := []Line{{Text: ui.Bridge(fmt.Sprintf("ROADS: %d", len(routes)), fmt.Sprintf("ROUTES: %d", len(routes))), Clr: ui.Palette.Gold}}
	for _, r := range routes {
		out = append(out, Line{Text: ui.Bridge(storyRoute(r), k8sRoute(r)), Clr: routeColor(r)})
	}
	return out
}

func storyRoute(r view.RouteView) string {
	if !r.Open {
		return fmt.Sprintf("%s: the road waits for its depot", r.Service)
	}
	return fmt.Sprintf("%s: to their depot, %d huts ready", r.Service, r.Ready)
}

func k8sRoute(r view.RouteView) string {
	if !r.Open {
		return fmt.Sprintf("%s %s: %s :%d, no backend yet", r.Route, r.Service, r.Listener, r.Port)
	}
	return fmt.Sprintf("%s %s: %s :%d -> %s (%d ready)", r.Route, r.Service, r.Listener, r.Port, r.Service, r.Ready)
}

func routeColor(r view.RouteView) color.RGBA {
	if r.Open {
		return ui.Palette.Text
	}
	return ui.Palette.TextDim
}
