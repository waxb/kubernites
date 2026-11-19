package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	KeyWarden       = "warden"
	KeyDescribeDeed = "describe-deed"
	KeyGetDeed      = "get-deed"
	courierImage    = "courier:"
	claimNS         = "default/"
)

var vaultCells = map[string]int{KeyDescribe: 0, KeyGet: 1, KeyDescribeDeed: 3, KeyGetDeed: 4}

func couriersCame(f Frame) bool {
	for _, p := range f.Pods {
		if strings.HasPrefix(p.Image, courierImage) {
			return true
		}
	}
	return false
}

func keepViewOf(f Frame, node string) (view.KeepView, bool) {
	for _, k := range f.State.Keeps {
		if k.Name == node {
			return k, true
		}
	}
	return view.KeepView{}, false
}

func WardenAction(node string, f Frame) (CardAction, bool) {
	k, ok := keepViewOf(f, node)
	if !ok || len(k.Daemons) > 0 || !couriersCame(f) {
		return CardAction{}, false
	}
	return CardAction{
		Key: KeyWarden, Label: ui.Bridge("PLACE WARDEN", "RUN WARDEN"), Command: strings.TrimPrefix(view.PlaceWardenCommand(node), "kubectl "),
		Tip: ui.Bridge("Place a post-warden on this Keep: couriers hand their satchels only to the warden on their own Keep.",
			"Run a warden Pod pinned to this node with a nodeSelector: the post Service (internalTrafficPolicy: Local) needs an endpoint here."),
		Act: view.Action{Action: view.ActPlaceWarden, Keep: node},
	}, true
}

func withWarden(acts []CardAction, node string, f Frame) []CardAction {
	if a, ok := WardenAction(node, f); ok {
		return append(append([]CardAction(nil), acts...), a)
	}
	return acts
}

func VaultOf(f Frame, name string) (view.VaultView, bool) {
	for _, v := range f.State.Storage.Vaults {
		if v.Name == name {
			return v, true
		}
	}
	return view.VaultView{}, false
}

func deedOf(f Frame, name string) (view.DeedView, bool) {
	for _, d := range f.State.Storage.Deeds {
		if d.Name == name {
			return d, true
		}
	}
	return view.DeedView{}, false
}

func vaultPortrait(s view.Structure, f Frame) string {
	v, _ := VaultOf(f, s.Object.Name)
	return SpritePortrait + world.VaultLook(v, f.State.Archive)
}

func VaultLines(s view.Structure, f Frame) []Line {
	v, ok := VaultOf(f, s.Object.Name)
	if !ok {
		return nil
	}
	out := []Line{{Text: ui.Bridge(fmt.Sprintf("%s  %s  ISLE %s", strings.ToUpper(v.Phase), strings.ToUpper(view.Barrels(float64(v.Shelves*roomsPerGiB))), v.Isle), fmt.Sprintf("%s  %dGi  zone %s", v.Phase, v.Shelves, v.Zone)), Clr: vaultTone(v.Phase)}}
	claim := strings.TrimPrefix(v.Claim, claimNS)
	switch {
	case claim != "":
		out = append(out, Line{Text: ui.Bridge("DEED "+claim+": its seal holds this Vault", "CLAIM "+v.Claim), Clr: ui.Palette.Text})
	default:
		out = append(out, Line{Text: ui.Bridge("NO DEED HOLDS IT", "no claimRef"), Clr: ui.Palette.TextDim})
	}
	return append(out, Line{Text: ui.Bridge(reclaimStory(v.Reclaim), "reclaimPolicy: "+v.Reclaim), Clr: ui.Palette.TextDim, Optional: true})
}

func vaultTone(phase string) color.RGBA {
	switch phase {
	case "Bound":
		return ui.Palette.Green
	case "Released", "Terminating", "Failed":
		return ui.Palette.RedText
	}
	return ui.Palette.GoldBright
}

func reclaimStory(policy string) string {
	if policy == "Delete" {
		return "When its Deed is torn up, the Vault is razed with it."
	}
	return "When its Deed is torn up, the Vault stands empty with a broken seal until cleared."
}

func VaultActions(s view.Structure, f Frame) []CardAction {
	name := s.Object.Name
	out := []CardAction{
		{Key: KeyDescribe, Label: "DESCRIBE", Command: "describe pv " + name, Details: DetailDescribe, Tip: ui.Bridge("kubectl describe: the Vault's shelves, its isle and the Deed that holds it", "kubectl describe persistentvolume")},
		{Key: KeyGet, Label: "GET", Command: "get pv " + name, Details: detailOutput, Tip: ui.Bridge("kubectl get: the Vault in one line", "kubectl get persistentvolume")},
	}
	v, ok := VaultOf(f, name)
	if claim := strings.TrimPrefix(v.Claim, claimNS); ok && claim != "" {
		out = append(out, deedActions(claim)...)
	}
	return out
}

func deedActions(claim string) []CardAction {
	return []CardAction{
		{Key: KeyDescribeDeed, Label: ui.Bridge("DEED", "PVC"), Command: "describe pvc " + claim, Details: DetailDescribe, Tip: ui.Bridge("kubectl describe: the Deed, its Vault and why it waits", "kubectl describe persistentvolumeclaim")},
		{Key: KeyGetDeed, Label: ui.Bridge("GET DEED", "GET PVC"), Command: "get pvc " + claim, Details: detailOutput, Tip: ui.Bridge("kubectl get: the Deed in one line", "kubectl get persistentvolumeclaim")},
	}
}

func withDeedActions(acts []CardAction, p model.PodView, f Frame) []CardAction {
	for _, c := range p.Claims {
		if _, ok := deedOf(f, c); ok {
			return append(append([]CardAction(nil), acts...), deedActions(c)[0])
		}
	}
	return acts
}

func WithDeed(c Card, p model.PodView, f Frame) Card {
	for _, name := range p.Claims {
		d, ok := deedOf(f, name)
		if !ok {
			continue
		}
		c.Lines = append(append([]Line(nil), c.Lines...), deedLine(d))
	}
	return c
}

func deedLine(d view.DeedView) Line {
	if d.Phase == "Bound" {
		return Line{Text: ui.Bridge("DEED "+d.Name+" IN "+d.Volume, "PVC "+d.Name+" -> PV "+d.Volume), Clr: ui.Palette.Green}
	}
	wait := d.Waiting
	if wait == "" {
		wait = ui.Bridge(fmt.Sprintf("waiting for a vault of %d shelves", d.Shelves), fmt.Sprintf("waiting for a %dGi PV", d.Shelves))
	}
	return Line{Text: ui.Bridge("DEED "+d.Name+" WAITING: "+wait, "PVC "+d.Name+" "+d.Phase+": "+wait), Clr: ui.Palette.GoldBright}
}
