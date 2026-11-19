package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestBorgsScrollShowsEveryPartWhole(t *testing.T) {
	h := New(nil, Backend{})
	c := questCard{title: "Operator brief", giver: "Operator", Body: "The operator who keeps this cluster has raised more of them than anyone remembers.",
		Parts: []view.QuestPartView{
			{Title: "The Codex", Purpose: "The store of record: every hut, crew and order you give is written in it; what is not written did not happen (etcd)"},
			{Title: "The Stewards", Purpose: "Hold the realm as you declared it: each crew at its count, and the roll of every depot that looks for a badge (kube-controller-manager)"},
		}}
	for _, mode := range []string{"story", "names"} {
		if mode == "names" {
			ui.SetNamer(names.Kubernetes(64))
		}
		_, parts := h.questBody(c)
		for _, rows := range parts {
			if last := rows[len(rows)-1]; strings.HasSuffix(last, "...") {
				t.Errorf("%s: a part is cut: %q", mode, rows)
			}
		}
		ui.SetNamer(nil)
	}
}
