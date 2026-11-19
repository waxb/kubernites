package world

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	rolloutGainFloor = .01
	rolloutRise      = 40.0
)

func RolloutText(c model.CrewChain, gain float64) (string, bool) {
	return RolloutTextWaiting(c, gain, 0)
}

func RolloutTextWaiting(c model.CrewChain, gain float64, waiting int) (string, bool) {
	f, ok := model.Newest(c)
	if !ok || !c.Rolling() {
		return "", false
	}
	_, tag, _ := strings.Cut(f.Image, ":")
	what, whatK8s := "CLASS "+tag, "ROLLOUT "+f.Image
	if change, changeK8s, same := rolloutChange(c, f); same {
		what, whatK8s = change, changeK8s
	}
	story := fmt.Sprintf("%s: %d OF %d REBUILT", what, f.Ready, c.Desired)
	k8s := fmt.Sprintf("%s: %d/%d UPDATED", whatK8s, f.Ready, c.Desired)
	switch {
	case waiting > 0:
		story, k8s = story+fmt.Sprintf(", %d WAIT FOR ROOM", waiting), k8s+fmt.Sprintf(", %d PENDING", waiting)
	case f.Ready == 0 && f.Current > 0:
		story, k8s = story+", THE FIRST IS STARTING", k8s+", FIRST POD NOT READY"
	case gain >= rolloutGainFloor:
		amount := goldText(gain)
		story, k8s = story+", +"+amount+" GOLD/S", k8s+", +$"+amount+"/S"
	}
	return ui.Bridge(story, k8s), true
}

func rolloutChange(c model.CrewChain, newest model.Foreman) (string, string, bool) {
	for _, f := range c.Foremen {
		if f.Newest || f.Desired == 0 && f.Current == 0 || f.Image != newest.Image {
			continue
		}
		switch {
		case f.Shutters != newest.Shutters:
			return "SHUTTERS", "READINESS PROBE", true
		case f.Bell != newest.Bell:
			return "THE BELL", "LIVENESS PROBE", true
		case f.Pledge != newest.Pledge:
			return "NEW PLEDGE", "NEW REQUESTS", true
		}
		return "REBUILD", "ROLLOUT", true
	}
	return "", "", false
}

func newestWaiting(c model.CrewChain, pods []model.PodView) int {
	f, ok := model.Newest(c)
	if !ok {
		return 0
	}
	n := 0
	for _, p := range pods {
		if p.Namespace == c.Namespace && p.State == model.HearthPending && slices.Contains(f.Pods, p.Name) {
			n++
		}
	}
	return n
}

func (r *Renderer) rolloutMarks(cam render.Camera, sc Scene) []UnservedMark {
	zoom := cam.Scale()
	out := []UnservedMark{}
	for _, c := range sc.Crews {
		if !c.Rolling() {
			continue
		}
		text, ok := RolloutTextWaiting(c, sc.Rollouts.Gain(c, sc.Freight), newestWaiting(c, sc.Pods))
		stall, found := DepotOf(sc.Site, c.Depot)
		if !ok || !found {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, Anchor(stall))
		if !onScreen(sx, sy, zoom) {
			continue
		}
		if box, ok := settleAbove(labelBox(text, sx, sy-rolloutRise*zoom), int(sx), sc.Panels); ok {
			out = append(out, UnservedMark{Kind: c.Name, Text: text, Box: box})
		}
	}
	return out
}

func (r *Renderer) drawTradeLabels(dst *ebiten.Image, cam render.Camera, sc Scene) {
	r.drawUnserved(dst, cam, sc)
	var font ui.Font
	for _, m := range r.rolloutMarks(cam, sc) {
		box, ok := r.settleLabel(m.Box, midX(m.Box), sc.Panels)
		if !ok {
			continue
		}
		m.Box = box
		ui.Fill(dst, m.Box.Offset(1, 1), ui.Palette.Shadow)
		ui.Fill(dst, m.Box, ui.Tint(ui.Palette.ConsoleEdge, 235))
		ui.Stroke(dst, m.Box, ui.Palette.Gold)
		font.Draw(dst, m.Text, m.Box.X+4, m.Box.Y+3, 1, ui.Palette.GoldBright)
	}
}
