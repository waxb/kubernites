package hud

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

func boardPanel(name string, f Frame) textPanel {
	return textPanel{name: name,
		bounds: func(h *HUD) ui.Rect {
			if rects := h.CrewRects(f); len(rects) == 1 {
				return rects[0]
			}
			return ui.Rect{}
		},
		draw:    func(h *HUD, dst *ebiten.Image, f Frame) { h.DrawCrewCard(dst, f) },
		buttons: (*HUD).CrewButtons}
}

func manyWorkersFrame(n int) Frame {
	f := boardFrame()
	f.Crews[0].Desired, f.Crews[0].Ready = n, n
	f.Pods, f.Crews[0].Foremen[0].Pods = nil, nil
	for i := range n {
		name := fmt.Sprintf("porters-7d9f-%02d", i)
		f.Pods = append(f.Pods, model.PodView{Namespace: "default", Name: name, State: model.HearthReady, Ready: true, Node: "k", Crew: "porters"})
		f.Crews[0].Foremen[0].Pods = append(f.Crews[0].Foremen[0].Pods, name)
	}
	f.Metrics.Nodes["lone-tower"] = view.NodeMetrics{CPUAllocatableMilli: 64000, CPURequestedMilli: int64(n) * 250, MemoryAllocatableBytes: 64 << 30}
	return f
}

func TestTheBoardsFitInBothNamings(t *testing.T) {
	idle := boardFrame()
	h, v := openBoard(t, idle)
	assertTextFits(t, h, idle, boardPanel("crew board idle", idle))
	h.Board.Drag, h.Board.Pick, h.Board.Top = DragKnob, v.Top, v.Top
	assertTextFits(t, h, idle, boardPanel("crew board drag", idle))
	full := boardFrame()
	full.Metrics.Nodes["lone-tower"] = view.NodeMetrics{CPUAllocatableMilli: 1000, CPURequestedMilli: 750, MemoryAllocatableBytes: 4 << 30}
	hf, vf := openBoard(t, full)
	hf.Board.Drag, hf.Board.Pick, hf.Board.Top = DragKnob, 8, vf.Top
	assertTextFits(t, hf, full, boardPanel("crew board full", full))
	horn := hornedCrewFrame()
	hh, _ := openBoard(t, horn)
	assertTextFits(t, hh, horn, boardPanel("crew board horn", horn))
	for _, n := range []int{12, 45} {
		crowd := manyWorkersFrame(n)
		hc, _ := openBoard(t, crowd)
		assertTextFits(t, hc, crowd, boardPanel(fmt.Sprintf("crew board %d", n), crowd))
	}
	bare := boardFrame()
	bare.State.Freight.Kinds, bare.Crews[0].Resources, bare.Crews[0].QoS = nil, model.Resources{}, model.QoSBestEffort
	hb, _ := openBoard(t, bare)
	assertTextFits(t, hb, bare, boardPanel("crew board besteffort", bare))
	depot := depotBoardFrame()
	hd := New(nil, Backend{})
	hd.Press(ChainButton, depot)
	assertTextFits(t, hd, depot, boardPanel("depot board", depot))
}

func TestTheOrderPagesAndTheBareListFitInBothNamings(t *testing.T) {
	for _, mode := range []string{CrewModeImage, CrewModeRes, CrewModeHorn, CrewModeRoll} {
		f := boardFrame()
		f.State.Reveals = append(f.State.Reveals, "soulvault")
		f.State.Unlocks[HornUnlock] = true
		f.State.Grimoire = []view.GrimoirePage{runes("resources.requests.cpu", "resources.requests.memory", "resources.limits.cpu", "resources.limits.memory")}
		h, _ := openBoard(t, f)
		pressed(t, h, f, "crew:mode:"+mode)
		if _, ok := h.BoardView(f); ok {
			t.Fatalf("%s shows the board", mode)
		}
		assertTextFits(t, h, f, boardPanel("crew page "+mode, f))
	}
	bare := boardFrame()
	lantern := model.CrewChain{Bare: true, Namespace: "default", Foremen: []model.Foreman{{Namespace: "default", Name: "lantern", Newest: true, Desired: 2, Ready: 1,
		Pods: []string{"porters-7d9f-a"}}}}
	bare.Crews, bare.Selection = []model.CrewChain{lantern}, Selection{Kind: SelectForeman, ID: "default/lantern"}
	h := New(nil, Backend{})
	h.Press(ChainButton, bare)
	h.Hover, h.Hovering = Button{ID: "crew:pick:rs:lantern"}, true
	assertTextFits(t, h, bare, boardPanel("bare foreman list", bare))
}

func TestAZeroPayoffSaysWhyAndStillFits(t *testing.T) {
	f := boardFrame()
	f.State.Crews[0].PerReplica.Income = 0
	h, v := openBoard(t, f)
	pieces := DeltaPieces(v, f)
	if len(pieces) < 2 || pieces[1].Text != "+0 GOLD/S: KEEPING UP" {
		t.Fatalf("pieces = %+v", pieces)
	}
	if b := crewButtonMap(v.Steps)[StepUpButton]; b.Label != "+1 HUT" {
		t.Fatalf("a zero change labels + as %q", b.Label)
	}
	assertTextFits(t, h, f, boardPanel("crew board zero change", f))
	h.Board.Drag, h.Board.Pick, h.Board.Top = DragKnob, v.Top, v.Top
	if drag, _ := h.BoardView(f); DeltaPieces(drag, f)[1].Text != "+0 GOLD/S: KEEPING UP" {
		t.Fatalf("drag pieces = %+v", DeltaPieces(drag, f))
	}
	f.State.Crews[0].Fits = 0
	if drag, _ := h.BoardView(f); DeltaPieces(drag, f)[1].Text != "+0 GOLD/S" {
		t.Fatalf("a drag onto no land = %+v", DeltaPieces(drag, f))
	}
}

func TestTheCrewLinkKeepsOpenTheBoardOnOneLine(t *testing.T) {
	defer ui.SetNamer(nil)
	for mode := range 2 {
		if mode == 0 {
			ui.SetStoryNamer(lexicon.StoryText)
		} else {
			ui.SetNamer(names.Kubernetes(64))
		}
		f := boardFrame()
		f.Selection = Selection{Kind: "hearth", Pod: f.Pods[0]}
		f.Crews[0].Name = "crasher-frontend"
		b, text, ok := crewLink(f)
		if !ok {
			t.Fatal("no crew link")
		}
		rows := crewLinkRows(text, b.Rect.W)
		if rows[len(rows)-1] != "open the board" && !strings.HasSuffix(rows[0], "open the board") {
			t.Errorf("mode %d: the link wraps as %q", mode, rows)
		}
	}
}
