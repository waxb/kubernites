package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func foremanBoardFrame() Frame {
	f := boardFrame()
	foreman := model.Foreman{Namespace: "default", Name: "porters", Desired: 3, Ready: 3, Current: 3, Newest: true, Image: "porter:1.0",
		Pods: []string{"porters-a", "porters-b", "porters-c"}}
	f.Crews = []model.CrewChain{{Namespace: "default", Name: "porters", Bare: true, Image: "porter:1.0", Container: "porter", Desired: 3, Ready: 3,
		Total: 3, Depot: "parcels", Foremen: []model.Foreman{foreman}}}
	f.Pods = nil
	for _, name := range foreman.Pods {
		f.Pods = append(f.Pods, model.PodView{Namespace: "default", Name: name, Owner: "ReplicaSet/porters", State: model.HearthReady, Crew: "porters", Ready: true, Node: "lone-tower"})
	}
	f.State.Crews[0].Keeper = view.KeeperReplicaSet
	return f
}

func TestAForemansCrewHasItsBoardAndScalesAsAReplicaSet(t *testing.T) {
	f := foremanBoardFrame()
	if _, ok := ChainOf(f); !ok {
		t.Fatal("selecting the shift's crew shows no card")
	}
	h, v := openBoard(t, f)
	x, y := sliderAt(v, 5)
	h.Update(Input{MX: x, MY: y}, f)
	h.Update(Input{MX: x, MY: y, Click: true, Held: true}, f)
	a, _ := actionOf(h.Update(Input{MX: x, MY: y}, f))
	if a.Action != view.ActScaleCrew || a.Name != "porters" || a.Replicas != 5 || CommandOf(a) != "kubectl scale rs porters --replicas=5" {
		t.Fatalf("release = %+v (%s)", a, CommandOf(a))
	}
}

func TestAForemansCrewCardIsTheForemansCard(t *testing.T) {
	c := CrewCard(foremanBoardFrame())
	if c.Title != "SHIFT PORTERS" || c.K8s != "ReplicaSet/porters" {
		t.Fatalf("card = %q %q", c.Title, c.K8s)
	}
}

func cardText(c Card) string {
	return strings.Join(cardTexts(c), "\n")
}

func TestAForemanWithoutARevisionShowsNone(t *testing.T) {
	f := foremanBoardFrame()
	if text := cardText(CrewCard(f)); strings.Contains(text, "REVISION") || strings.Contains(text, "pod-template-hash") {
		t.Fatalf("a bare shift's card shows a revision:\n%s", text)
	}
	adopted := f.Crews[0].Foremen[0]
	adopted.Revision = 1
	if text := cardText(ForemanCard(f.Crews[0], adopted)); !strings.Contains(text, "REVISION 1") || strings.Contains(text, "pod-template-hash") {
		t.Fatalf("an adopted shift keeps no hash label:\n%s", text)
	}
	hashed := adopted
	hashed.Hash = "7d9f"
	if text := cardText(ForemanCard(f.Crews[0], hashed)); !strings.Contains(text, "REVISION 1  pod-template-hash 7d9f") {
		t.Fatalf("a Deployment's shift names its hash:\n%s", text)
	}
}

func TestACrewJustRaisedOverAWorkingShiftReadsNoFalseZero(t *testing.T) {
	c := model.CrewChain{Namespace: "default", Name: "porters", Image: "porter:1.0", Desired: 7, Uncounted: true}
	text := cardText(DeploymentCard(c))
	if strings.Contains(text, "READY 0") || !strings.Contains(text, "COUNTING ITS HUTS") {
		t.Fatalf("a crew the Stewards have not counted yet reads READY 0 while its shift works:\n%s", text)
	}
	v := BoardView{Chain: c}
	pieces := UsePieces(v)
	joined := ""
	for _, p := range pieces {
		joined += p.Text + "|"
	}
	if strings.Contains(joined, "NO HUT RUNS YET") {
		t.Fatalf("the board of a crew not yet counted says no hut runs: %s", joined)
	}
}
