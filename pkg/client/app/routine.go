package app

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

var waitingForReadings = []string{"no metrics returned from resource metrics API", "did not receive metrics for targeted pods"}

type routineContext struct {
	carried   map[string]bool
	rising    bool
	waiting   map[string]bool
	turning   map[string]bool
	advancing map[string]bool
}

func routineOf(src AlertSources) routineContext {
	return routineContext{carried: carriedGroups(src.Pods), rising: keepsRising(src.Site), waiting: waitingHorns(src.Horns), turning: turningGroups(src.Pods), advancing: src.Advancing}
}

func hearthGroup(p model.PodView) string {
	if p.Depot != "" {
		return "depot:" + p.Depot
	}
	return "trade:" + p.Profession
}

func carriedGroups(pods []model.PodView) map[string]bool {
	out := map[string]bool{}
	for _, p := range pods {
		if p.Ready && !p.System && !p.Terminating {
			out[hearthGroup(p)] = true
		}
	}
	return out
}

func turningGroups(pods []model.PodView) map[string]bool {
	owners := map[string]map[string]bool{}
	moving := map[string]bool{}
	for _, p := range pods {
		if p.System || p.Owner == "" {
			continue
		}
		group := hearthGroup(p)
		if owners[group] == nil {
			owners[group] = map[string]bool{}
		}
		owners[group][p.Owner] = true
		if p.Terminating || p.Node != "" && !p.Ready {
			moving[group] = true
		}
	}
	out := map[string]bool{}
	for group, set := range owners {
		out[group] = len(set) > 1 && moving[group]
	}
	return out
}

func flakyByDesign(image string) bool {
	prof, ok := soulstone.Lookup(image)
	return ok && prof.MTBFSeconds > 0 && !prof.Cursed
}

func keepsRising(site view.Site) bool {
	for _, s := range site.Structures {
		if s.Kind == view.StructKeep && (s.State == view.StateConstruction || s.State == view.StateJoining) {
			return true
		}
	}
	return false
}

func waitingHorns(horns []model.Horn) map[string]bool {
	out := map[string]bool{}
	for _, h := range horns {
		for _, w := range waitingForReadings {
			if strings.Contains(h.Message, w) {
				out[h.Name] = true
			}
		}
	}
	return out
}

func (c routineContext) hearth(kind string, p model.PodView) bool {
	switch kind {
	case hud.AlertFire:
		return flakyByDesign(p.Image) && !p.OOM() && c.carried[hearthGroup(p)]
	case hud.AlertPending:
		return c.rising || c.turning[hearthGroup(p)] || c.advancing[p.OwnerKey()]
	}
	return false
}

func (c routineContext) structure(s view.Structure) bool {
	return s.Kind == view.StructWarHorn && c.waiting[s.Object.Name]
}
