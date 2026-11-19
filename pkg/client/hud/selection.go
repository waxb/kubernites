package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var hearthWords = map[string]string{
	model.HearthReady: "BURNING", model.HearthKindling: "KINDLING", model.HearthGuttering: "GUTTERING",
	model.HearthGhost: "THE GRADUATE NEVER ARRIVED", model.HearthBanished: "FAILED", model.HearthPending: "UNHOUSED",
}

var hearthColors = map[string]color.RGBA{
	model.HearthReady: ui.Palette.Green, model.HearthKindling: ui.Palette.GoldBright, model.HearthGuttering: ui.Palette.Red,
	model.HearthGhost: ui.Palette.Purple, model.HearthBanished: ui.Palette.Muted, model.HearthPending: ui.Palette.TextDim,
}

type Card struct {
	Portrait string `json:"-"`
	Title    string `json:"-"`
	K8s      string `json:"-"`
	Lines    []Line `json:"-"`
	progress float64
}

type Line struct {
	Text   string     `json:"-"`
	Clr    color.RGBA `json:"-"`
	Bar    bool       `json:"-"`
	Value  float64    `json:"-"`
	Fill   color.RGBA `json:"-"`
	Suffix string     `json:"-"`

	reserved    float64
	reserveFill color.RGBA

	Optional bool `json:"-"`
	lesson   string
	plain    string
	status   bool
}

func PortraitBox() ui.Rect {
	return ui.Rect{X: CardBox.X + 5, Y: CardBox.Y + 5, W: 56, H: 64}
}

func (f Frame) cardSelection() bool {
	return !f.Selection.Empty() && !f.Teaching(SurfaceGate)
}

func (h *HUD) SelectionButtons(f Frame) []Button {
	if !f.cardSelection() {
		return nil
	}
	out := []Button{selectionClose()}
	if f.Selection.Gone {
		return out
	}
	if b, _, ok := crewLink(f); ok {
		out = append(out, b)
	}
	return append(out, h.cardLineButtons(f)...)
}

func StructureCard(s view.Structure, f Frame) Card {
	c := Card{Portrait: gfx.StructurePortrait(s, gfx.AgeTier(f.State.Age)), Title: strings.ToUpper(s.Label), progress: -1}
	if s.Kind == view.StructVault {
		c.Portrait = vaultPortrait(s, f)
	}
	if s.Object.Kind != "" {
		c.K8s = s.Object.Kind + "/" + s.Object.Name
	}
	switch s.State {
	case view.StateConstruction:
		c.Lines = append(c.Lines, Line{Text: fmt.Sprintf("UNDER CONSTRUCTION %d%%", int(s.Progress*100)), Clr: ui.Palette.GoldBright})
		c.progress = s.Progress
	case view.StateDark:
		c.Lines = append(c.Lines, Line{Text: DarkText(s.Kind), Clr: ui.Palette.RedText})
	case view.StateRuined:
		c.Lines = append(c.Lines, Line{Text: "RUINED", Clr: ui.Palette.RedText})
	case view.StateJoining:
		c.Lines = append(c.Lines, Line{Text: joiningText, Clr: ui.Palette.GoldBright})
	default:
		c.Lines = append(c.Lines, Line{Text: "STANDING", Clr: ui.Palette.Green})
	}
	c.Lines = append(c.Lines, KindLines(s, f)...)
	desc := describeKind(s.Kind, f)
	if desc != "" {
		c.Lines = append(c.Lines, Line{Text: desc, Clr: ui.Palette.Text, Optional: true})
	}
	return c
}

func describeKind(kind string, f Frame) string {
	for _, b := range f.State.Buildables {
		if b.Kind == kind {
			return b.K8s + ". " + b.Description
		}
	}
	if text, ok := lore[kind]; ok {
		return text
	}
	return structureLore[kind]
}

var structureLore = map[string]string{
	view.StructHerald:      "CoreDNS. The Herald answers when a hut asks where a Service lives.",
	view.StructWayfinder:   "kube-proxy. The Wayfinder routes boxes from a depot (Service) to the huts behind it.",
	view.StructObservatory: "metrics-server. The Observatory measures the food and housing of every worker.",
}

func PodCard(p model.PodView, m view.PodMetrics) Card {
	c := Card{Portrait: "portrait/" + p.Sprite, Title: strings.ToUpper(p.Profession), K8s: "Pod/" + p.Name, progress: -1}
	if p.Static {
		c.Title = staticTitle
		c.Lines = append(c.Lines, Line{Text: staticLine, Clr: ui.Palette.Cream})
	}
	if p.Focus != "" {
		c.Lines = append(c.Lines, Line{Text: KuberniteLine(p), Clr: ui.Palette.Cream})
	}
	state := HearthWord(p)
	c.Lines = append(c.Lines, Line{Text: "HUT " + state, Clr: hearthColors[p.State]})
	if p.Reason != "" {
		c.Lines = append(c.Lines, Line{Text: p.Reason, Clr: ui.Palette.TextDim})
	}
	keep := p.Node
	if keep == "" {
		keep = "none yet"
	}
	c.Lines = append(c.Lines,
		Line{Text: fmt.Sprintf("RESTARTS %d  KEEP %s", p.Restarts, keep), Clr: ui.Palette.Text, lesson: restartsLesson, plain: "KEEP " + keep},
		Line{Text: ui.Bridge("TRADE "+p.Image, "IMAGE "+p.Image), Clr: ui.Palette.TextDim, lesson: classLesson},
		Line{Text: LaborLine(m), Clr: ui.Palette.Blue, lesson: laborLesson})
	return c
}

func KuberniteLine(p model.PodView) string {
	word, name := ui.Bridge("WORKER", "CONTAINER"), p.Focus
	if ui.StoryNaming() {
		name = strings.TrimPrefix(ui.Name(name), "the ")
	}
	for i, k := range p.People {
		if k.Name == p.Focus {
			return fmt.Sprintf("%s %s (%d OF %d)", word, name, i+1, len(p.People))
		}
	}
	return word + " " + name
}

func WithQoS(c Card, p model.PodView) Card {
	if model.QoSTitle(p.QoS) == "" || p.System {
		return c
	}
	at := 0
	for i, l := range c.Lines {
		if strings.HasPrefix(l.Text, "HUT ") {
			at = i + 1
		}
	}
	lines := append(append(append([]Line(nil), c.Lines[:at]...), Line{Text: model.QoSTitle(p.QoS), Clr: qosColors[p.QoS]}), c.Lines[at:]...)
	c.Lines = append(lines, Line{Text: model.QoSSentence(p.QoS), Clr: ui.Palette.TextDim, Optional: true})
	return c
}

func (h *HUD) CardFor(f Frame) Card {
	switch f.Selection.Kind {
	case "structure":
		return StructureCard(f.Selection.Structure, f)
	case SelHaven:
		return HarborCard(f)
	case "hearth", "walker":
		p := f.Selection.Pod
		if f.Selection.Gone {
			return goneCard(p, f)
		}
		c := taughtLines(PodCard(p, f.Metrics.Pods[p.Namespace+"/"+p.Name]), f)
		if f.Revealed(QosReveal) {
			c = WithQoS(c, p)
		}
		c.Lines = append(c.Lines, rollLines(p, f)...)
		return withWorker(WithDeed(c, p, f), f)
	case SelectCrew, SelectForeman:
		return CrewCard(f)
	}
	return Card{}
}

func selectionClose() Button {
	return closeButtonAs(CardBox, SelClose, "Deselect [Esc or right click]")
}

func (h *HUD) DrawSelection(dst *ebiten.Image, f Frame) {
	if !f.cardSelection() && h.PageID == PageSummon && !f.Teaching(SurfaceGate) {
		h.drawSummonCard(dst, f)
		return
	}
	if !f.cardSelection() {
		h.drawRealmCard(dst, f)
		return
	}
	c := h.CardFor(f)
	box := PortraitBox()
	ui.Well(dst, box)
	if !h.DrawWorkerView(dst, box.Inset(4), f) {
		h.drawCardPortrait(dst, c.Portrait, box.Inset(4))
	}
	x, y, w := box.X+box.W+8, CardBox.Y+6, CardBox.X+CardBox.W-box.X-box.W-14
	head := CardHeadWidth()
	title, ref := SplitCardHead(c.Title, c.K8s, ui.MaxChars(head, 1))
	h.Font.DrawShadow(dst, ui.Clip(title, ui.MaxChars(head, 1)), x, y, 1, ui.Palette.GoldBright)
	y += ui.LineHeight
	if ref != "" {
		RefFont().Draw(dst, RefFont().Clip(ref, ui.MaxChars(head, 1)), x, y, 1, ui.Palette.Blue)
		y += ui.LineHeight
	}
	h.drawCardLines(dst, h.FitLines(c.Lines, w, (CardBottom(c, f)-y)/ui.LineHeight), x, y, w, CardBottom(c, f))
	if c.progress >= 0 {
		ui.Bar(dst, ui.Rect{X: x, Y: CardBox.Y + CardBox.H - 12, W: w, H: 7}, c.progress, ui.Palette.Green)
	}
	h.drawCrewLink(dst, f)
	h.drawButton(dst, selectionClose())
}

func RefFont() ui.Font {
	if ui.StoryNaming() {
		return ui.Font{}
	}
	return Literal
}

func objectRef(kind, group, name string) string {
	if ui.StoryNaming() {
		return kind + "/" + name
	}
	return ui.Settle(group + "/" + name)
}

func CardHeadWidth() int {
	box := PortraitBox()
	return CardBox.X + CardBox.W - box.X - box.W - 14 - selectionClose().Rect.W
}

func SplitCardHead(title, ref string, chars int) (string, string) {
	fits := func(text string) bool { return len([]rune(ui.Name(text))) <= chars }
	if !fits(ref) {
		if _, name, slashed := strings.Cut(ref, "/"); slashed {
			ref = name
		}
	}
	if !fits(title) {
		if kind, _, found := strings.Cut(title, " "); found {
			title = kind
		}
	}
	return title, ref
}

func cardFloor() int {
	return CardBox.Y + CardBox.H - 4
}

func CardBottom(c Card, f Frame) int {
	bottom := cardFloor()
	if c.progress >= 0 {
		bottom -= 12
	}
	if b, _, ok := crewLink(f); ok {
		bottom -= b.Rect.H
	}
	return bottom
}

func (h *HUD) drawCardLines(dst *ebiten.Image, lines []Line, x, y, w, bottom int) {
	room := (bottom - y) / ui.LineHeight
	for _, l := range lines {
		if l.Bar {
			h.drawBarLine(dst, l, x, y, w)
			y += ui.LineHeight
			if room--; room <= 0 {
				return
			}
			continue
		}
		rows := h.Font.WrapLines(l.Text, w, 1, room)
		for _, row := range rows {
			h.Font.Draw(dst, row, x, y, 1, l.Clr)
			y += ui.LineHeight
		}
		if room -= len(rows); room <= 0 {
			return
		}
	}
}

var qosColors = map[string]color.RGBA{model.QoSBestEffort: ui.Palette.RedText, model.QoSBurstable: ui.Palette.GoldBright, model.QoSGuaranteed: ui.Palette.Green}

func Hands(milli float64) string {
	return ui.Bridge(fmt.Sprintf("%.2fH", milli/1000), fmt.Sprintf("%.0fm", milli))
}

func LaborLine(m view.PodMetrics) string {
	return ui.Bridge(fmt.Sprintf("FOOD %s  HOUSING %s  %.1f/S", foodCount(m.CPUMilli), housingCount(m.MemoryBytes), m.ServedRate),
		fmt.Sprintf("CPU %s  MEMORY %dMi  %.1f/S", Hands(m.CPUMilli), int(m.MemoryBytes/(1<<20)), m.ServedRate))
}

func HearthWord(p model.PodView) string {
	switch {
	case p.Terminating:
		return "LAST RITES"
	case p.Reason == "Evicted":
		return "BANISHED"
	case p.Reason == "OOMKilled":
		return "VOIDTAKEN"
	case firstExit(p):
		return ui.Bridge("STOPPED", strings.ToUpper(p.Reason))
	}
	return hearthWords[p.State]
}

func hearthSelected(f Frame) bool {
	return f.Selection.Kind == "hearth" || f.Selection.Kind == "walker"
}

func firstExit(p model.PodView) bool {
	return p.State == model.HearthGuttering && p.Reason != "" && p.Reason != "CrashLoopBackOff"
}
