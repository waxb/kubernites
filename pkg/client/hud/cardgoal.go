package hud

import (
	"fmt"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func SelectionMatches(sel Selection, target string) bool {
	kind, name, _ := strings.Cut(target, ":")
	switch kind {
	case "structure":
		return sel.Kind == "structure" && (sel.Structure.ID == name || sel.Structure.Kind == name)
	case "keep":
		return sel.Kind == "structure" && sel.Structure.Kind == view.StructKeep && sel.Structure.Object.Name == name
	case "pod":
		_, pod, found := strings.Cut(name, "/")
		if !found {
			pod = name
		}
		return (sel.Kind == "hearth" || sel.Kind == "walker") && sel.Pod.Name == pod
	case "crew":
		_, crew, _ := strings.Cut(name, "/")
		return sel.Kind == "crew" && sel.ID == name || (sel.Kind == "hearth" || sel.Kind == "walker") && sel.Pod.Crew == crew && crew != ""
	}
	return false
}

func CardGoal(f Frame) string {
	g := f.Guide
	if g.Button == "" || !SelectionMatches(f.Selection, g.Target) {
		return g.Target
	}
	key, mine := strings.CutPrefix(g.Button, ActionPrefix+":")
	if !mine {
		return g.Button
	}
	if a, ok := ActionByKey(SelectionActions(f), key); ok && !a.Disabled {
		return g.Button
	}
	return g.Target
}

func ServiceTypeLine(service string) string {
	d, ok := depotOfStructure(view.Structure{Object: view.ObjectRef{Kind: "Service", Name: service}})
	if !ok {
		return ""
	}
	return ui.Bridge(fmt.Sprintf("DOOR %d -> WORKER DOOR %d", d.Port, d.TargetPort), fmt.Sprintf("ClusterIP  PORT %d -> %d", d.Port, d.TargetPort))
}

func withServiceType(s view.Structure, lines []Line) []Line {
	text := ServiceTypeLine(s.Object.Name)
	if text == "" {
		return lines
	}
	return append([]Line{{Text: text, Clr: ui.Palette.Blue}}, lines...)
}
