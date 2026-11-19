package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	FromKeepCard        = "the Keep card"
	FromHearthCard      = "the hut card"
	FromDepotCard       = "the depot card"
	FromCodexCard       = "the Codex card"
	FromChanceryCard    = "the Chancery card"
	FromObservatoryCard = "the Observatory card"
	FromCrewCard        = "the crew card"
	FromCommandCard     = "the command card"
	FromObjectiveCard   = "the objective card"
	FromDetails         = "the details panel"
	FromCouncil         = "the Council"
	FromCalamityCard    = "the calamity card"
	FromStoneButton     = "the Stone's DO IT"
	FromButton          = "a button"
	echoSeparator       = "  (from "
)

func EchoText(cmd, from string) string {
	text := EchoFor(cmd)
	if from == "" {
		return text
	}
	return text + echoSeparator + ui.Name(from) + ")"
}

func (h *HUD) Echo(cmd, from string) {
	h.Stone.Append(LineEcho, EchoText(cmd, from))
}

func (h *HUD) EchoAnswer(output string, ok bool) {
	if strings.TrimSpace(output) == "" {
		return
	}
	kind := LineOutput
	if !ok {
		kind = LineError
	}
	h.Stone.Append(kind, StoryAnswer(output, ok)...)
}

func EchoIntent(cmd, output string, ok bool) Intent {
	return Intent{Kind: IntentEcho, Command: BareCommand(cmd), Target: FirstLine(output), Tab: echoState(ok)}
}

func (h *HUD) EchoFrom(command, from, output string, ok bool) []Intent {
	h.Echo(command, from)
	h.EchoAnswer(output, ok)
	return []Intent{EchoIntent(command, output, ok), {Kind: IntentKubectl}}
}

func (h *HUD) RunFrom(command, from string) (view.CommandResult, []Intent) {
	res := h.kubectl(BareCommand(command))
	return res, h.EchoFrom(command, from, res.Output, res.OK)
}

func ActFrom(a view.Action, from string) Intent {
	return Intent{Kind: IntentAction, Action: a, From: from}
}

func EchoSource(it Intent) string {
	if it.From != "" {
		return it.From
	}
	return FromButton
}
