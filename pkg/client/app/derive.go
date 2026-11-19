package app

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const MaxTidings = 80

func EventTime(e *corev1.Event) float64 {
	switch {
	case !e.LastTimestamp.IsZero():
		return float64(e.LastTimestamp.UnixNano()) / 1e9
	case !e.EventTime.IsZero():
		return float64(e.EventTime.UnixNano()) / 1e9
	}
	return float64(e.CreationTimestamp.UnixNano()) / 1e9
}

func RefreshSelection(sel hud.Selection, site view.Site, pods []model.PodView) hud.Selection {
	switch sel.Kind {
	case world.HitStructure:
		for _, s := range site.Structures {
			if s.ID == sel.ID {
				sel.Structure = s
				sel.Pod = PodNamed(pods, s.Object)
				return sel
			}
		}
		return hud.Selection{}
	case world.HitHearth, world.HitWalker:
		for _, p := range pods {
			if p.Namespace+"/"+p.Name == sel.ID {
				sel.Pod, sel.Gone = p.Focused(sel.Pod.Focus), false
				return sel
			}
		}
		if sel.Kind == world.HitWalker && sel.Pod.Name != "" {
			sel.Gone = true
			return sel
		}
		return hud.Selection{}
	}
	return sel
}

func PodNamed(pods []model.PodView, ref view.ObjectRef) model.PodView {
	if ref.Kind != "Pod" {
		return model.PodView{}
	}
	for _, p := range pods {
		if p.Namespace == ref.Namespace && p.Name == ref.Name {
			return p
		}
	}
	return model.PodView{}
}

func SelectionOf(hit world.Hit, pods []model.PodView) hud.Selection {
	sel := hud.Selection{Kind: hit.Kind, ID: hit.ID, Structure: hit.Structure, Pod: hit.Pod}
	if hit.Kind == world.HitSignpost || hit.Kind == world.HitFamiliar {
		sel.Kind = world.HitHearth
	}
	if hit.Kind == world.HitStructure {
		sel.Pod = PodNamed(pods, hit.Structure.Object)
	}
	return sel
}

var eventEffects = map[string]string{"Tremor": "dust", "OOMKilling": "void", "Evicted": "dust", "Killing": "dust", "Preempted": "dust"}

func EventTransients(events []*corev1.Event, seen map[string]bool, homes func(string) (world.Vec, bool), now float64) []world.Transient {
	out := []world.Transient{}
	for _, e := range events {
		key := string(e.UID) + "/" + e.ResourceVersion
		if seen[key] {
			continue
		}
		seen[key] = true
		kind, ok := eventEffects[e.Reason]
		if !ok {
			continue
		}
		home, ok := homes(e.InvolvedObject.Name)
		if !ok {
			continue
		}
		out = append(out, world.Transient{Kind: kind, At: world.Vec{X: home.X + .5, Y: home.Y + .5}, Start: now, Life: 1.2})
	}
	return out
}

func PruneTransients(ts []world.Transient, now float64) []world.Transient {
	out := make([]world.Transient, 0, len(ts))
	for _, t := range ts {
		if now-t.Start <= t.Life {
			out = append(out, t)
		}
	}
	return out
}

func Crisis(pods []model.PodView) bool {
	bad := 0
	for _, p := range pods {
		if !p.System && (p.State == model.HearthGuttering || p.State == model.HearthGhost) {
			bad++
		}
	}
	return bad >= 2
}
