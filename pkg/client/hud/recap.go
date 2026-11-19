package hud

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	RecapButton    = "recall"
	recapTitle     = "WHAT IS IT IN KUBERNETES?"
	recapStart     = "ASK ME: FOUR QUICK PICKS"
	recapIntro     = "Optional: four quick picks, no score. Leave them if you like."
	recapOptionW   = 112
	recapOptionH   = 14
	recapRevealMax = 2
	recapTop       = 50
	recapRight     = "[x] "
	recapWrong     = "[ ] "
)

type recapQuestion struct {
	ask     string
	options []string
	Answer  int `json:"-"`
	reveal  string
	page    string
}

var DarkRecap = []recapQuestion{
	{ask: "A porter hut is", options: []string{"a Node", "a Pod", "a Service"}, Answer: 1, page: "hut",
		reveal: "A Pod: one or more containers that share a network address, running on one Node."},
	{ask: "The crew that rolls your porters out version by version, with one shift for each version, is", options: []string{"a ReplicaSet", "a Service", "a Deployment"}, Answer: 2, page: "crew",
		reveal: "A Deployment: it keeps one ReplicaSet per Pod template and rolls from one to the next. Each ReplicaSet (your shift) keeps its own count."},
	{ask: "The parcels depot is", options: []string{"a Service", "a Pod", "a ConfigMap"}, Answer: 0, page: "depot",
		reveal: "A Service: one stable name in front of every ready Pod its selector matches."},
	{ask: "A Keep is", options: []string{"a Namespace", "a Node", "a Pod"}, Answer: 1, page: "keep",
		reveal: "A Node: a machine whose kubelet runs the Pods the scheduler places on it."},
}

type recapState struct {
	age     string
	started bool
	Picks   map[int]int `json:"-"`
}

func (h *HUD) RecapShown(f Frame) bool {
	return h.DawnLive(f) && h.Dawn.More && f.State.Age == view.AgeFeudal && !loomOpen(f) && !ui.Naming()
}

func (h *HUD) syncRecap(f Frame) {
	if h.Kube.Recap.age != f.State.Age {
		h.Kube.Recap = recapState{age: f.State.Age, Picks: map[int]int{}}
	}
}

func (h *HUD) recapNext() int {
	for i := range DarkRecap {
		if _, ok := h.Kube.Recap.Picks[i]; !ok {
			return i
		}
	}
	return len(DarkRecap)
}

func recapRect() ui.Rect {
	r := ageRight()
	return ui.Rect{X: r.X, Y: r.Y + recapTop, W: r.W, H: r.H - recapTop}
}

type recapRow struct {
	Text string `json:"-"`
	clr  color.RGBA
	gap  int
	ask  int
}

func (h *HUD) RecapLines() []recapRow {
	r := recapRect()
	out := []recapRow{{Text: recapTitle, clr: ui.Palette.Gold}, {Text: recapIntro, clr: ui.Palette.TextDim, gap: 4}}
	if !h.Kube.Recap.started {
		return out
	}
	for i, q := range DarkRecap {
		pick, done := h.Kube.Recap.Picks[i]
		if !done {
			out = append(out, recapRow{Text: q.ask + "...", clr: ui.Palette.Text, ask: i + 1})
			return out
		}
		mark, clr := recapRight, ui.Palette.Green
		answer := q.options[q.Answer]
		if pick != q.Answer {
			mark, clr = recapWrong, ui.Palette.TextDim
			answer += ", not " + q.options[pick]
		}
		for i, row := range Literal.WrapLines(q.ask+" "+answer, r.W-len(mark)*ui.Advance, 1, recapRevealMax) {
			lead := strings.Repeat(" ", len(mark))
			if i == 0 {
				lead = mark
			}
			out = append(out, recapRow{Text: lead + row, clr: clr})
		}
		for _, row := range Literal.WrapLines(q.reveal, r.W-4*ui.Advance, 1, recapRevealMax) {
			out = append(out, recapRow{Text: "    " + row, clr: ui.Palette.Blue})
		}
	}
	right := 0
	for i, q := range DarkRecap {
		if h.Kube.Recap.Picks[i] == q.Answer {
			right++
		}
	}
	return append(out, recapRow{Text: fmt.Sprintf("You named %d of %d. Each page waits in the Almanac, Kubernetes side.", right, len(DarkRecap)),
		clr: ui.Palette.GoldBright})
}

func (h *HUD) recapButtons(f Frame) []Button {
	if !h.RecapShown(f) {
		return nil
	}
	h.syncRecap(f)
	r := recapRect()
	y := r.Y
	out := []Button{}
	if !h.Kube.Recap.started {
		lines := h.RecapLines()
		return []Button{{ID: RecapButton + ":start", Rect: ui.Rect{X: r.X, Y: y + len(lines)*ui.LineHeight + 6, W: 2 * recapOptionW, H: recapOptionH + 2},
			Label: recapStart, Tip: "Four questions about what you built in the Dark Age, each answered as soon as you pick"}}
	}
	for _, row := range h.RecapLines() {
		y += ui.LineHeight + row.gap
		if row.ask == 0 {
			continue
		}
		q := DarkRecap[row.ask-1]
		for i, opt := range q.options {
			out = append(out, Button{ID: fmt.Sprintf("%s:%d:%d", RecapButton, row.ask-1, i), Rect: ui.Rect{X: r.X + i*(recapOptionW+6), Y: y + 2, W: recapOptionW, H: recapOptionH},
				Label: ui.Settle(opt), Tip: "No score: pick what you believe, then see the answer"})
		}
	}
	return out
}

func (h *HUD) drawRecap(dst *ebiten.Image, f Frame) {
	if !h.RecapShown(f) {
		return
	}
	h.syncRecap(f)
	r := recapRect()
	y := r.Y
	for _, row := range h.RecapLines() {
		Literal.Draw(dst, Literal.Clip(row.Text, ui.MaxChars(r.W, 1)), r.X, y, 1, row.clr)
		y += ui.LineHeight + row.gap
	}
}

func (h *HUD) pressRecap(rest string, f Frame) []Intent {
	if !h.RecapShown(f) {
		return nil
	}
	h.syncRecap(f)
	if rest == "start" {
		h.Kube.Recap.started = true
		return []Intent{{Kind: IntentSound, Command: "click"}}
	}
	q, opt, ok := strings.Cut(rest, ":")
	qi, err1 := strconv.Atoi(q)
	oi, err2 := strconv.Atoi(opt)
	if !ok || err1 != nil || err2 != nil || qi != h.recapNext() || qi >= len(DarkRecap) || oi < 0 || oi >= len(DarkRecap[qi].options) {
		return nil
	}
	h.Kube.Recap.Picks[qi] = oi
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func init() {
	prefixHandlers[RecapButton] = (*HUD).pressRecap
}

func (h *HUD) kubeAgeButtons(f Frame) []Button {
	return append(h.recapButtons(f), h.endingAgeButton(f)...)
}

func (h *HUD) drawKubeAge(dst *ebiten.Image, f Frame) {
	h.drawRecap(dst, f)
}
