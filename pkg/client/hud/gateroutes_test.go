package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

func gateFrame(t *testing.T) (Frame, view.Structure) {
	f := fullFrame(t)
	f.State.Routes = []view.RouteView{
		{Kind: "parcel", Service: "parcels", Route: "HTTPRoute", Listener: "HTTP", Port: 80, Open: true, Ready: 3},
		{Kind: "ledger", Service: "ledgers", Route: "TCPRoute", Listener: "TCP", Port: 5432},
	}
	gate := view.Structure{ID: "gate-1", Kind: view.StructGate, Label: "The Gate", State: view.StateReady}
	return f, gate
}

func TestTheGateCardListsItsRoutes(t *testing.T) {
	defer ui.SetNamer(nil)
	f, gate := gateFrame(t)
	ui.SetStoryNamer(lexicon.StoryText)
	story := strings.Join(texts2(KindLines(gate, f)), " | ")
	if !strings.Contains(story, "parcels") || !strings.Contains(story, "ledgers") || lexicon.KubernetesWords.MatchString(ui.Name(story)) {
		t.Fatalf("story Gate card %q", story)
	}
	ui.SetNamer(names.Kubernetes(64))
	k8s := strings.Join(texts2(KindLines(gate, f)), " | ")
	for _, want := range []string{"HTTPRoute parcels", "HTTP :80", "TCPRoute ledgers", "TCP :5432"} {
		if !strings.Contains(k8s, want) {
			t.Errorf("Kubernetes Gate card %q lacks %q", k8s, want)
		}
	}
}

func TestAGateWithNoRoutesSaysHowToGetOne(t *testing.T) {
	f, gate := gateFrame(t)
	f.State.Routes = nil
	if lines := KindLines(gate, f); len(lines) != 1 || !strings.Contains(lines[0].Text, "depot") {
		t.Fatalf("no-route Gate card %+v", lines)
	}
}

func texts2(lines []Line) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.Text
	}
	return out
}
