package hud

import (
	"fmt"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

const stallPrefix = "stall"

var (
	stallIcons    = map[string]string{soulstone.KindSealed: "icon/sealed"}
	serviceBuilds = map[string]string{view.StructGuildhall: soulstone.KindParcel, view.StructArchive: soulstone.KindLedger}
)

func init() {
	prefixHandlers[stallPrefix] = pressStall
}

func StallTarget(kind string) string { return stallPrefix + ":" + kind }

func stallRows(f Frame) []RaiseRow {
	out := []RaiseRow{}
	for _, r := range RaiseRows(f) {
		if _, ok := DepotAction(r); ok {
			out = append(out, r)
		}
	}
	return out
}

func stallButtons(f Frame) []Button {
	out := []Button{}
	for _, r := range stallRows(f) {
		id := StallTarget(r.Kind)
		name := strings.ToUpper(r.Depot)
		b := Button{ID: id, Label: ui.Bridge(name+" DEPOT", r.Depot+" SERVICE"), Icon: stallIcon(r.Kind),
			Tip: stallTip(r, f), Selected: f.Guide.Button == id}
		if !f.Shown(CmdDepot) && f.State.Roll.Hired < view.HandPorterTarget {
			b.Why = ui.Bridge("HIRE FIRST", "PODS FIRST")
			b.Disabled, b.Tip = true, ui.Bridge("Opens once three porters are hired: a depot needs huts to hand its boxes to.",
				"Opens once three Pods run: a Service needs Pods to send its requests to.")
		}
		out = append(out, b)
	}
	return out
}

func stallIcon(kind string) string {
	if icon, ok := stallIcons[kind]; ok {
		return icon
	}
	return "icon/kind-" + kind
}

func stallTip(r RaiseRow, f Frame) string {
	if handDepot(r, f) {
		return fmt.Sprintf(ui.Bridge("Open the %s depot: it stands on the Forum's rim and hands each box to a hut on the roll you keep by hand. The roll starts with the door porter.",
			"Create Service/%s with no selector: it sends each request to an address on the EndpointSlice you write by hand, which starts with the door Pod."), r.Depot)
	}
	text := fmt.Sprintf(ui.Bridge("Open the %s depot: it stands on the Forum's rim and hands each box to a ready hut wearing app=%s.",
		"Expose Service/%s: it sends each request to a ready Pod labelled app=%s."), r.Depot, r.Crew)
	return text + SpeakTip(DepotCommandOf(r, f))
}

func handDepot(r RaiseRow, f Frame) bool {
	return HandDepot(r.Kind, f)
}

func HandDepot(kind string, f Frame) bool {
	return kind == soulstone.KindParcel && !f.State.Unlocks[view.BadgesPage]
}

func pressStall(_ *HUD, kind string, _ Frame) []Intent {
	return []Intent{ActFrom(view.Action{Action: "openDepot", Kind: kind}, FromCommandCard)}
}

func stalled(b Button, f Frame) bool {
	kind, ok := serviceBuilds[strings.TrimPrefix(b.ID, "build:")]
	if !ok {
		return false
	}
	if ladderGated(b, f) {
		return true
	}
	for _, r := range RaiseRows(f) {
		if r.Kind == kind && (r.DepotOpen || StallOffered(f, kind)) {
			return true
		}
	}
	return false
}

func ladderGated(b Button, f Frame) bool {
	for _, d := range f.State.Buildables {
		if "build:"+d.Kind == b.ID && d.Reason == view.LadderDepotGate {
			return true
		}
	}
	return false
}

func StallOffered(f Frame, kind string) bool {
	for _, r := range stallRows(f) {
		if r.Kind == kind {
			return true
		}
	}
	return false
}

func stallGoal(goal string, f Frame) string {
	if goal != CmdDepot {
		return goal
	}
	rows := stallRows(f)
	if l, ok := DepotLine(f); ok && l.Action != nil && StallOffered(f, l.Action.Kind) {
		return StallTarget(l.Action.Kind)
	}
	if len(rows) > 0 {
		return StallTarget(rows[0].Kind)
	}
	return ""
}

func gridGoal(goal string, f Frame) string {
	return portGoal(stallGoal(goal, f), f)
}
