package app

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

func StorageTarget(target string, st view.GameState, site view.Site, home func(string) (world.Vec, bool)) (world.Vec, bool) {
	kind, name, _ := strings.Cut(target, ":")
	switch kind {
	case "pv":
		return vaultSpot(site, name)
	case "deed":
		return deedSpot(st, site, name, home)
	case "sts":
		return statefulSpot(st, name, home)
	}
	return world.Vec{}, false
}

func vaultSpot(site view.Site, name string) (world.Vec, bool) {
	for _, s := range site.Structures {
		if s.Kind == view.StructVault && (s.ID == name || s.Object.Name == name) {
			return world.Anchor(s), true
		}
	}
	return world.Vec{}, false
}

func deedSpot(st view.GameState, site view.Site, name string, home func(string) (world.Vec, bool)) (world.Vec, bool) {
	for _, d := range st.Storage.Deeds {
		if d.Name != name || d.Volume == "" {
			continue
		}
		if v, ok := vaultSpot(site, d.Volume); ok {
			return v, true
		}
	}
	for _, set := range st.StatefulSets {
		for _, p := range set.Pods {
			if p.Claim == name {
				return podSpot(p.Name, home)
			}
		}
	}
	return world.Vec{}, false
}

func statefulSpot(st view.GameState, name string, home func(string) (world.Vec, bool)) (world.Vec, bool) {
	for _, set := range st.StatefulSets {
		if set.Name != name {
			continue
		}
		for _, p := range set.Pods {
			if v, ok := podSpot(p.Name, home); ok {
				return v, true
			}
		}
	}
	return world.Vec{}, false
}

func podSpot(pod string, home func(string) (world.Vec, bool)) (world.Vec, bool) {
	if v, ok := home(pod); ok {
		return world.Vec{X: v.X + .5, Y: v.Y + .5}, true
	}
	return world.Vec{}, false
}
