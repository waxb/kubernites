package hud

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	SelHaven         = world.HitHaven
	BerthButton      = "harbor:berth"
	CloseBerthButton = "harbor:close"
	SurfaceBerths    = "surface:berths"
	BerthsReveal     = "berths"
	GateHaven        = "haven:" + world.GateHavenID
	IntentHarbor     = "harbor"
	SpritePortrait   = "sprite:"
	GateHarborTitle  = "GATE HARBOR"
	closeBerthTip    = "Sheds load: fewer ships come. No refund."
	noBerthTip       = "No berth is open"
)

const (
	harborLore = "A harbor is where ships land the realm's boxes: the traffic that comes to your cluster. It is not a Kubernetes object. " +
		"Each berth you open draws more ships, as a production launch opens a service to more users. " +
		"A berth opens only while your crews hold the Nine; if they cannot keep up, you may close one to shed ships."
	harborLoreK8s = "Ships are the traffic that comes to your cluster, and every box they land is a request. " +
		"Each production launch opens the service to more users, so more requests arrive. " +
		"A launch widens only while your Deployments meet their SLO; if they cannot keep up, you may roll one back to shed load."
)

func HarborLore() string { return ui.Bridge(harborLore, harborLoreK8s) }

func HavenSelected(f Frame) bool { return f.Selection.Kind == SelHaven }

func BerthsOffered(f Frame) bool {
	return f.Revealed(BerthsReveal) && f.Shown(SurfaceBerths)
}

func HarborStep(target string, sel Selection) string {
	if (target == BerthButton || target == CloseBerthButton || target == SurfaceBerths) && sel.Kind != SelHaven {
		return GateHaven
	}
	if _, id, ok := PortTarget(target); ok && (sel.Kind != SelHaven || sel.ID != id) {
		return "haven:" + id
	}
	return target
}

func havenNamed(site view.Site, id string) (view.Haven, bool) {
	for _, h := range world.SiteHavens(site) {
		if h.ID == id {
			return h, true
		}
	}
	return view.Haven{}, false
}

func HarborTitle(f Frame) string {
	if h, ok := havenNamed(f.Site, f.Selection.ID); ok && h.Name != "" {
		return strings.ToUpper(h.Name)
	}
	return GateHarborTitle
}

func HarborCard(f Frame) Card {
	if p, ok := SelectedPort(f); ok {
		return PortCard(f, p)
	}
	fv := f.State.Freight
	c := Card{Portrait: SpritePortrait + gfx.ShipName(world.ShipClassOf(fv.Tier), 0, "se", 0), Title: HarborTitle(f), progress: -1}
	c.Lines = append(c.Lines, Line{Text: TierLine(fv), Clr: ui.Palette.GoldBright}, tierBar(fv))
	c = withTreasure(withStranded(withUnserved(c, f), f), f)
	if !BerthsOffered(f) {
		return c
	}
	b := f.State.Berths
	c.Lines = append(c.Lines, Line{Text: BerthsLine(b), Clr: ui.Palette.Text})
	if b.Gate != "" {
		return withLine(c, Line{Text: upperFirst(b.Gate) + ".", Clr: ui.Palette.RedText})
	}
	return withLine(c, Line{Text: "Next: " + b.Effect, Clr: ui.Palette.TextDim})
}

func withLine(c Card, l Line) Card {
	next := c
	next.Lines = append(append([]Line(nil), c.Lines...), l)
	return next
}

func TierLine(fv view.FreightView) string {
	tier := max(fv.Tier, 1)
	return fmt.Sprintf("Tier %d: %s", tier, view.HarborTierOf(tier).Ships)
}

func tierBar(fv view.FreightView) Line {
	if fv.NextTier <= 0 {
		return Line{Text: "The top of the harbor ladder", Clr: ui.Palette.Cream}
	}
	return Line{Text: "NEXT TIER", Clr: ui.Palette.Text, Bar: true, Value: fv.TierProgress, Fill: ui.Palette.Green,
		Suffix: Compact(fv.TierProgress*fv.NextTier) + "/" + Compact(fv.NextTier)}
}

func BerthsLine(b view.BerthView) string {
	open := "No berth open"
	switch {
	case b.Open == 1:
		open = "1 berth open"
	case b.Open > 1:
		open = fmt.Sprintf("%d berths open", b.Open)
	}
	return fmt.Sprintf("%s: ships x%s", open, strconv.FormatFloat(max(b.Multiplier, 1), 'f', -1, 64))
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func OpenBerth(f Frame) Button {
	b := f.State.Berths
	btn := Button{ID: BerthButton, Label: "OPEN BERTH " + Gold(b.NextCost), Tip: withEffect("Draws more ships to your harbors.", b.Effect)}
	switch {
	case b.Gate != "":
		btn.Disabled, btn.Tip = true, upperFirst(b.Gate)+"."
	case f.State.Gold < b.NextCost:
		btn.Disabled, btn.Tip = true, fmt.Sprintf("A berth costs %s gold: not enough gold yet.", Compact(b.NextCost))
	}
	return btn
}

func CloseBerth(f Frame) Button {
	btn := Button{ID: CloseBerthButton, Label: "CLOSE BERTH", Tip: closeBerthTip}
	if f.State.Berths.Open == 0 {
		btn.Disabled, btn.Tip = true, noBerthTip
	}
	return btn
}

func (h *HUD) drawCardPortrait(dst *ebiten.Image, name string, box ui.Rect) {
	if sprite, ok := strings.CutPrefix(name, SpritePortrait); ok {
		h.Lib.Fitted(dst, sprite, box)
		return
	}
	h.Lib.Portrait(dst, strings.TrimPrefix(name, "portrait/"), box)
}
