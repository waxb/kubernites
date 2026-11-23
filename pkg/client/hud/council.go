package hud

import (
	"cmp"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	OverlayCouncil   = "council"
	TabQuartermaster = "quartermaster"
	TabStewards      = "stewards"
	TabWayfinder     = "wayfinder"
	TabTreasurer     = "treasurer"
	TabAugur         = "augur"
	TabDecrees       = "decrees"

	CouncilPrefix  = "council"
	CouncilReveal  = "council"
	CodexReveal    = "codex-bloat"
	QosReveal      = "qos"
	LensReveal     = "overlays"
	heraldryReveal = "heraldry"
	PageTop        = 50
	councilRowH    = 12
	sectionGap     = 6
	tabGap         = 4
	tabH           = 22
	rankW          = 56
	lensW          = 104
	rebindW        = 146
	footH          = 16
	councilTip     = "The Council: your advisors and the realm's ratings [C]"
	RankLockTip    = "Learn Ranks in the Grimoire to rank a crew (priorityClassName)."
)

type advisor struct {
	Id, Label, portrait, fallback, tip string `json:"-"`
}

var advisorK8s = map[string]string{TabDecrees: "ORDERS"}

func (a advisor) title() string {
	if short, ok := advisorK8s[a.Id]; ok {
		return ui.Bridge(a.Label, short)
	}
	return a.Label
}

var Advisors = []advisor{
	{TabQuartermaster, "QUARTERMASTER", "advisor-quartermaster", "quartermaster", "The Quartermaster: food and housing on every Keep, and the Unhoused"},
	{TabStewards, "STEWARDS", "steward", "stewards-hall", "The Stewards: crews, their huts, Ranks and War Horns"},
	{TabWayfinder, "WAYFINDER", "advisor-wayfinder", "wayfinder", "The Wayfinder: Guildhalls and their Guild Rolls, harbors and ships"},
	{TabTreasurer, "TREASURER", "advisor-treasurer", "tallyman", "The Treasurer: gold, upkeep and tributes"},
	{TabAugur, "AUGUR", "augur", "observatory", "The Augur: the busiest huts, the Keeps' ground and the Codex"},
	{TabDecrees, "CONTRACTS", "herald", "herald", "The merchants' contracts: deliveries, deadlines and rewards"},
}

func CouncilOpen(f Frame) bool {
	return f.Revealed(CouncilReveal) && f.Shown("cmd:council")
}

func CouncilBody() ui.Rect {
	return ui.Rect{X: Overlay.X + 10, Y: Overlay.Y + 110, W: Overlay.W - 20, H: Overlay.H - 118}
}

func CouncilTabID(i int) ui.Rect {
	return councilTabAt(i, len(Advisors))
}

func councilTabAt(i, n int) ui.Rect {
	w := (Overlay.W - 20 - (n-1)*tabGap) / n
	return ui.Rect{X: Overlay.X + 10 + i*(w+tabGap), Y: Overlay.Y + 84, W: w, H: tabH}
}

func FootRect(i int) ui.Rect {
	b := CouncilBody()
	return ui.Rect{X: b.X + 4 + i*(lensW+4), Y: b.Y + b.H - footH - 2, W: lensW, H: footH - 2}
}

func (h *HUD) CouncilTab() string {
	if h.CouncilTabID == "" {
		return TabQuartermaster
	}
	return h.CouncilTabID
}

func (h *HUD) councilTabOf(f Frame) string {
	tab := h.CouncilTab()
	for _, a := range councilAdvisors(f) {
		if a.Id == tab {
			return tab
		}
	}
	return TabQuartermaster
}

func (h *HUD) OpenCouncil(tab string) {
	h.OverlayID = OverlayCouncil
	if tab != "" {
		h.CouncilTabID = tab
	}
}

func CouncilPageFor(tab string, f Frame) CouncilPage {
	if f.Live {
		return liveCouncilPage(tab, f)
	}
	switch tab {
	case TabStewards:
		return StewardsPage(f)
	case TabWayfinder:
		return WayfinderPage(f, world.BerthCount)
	case TabTreasurer:
		return TreasurerPage(f)
	case TabAugur:
		return AugurPage(f)
	case TabDecrees:
		return DecreesPage(f)
	}
	return QuartermasterPage(f)
}

func SectionRows(p CouncilPage, top int) [][]int {
	out := make([][]int, len(p.Sections))
	y := top
	for i, s := range p.Sections {
		y += ui.LineHeight
		if len(s.Head) > 0 {
			y += ui.LineHeight
		}
		rows := max(len(s.Rows), 1)
		out[i] = make([]int, rows)
		for r := range rows {
			out[i][r] = y + r*councilRowH
		}
		y += rows*councilRowH + sectionGap
	}
	return out
}

func (h *HUD) CouncilButtons(f Frame) []Button {
	out := append([]Button{closeButton(Overlay)}, dialButtons(f)...)
	out = append(out, counselButtons(f)...)
	active := h.councilTabOf(f)
	advisors := councilAdvisors(f)
	for i, a := range advisors {
		tip := a.tip
		if f.Live {
			tip = liveAdvisorTip(a.Id)
		}
		out = append(out, Button{ID: CouncilPrefix + ":tab:" + a.Id, Rect: councilTabAt(i, len(advisors)), Label: a.title(), Tip: tip, Selected: active == a.Id})
	}
	if f.Live {
		return out
	}
	switch active {
	case TabQuartermaster:
		out = append(out, h.lensButtons(f, []string{LensUnhoused, LensLabor})...)
	case TabStewards:
		out = append(out, rankButtons(f)...)
	case TabAugur:
		out = append(out, h.lensButtons(f, []string{LensLabor, LensMidden, LensCodex})...)
		out = append(out, codexButtons(f)...)
	}
	return out
}

func rankButtons(f Frame) []Button {
	if !RankReachable() {
		return nil
	}
	rows := SectionRows(StewardsPage(f), CouncilBody().Y+PageTop)[0]
	out := []Button{}
	for i, c := range rankedCrews(f) {
		name := CrewName(c)
		for k, r := range model.Ranks {
			rect := ui.Rect{X: CouncilBody().X + 4 + stewardCols[2] + k*(rankW+2), Y: rows[i] - 2, W: rankW, H: councilRowH - 1}
			b := Button{ID: CouncilPrefix + ":rank:" + name + ":" + r, Rect: rect, Label: strings.ToUpper(r), Tip: RankTip(name, r), Selected: c.Rank == r}
			if !RuneKnown(f, view.RankRune) {
				b.Disabled, b.Tip = true, RankLockTip
			}
			out = append(out, b)
		}
	}
	return out
}

var rankValues = map[string]string{model.RankHigh: "1000", model.RankNormal: "100", model.RankLow: "0"}

func RankTip(crew, rank string) string {
	return "Rank the crew " + crew + " " + rank + ": PriorityClass " + model.RankClass(rank) + " (value " + rankValues[rank] +
		"). Its rollout carries the Rank to every new hut."
}

func SetRank(name, rank string) view.Action {
	return view.Action{Action: "setRank", Name: name, Rank: rank}
}

func RebindCodex() view.Action { return view.Action{Action: "rebindCodex"} }

func codexButtons(f Frame) []Button {
	c := f.Caesar.Codex
	out := []Button{}
	if c.Large() {
		b := CouncilBody()
		out = append(out, Button{ID: CouncilPrefix + ":rebind", Rect: ui.Rect{X: b.X + b.W - rebindW - 4, Y: FootRect(0).Y, W: rebindW, H: footH - 2},
			Label: "REBIND THE CODEX", Tip: rebindTip})
	}
	if c.Alarm {
		out = append(out, Button{ID: CouncilPrefix + ":disarm", Rect: disarmRect(FootRect(3)), Tip: disarmTip})
	}
	return out
}

const (
	rebindTip = "Rebind the Codex (etcdctl defrag): write it afresh and give back the space its old pages hold."
	disarmTip = "Opens the Speaking Stone with this command. After a rebind, disarm the NOSPACE alarm."
)

func disarmRect(after ui.Rect) ui.Rect {
	w := len(SpeakLabel(DisarmCommand))*ui.Advance + chipPad
	return ui.Rect{X: after.X, Y: after.Y + 1, W: w, H: ChipH}
}

func (h *HUD) pressCouncil(rest string, f Frame) []Intent {
	click := Intent{Kind: IntentSound, Command: "click"}
	kind, arg, _ := strings.Cut(rest, ":")
	switch kind {
	case "tab":
		h.CouncilTabID = arg
	case "open":
		h.OpenCouncil(arg)
	case "rank":
		name, rank, _ := strings.Cut(arg, ":")
		return []Intent{act(SetRank(name, rank))}
	case "rebind":
		return []Intent{ActFrom(RebindCodex(), FromCouncil)}
	case "disarm":
		h.OpenStone(DisarmCommand)
	case "fix":
		h.OverlayID = OverlayNone
		return h.PressFix(f)
	case "show":
		h.OverlayID = OverlayNone
		return h.FollowNext(f)
	}
	return []Intent{click}
}

const counselBtnW = 120

func counselButtons(f Frame) []Button {
	if f.Live {
		return nil
	}
	box := CounselBox()
	rect := ui.Rect{X: box.X + box.W - counselBtnW, Y: box.Y + (counselLines-1)*ui.LineHeight + 2, W: counselBtnW, H: 14}
	if fix := EconomyOf(f.State).Fix; fix != nil && FixFade(f, *fix) != view.FixNames {
		label := ui.Clip("DO IT: "+ShortFixLabel(*fix), ui.MaxChars(counselBtnW-8, 1))
		b := Button{ID: CouncilPrefix + ":fix", Rect: rect, Label: label, Tip: FixTip(*fix)}
		if price := priceOf(f, *fix); price.closed {
			b.Disabled, b.Tip = true, cmp.Or(price.reason, b.Tip)
		}
		return []Button{b}
	}
	if f.State.Advice != "" && Counsel(f) == f.State.Advice {
		return []Button{{ID: CouncilPrefix + ":show", Rect: rect, Label: "SHOW ME", Tip: "Close the Council and go where the advice points."}}
	}
	return nil
}

func (h *HUD) toggleCouncil() []Intent {
	h.toggleOverlay(OverlayCouncil)
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func councilWellButton(f Frame) (Button, bool) {
	if !CouncilOpen(f) {
		return Button{}, false
	}
	w, tip := 60, councilTip
	if f.Live {
		tip = liveCouncilTip
	}
	return Button{ID: "cmd:council", Rect: ui.Rect{X: InfoWell.X + InfoWell.W - 5 - w, Y: InfoWell.Y + 4, W: w, H: 14}, Label: councilLabel, Tip: tip}, true
}
