package hud

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func TestStoneEditing(t *testing.T) {
	s := NewStone()
	for _, r := range "get pod" {
		s.Insert(r)
	}
	s.Insert('\n')
	s.Move(-3)
	s.Insert('s')
	s.Backspace()
	s.Delete()
	if string(s.Input) != "get od" {
		t.Errorf("input = %q", string(s.Input))
	}
	s.Move(100)
	if s.Submit() != "get od" || s.Submit() != "" {
		t.Error("submit")
	}
	s.Input = []rune("x")
	s.Submit()
	s.Recall(-1)
	s.Recall(-1)
	if string(s.Input) != "get od" {
		t.Errorf("recall = %q", string(s.Input))
	}
	s.Recall(5)
	if len(s.Input) != 0 {
		t.Error("recall past the end clears the line")
	}
	s.Cursor = 0
	s.Backspace()
	for i := 0; i < maxStoneLines+10; i++ {
		s.Append(LineOutput, "line")
	}
	if len(s.Output) != maxStoneLines {
		t.Errorf("output cap = %d", len(s.Output))
	}
	empty := Stone{}
	empty.Recall(-1)
	for _, kind := range []int{LineOutput, LineError, LineEcho, LineHint} {
		_ = StoneLine{Kind: kind}.color()
	}
}

func TestGrimoireLayout(t *testing.T) {
	pages := []view.GrimoirePage{{ID: "a"}, {ID: "b", Requires: []string{"a"}}, {ID: "c", Requires: []string{"b", "missing"}}, {ID: "h", State: "hidden"}, {ID: "loop", Requires: []string{"loop"}}}
	l := LayoutKnowledge(pages, KmapRowsTiles)
	a, b, c := mustFind(t, l, "a"), mustFind(t, l, "b"), mustFind(t, l, "c")
	if _, ok := l.Find("h"); ok || b.Col <= a.Col || c.Col <= b.Col {
		t.Errorf("layout = %+v", l.Plaques)
	}
	if _, ok := l.Find("loop"); !ok {
		t.Error("a page that requires itself still shows")
	}
}

func TestCards(t *testing.T) {
	f := fullFrame(t)
	for _, state := range []string{view.StateConstruction, view.StateDark, view.StateRuined, view.StateReady} {
		c := StructureCard(view.Structure{Kind: "codex", Label: "The Codex", State: state, Progress: .5, Object: view.ObjectRef{Kind: "Pod", Name: "etcd"}}, f)
		if c.Title != "THE CODEX" || len(c.Lines) < 2 {
			t.Errorf("card %s = %+v", state, c)
		}
	}
	if describeKind(view.StructHerald, f) == "" || describeKind(view.StructTreasury, f) == "" {
		t.Error("lore missing")
	}
	p := f.Pods[1]
	p.Terminating = true
	if c := PodCard(p, view.PodMetrics{}); c.Lines[0].Text != "HUT LAST RITES" {
		t.Errorf("pod card = %+v", c.Lines)
	}
	for reason, want := range map[string]string{"Evicted": "BANISHED", "OOMKilled": "VOIDTAKEN", "": "THE GRADUATE NEVER ARRIVED"} {
		q := f.Pods[1]
		q.Reason = reason
		if got := HearthWord(q); got != want {
			t.Errorf("hearth word for %q = %q", reason, got)
		}
	}
	calm := f
	calm.State.Bottleneck = ""
	if got := Cause(calm); got != "1 of 2 huts ready, 14 boxes on the quay" {
		t.Errorf("calm cause = %q", got)
	}
}

func drawAll(t *testing.T, h *HUD, f Frame) {
	t.Helper()
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	overlays := []string{OverlayNone, OverlayStone, OverlayScriptorium, OverlayGrimoire, OverlaySoulvault, OverlayChronicles, OverlayAge, OverlayReforge, OverlayMenu, OverlaySiege}
	for _, o := range overlays {
		for _, tab := range []string{TabEvents, TabTech, TabKube, TabYAML} {
			h.OverlayID, h.TabID = o, tab
			h.Draw(dst, f)
		}
	}
	h.OpenDetails(Selection{Kind: "hearth", ID: "default/porter-a", Pod: f.Pods[0]}, DetailDescribe)
	h.Draw(dst, f)
}

func TestDrawEverything(t *testing.T) {
	runner := &fakeRunner{}
	f := fullFrame(t)
	for _, lib := range []*gfx.Library{nil, gfx.Load(art.Build())} {
		h := New(lib, runner.backend())
		h.Toast("hello", false)
		h.Toast("bad", true)
		h.Banner("KEEP NODE-A CORDONED")
		h.Vignette()
		h.gate.reopened = true
		h.Stone.Append(LineOutput, "x")
		h.Script.Result, h.Script.ErrLine = []string{"error: line 2"}, 2
		h.Stone.SetInput("get pods lan")
		for _, page := range []string{PageMain, PageBuild, PageSummon, PageMore} {
			h.PageID = page
			buttons := h.Buttons(f)
			h.Update(Input{MX: buttons[len(buttons)-1].Rect.X + 1, MY: buttons[len(buttons)-1].Rect.Y + 1}, f)
			drawAll(t, h, f)
		}
		for _, sel := range []Selection{
			{Kind: "structure", ID: "s", Structure: view.Structure{Kind: "keep", Label: "Lone Tower", State: view.StateConstruction, Progress: .3}},
			{Kind: "walker", ID: "default/porter-b", Pod: f.Pods[1]},
		} {
			f.Selection = sel
			drawAll(t, h, f)
		}
	}
	empty := fullFrame(t)
	empty.Events, empty.State.Chronicles, empty.State.AgeUp, empty.State.Objectives = nil, nil, view.AgeUpView{}, nil
	empty.State.Mastery, empty.State.Reveals, empty.State.Reforge.Available = view.MasteryView{}, []string{"serve", "gold"}, false
	h := New(nil, Backend{})
	drawAll(t, h, empty)
	resolved := fullFrame(t)
	resolved.State.Chronicles[0].Resolved, resolved.State.Chronicles[0].Correct = true, true
	resolved.State.Reforge.Available = false
	resolved.Events = append(resolved.Events, Tiding{Kind: "omen", Text: "A drain is running at porter-a!", Reason: "Omen"})
	h.Stone.Tall, h.Reduce = true, true
	h.Script.Result, h.Script.OK, h.Script.ResultAt, h.Script.Confirm = []string{"configured"}, true, h.Now, "SCROLL"
	drawAll(t, h, resolved)
	resolved.State.AgeUp = view.AgeUpView{}
	h.Script.Confirm = ""
	drawAll(t, h, resolved)
}

func TestTitle(t *testing.T) {
	title := &Title{}
	if got := title.Update(Input{Keys: []ebiten.Key{ebiten.KeyEnter}}); got != nil {
		t.Error("enter needs a save")
	}
	if _, ok := hasIntent(title.Update(Input{Keys: []ebiten.Key{ebiten.KeyN}}), IntentNew); !ok {
		t.Error("N does not start the first mode")
	}
	if _, ok := HitID(title.Buttons(), "title:load"); ok {
		t.Error("continue shows only when a save exists")
	}
	title.HasSave, title.Note = true, "note"
	b := title.Buttons()[1]
	if b.ID != "title:load" {
		t.Fatalf("continue stands under the new-game button: %s", b.ID)
	}
	if _, ok := hasIntent(title.Update(Input{MX: b.Rect.X + 1, MY: b.Rect.Y + 1, Click: true}), IntentLoad); !ok {
		t.Error("continue loads")
	}
	if _, ok := hasIntent(title.Update(Input{Keys: []ebiten.Key{ebiten.KeyEnter}}), IntentLoad); !ok {
		t.Error("enter continues")
	}
	if title.Update(Input{MX: 1, MY: 1, Click: true}) != nil {
		t.Error("clicking nothing does nothing")
	}
	dst := ebiten.NewImage(Width, Height)
	title.Draw(dst)
	dst.Deallocate()
	_ = model.HearthReady
}
