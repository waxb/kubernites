package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func scaleFix() *view.CommandOffer {
	return &view.CommandOffer{ID: "scale:porters", Label: "+2 porters", Command: "kubectl scale deployment porters --replicas=4",
		Action: view.Action{Action: view.ActScaleCrew, Name: "porters", Replicas: 4}}
}

func flowEconomy() view.EconomyView {
	return view.EconomyView{Income: 4.4, Upkeep: .5, Profit: 3.9, Arriving: 1.7, Carried: .7, Lost: 1, Bottleneck: view.FlowCrew, Fix: scaleFix()}
}

func flowFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Economy = flowEconomy()
	return f
}

func segmentTexts(segs []Segment) string {
	out := []string{}
	for _, s := range segs {
		out = append(out, s.Text)
	}
	return strings.Join(out, " | ")
}

func TestTheFlowRowReadsArrivingCarriedAndLostInRed(t *testing.T) {
	segs := FlowSegments(flowEconomy(), BoardW-2*BoardPad)
	if got := segmentTexts(segs); got != "IN 1.7/S | CARRIED 0.70/S | LOST 1.0/S" {
		t.Errorf("flow = %q", got)
	}
	if segs[2].Clr != ui.Palette.RedText || segs[1].Clr != ui.Palette.Green {
		t.Errorf("lost is red and carried green: %+v", segs)
	}
	calm := flowEconomy()
	calm.Lost = 0
	if segs := FlowSegments(calm, BoardW-2*BoardPad); segs[2].Clr == ui.Palette.RedText {
		t.Error("nothing lost reads calm, not red")
	}
}

func TestTheMoneyRowShowsIncomeAndUpkeepWithoutProfit(t *testing.T) {
	defer ui.SetNamer(nil)
	loss := flowEconomy()
	loss.Income, loss.Upkeep, loss.Profit = 1, 3, -2
	for _, namer := range []func(string) string{nil, names.Kubernetes(64)} {
		ui.SetNamer(namer)
		for _, e := range []view.EconomyView{flowEconomy(), loss} {
			for _, width := range []int{BoardW - 2*BoardPad, 150, 100} {
				got := segmentTexts(MoneySegments(e, width))
				if strings.Contains(got, "PROFIT") || strings.Contains(got, "NET") {
					t.Errorf("the money row keeps the balance, not the net rate: %q", got)
				}
			}
		}
	}
	segs := MoneySegments(flowEconomy(), BoardW-2*BoardPad)
	if len(segs) != 2 || !strings.HasPrefix(segs[0].Text, "INCOME ") || !strings.HasPrefix(segs[1].Text, "UPKEEP ") {
		t.Errorf("money = %q", segmentTexts(segs))
	}
}

func TestTheMoneyRowNeverReadsLikeTheArrivalsRowAndKeepsSmallUpkeep(t *testing.T) {
	defer ui.SetNamer(nil)
	small := view.EconomyView{Income: 1.5, Upkeep: .02, Profit: 1.48}
	for _, namer := range []func(string) string{nil, names.Kubernetes(64)} {
		ui.SetNamer(namer)
		segs := MoneySegments(small, BoardW-2*BoardPad)
		got := segmentTexts(segs)
		if strings.HasPrefix(got, "IN ") {
			t.Errorf("the money row reads like arrivals: %q", got)
		}
		if len(segs) != 2 || !strings.Contains(segs[1].Text, "0.02") {
			t.Errorf("a small cost rounds away: %q", got)
		}
	}
}

func TestFitSegmentsFallsBackToShorterWords(t *testing.T) {
	big := view.EconomyView{Income: 12345, Upkeep: 1234, Profit: -11111}
	for _, width := range []int{BoardW - 2*BoardPad, 150} {
		segs := MoneySegments(big, width)
		total := FlowbarMinGap * (len(segs) - 1)
		for _, s := range segs {
			total += ui.Font{}.Width(s.Text, 1)
		}
		if total > width && width >= BoardW-2*BoardPad {
			t.Errorf("money at %dpx = %q (%dpx)", width, segmentTexts(segs), total)
		}
	}
	if got := FitSegments([][]Segment{{{Text: "WIDE WIDE WIDE"}}, {{Text: "NARROW"}}}, 40); got[0].Text != "NARROW" {
		t.Errorf("fit = %+v", got)
	}
}

func TestTheBoardFoldsTheLossesInAsOnePanel(t *testing.T) {
	f := flowFrame(t)
	f.State.Freight.Losses.QuayFull, f.State.Freight.Losses.Blocked = .4, .2
	if rects := FreightRects(f); len(rects) != 1 || rects[0] != FreightRect(f) {
		t.Fatalf("one panel, got %+v", rects)
	}
	if !Within(LossesRect(f), FreightRect(f)) {
		t.Errorf("losses %+v leave the board %+v", LossesRect(f), FreightRect(f))
	}
	if len(StateLossLines(f.State)) <= FlowbarLossMax || FlowbarLossRows(f) != FlowbarLossMax {
		t.Errorf("at most %d loss reasons, got %d rows", FlowbarLossMax, FlowbarLossRows(f))
	}
	if FreightRect(f).H > BoardMaxH || Overlaps(FreightRect(f), SiegeStrip) {
		t.Errorf("the full board %+v outgrows its room above the reserved strip %+v", FreightRect(f), SiegeStrip)
	}
	f.State.Freight.Losses = view.LossView{}
	for i := range f.State.Freight.Kinds {
		f.State.Freight.Kinds[i].Losses = view.LossView{}
	}
	if LossesShown(f) || FlowbarLossRows(f) != 0 {
		t.Error("with nothing lost the loss rows fold away")
	}
}

func TestTheFixButtonActsThroughTheOfferAndEchoPath(t *testing.T) {
	h := New(nil, Backend{})
	f := flowFrame(t)
	b, ok := HitID(h.Buttons(f), FlowbarFix)
	if !ok || b.Label != "+2 PORTERS" || !strings.Contains(b.Tip, "kubectl scale deployment porters --replicas=4") {
		t.Fatalf("fix button = %+v %v", b, ok)
	}
	it, ok := hasIntent(clickOn(t, h, f, FlowbarFix), IntentAction)
	if !ok || it.Action.Action != view.ActScaleCrew || it.Action.Replicas != 4 || it.From != FromHarborBoard {
		t.Errorf("the fix acts = %+v %v", it, ok)
	}
	f.State.Economy.Fix = nil
	if _, ok := HitID(h.Buttons(f), FlowbarFix); ok {
		t.Error("no fix offered, no button")
	}
	if intents := h.PressFix(f); intents != nil {
		t.Errorf("a stale press does nothing, got %+v", intents)
	}
}

func TestTheBottleneckIsNamedInBothModes(t *testing.T) {
	defer ui.SetNamer(nil)
	for neck := range NeckNames {
		if NeckName(neck) == "" || NeckLine(neck) == "" {
			t.Errorf("%s has no name or line", neck)
		}
	}
	if NeckName("") != "" || NeckLine("nonsense") != "" {
		t.Error("no bottleneck, no name")
	}
	ui.SetNamer(names.Kubernetes(64))
	if NeckName(view.FlowCrew) != "REPLICAS" || !strings.Contains(NeckTip(flowEconomy()), "+2 porters") {
		t.Errorf("k8s neck %q tip %q", NeckName(view.FlowCrew), NeckTip(flowEconomy()))
	}
}

func harborPanel(name string) textPanel {
	return textPanel{name: name, bounds: func(h *HUD) ui.Rect { return ui.Rect{} },
		draw: (*HUD).DrawBoard, buttons: (*HUD).FreightButtons}
}

func assertBoardFits(t *testing.T, name string, f Frame) {
	t.Helper()
	h := New(nil, Backend{})
	p := harborPanel(name)
	p.bounds = func(*HUD) ui.Rect { return FreightRect(f) }
	assertTextFits(t, h, f, p)
}

func TestTheFoldedBoardFitsInBothNamesModes(t *testing.T) {
	assertBoardFits(t, "board with a fix", flowFrame(t))
	big := flowFrame(t)
	big.State.Economy = view.EconomyView{Income: 12345, Upkeep: 23456, Profit: -11111, Arriving: 123.4, Carried: 99.9, Lost: 23.5,
		Bottleneck: view.FlowPending, Fix: &view.CommandOffer{Label: "No room: add a Keep", Action: view.Action{Action: "buyKeep"}}}
	big.State.Freight.Losses.QuayFull = 3.3
	assertBoardFits(t, "board at scale", big)
	calm := flowFrame(t)
	calm.State.Economy = view.EconomyView{Income: 2, Profit: 2, Arriving: 1, Carried: 1}
	assertBoardFits(t, "calm board", calm)
	if b, ok := FixButton(big, BoardRowRect(FreightRect(big), FlowbarFirst(big))); !ok || b.Label != "ADD A KEEP" {
		t.Errorf("the fix names the act, the neck the reason: %+v", b)
	}
	for neck := range NeckNames {
		f := flowFrame(t)
		f.State.Economy.Bottleneck = neck
		assertBoardFits(t, "board neck "+neck, f)
	}
}

func TestTheBoardDrawsTheFlowBar(t *testing.T) {
	h := New(nil, Backend{})
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	f := flowFrame(t)
	h.DrawBoard(dst, f)
	f.State.Economy.Arriving = 0
	h.DrawBoard(dst, f)
	f.Guide = Guide{Staged: true, Intro: &Intro{ID: "losses", Target: SurfaceLosses, Via: ViaAck}, Introduced: []string{SurfaceFreight}}
	f.State.Freight.Losses = view.LossView{}
	for i := range f.State.Freight.Kinds {
		f.State.Freight.Kinds[i].Losses = view.LossView{}
	}
	runs := recordRuns(func() { h.DrawBoard(dst, f) })
	if !hasRun(runs, NoLossesRow) {
		t.Error("the losses intro shows the calm row")
	}
}

func TestClearAvoidingNeverTakesAHardBlockWhileASafeSpotRemains(t *testing.T) {
	board := ui.Rect{X: 100, Y: 0, W: 100, H: 100}
	card := ui.Rect{X: 0, Y: 0, W: 90, H: 100}
	spots := []ui.Rect{{X: 150, Y: 50, W: 20, H: 20}, {X: 10, Y: 10, W: 50, H: 50}}
	if got := ClearAvoiding(spots, []ui.Rect{board}, []ui.Rect{card}); got != spots[1] {
		t.Errorf("chose %+v over the board", got)
	}
	if got := ClearAvoiding(spots[:1], []ui.Rect{board}, nil); got != spots[0] {
		t.Errorf("with no safe spot the least blocked wins, got %+v", got)
	}
}

func TestTheFixLabelShortensToFitBesideTheBottleneck(t *testing.T) {
	f := flowFrame(t)
	f.State.Economy.Bottleneck = view.FlowService
	f.State.Economy.Fix = &view.CommandOffer{Label: "Open the ledgers depot", Action: view.Action{Action: "openDepot", Kind: "ledger"}}
	b, ok := FixButton(f, BoardRowRect(FreightRect(f), FlowbarFirst(f)))
	if !ok || b.Label != "OPEN LEDGERS" {
		t.Errorf("depot fix = %+v", b)
	}
	cases := map[string]view.Action{"RAISE CLERKS": {Action: "raiseCrew", Name: "clerks"}, "BUY A KEEP": {Action: "buyKeep"},
		"SCALE TO 5": {Action: view.ActScaleCrew, Replicas: 5}, "ROLL IT BACK": {Action: view.ActRolloutCrew}}
	for want, a := range cases {
		if got := ShortFixLabel(view.CommandOffer{Label: "Roll it back (undo)", Action: a}); got != want {
			t.Errorf("short label for %s = %q, want %q", a.Action, got, want)
		}
	}
	assertBoardFits(t, "board with a depot fix", f)
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	if got := ShortFixLabel(*f.State.Economy.Fix); got != "EXPOSE CLERKS" {
		t.Errorf("k8s depot fix = %q", got)
	}
}

func TestEveryFixButtonReadsWholeInBothModes(t *testing.T) {
	defer ui.SetNamer(nil)
	crew := view.CrewOrder{Container: "porter", Requests: "cpu=200m"}
	limit := view.CrewOrder{Container: "porter", Limits: "cpu=1000m"}
	offers := []view.CommandOffer{
		{Label: "Pull leaner porters at the registry (400 MiB)", Action: view.Action{Action: "buySoulstone", Image: "porter:2.0"}},
		{Label: "Pledge less: 200m of CPU per porters worker", Action: view.Action{Action: view.ActCrewResources, Name: "porters", Crew: crew}},
		{Label: "Loosen the binding: 1000m of CPU per porters worker", Action: view.Action{Action: view.ActCrewResources, Name: "porters", Crew: limit}},
		{Label: "Raze a hut on node-a: its shift raises a new one on a Keep with room", Action: view.Action{Action: view.ActMoveWorker, Name: "porters"}},
		{Label: "Roll the leaner porter:2.0 out", Action: view.Action{Action: view.ActCrewImage, Name: "porters", Image: "porter:2.0"}},
		{Label: "Close a berth: fewer ships until the crews cope", Action: view.Action{Action: view.ActCloseBerth}},
		{Label: "Roll porters back", Action: view.Action{Action: view.ActRolloutCrew, Name: "porters"}},
	}
	for _, mode := range namesModes() {
		mode.apply()
		for _, o := range offers {
			o := o
			f := flowFrame(t)
			f.State.Economy.Bottleneck, f.State.Economy.Fix = view.FlowCPU, &o
			f.State.Reveals = append(f.State.Reveals, allReveals...)
			row := BoardRowRect(FreightRect(f), FlowbarFirst(f))
			b, ok := FixButton(f, row)
			if !ok || b.Label == "" {
				continue
			}
			if strings.Contains(b.Label, "...") || NamedFont.Width(ui.Name(b.Label), 1)+FlowbarFixPad > b.Rect.W+2 {
				t.Errorf("[%s] %s: the fix button reads %q in %dpx", mode.name, o.Action.Action, b.Label, b.Rect.W)
			}
		}
	}
}

func TestDroppedAboveInSaysWhy(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	e := view.EconomyView{Arriving: 2.7, Carried: 1.4, Lost: 3.1}
	tip := RateTip(e)
	if !strings.Contains(tip, "queued ones included") || !strings.Contains(tip, "DROPPED 3.1/S runs above IN 2.7/S") {
		t.Errorf("'IN 2.7  SERVED 1.4' beside 'DROPPED 3.1/S' with no why; the tip reads %q", tip)
	}
	if note := LostNote(e); note != "DROPPED INCLUDES THE QUEUE" {
		t.Errorf("the board's note reads %q", note)
	}
	if LostNote(view.EconomyView{Arriving: 2.7, Carried: 2.5, Lost: 0.2}) != "" || strings.Contains(RateTip(view.EconomyView{Arriving: 2.7, Lost: 0.2}), "runs above") {
		t.Error("a loss below what arrives needs no note")
	}
	ui.SetNamer(nil)
	if note := LostNote(e); note != "LOST INCLUDES THE QUAY'S PILE" || !strings.Contains(RateTip(e), "the ones waiting on the quay included") {
		t.Errorf("story note %q, tip %q", note, RateTip(e))
	}
}
