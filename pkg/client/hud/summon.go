package hud

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	summonCost      = "A hut costs no gold: its worker pledges a Keep's food (CPU)."
	summonFull      = "A full Keep leaves new huts Unhoused (Pending)."
	summonNoRoom    = "FULL"
	summonNoRoomTip = "No Keep has room now: a new hut waits Unhoused until one does. Making room comes later."
	summonKeeps     = 2
	staticTitle     = "HUT AT THE DOOR"
	staticLine      = "STATIC POD, RUN BY ITS KEEP"
	staticDetail    = "Hut at the door (static Pod)"
)

var staticLore = []string{
	"Hut at the door (static Pod).",
	"Its Keep runs it by itself from a file (/etc/kubernetes/manifests), with no Chancery.",
	"It holds the Keep's door (hostPort 80). Once the Chancery stands, the Keep shows a mirror of the hut there; kubectl delete removes only that mirror, and the Keep raises it again.",
	"",
}

var chapterStaticLore = []string{
	"A static Pod: its Node's kubelet runs it from a file (/etc/kubernetes/manifests), with no kube-apiserver.",
	"It serves on the Node's own port 80 (hostNetwork: true). Once a kube-apiserver answers, the kubelet posts a mirror Pod there so kubectl can see it; kubectl delete removes only that mirror, and the kubelet writes it again.",
	"",
}

const (
	refusedHint        = "The Chancery (kube-apiserver) does not stand yet. Build the Codex and the Chancery to speak with the realm."
	chapterRefusedHint = "No kube-apiserver answers yet, and kubectl reads the cluster only through it."
)

func (h *HUD) StaticLore() []string {
	if h.InChapter() {
		return chapterStaticLore
	}
	return staticLore
}

func (h *HUD) RefusedHint() string {
	if h.InChapter() {
		return chapterRefusedHint
	}
	return refusedHint
}

type keepHands struct {
	label string
	free  float64
}

func freeHands(f Frame) []keepHands {
	out := []keepHands{}
	for _, s := range f.Site.Structures {
		if s.Kind != view.StructKeep || s.State != view.StateReady {
			continue
		}
		m, ok := f.Metrics.Nodes[s.Object.Name]
		if !ok || m.CPUAllocatableMilli <= 0 {
			continue
		}
		out = append(out, keepHands{label: s.Label, free: float64(m.CPUAllocatableMilli - m.CPURequestedMilli)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].free > out[j].free })
	return out[:min(len(out), summonKeeps)]
}

func PodsLine(f Frame) string {
	s := f.State.Summon
	return fmt.Sprintf("Pods: %d running, %d waiting for room.", s.Running, s.Waiting)
}

func handsLines(f Frame) []string {
	out := []string{}
	for _, k := range freeHands(f) {
		out = append(out, fmt.Sprintf("The %s has %s free.", k.label, foodOf(max(k.free, 0))))
	}
	return out
}

func SummonLines(f Frame) []string {
	out := append([]string{summonCost, PodsLine(f)}, handsLines(f)...)
	return append(out, summonFull)
}

func summonTip(f Frame, tip string) string {
	lines := append([]string{tip, PodsLine(f)}, handsLines(f)...)
	if line := workerTrio(f); line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func workerTrio(f Frame) string {
	lr := f.State.LimitRange
	if lr == nil || lr.CPU <= 0 && lr.CPULimit <= 0 {
		return ""
	}
	story := "its use is not measured through a noon yet"
	k8s := "its use is not measured through a peak yet"
	if cv, ok := crewByName(f, "porters"); ok && cv.Peak.Known() {
		story = "a porter uses about " + foodOf(float64(cv.Peak.CPU)) + " at the noon peak of the last day"
		k8s = fmt.Sprintf("a porter uses about %dm at the peak of the last day", cv.Peak.CPU)
	}
	return ui.Bridge(fmt.Sprintf("Each worker pledges %s (its Pledge), %s, and the realm's Binding (the most a worker may use) caps it at %s.", foodOf(float64(lr.CPU)), story, foodOf(float64(lr.CPULimit))),
		fmt.Sprintf("Each Pod requests %dm CPU, %s, and is capped at %dm CPU (its limit).", lr.CPU, k8s, lr.CPULimit))
}

func summonCell(b Button, f Frame) Button {
	next := b
	next.Tip = summonTip(f, b.Tip)
	if f.State.Summon.Waiting > 0 {
		next.Cost, next.Tip = summonNoRoom, summonNoRoomTip+"\n"+next.Tip
	}
	return next
}

func (h *HUD) drawSummonCard(dst *ebiten.Image, f Frame) {
	x, y := CardBox.X+8, CardBox.Y+6
	h.Font.DrawShadow(dst, "HIRE BY HAND", x, y, 1, ui.Palette.Gold)
	y += ui.LineHeight + 2
	lines := SummonLines(f)
	for i, l := range lines {
		clr := ui.Palette.Text
		if i == 0 {
			clr = ui.Palette.Cream
		}
		for _, row := range h.Font.WrapLines(l, CardBox.W-16, 1, 2) {
			h.Font.Draw(dst, row, x, y, 1, clr)
			y += ui.LineHeight
		}
	}
}
