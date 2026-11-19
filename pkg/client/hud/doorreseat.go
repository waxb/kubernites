package hud

import (
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	keyReseat   = "reseat"
	doorClassOf = "porter:1.0"
)

func withDoorReseat(acts []CardAction, p model.PodView, f Frame) []CardAction {
	if !p.Static || p.Namespace != defaultNS || p.Crew != "porters" {
		return acts
	}
	choices := []CardAction{}
	for _, image := range OwnedClasses(f, doorClassOf) {
		if image != p.Image {
			choices = append(choices, CardAction{Key: keyReseat + ":" + image, Label: "Switch to " + image,
				Tip: "The Reeve re-reads the door's manifest and replaces the static Pod with " + image + ".",
				Act: view.Action{Action: view.ActReseatDoor, Image: image}})
		}
	}
	reseat := CardAction{Key: keyReseat, Label: "RESEAT", Choices: choices,
		Tip: "Switch the door porter to another owned class: a static Pod changes only when the file its Keep reads changes."}
	if len(choices) == 0 {
		reseat.Disabled, reseat.Tip = true, "No other porter class is owned yet: the door porter keeps its class."
	}
	return append(acts, reseat)
}
