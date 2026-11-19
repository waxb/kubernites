package app

import (
	"fmt"
	"maps"
	"slices"
)

func (d *Driver) bridgePresses() []string {
	if d.Sess() == nil {
		return nil
	}
	now := map[string]int{}
	for page, mark := range d.Sess().Realm.GameState().Kube.Bridged {
		now[page] = mark.Presses
	}
	out := []string{}
	total := 0
	for _, n := range now {
		total += n
	}
	for _, page := range slices.Sorted(maps.Keys(now)) {
		if now[page] > d.bridged[page] {
			out = append(out, fmt.Sprintf("bridge: SEE IT IN KUBERNETES pressed for %s (%d presses of the bridge so far)", page, total))
		}
	}
	d.bridged = now
	return out
}
