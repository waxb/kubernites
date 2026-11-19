package hud

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

type textPanel struct {
	name    string
	bounds  func(h *HUD) ui.Rect
	draw    func(h *HUD, dst *ebiten.Image, f Frame)
	buttons func(h *HUD, f Frame) []Button
}

type namesMode struct {
	name  string
	namer func(string) string
}

func namesModes() []namesMode {
	return []namesMode{{"STORY", lexicon.StoryText}, {"KUBERNETES", names.Kubernetes(4096)}}
}

func (m namesMode) apply() {
	if m.name == "STORY" {
		ui.SetStoryNamer(m.namer)
		return
	}
	ui.SetNamer(m.namer)
}

func (m namesMode) story() bool { return m.name == "STORY" }

func recordRuns(draw func()) []ui.TextRun {
	var runs []ui.TextRun
	stop := ui.RecordText(func(r ui.TextRun) { runs = append(runs, r) })
	defer stop()
	draw()
	return runs
}

func textProblems(runs []ui.TextRun, bounds ui.Rect) []string {
	var out []string
	for i, r := range runs {
		if !inRect(r.Rect, bounds) {
			out = append(out, fmt.Sprintf("%q at %+v leaves %+v", r.Text, r.Rect, bounds))
		}
		for _, o := range runs[i+1:] {
			if o != r && runsCollide(r.Rect, o.Rect) {
				out = append(out, fmt.Sprintf("%q at %+v collides with %q at %+v", r.Text, r.Rect, o.Text, o.Rect))
			}
		}
	}
	return out
}

func labelProblems(buttons []Button) []string {
	var out []string
	var font ui.Font
	for _, b := range buttons {
		face := b.Rect
		if b.Face.W > 0 {
			face = b.Face
		}
		lines := faceLines(b.Label, face)
		for _, l := range lines {
			if w := font.Width(l, 1); w > face.W-2 {
				out = append(out, fmt.Sprintf("button %s label %q is %dpx in a %dpx face", b.ID, ui.Name(b.Label), w, face.W))
			}
		}
		if len(lines) > 1 && strings.Join(lines, " ") != ui.Name(b.Label) {
			out = append(out, fmt.Sprintf("button %s label %q is cut to %q", b.ID, ui.Name(b.Label), lines))
		}
	}
	return out
}

func faceLines(label string, face ui.Rect) []string {
	if label == "" {
		return nil
	}
	if face.H >= ChipH+ui.LineHeight {
		return ui.Font{}.WrapLines(label, face.W-ButtonPad, 1, RemedyMaxLines)
	}
	return []string{label}
}

func coveredText(runs []ui.TextRun, buttons []Button) []string {
	var out []string
	for _, r := range runs {
		for _, b := range buttons {
			face := b.Rect
			if b.Face.W > 0 {
				face = b.Face
			}
			if runsCollide(r.Rect, face) && !inRect(r.Rect, face) {
				out = append(out, fmt.Sprintf("%q at %+v runs under button %s at %+v", r.Text, r.Rect, b.ID, face))
			}
		}
	}
	return out
}

func inRect(r, b ui.Rect) bool {
	return r.X >= b.X && r.Y >= b.Y && r.X+r.W <= b.X+b.W && r.Y+r.H <= b.Y+b.H
}

func runsCollide(a, b ui.Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

func assertTextFits(t *testing.T, h *HUD, f Frame, p textPanel) {
	t.Helper()
	defer ui.SetNamer(nil)
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	cut := map[string][]string{}
	for _, mode := range namesModes() {
		mode.apply()
		f.TrueNames = !mode.story()
		runs := recordRuns(func() { p.draw(h, dst, f) })
		cut[mode.name] = truncated(runs)
		problems := textProblems(runs, p.bounds(h))
		if p.buttons != nil {
			buttons := p.buttons(h, f)
			problems = append(problems, labelProblems(buttons)...)
			problems = append(problems, coveredText(runs, buttons)...)
		}
		for _, msg := range problems {
			t.Errorf("%s [%s]: %s", p.name, mode.name, msg)
		}
	}
	if story, k8s := cut["STORY"], cut["KUBERNETES"]; len(k8s) > len(story) {
		t.Errorf("%s: Kubernetes names cut more text (%q) than story names (%q)", p.name, k8s, story)
	}
}

func truncated(runs []ui.TextRun) []string {
	var out []string
	for _, r := range runs {
		if strings.HasSuffix(r.Text, "...") {
			out = append(out, r.Text)
		}
	}
	return out
}

func fixed(r ui.Rect) func(*HUD) ui.Rect { return func(*HUD) ui.Rect { return r } }

func TestTextFitsTheRegistry(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	assertTextFits(t, h, f, textPanel{name: "registry", bounds: fixed(Overlay), draw: (*HUD).DrawAcademy, buttons: (*HUD).AcademyButtons})
}

func TestRecorderSeesTheRegistryTitleOnce(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	ui.SetNamer(names.Kubernetes(64))
	defer ui.SetNamer(nil)
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	runs := recordRuns(func() { h.DrawAcademy(dst, f) })
	for _, want := range []string{"PORTER PODS, TAG 1.0", "porter:1.0"} {
		if !hasRun(runs, want) {
			t.Errorf("no %q drawn in %+v", want, runs)
		}
	}
}

func hasRun(runs []ui.TextRun, text string) bool {
	for _, r := range runs {
		if r.Text == text {
			return true
		}
	}
	return false
}

func TestCellLabelWrapsTheNamedLabel(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, mode := range namesModes() {
		mode.apply()
		for _, label := range []string{"STONE KEEP", "GREAT KEEP", "CITADEL KEEP", "PORTERS", "SERVE", "RESTART A WEB CREW"} {
			rows := CellLabel(label, 11)
			if len(rows) > 2 {
				t.Errorf("[%s] %q takes %d rows", mode.name, label, len(rows))
			}
			for _, row := range rows {
				if n := len([]rune(ui.Name(row))); n > 11 {
					t.Errorf("[%s] %q row %q draws %d chars", mode.name, label, ui.Name(row), n)
				}
			}
		}
	}
}

var overlayBounds = map[string]func(h *HUD) ui.Rect{
	OverlayMenu: fixed(MenuBox), OverlayBook: fixed(BookBox), OverlayRoad: fixed(RoadBox), OverlayEnding: fixed(RoadBox), OverlaySiege: fixed(SiegeBox), OverlayWelcome: fixed(WelcomeBox),
	OverlayChapters: fixed(KnowledgeBox), OverlayDebrief: func(h *HUD) ui.Rect { return h.DebriefRect() },
	OverlayStone: func(h *HUD) ui.Rect { return h.StoneRect() }, OverlayChooser: func(h *HUD) ui.Rect { return h.ChooserRect() }, OverlayGrimoire: fixed(ui.Rect{X: -Width, Y: 0, W: 3 * Width, H: Height}),
}

func overlayPanel(key string, spec OverlaySpec) textPanel {
	bounds, ok := overlayBounds[key]
	if !ok {
		bounds = fixed(Overlay)
	}
	return textPanel{name: "overlay " + key, bounds: bounds,
		draw: func(h *HUD, dst *ebiten.Image, f Frame) { h.OverlayID = key; spec.Draw(h, dst, f) },
		buttons: func(h *HUD, f Frame) []Button {
			h.OverlayID = key
			return spec.Buttons(h, f)
		}}
}

func TestTextFitsEveryOverlay(t *testing.T) {
	f := fullFrame(t)
	for key, spec := range Overlays {
		if key == OverlayReforge {
			continue
		}
		h := New(nil, Backend{})
		if key == OverlayDetails {
			h.OpenDetails(Selection{Kind: "hearth", ID: "default/porter-a", Pod: f.Pods[0]}, DetailDescribe)
		}
		assertTextFits(t, h, f, overlayPanel(key, spec))
	}
}

func TestTextFitsEveryLayer(t *testing.T) {
	f := fullFrame(t)
	skip := map[string]bool{"overlay": true, "tooltip": true, "worldtip": true, GuideLayer: true, "vignette": true}
	for _, l := range HudLayers {
		if skip[l.Name] {
			continue
		}
		h := New(nil, Backend{})
		h.Toast("The porters carry the boxes now.", false)
		assertTextFits(t, h, f, textPanel{name: "layer " + l.Name, bounds: fixed(ui.Rect{W: Width, H: Height}), draw: l.Draw})
	}
}

func TestGlossedDropsAnEcho(t *testing.T) {
	defer ui.SetNamer(nil)
	if title, gloss := Glossed("THE UNHOUSED", "Pending (FailedScheduling)"); title != "THE UNHOUSED" || gloss == "" {
		t.Errorf("story = %q %q", title, gloss)
	}
	ui.SetNamer(names.Kubernetes(64))
	if title, gloss := Glossed("THE UNHOUSED", "Pending (FailedScheduling)"); title != "Pending (FailedScheduling)" || gloss != "" {
		t.Errorf("kubernetes = %q %q", title, gloss)
	}
	if title, gloss := Glossed("A Stray Deletion", "Pod deleted"); title != "A Stray Deletion" || gloss != "Pod deleted" {
		t.Errorf("a different gloss stays = %q %q", title, gloss)
	}
}

func TestTextFitsTheContracts(t *testing.T) {
	f := contractFrame(t)
	h := New(nil, Backend{})
	assertTextFits(t, h, f, overlayPanel(OverlayContracts, Overlays[OverlayContracts]))
}

func TestContractRewardsFitWithKubernetesNames(t *testing.T) {
	defer ui.SetNamer(nil)
	reward := view.ContractReward{Gold: 170, Wisdom: 4, Favor: 6, Boost: 7960}
	for _, mode := range namesModes() {
		mode.apply()
		for _, l := range (ui.Font{}).WrapLines(RewardText(reward), ContractRight-12, 1, 2) {
			if strings.HasSuffix(l, "...") {
				t.Errorf("[%s] reward is cut: %q", mode.name, l)
			}
		}
	}
}

func TestGlossTipDropsTheEchoedTerm(t *testing.T) {
	defer ui.SetNamer(nil)
	if got := GlossTip("Guildhall", "Service (parcels)", "It sends."); got != "Guildhall (Service): It sends." {
		t.Errorf("story tip = %q", got)
	}
	ui.SetNamer(names.Kubernetes(64))
	if got := ui.Name(GlossTip("Guildhall", "Service (parcels)", "It sends.")); got != "Service (web requests): It sends." {
		t.Errorf("kubernetes tip = %q", got)
	}
}

func TestTextFitsTheSelectionCards(t *testing.T) {
	base := fullFrame(t)
	sels := []Selection{
		{Kind: "hearth", ID: "default/porter-a", Pod: base.Pods[0]},
		{Kind: "structure", ID: "k", Structure: view.Structure{ID: "k", Kind: view.StructKeep, Label: "Lone Tower", State: view.StateReady,
			Object: view.ObjectRef{Kind: "Node", Name: "lone-tower"}}},
		{Kind: "structure", ID: "g", Structure: view.Structure{ID: "g", Kind: view.StructGuildhall, Label: "Guildhall", State: view.StateReady}},
	}
	for _, sel := range sels {
		f := fullFrame(t)
		f.Selection = sel
		for _, l := range HudLayers {
			if l.Name != "console" && l.Name != "windows" {
				continue
			}
			h := New(nil, Backend{})
			assertTextFits(t, h, f, textPanel{name: "card " + sel.Kind + " " + sel.ID + " " + l.Name, bounds: fixed(ui.Rect{W: Width, H: Height}),
				draw: l.Draw, buttons: (*HUD).SelectionButtons})
		}
	}
}

func TestToastsWaitWhileAWindowCoversThem(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.Clock = 10
	h.Update(Input{}, f)
	h.Toast("The porters carry the boxes now.", false)
	h.Open(OverlaySoulvault)
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	for step := 1; step <= 10; step++ {
		f.Clock = 10 + float64(step)
		h.Update(Input{}, f)
		if hasRun(recordRuns(func() { h.DrawToasts(dst, f) }), "The porters carry the boxes now.") {
			t.Fatal("a toast is drawn over the open registry")
		}
	}
	if len(h.Toasts()) != 1 {
		t.Fatalf("the toast expired behind the window: %+v", h.Toasts())
	}
	h.OverlayID = OverlayNone
	h.Update(Input{}, f)
	if !hasRun(recordRuns(func() { h.DrawToasts(dst, f) }), "The porters carry the boxes now.") {
		t.Error("the toast shows once the window closes")
	}
	f.Clock += ToastLife + 1
	h.Update(Input{}, f)
	if len(h.Toasts()) != 0 {
		t.Error("the toast still expires in the open")
	}
}

func TestUnechoedDropsARepeatedTerm(t *testing.T) {
	defer ui.SetNamer(nil)
	items := []string{"War Horns: HorizontalPodAutoscaler", "Bindings: resources.limits", "no colon here"}
	for _, item := range items {
		if got := Unechoed(item); got != item {
			t.Errorf("story %q became %q", item, got)
		}
	}
	ui.SetNamer(names.Kubernetes(64))
	want := []string{"HorizontalPodAutoscaler", "Bindings: resources.limits", "no colon here"}
	for i, item := range items {
		if got := Unechoed(item); got != want[i] {
			t.Errorf("kubernetes %q became %q, want %q", item, got, want[i])
		}
	}
}
