package hud

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	PanelKind   = "kindcard"
	KindKey     = "kind"
	KindW       = 360
	KindPad     = 8
	KindHead    = 34
	kindBtnW    = 72
	KindBtnH    = 16
	kindMaxRows = 14
	dsPrefix    = "ds:"
	stsPrefix   = "sts:"
	DsRaiseID   = "ds-raise"
)

func kindRows(f Frame) []CrewsRow {
	out := []CrewsRow{}
	for _, d := range f.State.DaemonSets {
		out = append(out, CrewsRow{kind: dsPrefix + d.Name, Standing: true})
	}
	for _, s := range f.State.StatefulSets {
		out = append(out, CrewsRow{kind: stsPrefix + s.Name, Standing: true})
	}
	if len(f.State.DaemonSets) == 0 && couriersCame(f) {
		out = append(out, CrewsRow{kind: DsRaiseID})
	}
	switch {
	case f.State.Archive != nil:
		out = append(out, CrewsRow{kind: archiveCardID, Standing: true})
	case sealwrightsCame(f):
		out = append(out, CrewsRow{kind: archiveRaiseID})
	}
	return out
}

func (r CrewsRow) KindWord() string {
	switch {
	case strings.HasPrefix(r.kind, dsPrefix) || r.kind == DsRaiseID:
		return ui.Bridge("WARDENS", "DAEMONSET")
	case strings.HasPrefix(r.kind, stsPrefix):
		return ui.Bridge("STATEFUL CREW", "STATEFULSET")
	case r.kind == archiveCardID, r.kind == archiveRaiseID:
		return ui.Bridge("BOOKS", "VOLUMES")
	case r.Standing:
		return crewKindWord(r.Chain)
	}
	return ""
}

func daemonSetNamed(f Frame, name string) (view.DaemonSetView, bool) {
	for _, d := range f.State.DaemonSets {
		if d.Name == name {
			return d, true
		}
	}
	return view.DaemonSetView{}, false
}

func statefulSetNamed(f Frame, name string) (view.StatefulSetView, bool) {
	for _, s := range f.State.StatefulSets {
		if s.Name == name {
			return s, true
		}
	}
	return view.StatefulSetView{}, false
}

func KindCard(f Frame, id string) (Card, bool) {
	if name, ok := strings.CutPrefix(id, dsPrefix); ok {
		d, found := daemonSetNamed(f, name)
		return daemonSetCard(d, f), found
	}
	if name, ok := strings.CutPrefix(id, stsPrefix); ok {
		s, found := statefulSetNamed(f, name)
		return statefulSetCard(s), found
	}
	if id == archiveCardID && f.State.Archive != nil {
		return archiveCard(f), true
	}
	return Card{}, false
}

func daemonSetCard(d view.DaemonSetView, f Frame) Card {
	c := Card{Title: ui.Settle(strings.ToUpper(d.Name)), K8s: objectRef("DaemonSet", "daemonset.apps", d.Name), progress: -1}
	c.Lines = []Line{
		{Text: fmt.Sprintf("DESIRED %d  CURRENT %d  READY %d  UP-TO-DATE %d", d.Desired, d.Current, d.Ready, d.Updated), Clr: readyTone(int(d.Ready), int(d.Desired))},
		{Text: ui.Bridge("TRADE "+d.Image, "IMAGE "+d.Image), Clr: ui.Palette.Text},
		{Text: ui.Bridge("One warden on every Keep: a new Keep gets its own, a razed one takes its warden along.", "One Pod on every node that matches the selector; new nodes get theirs, removed nodes take it along."), Clr: ui.Palette.TextDim},
	}
	if d.Selector != "" {
		c.Lines = append(c.Lines, Line{Text: ui.Bridge("ONLY ON KEEPS MARKED ", "nodeSelector ") + d.Selector, Clr: ui.Palette.Blue})
	}
	for _, k := range f.State.Keeps {
		c.Lines = append(c.Lines, keepWardenLine(k))
	}
	return c
}

func keepWardenLine(k view.KeepView) Line {
	switch {
	case len(k.Daemons) == 0:
		return Line{Text: ui.Bridge(k.Name+"  NO WARDEN", k.Name+"  no Pod"), Clr: ui.Palette.RedText}
	case k.Daemons[0].Ready:
		return Line{Text: ui.Bridge(k.Name+"  "+k.Daemons[0].Name+"  AT THE POST", k.Name+"  "+k.Daemons[0].Name+"  Ready"), Clr: ui.Palette.Green}
	}
	return Line{Text: ui.Bridge(k.Name+"  "+k.Daemons[0].Name+"  COMING", k.Name+"  "+k.Daemons[0].Name+"  not Ready"), Clr: ui.Palette.GoldBright}
}

func statefulSetCard(s view.StatefulSetView) Card {
	c := Card{Title: ui.Settle(strings.ToUpper(s.Name)), K8s: objectRef("StatefulSet", "statefulset.apps", s.Name), progress: -1}
	c.Lines = []Line{
		{Text: fmt.Sprintf("READY %d/%d", s.Ready, s.Replicas), Clr: readyTone(s.Ready, s.Replicas)},
		{Text: ui.Bridge("Each archivist keeps its number, its name and its own Deed, whatever Keep it lands on.", "Each Pod keeps its ordinal, its name and its own PersistentVolumeClaim across rescheduling."), Clr: ui.Palette.TextDim},
	}
	if s.Updating {
		c.Lines = append(c.Lines, Line{Text: ui.Bridge("NEW CLASS ROLLING OUT, HIGHEST NUMBER FIRST", "rolling update in progress, highest ordinal first"), Clr: ui.Palette.GoldBright})
	}
	for _, p := range s.Pods {
		keep := p.Keep
		if keep == "" {
			keep = "(pending)"
		}
		c.Lines = append(c.Lines, Line{Text: ui.Settle(p.Name + "  " + keep + "  " + p.Claim), Clr: ui.Palette.Text})
	}
	return c
}

func kindResource(id string, f Frame) (string, string) {
	if name, ok := strings.CutPrefix(id, dsPrefix); ok {
		return "daemonset", name
	}
	if id == archiveCardID {
		return archiveObject(f), view.ArchiveName
	}
	name, _ := strings.CutPrefix(id, stsPrefix)
	return "statefulset", name
}

func (h *HUD) KindShown(f Frame) bool {
	if h.KindCard == "" || h.OverlayID != OverlayNone || h.introCovers(f) {
		return false
	}
	_, ok := KindCard(f, h.KindCard)
	return ok
}

func kindLineRows(c Card) int {
	rows := 0
	for _, l := range c.Lines {
		rows += len(ui.Font{}.WrapLines(l.Text, KindW-2*KindPad, 1, kindMaxRows))
	}
	return min(rows, kindMaxRows)
}

func KindRect(f Frame, c Card) ui.Rect {
	return kindRectWith(f, c, 0)
}

func kindRectWith(f Frame, c Card, verbs int) ui.Rect {
	rows := kindLineRows(c)
	height := KindHead + rows*ui.LineHeight + verbs + KindBtnH + 2*KindPad
	return f.placed(PanelKind, ui.Rect{X: CardBox.X, Y: ConsoleY - crewFloat - height, W: KindW, H: height})
}

func kindWindow(h *HUD, f Frame) (ui.Rect, bool) {
	if !h.KindShown(f) {
		return ui.Rect{}, false
	}
	c, _ := KindCard(f, h.KindCard)
	return kindRectWith(f, c, verbBlock(f, h.KindCard)), true
}

func (h *HUD) kindButtons(f Frame) []Button {
	box, ok := kindWindow(h, f)
	if !ok {
		return nil
	}
	res, name := kindResource(h.KindCard, f)
	y := box.Y + box.H - KindPad - KindBtnH
	out := []Button{
		closeButtonAs(box, KindKey+":close", ""),
		{ID: KindKey + ":describe", Label: "DESCRIBE", Rect: ui.Rect{X: box.X + KindPad, Y: y, W: kindBtnW, H: KindBtnH}, Tip: "kubectl describe " + res + " " + name},
		{ID: KindKey + ":get", Label: "GET", Rect: ui.Rect{X: box.X + KindPad + kindBtnW + 4, Y: y, W: kindBtnW, H: KindBtnH}, Tip: "kubectl get " + res + " " + name},
	}
	return append(out, verbButtons(box, f, h.KindCard)...)
}

func (h *HUD) drawKindCard(dst *ebiten.Image, f Frame) {
	box, ok := kindWindow(h, f)
	if !ok {
		return
	}
	c, _ := KindCard(f, h.KindCard)
	drawPanel(dst, box)
	h.Font.DrawShadow(dst, c.Title, box.X+KindPad, box.Y+6, 1, ui.Palette.GoldBright)
	h.Font.Draw(dst, c.K8s, box.X+KindPad, box.Y+18, 1, ui.Palette.Blue)
	h.drawCardLines(dst, c.Lines, box.X+KindPad, box.Y+KindHead, box.W-2*KindPad, box.Y+box.H-KindPad-KindBtnH-verbBlock(f, h.KindCard))
	for _, b := range h.kindButtons(f) {
		h.drawButton(dst, b)
	}
}

func (h *HUD) PressKind(rest string, f Frame) []Intent {
	click := Intent{Kind: IntentSound, Command: "click"}
	if rest == "close" || h.KindCard == "" {
		h.KindCard = ""
		return []Intent{click}
	}
	if key, ok := strings.CutPrefix(rest, verbPrefix); ok {
		return h.pressVerb(key, f, click)
	}
	res, name := kindResource(h.KindCard, f)
	cmd := rest + " " + res + " " + name
	out, intents := h.RunFrom(cmd, FromButton)
	h.DetailsCard = Details{Title: strings.ToUpper(name), Tab: detailOutput, OK: out.OK, Lines: h.AnswerLines(SpeakLabel(cmd), out)}
	h.OverlayID = OverlayDetails
	return append(intents, click)
}

func init() {
	prefixHandlers[KindKey] = (*HUD).PressKind
	WindowSpecs[PanelKind] = windowSpec{Rect: kindWindow, draw: (*HUD).drawKindCard, buttons: (*HUD).kindButtons}
}
