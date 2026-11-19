package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func TestAHutCardShowsOnlyWhatWasTaught(t *testing.T) {
	f := Frame{}
	f.Selection = Selection{Kind: "hearth", Pod: model.PodView{Name: "porter-a", Namespace: defaultNS, Profession: "Porter", Node: "lone-tower", Image: "porter:1.0", State: "running"}}
	text := func() string {
		out := []string{}
		for _, l := range (&HUD{}).CardFor(f).Lines {
			out = append(out, l.Text)
		}
		return strings.Join(out, "|")
	}
	if got := text(); strings.Contains(got, "RESTARTS") || strings.Contains(got, "TRADE") || strings.Contains(got, "FOOD") || !strings.Contains(got, "KEEP lone-tower") {
		t.Fatalf("a fresh hut card = %s", got)
	}
	f.State.Reveals = []string{restartsLesson, classLesson, laborLesson}
	if got := text(); !strings.Contains(got, "RESTARTS 0  KEEP lone-tower") || !strings.Contains(got, "TRADE porter:1.0") || !strings.Contains(got, "FOOD") {
		t.Fatalf("a taught hut card = %s", got)
	}
}

func TestACardCommandComesWithItsLesson(t *testing.T) {
	f := Frame{}
	f.State.Age = view.AgeDark
	keys := func(acts []CardAction) string {
		out := []string{}
		for _, a := range acts {
			out = append(out, a.Key)
		}
		return strings.Join(out, ",")
	}
	all := HearthActions(model.PodView{Name: "porter-a", Namespace: defaultNS})
	if got := keys(taughtActions(all, f)); got != "describe,logs" {
		t.Fatalf("before any hint the hut offers %s", got)
	}
	f.State.Reveals = []string{"summon", "depot", "chronicles", "scriptorium"}
	if got := keys(taughtActions(all, f)); got != keys(all) {
		t.Fatalf("after the hints the hut offers %s, want %s", got, keys(all))
	}
	f.State.Age = view.AgeDark
	keep := []CardAction{{Key: KeyTaint}, {Key: KeyCordon}, {Key: KeyDrain}, {Key: KeyDescribe}}
	if got := keys(taughtActions(keep, f)); got != "describe" {
		t.Fatalf("an early Keep offers %s", got)
	}
	f.State.Age = view.AgeFeudal
	if got := keys(taughtActions(keep, f)); got != "taint,cordon,drain,describe" {
		t.Fatalf("a later Keep offers %s", got)
	}
}
