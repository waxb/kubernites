package app

import (
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live"
)

type lister interface {
	Listing() []live.Listed
}

func listedReads(r any) []hud.ListedRead {
	l, ok := r.(lister)
	if !ok {
		return nil
	}
	var out []hud.ListedRead
	for _, k := range l.Listing() {
		read := hud.ListedRead{List: k.List, Every: int(k.Every.Seconds())}
		if k.PastCap {
			read.Cap = live.CustomWatchCap
		}
		out = append(out, read)
	}
	return out
}
