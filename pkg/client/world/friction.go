package world

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	FrictionUnhoused   = "unhoused"
	FrictionWarded     = "warded"
	FrictionIdle       = "idle"
	FrictionUnreplaced = "unreplaced"
	FrictionStrained   = "strained"
	FrictionUnrecorded = "unrecorded"
	FrictionQueue      = "queue"

	idleQueue    = 1.0
	scrollFloor  = .1
	ledgerObject = "service/default/ledgers"
	podPrefix    = "pod/"
	nodePrefix   = "node/"
)

var scrollSteps = [3]float64{.35, .7, .9}

type Friction struct {
	Kind   string
	Object string
	Text   string
}

func eachDerived(sc Scene, c *census, emit func(Friction)) {
	eachIdle(sc, c, func(p model.PodView) { emit(Friction{Kind: FrictionIdle, Object: PodObject(p)}) })
	for i, p := range sc.Pods {
		switch {
		case p.System || p.Terminating:
		case p.State == model.HearthPending:
			emit(Friction{Kind: FrictionUnhoused, Object: PodObject(p)})
		case p.State == model.HearthReady && c.stats[i].m.Throttled:
			emit(Friction{Kind: FrictionStrained, Object: PodObject(p)})
		}
	}
	eachDark(sc.Site, func(k view.Structure) { emit(Friction{Kind: FrictionWarded, Object: NodeObject(k.Object.Name)}) })
	if sc.Ledger.Open && sc.Ledger.Share >= ledgerWarn {
		emit(Friction{Kind: FrictionUnrecorded, Object: ledgerObject})
	}
}

func eachDark(site view.Site, fn func(view.Structure)) {
	for _, k := range site.Structures {
		if k.Kind == view.StructKeep && k.Object.Name != "" && k.State == view.StateDark {
			fn(k)
		}
	}
}

func eachIdle(sc Scene, c *census, fn func(model.PodView)) {
	if sc.Queue < idleQueue {
		return
	}
	_, guild := FirstOfKind(sc.Site, view.StructGuildhall, true)
	for i, p := range sc.Pods {
		st := c.stats[i]
		if declaredHearth(p) && (!guild || (st.measured && st.m.ServedRate <= 0)) {
			fn(p)
		}
	}
}

type frictionSets struct {
	warded     map[string]bool
	unreplaced bool
}

func (fs *frictionSets) fill(sc Scene, c *census) {
	if fs.warded == nil {
		fs.warded = map[string]bool{}
	}
	clear(fs.warded)
	fs.unreplaced = false
	for _, f := range sc.Frictions {
		switch f.Kind {
		case FrictionWarded:
			if node, ok := strings.CutPrefix(f.Object, nodePrefix); ok {
				fs.warded[node] = true
			}
		case FrictionUnreplaced:
			fs.unreplaced = true
		}
	}
	eachDark(sc.Site, func(k view.Structure) { fs.warded[k.Object.Name] = true })
}

func PodObject(p model.PodView) string {
	return podPrefix + p.Namespace + "/" + p.Name
}

func NodeObject(name string) string {
	return nodePrefix + name
}

func MergeFrictions(lists ...[]Friction) []Friction {
	seen := map[[2]string]bool{}
	out := []Friction{}
	for _, list := range lists {
		for _, f := range list {
			key := [2]string{f.Kind, f.Object}
			if !seen[key] {
				seen[key] = true
				out = append(out, f)
			}
		}
	}
	return out
}

func FrictionOn(fs []Friction, kind, object string) bool {
	for _, f := range fs {
		if f.Kind == kind && f.Object == object {
			return true
		}
	}
	return false
}

func ScrollPile(l Ledger) int {
	if !l.Open || l.Share < scrollFloor {
		return -1
	}
	return stepLevel(l.Share, scrollSteps)
}
