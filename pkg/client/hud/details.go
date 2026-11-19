package hud

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	DetailDescribe   = "describe"
	DetailYAML       = "yaml"
	DetailLogs       = "logs"
	DetailExplain    = "explain"
	detailTitleChars = 32
)

var lore = map[string]string{
	view.StructGate: "The Gate is where the harbor's boxes enter the realm. Each box is a request. " +
		"In Kubernetes terms the Gate is the edge of the cluster: the load balancer or Ingress " +
		"that sends traffic to your Services. Carry boxes by hand until a crew can do it.",
	view.StructTreasury: "The Treasury holds the gold your realm earns when boxes are delivered. " +
		"It is not a Kubernetes object: it is your score. Gold pays for buildings, Soulstones and berths.",
}

type Details struct {
	Title  string
	Sel    Selection
	Tab    string
	Lines  []string
	OK     bool
	Scroll ui.Scroll
}

type target struct {
	kind, namespace, name string
	restarts              int
}

func targetOf(sel Selection) (target, bool) {
	switch sel.Kind {
	case "hearth", "walker":
		return target{kind: "pod", namespace: sel.Pod.Namespace, name: sel.Pod.Name, restarts: int(sel.Pod.Restarts)}, true
	case SelectCrew, SelectForeman:
		return crewDetailTarget(sel)
	case "structure":
		o := sel.Structure.Object
		if o.Kind == "" {
			return target{}, false
		}
		return target{kind: kubectlKind(o.Kind), namespace: o.Namespace, name: o.Name, restarts: int(sel.Pod.Restarts)}, true
	}
	return target{}, false
}

var shortKinds = map[string]string{"HorizontalPodAutoscaler": "hpa"}

func kubectlKind(kind string) string {
	if short, ok := shortKinds[kind]; ok {
		return short
	}
	return strings.ToLower(kind)
}

func nsFlag(ns string) string {
	if ns == "" {
		return ""
	}
	return " -n " + ns
}

func DetailCommands(sel Selection, tab string) []string {
	t, ok := targetOf(sel)
	if !ok {
		return nil
	}
	switch tab {
	case DetailYAML:
		return []string{"get " + t.kind + " " + t.name + nsFlag(t.namespace) + " -o yaml"}
	case DetailExplain:
		return []string{"explain " + t.kind}
	case DetailLogs:
		if t.kind != "pod" {
			return nil
		}
		base := "logs " + t.name + nsFlag(t.namespace)
		if t.restarts > 0 {
			return []string{base, base + " --previous"}
		}
		return []string{base}
	}
	return []string{"describe " + t.kind + " " + t.name + nsFlag(t.namespace)}
}

func HasLogs(sel Selection) bool {
	t, ok := targetOf(sel)
	return ok && t.kind == "pod"
}

func SelectionTitle(sel Selection) string {
	switch sel.Kind {
	case "structure":
		return sel.Structure.Label
	case SelectCrew, SelectForeman:
		return crewSelectionTitle(sel)
	case SelHaven:
		return "The harbor"
	case "hearth", "walker":
		if sel.Pod.Static {
			return staticDetail + " - " + sel.Pod.Name
		}
		return sel.Pod.Profession + " - " + sel.Pod.Name
	}
	return ""
}

func (h *HUD) OpenDetails(sel Selection, tab string) {
	h.DetailsCard = Details{Title: SelectionTitle(sel), Sel: sel, Tab: tab}
	h.OverlayID = OverlayDetails
	h.fillDetails()
}

func (h *HUD) Details() Details { return h.DetailsCard }

func (h *HUD) fillDetails() {
	d := &h.DetailsCard
	d.Scroll = ui.Scroll{}
	cmds := DetailCommands(d.Sel, d.Tab)
	if len(cmds) == 0 {
		d.OK, d.Lines = true, h.LoreLines(d.Sel)
		return
	}
	d.OK, d.Lines = true, nil
	if d.Sel.Pod.Static && d.Sel.Kind != "structure" {
		for _, l := range h.StaticLore() {
			d.Lines = append(d.Lines, h.Font.Wrap(l, detailBody().W-8, 1)...)
		}
	}
	for i, cmd := range cmds {
		if i > 0 {
			d.Lines = append(d.Lines, "", ui.Bridge("--- its life before ---", "--- previous container (kubectl "+cmd+") ---"))
		}
		res := h.inspect(cmd)
		h.Echo(cmd, FromDetails)
		d.OK = d.OK && res.OK
		d.Lines = append(d.Lines, appLines(d.Tab, h.AnswerLines("kubectl "+cmd, res))...)
		if !res.OK && strings.Contains(res.Output, "refused") {
			d.Lines = append(d.Lines, "", h.RefusedHint())
		}
	}
}

func (h *HUD) AnswerLines(label string, res view.CommandResult) []string {
	out := []string{"$ " + label}
	if res.Summary != "" && ui.StoryNaming() {
		out = append(out, h.Font.Wrap(res.Summary, detailBody().W-8, 1)...)
		out = append(out, "")
	}
	answer := StoryAnswer(res.Output, res.OK)
	if res.OK && ui.StoryNaming() {
		answer, h.Stone.folded = foldSheet(answer)
	}
	return append(out, answer...)
}

func appLines(tab string, lines []string) []string {
	if tab != DetailLogs || !ui.Naming() {
		return lines
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l
		if i > 0 {
			out[i] = ui.Name(l)
		}
	}
	return out
}

func (h *HUD) inspect(cmd string) view.CommandResult {
	return call(h.backend.Inspect, cmd)
}

func (h *HUD) kubectl(cmd string) view.CommandResult {
	return call(h.backend.Kubectl, cmd)
}

func call(run Runner, cmd string) view.CommandResult {
	if run == nil {
		return view.CommandResult{Output: "the Speaking Stone is silent"}
	}
	return run(cmd)
}

func (h *HUD) LoreLines(sel Selection) []string {
	text, ok := lore[sel.Structure.Kind]
	if sel.Kind == SelHaven {
		text, ok = HarborLore(), true
	}
	if !ok {
		text = "Nothing more is known of this place."
	}
	return h.Font.Wrap(text, detailBody().W-8, 1)
}

func SplitOutput(out string) []string {
	out = strings.ReplaceAll(strings.TrimRight(out, "\n"), "\t", "    ")
	if out == "" {
		return []string{"(no output)"}
	}
	return strings.Split(out, "\n")
}

func (h *HUD) detailKeys(in Input) bool {
	page := max(detailVisible()-1, 1)
	steps := map[ebiten.Key]int{ebiten.KeyPageDown: page, ebiten.KeyPageUp: -page, ebiten.KeyArrowDown: 1, ebiten.KeyArrowUp: -1}
	for k, step := range steps {
		if in.Pressed(k) {
			h.DetailsCard.scrollBy(step)
			return true
		}
	}
	return false
}

func (d *Details) scrollBy(step int) {
	d.Scroll = d.Scroll.By(step, len(d.Lines), detailVisible())
}

func detailBody() ui.Rect {
	return ui.Rect{X: Overlay.X + 8, Y: Overlay.Y + 28, W: Overlay.W - 16, H: Overlay.H - 36}
}

func detailVisible() int {
	return ui.VisibleLines(detailBody().H-6, 1)
}

func (h *HUD) pressSelection(rest string, f Frame) []Intent {
	if intents, ok := h.pressView(rest, f); ok {
		return intents
	}
	if rest == "close" {
		return append(h.leaveObject(f), Intent{Kind: IntentSound, Command: "click"})
	}
	tab := DetailDescribe
	if rest == "logs" {
		tab = DetailLogs
	}
	h.OpenDetails(f.Selection, tab)
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) pressDetails(rest string, _ Frame) []Intent {
	if rest == "close" {
		h.OverlayID = OverlayNone
		return nil
	}
	if rest == EditTab {
		return h.editSelected()
	}
	h.DetailsCard.Tab = rest
	h.fillDetails()
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) detailButtons() []Button {
	d := h.DetailsCard
	out := []Button{}
	tabs := []string{DetailDescribe, DetailYAML, DetailLogs, DetailExplain}
	x := Overlay.X + Overlay.W - 32 - len(tabs)*68
	for _, tab := range tabs {
		_, hasTarget := targetOf(d.Sel)
		disabled := !hasTarget || (tab == DetailLogs && !HasLogs(d.Sel))
		out = append(out, Button{ID: "det:" + tab, Rect: ui.Rect{X: x, Y: Overlay.Y + 6, W: 64, H: 14}, Label: strings.ToUpper(tab),
			Icon: "icon/" + tab, Selected: d.Tab == tab, Disabled: disabled, Tip: "kubectl " + strings.Join(DetailCommands(d.Sel, tab), " / ")})
		x += 68
	}
	out = append(out, h.editTabButton(d)...)
	out = append(out, closeButton(Overlay))
	return out
}

func (h *HUD) DetailLineAt(contains string) (ui.Rect, bool) {
	body := detailBody()
	for i, l := range h.DetailsCard.Lines {
		if i >= detailVisible() {
			break
		}
		if strings.Contains(l, contains) {
			return ui.Rect{X: body.X + 4, Y: body.Y + 3 + i*ui.LineHeight, W: body.W - 8, H: ui.LineHeight}, true
		}
	}
	return ui.Rect{}, false
}
