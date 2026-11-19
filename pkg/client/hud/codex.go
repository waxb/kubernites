package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	gaugeW       = 26
	gaugeGap     = 4
	etcdctlVerb  = "etcdctl"
	kubectlVerb  = "kubectl "
	codexStatTip = "The Codex (etcd): its size against its quota. Click to ask the Augur."
	codexSlot    = "CODEX 100%"
)

func toneColor(tone int) color.RGBA {
	switch tone {
	case ToneDim:
		return ui.Palette.TextDim
	case ToneGood:
		return ui.Palette.Green
	case ToneWarn:
		return ui.Palette.GoldBright
	case ToneBad:
		return ui.Palette.RedText
	case ToneGold:
		return ui.Palette.Gold
	}
	return ui.Palette.Text
}

func codexShown(f Frame) bool {
	return f.Revealed(CodexReveal) && f.Caesar.Codex.Known() && standing(f.Site, view.StructCodex)
}

func CodexStat(c model.Codex) (string, color.RGBA) {
	tone := gaugeTone(model.GaugeLevel(c.Fill(), c.Alarm))
	if c.Alarm {
		return "CODEX FULL", toneColor(tone)
	}
	return fmt.Sprintf("CODEX %d%%", percent(c.Fill())), toneColor(tone)
}

func codexStat(f Frame) (Stat, bool) {
	if !codexShown(f) {
		return Stat{}, false
	}
	c := f.Caesar.Codex
	text, clr := CodexStat(c)
	st := Stat{Text: text, slot: codexSlot, clr: clr, Tip: codexStatTip, gauge: true, fill: c.Fill(), Meter: MeterCodex}
	if CouncilOpen(f) {
		st.Id = CouncilPrefix + ":open:" + TabAugur
	}
	return st, true
}

func CodexLines(c model.Codex) []Line {
	if !c.Known() {
		return nil
	}
	fill := c.Fill()
	gauge := toneColor(gaugeTone(model.GaugeLevel(fill, c.Alarm)))
	out := []Line{
		{Text: "CODEX", Clr: ui.Palette.Text, Bar: true, Value: fill, Fill: gauge, Suffix: fmt.Sprintf("%d%% of %s", percent(fill), model.Bytes(c.QuotaBytes))},
		{Text: fmt.Sprintf("SIZE %s, IN USE %s", model.Bytes(c.SizeBytes), model.Bytes(c.InUseBytes)), Clr: ui.Palette.Text},
	}
	switch {
	case c.Alarm:
		out = append(out, Line{Text: "ALARM NOSPACE: every write fails. Rebind the Codex, then disarm.", Clr: ui.Palette.RedText})
	case c.Large():
		out = append(out, Line{Text: "Swelling: a rebind reclaims " + model.Bytes(c.Reclaimable()) + ".", Clr: ui.Palette.GoldBright})
	}
	return out
}

func codexSelected(f Frame) bool {
	return f.Revealed(CodexReveal) && f.Selection.Kind == "structure" && f.Selection.Structure.Kind == view.StructCodex
}

var NodePrompt = "you@" + view.TowerName + ":~$ "

var nodeVerbs = map[string]bool{etcdctlVerb: true, "sudo": true, "mv": true, "ls": true, "cat": true}

func NodeCommand(cmd string) bool {
	fields := strings.Fields(cmd)
	return len(fields) > 0 && nodeVerbs[fields[0]]
}

func SpeakLabel(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if NodeCommand(cmd) {
		return cmd
	}
	return ChipLabel(cmd)
}

func ShownCommand(cmd string) string {
	if ui.StoryNaming() && !lexicon.StoryCommandShown(cmd) {
		return ""
	}
	return cmd
}

func SpeakTip(cmd string) string {
	if ShownCommand(cmd) == "" {
		return ""
	}
	return "\n$ " + EchoLabel(cmd)
}

func PromptFor(input string) string {
	if NodeCommand(input) {
		return NodePrompt
	}
	return "$ " + kubectlVerb
}

func EchoFor(cmd string) string {
	if NodeCommand(cmd) && !ui.StoryNaming() {
		return NodePrompt + strings.TrimSpace(cmd)
	}
	return "> " + EchoLabel(cmd)
}

func EchoLabel(cmd string) string {
	if ui.StoryNaming() {
		return SpeakLabel(lexicon.CommandHead(cmd))
	}
	return SpeakLabel(cmd)
}
