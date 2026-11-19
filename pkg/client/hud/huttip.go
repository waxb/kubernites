package hud

import (
	"fmt"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const hutMore = "click for more"

type hutState struct{ story, kube string }

var hutStates = map[string]hutState{
	model.HearthReady:     {"Burning: up and ready", "Running"},
	model.HearthKindling:  {"Kindling: getting ready", "ContainerCreating"},
	model.HearthGuttering: {"Guttering: it keeps failing", "CrashLoopBackOff"},
	model.HearthGhost:     {"Trade not found: it cannot fetch its trade", "ImagePullBackOff"},
	model.HearthBanished:  {"Failed: it was sent away", "Failed"},
	model.HearthPending:   {"Unhoused: no Keep has room yet", "Pending: no Node has room yet"},
}

func hutStateWord(p model.PodView) string {
	if p.Terminating {
		return ui.Bridge("Last Rites: shutting down", "Terminating")
	}
	if p.State == model.HearthPending && p.Untried {
		return ui.Bridge("Unhoused: the Quartermaster has not placed it yet", "Pending: not scheduled yet")
	}
	if firstExit(p) {
		return ui.Bridge("Stopped: its worker exited, and the Reeve relights it at once", p.Reason+": the container exited, and the kubelet restarts it at once")
	}
	if s, ok := hutStates[p.State]; ok {
		return ui.Bridge(s.story, s.kube)
	}
	return p.State
}

func hutCarries(p model.PodView, m view.PodMetrics) string {
	switch {
	case p.State != model.HearthReady:
		return ui.Bridge("nothing yet", "no traffic yet")
	case m.ServedRate > 0:
		return ui.Bridge(fmt.Sprintf("%.1f boxes a second", m.ServedRate), fmt.Sprintf("%.1f requests a second", m.ServedRate))
	case p.Depot == "" && p.NodePort > 0 && p.HostNetwork:
		return ui.Bridge("at its Keep's door", fmt.Sprintf("requests at its Node's port %d (hostNetwork)", p.NodePort))
	case p.Depot == "" && p.NodePort > 0:
		return ui.Bridge("at its Keep's door", fmt.Sprintf("requests at its Node's port %d (hostPort)", p.NodePort))
	case p.Depot == "":
		return ui.Bridge("idle: on no depot's roll", "idle: no Service selects it")
	}
	return ui.Bridge("0.0 boxes a second", "0.0 requests a second")
}

func hutUsed(pledged, used float64, food bool) string {
	switch {
	case pledged <= 0 && food:
		return ui.Bridge("food: no pledge", "cpu: no request")
	case pledged <= 0:
		return ui.Bridge("rooms: no pledge", "memory: no request")
	case food:
		return ui.Bridge(fmt.Sprintf("%s rations now, of %s pledged", foodCount(used), foodCount(pledged)), fmt.Sprintf("%.0fm CPU now, of %.0fm requested", used, pledged))
	}
	return ui.Bridge(fmt.Sprintf("%s rooms now, of %s pledged", housingCount(used), housingCount(pledged)), fmt.Sprintf("%.0fMi memory now, of %.0fMi requested", used/mebi, pledged/mebi))
}

func HutTip(p model.PodView, m view.PodMetrics, why, fix string) string {
	keep := p.Node
	if keep == "" {
		keep = ui.Bridge("none yet", "none")
	}
	parts := []TipPart{TipTitle(p.Name + " - " + hutStateWord(p))}
	parts = append(parts, TipRow(ui.Bridge("Trade", "Image"), p.Image+" ("+ui.Bridge("Keep", "Node")+" "+keep+")"))
	if why == "" {
		cpu, mem := p.RequestSum()
		food, room := hutUsed(float64(cpu), m.CPUMilli, true), hutUsed(float64(mem), m.MemoryBytes, false)
		parts = append(parts, TipRow(ui.Bridge("Carries", "Serves"), hutCarries(p, m)),
			TipRow(ui.Bridge("Eats", "CPU"), food), TipRow(ui.Bridge("Room", "Memory"), room))
	} else {
		parts = append(parts, TipGold("Why: "+why))
		if fix != "" {
			parts = append(parts, TipGold("Fix: "+fix))
		}
	}
	return StructTip(append(parts, TipDim(hutMore))...)
}

func CauseTipText(title, pod, keep, why, fix string) string {
	parts := []TipPart{TipTitle(title)}
	if pod != "" {
		parts = append(parts, TipRow(ui.Bridge("Hut", "Pod"), pod))
	}
	if keep != "" {
		parts = append(parts, TipRow(ui.Bridge("Keep", "Node"), keep))
	}
	if why != "" {
		parts = append(parts, TipGold("Why: "+why))
	}
	if fix != "" {
		parts = append(parts, TipGold("Fix: "+fix))
	}
	return StructTip(append(parts, TipDim("click the hut for more"))...)
}
