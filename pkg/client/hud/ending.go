package hud

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	OverlayEnding    = "ending"
	endingPrefix     = "ending"
	EndingOpen       = endingPrefix + ":open"
	EndingPlayOn     = endingPrefix + ":playon"
	endingBook       = endingPrefix + ":almanac"
	endingInset      = 10
	endingLeftW      = 452
	endingNodes      = 5
	endingCrews      = 6
	endingYAML       = 14
	endingCloseLines = 3
	endingPlayOnRows = 7
	endingCrewYAML   = "porters"

	experimentalStepTitle = "EXPERIMENTAL STEP"
)

var endingDropped = []string{"creationTimestamp", "uid:", "resourceVersion", "generation:", "terminationMessage", "dnsPolicy",
	"restartPolicy", "schedulerName", "securityContext", "terminationGracePeriodSeconds", "imagePullPolicy",
	"progressDeadlineSeconds", "revisionHistoryLimit", "resources: {}", "namespace:", "annotations:", "deployment.kubernetes.io", "strategy:", "selector:", "labels:"}

type clusterBlock struct {
	command string
	lines   []string
}

type endingState struct {
	opened  bool
	Cluster []clusterBlock `json:"-"`
}

func endingOf(f Frame) (view.EndingView, bool) {
	if f.State.Kube.Ending == nil {
		return view.EndingView{}, false
	}
	return *f.State.Kube.Ending, true
}

func (h *HUD) OpenEnding(f Frame) []Intent {
	v, ok := endingOf(f)
	if !ok {
		h.Kube.Ending.opened = false
		return nil
	}
	if v.Read || h.Kube.Ending.opened || h.OverlayID != OverlayNone || !h.onStage(actEnding) {
		return nil
	}
	h.showEnding(f)
	return []Intent{act(view.Action{Action: view.ActReadEnding}), {Kind: IntentSound, Command: "victory"}}
}

func (h *HUD) showEnding(f Frame) {
	h.Kube.Ending = endingState{opened: true, Cluster: h.clusterBlocks(f)}
	h.OverlayID = OverlayEnding
}

func (h *HUD) clusterLines(command string) []string {
	if h.backend.Inspect == nil {
		return nil
	}
	res := h.backend.Inspect(command)
	if !res.OK {
		return nil
	}
	return SplitOutput(res.Output)
}

func (h *HUD) clusterBlocks(f Frame) []clusterBlock {
	crew := endingCrewYAML
	if _, ok := crewByName(f, crew); !ok && len(f.State.Crews) > 0 {
		crew = f.State.Crews[0].Name
	}
	yaml := "kubectl get deployment " + crew + " -o yaml"
	return []clusterBlock{
		{command: "kubectl get nodes", lines: firstLines(h.clusterLines("kubectl get nodes"), endingNodes)},
		{command: "kubectl get deployments", lines: firstLines(h.clusterLines("kubectl get deployments"), endingCrews)},
		{command: yaml, lines: firstLines(TrimmedYAML(h.clusterLines(yaml)), endingYAML)},
	}
}

func firstLines(lines []string, n int) []string {
	if len(lines) <= n {
		return lines
	}
	return append(append([]string(nil), lines[:n-1]...), "  ...")
}

func TrimmedYAML(lines []string) []string {
	out := []string{}
	skipUnder := -1
	for _, l := range lines {
		indent := len(l) - len(strings.TrimLeft(l, " "))
		trimmed := strings.TrimSpace(l)
		if skipUnder >= 0 && indent > skipUnder {
			continue
		}
		skipUnder = -1
		if indent == 0 && strings.HasPrefix(trimmed, "status:") {
			break
		}
		if dropped(trimmed) {
			if strings.HasSuffix(trimmed, ":") {
				skipUnder = indent
			}
			continue
		}
		out = append(out, l)
	}
	return withoutEmptyParents(out)
}

func withoutEmptyParents(lines []string) []string {
	out := []string{}
	for i, l := range lines {
		parent := strings.HasSuffix(strings.TrimSpace(l), ":")
		indent := len(l) - len(strings.TrimLeft(l, " "))
		if parent && (i+1 >= len(lines) || len(lines[i+1])-len(strings.TrimLeft(lines[i+1], " ")) <= indent && !strings.HasPrefix(strings.TrimSpace(lines[i+1]), "- ")) {
			continue
		}
		out = append(out, l)
	}
	return out
}

func dropped(line string) bool {
	for _, d := range endingDropped {
		if strings.HasPrefix(line, d) {
			return true
		}
	}
	return false
}

func (h *HUD) endingButtons(f Frame) []Button {
	foot := ui.Rect{X: RoadBox.X + endingInset, Y: RoadBox.Y + RoadBox.H - 26, W: RoadBox.W - 2*endingInset, H: 20}
	out := []Button{closeButtonAs(RoadBox, endingPrefix+":close", "Close [Esc]"),
		{ID: EndingPlayOn, Rect: ui.Rect{X: foot.X + foot.W - 120, Y: foot.Y, W: 120, H: 18}, Label: "PLAY ON", Selected: true,
			Tip: "Your realm keeps running; the next steps are experimental"}}
	if len(f.State.Almanac) > 0 {
		out = append(out, Button{ID: endingBook, Rect: ui.Rect{X: foot.X + foot.W - 120 - 4 - 80, Y: foot.Y, W: 80, H: 18}, Label: AlmanacLabel,
			Tip: "Every page you met, with its Kubernetes side"})
	}
	return out
}

func (h *HUD) pressEnding(rest string, f Frame) []Intent {
	switch rest {
	case "open":
		if _, ok := endingOf(f); ok {
			h.showEnding(f)
			return []Intent{{Kind: IntentSound, Command: "click"}}
		}
	case "playon", "close":
		h.OverlayID = OverlayNone
		return []Intent{{Kind: IntentSound, Command: "click"}}
	case "almanac":
		return h.OpenBook("", f)
	}
	return nil
}

func (h *HUD) endingAgeButton(f Frame) []Button {
	if _, ok := endingOf(f); !ok {
		return nil
	}
	b := AgeBottom()
	return []Button{{ID: EndingOpen, Rect: ui.Rect{X: b.X + b.W - 204 - 4 - 104 - 4 - 104, Y: b.Y + 2, W: 104, H: 18}, Label: "THE DEMO'S END",
		Tip: "What you built, your realm as a cluster, and what the full release adds"}}
}

func (h *HUD) drawEnding(dst *ebiten.Image, f Frame) {
	ui.DarkPanel(dst, RoadBox)
	v, _ := endingOf(f)
	x, y := RoadBox.X+endingInset, RoadBox.Y+6
	h.Font.DrawShadow(dst, view.EndingTitle, x, y, 2, ui.Palette.Gold)
	h.Font.Draw(dst, view.EndingSubtitle, x, y+22, 1, ui.Palette.TextDim)
	y += 40
	y = h.endingSection(dst, "WHAT YOU BUILT", bullets(v.Built), x, y, ui.Palette.Text)
	y = h.endingSection(dst, "PLAY ON, AS IT IS", h.Font.WrapLines(v.PlayOn, endingLeftW, 1, endingPlayOnRows), x, y, ui.Palette.Cream)
	y = h.endingSection(dst, "ONE NEXT GOAL", h.Font.WrapLines(v.Goal, endingLeftW, 1, 2), x, y, ui.Palette.GoldBright)
	h.endingSection(dst, "THE FULL RELEASE", h.Font.WrapLines(v.Close, endingLeftW, 1, endingCloseLines), x, y, ui.Palette.Cream)
	h.drawCluster(dst, RoadBox.X+endingInset+endingLeftW+16, RoadBox.Y+8)
	h.DrawOverlayButtons(dst, h.endingButtons(f))
}

func bullets(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, "- "+l)
	}
	return out
}

func (h *HUD) endingSection(dst *ebiten.Image, title string, rows []string, x, y int, clr color.RGBA) int {
	h.Font.DrawShadow(dst, title, x, y, 1, ui.Palette.Gold)
	y += ui.LineHeight + 2
	for _, r := range rows {
		h.Font.Draw(dst, ui.Clip(r, ui.MaxChars(endingLeftW, 1)), x, y, 1, clr)
		y += ui.LineHeight
	}
	return y + 8
}

func (h *HUD) drawCluster(dst *ebiten.Image, x, y int) {
	w := RoadBox.X + RoadBox.W - endingInset - x
	Literal.DrawShadow(dst, "SEE IT IN KUBERNETES: YOUR REALM AS A CLUSTER", x, y, 1, ui.Palette.Gold)
	y += ui.LineHeight + 4
	cells := ui.MaxChars(w, 1)
	for _, b := range h.Kube.Ending.Cluster {
		Literal.Draw(dst, Literal.Clip("$ "+b.command, cells), x, y, 1, ui.Palette.GoldBright)
		y += ui.LineHeight
		for _, l := range b.lines {
			Literal.Draw(dst, Literal.Clip(l, cells), x, y, 1, ui.Palette.Blue)
			y += ui.LineHeight
		}
		y += 4
	}
}

func init() {
	Overlays[OverlayEnding] = OverlaySpec{(*HUD).endingButtons, (*HUD).drawEnding}
	prefixHandlers[endingPrefix] = (*HUD).pressEnding
}
