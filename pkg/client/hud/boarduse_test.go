package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func useFrame() Frame {
	f := boardFrame()
	f.Metrics.Pods = map[string]view.PodMetrics{}
	for i := range f.Pods {
		p := &f.Pods[i]
		p.Requests = []model.ContainerRequest{{Name: "c", CPUMilli: 250, MemoryBytes: 64 << 20}}
		f.Metrics.Pods[p.Namespace+"/"+p.Name] = view.PodMetrics{CPUMilli: 100, MemoryBytes: 32 << 20}
	}
	f.State.Crews[0].Peak = view.UsePeakView{CPU: 150, Memory: 32 << 20, Window: 720, Noon: true}
	return f
}

func pieceText(pieces []deltaPiece) string {
	parts := make([]string, len(pieces))
	for i, p := range pieces {
		parts[i] = p.Text
	}
	return strings.Join(parts, " | ")
}

func TestACrewShowsItsUseAgainstItsRequestsAndTheRoom(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	_, v := openBoard(t, useFrame())
	got := pieceText(UsePieces(v))
	for _, want := range []string{"PER CONTAINER", "CPU REQ 250m, USES 150m", "MEM REQ 64Mi, USES 32Mi", "NODES FREE 1250m CPU"} {
		if !strings.Contains(got, want) {
			t.Errorf("the use row %q lacks %q", got, want)
		}
	}
}

func TestTheUseRowSpeaksStoryWords(t *testing.T) {
	ui.SetNamer(nil)
	_, v := openBoard(t, useFrame())
	got := pieceText(UsePieces(v))
	for _, want := range []string{"EACH WORKER", "PLEDGE 250, USES 150", "PLEDGE 64, USES 32", "KEEPS FREE 1,250 RATIONS"} {
		if !strings.Contains(got, want) {
			t.Errorf("the use row %q lacks %q", got, want)
		}
	}
}

func TestEachWorkersPledgeKeepsItsSquiresApart(t *testing.T) {
	ui.SetNamer(nil)
	f := useFrame()
	f.State.Crews[0].PerReplica.SquireCPU = 50
	_, v := openBoard(t, f)
	got := pieceText(UsePieces(v))
	if !strings.Contains(got, "PLEDGE 200, USES 150") || !strings.Contains(got, "SQUIRES ADD") {
		t.Errorf("the use row %q gives the hut's pledge as the worker's", got)
	}
	if n := CrewPanelNumbers(f); len(n) == 0 || n[0].Pledge != 200 {
		t.Errorf("the panel numbers read the worker's pledge as %+v", n)
	}
}

func TestTheSliderPreviewsWhatNewReplicasReserveAndWhetherTheyFit(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	f := useFrame()
	h, v := openBoard(t, f)
	x, y := sliderAt(v, 6)
	h.Update(Input{MX: x, MY: y}, f)
	h.Update(Input{MX: x, MY: y, Click: true, Held: true}, f)
	x5, _ := sliderAt(v, 5)
	h.Update(Input{MX: x5, MY: y, Held: true}, f)
	fit, _ := h.BoardView(f)
	got := pieceText(UsePieces(fit))
	if !strings.Contains(got, "1250m CPU") || !strings.Contains(got, "750m CPU") || !strings.Contains(got, "ALL SCHEDULE") {
		t.Errorf("a fitting drag reads %q", got)
	}
	x9, _ := sliderAt(v, 9)
	h.Update(Input{MX: x9, MY: y, Held: true}, f)
	over, _ := h.BoardView(f)
	if got := pieceText(UsePieces(over)); !strings.Contains(got, "5 OF 6 SCHEDULE") {
		t.Errorf("an over-full drag reads %q", got)
	}
}

func TestThePerContainerRowKeepsItsMemoryHalfInNamesMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	f := useFrame()
	f.State.Crews[0].Peak = view.UsePeakView{CPU: 500, Memory: 64 << 20, Window: 720, Noon: true}
	for i := range f.Pods {
		f.Pods[i].Requests = []model.ContainerRequest{{Name: "c", CPUMilli: 200, MemoryBytes: 64 << 20}}
	}
	f.Crews[0].Resources.CPULimit, f.Crews[0].Resources.MemLimit = 500, 1024
	h, v := openBoard(t, f)
	got := pieceText(h.fitPieces(UsePieces(v), v.box.W-2*cbPad))
	if !strings.Contains(got, "MEM") || !strings.Contains(got, "64Mi") {
		t.Errorf("the NAMES row %q drops its memory half (crew-board-full: \"PER CONTAINER CPU REQUEST 150m, USES 500m, LIMIT 500m\" while the story row shows both)", got)
	}
}
