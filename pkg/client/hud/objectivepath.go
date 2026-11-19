package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	orSpeak  = "OR SPEAK"
	orSpeakW = 54
)

var readVerbs = []string{"get", "describe", "top", "logs", "explain", "version", "cluster-info", "api-resources"}

func ButtonFirst(o view.ObjectiveView) bool {
	return o.Target != "" && o.Target != SurfaceStone && o.Target != SurfaceScriptorium
}

func objectivePath(l CommandLine, o view.ObjectiveView) CommandLine {
	next := l
	if ButtonFirst(o) {
		next.Action, next.Run = nil, false
		return next
	}
	next.Run = true
	return next
}

func readCommand(cmd string) bool {
	w := commandWords(cmd)
	if len(w) == 0 {
		return false
	}
	for _, v := range readVerbs {
		if w[0] == v {
			return true
		}
	}
	return false
}

func (h *HUD) doObjective(l CommandLine) []Intent {
	return h.doObjectiveStep(l, "")
}

func (h *HUD) doObjectiveStep(l CommandLine, step string) []Intent {
	if l.Action != nil {
		return h.doLine(l, FromObjectiveCard)
	}
	if !l.Run || strings.TrimSpace(l.Command) == "" {
		return nil
	}
	res, intents := h.RunFrom(l.Command, FromObjectiveCard)
	if readCommand(l.Command) {
		h.showCommand(l.Command, res)
		if step != "" {
			h.DetailsCard.Lines = append([]string{"Your step: " + step, ""}, h.DetailsCard.Lines...)
		}
		return append(intents, Intent{Kind: IntentSound, Command: "click"})
	}
	h.Toast(answerLine(res.Output), !res.OK)
	return append(intents, Intent{Kind: IntentSound, Command: soundFor(res.OK)})
}

func soundFor(ok bool) string {
	if ok {
		return "click"
	}
	return "error"
}

func (h *HUD) showCommand(cmd string, res view.CommandResult) {
	h.DetailsCard = Details{Title: ResultTitle(cmd), Tab: detailOutput, OK: res.OK}
	h.DetailsCard.Lines = h.AnswerLines(SpeakLabel(cmd), res)
	h.OverlayID = OverlayDetails
}
