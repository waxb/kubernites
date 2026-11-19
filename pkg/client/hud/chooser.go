package hud

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/soulstone"
)

const (
	OverlayChooser = "chooser"
	pickPrefix     = "pick"
	chooserW       = 250
	chooserMaxW    = 360
	chooserHead    = 28
	chooserRowH    = 15
	chooserGap     = 3
	chooserPad     = 6
	debugKey       = "debug"
	appKey         = "app"
)

type chooserState struct {
	title   string
	from    string
	sel     string
	x       int
	Options []CardAction `json:"-"`
}

func (h *HUD) openChooser(a CardAction, f Frame) {
	h.Ops.Chooser = chooserState{title: a.Label + " " + SelectionTitle(f.Selection), from: CardSource(f), sel: f.Selection.ID,
		x: chooserX(h.CrewRects(f)), Options: a.Choices}
	h.OverlayID = OverlayChooser
}

func chooserX(beside []ui.Rect) int {
	x := CardBox.X + cardLineX
	for _, r := range beside {
		x = max(x, r.X+r.W+4)
	}
	return min(x, Width-chooserW-4)
}

func (h *HUD) syncChooser(f Frame) {
	if h.OverlayID == OverlayChooser && h.Ops.Chooser.sel != "" && f.Selection.ID != h.Ops.Chooser.sel {
		h.OverlayID = OverlayNone
	}
}

func chooserRect(x, n, w int) ui.Rect {
	height := chooserHead + n*(chooserRowH+chooserGap) + chooserPad
	if x == 0 {
		x = CardBox.X + cardLineX
	}
	return ui.Rect{X: min(x, Width-w-4), Y: CardBox.Y - height - 4, W: w, H: height}
}

func (h *HUD) ChooserRect() ui.Rect {
	return chooserRect(h.Ops.Chooser.x, len(h.Ops.Chooser.Options), h.chooserWidth())
}

func (h *HUD) chooserWidth() int {
	c := h.Ops.Chooser
	w := max(chooserW, h.Font.Width(strings.ToUpper(c.title), 1)+40)
	for _, o := range c.Options {
		w = max(w, h.Font.Width(o.Label, 1)+8+2*chooserPad)
	}
	return min(w, chooserMaxW)
}

func pickRect(box ui.Rect, i int) ui.Rect {
	return ui.Rect{X: box.X + chooserPad, Y: box.Y + chooserHead + i*(chooserRowH+chooserGap), W: box.W - 2*chooserPad, H: chooserRowH}
}

func (h *HUD) chooserButtons(_ Frame) []Button {
	c := h.Ops.Chooser
	box := h.ChooserRect()
	out := []Button{}
	for i, o := range c.Options {
		out = append(out, Button{ID: fmt.Sprintf("%s:%d", pickPrefix, i), Rect: pickRect(box, i), Label: o.Label, Tip: o.Tip, Disabled: o.Disabled})
	}
	return append(out, closeButton(box))
}

func (h *HUD) pressPick(rest string, f Frame) []Intent {
	c := h.Ops.Chooser
	var i int
	if _, err := fmt.Sscan(rest, &i); err != nil || i < 0 || i >= len(c.Options) || c.sel != f.Selection.ID || c.Options[i].Disabled {
		return nil
	}
	h.OverlayID = OverlayNone
	return h.RunAction(c.Options[i], f.Selection, c.from)
}

func (h *HUD) drawChooser(dst *ebiten.Image, _ Frame) {
	c := h.Ops.Chooser
	box := h.ChooserRect()
	drawPanel(dst, box)
	h.Font.DrawShadow(dst, ui.Clip(strings.ToUpper(c.title), ui.MaxChars(box.W-40, 1)), box.X+8, box.Y+6, 1, ui.Palette.Gold)
	for _, b := range h.chooserButtons(Frame{}) {
		if IsClose(b) {
			h.drawButton(dst, b)
			continue
		}
		h.drawPick(dst, b)
	}
}

func (h *HUD) drawPick(dst *ebiten.Image, b Button) {
	b = h.liveGate(b)
	state := ui.ButtonState{Hovered: h.Hovering && h.Hover.ID == b.ID, Disabled: b.Disabled}
	text := ui.ButtonFace(dst, b.Rect, state)
	h.Font.Draw(dst, ui.Clip(b.Label, ui.MaxChars(b.Rect.W-8, 1)), b.Rect.X+4, b.Rect.Y+4, 1, text)
}

func TaintChoices(node string, n model.NodeView) []CardAction {
	out := []CardAction{
		wardChoice(node, "NoSchedule", "Ward it: keep new huts away", n),
		wardChoice(node, "PreferNoSchedule", "Soft Ward: only when no room is left", n),
		wardChoice(node, "NoExecute", "Harsh Ward: also evict its huts", n),
	}
	for _, spec := range n.TaintSpecs {
		if strings.HasPrefix(spec, "node.kubernetes.io/") {
			continue
		}
		key, effect := taintParts(spec)
		lift := key + ":" + effect + "-"
		out = append(out, CardAction{Key: lift, Label: "Lift " + taintName(key) + " (" + effect + ")",
			Command: "taint node " + node + " " + lift, Tip: "kubectl taint node " + node + " " + lift})
	}
	return out
}

func taintParts(spec string) (string, string) {
	head, effect, _ := strings.Cut(spec, ":")
	key, _, _ := strings.Cut(head, "=")
	return key, effect
}

func taintName(key string) string {
	switch key {
	case wardKey:
		return "the Ward"
	case "node-role.kubernetes.io/control-plane":
		return "the Royal Ward"
	}
	return key
}

func systemLabel(key string) bool {
	return strings.Contains(key, "kubernetes.io") || strings.Contains(key, "k8s.io") || strings.Contains(key, "kubernites.dev")
}

func labelChoice(kind, name, ns, change, label, tip string) CardAction {
	cmd := "label " + kind + " " + name + ns + " " + change
	return CardAction{Key: change, Label: label, Command: cmd, Tip: tip + "\n$ kubectl " + cmd}
}

func NodeLabelChoices(node string, n model.NodeView) []CardAction {
	out := []CardAction{}
	if n.Labels["disktype"] != "ssd" {
		out = append(out, labelChoice("node", node, "", "disktype=ssd", "Badge it: fast floors (disktype=ssd)", "A badge a hut's nodeSelector can ask for."))
	}
	for _, key := range sortedKeys(n.Labels) {
		if !systemLabel(key) {
			out = append(out, labelChoice("node", node, "", key+"-", "Remove the badge "+key, "Take the label off the Keep."))
		}
	}
	return out
}

func PodLabelChoices(p model.PodView, ns string) []CardAction {
	out := []CardAction{}
	if crew := p.Labels[appKey]; crew != "" {
		out = append(out, labelChoice("pod", p.Name, ns, appKey+"-", "Take it out of the depot ("+appKey+"-)",
			"Its depot stops sending it boxes and its crew raises another: the hut stays for you to study."))
	} else if crew := crewOfImage(p.Image); crew != "" {
		out = append(out, labelChoice("pod", p.Name, ns, appKey+"="+crew, "Return it to the depot ("+appKey+"="+crew+")",
			"The depot sends it boxes again, and its crew may adopt it back."))
	}
	if p.Labels[debugKey] == "" {
		return append(out, labelChoice("pod", p.Name, ns, debugKey+"=true", "Badge it for study (debug=true)", "A badge you can select on: kubectl get pods -l debug=true"))
	}
	return append(out, labelChoice("pod", p.Name, ns, debugKey+"-", "Clear the study badge (debug-)", "Take the debug label off."))
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func crewOfImage(image string) string {
	p, ok := soulstone.Lookup(image)
	if !ok {
		return ""
	}
	for _, kind := range p.Kinds {
		if d, ok := soulstone.DepotFor(kind); ok {
			return d.Crew
		}
	}
	return ""
}
