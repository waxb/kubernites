package hud

import (
	"fmt"
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestNavPathWalksThroughMenus(t *testing.T) {
	cases := []struct {
		name   string
		target string
		st     NavState
		want   []string
	}{
		{"behind the manual", "build:keep", NavState{Overlay: OverlayGrimoire}, []string{"overlay:close", "cmd:build", "build:keep"}},
		{"behind BACK", "cmd:summon", NavState{Page: PageBuild}, []string{"cmd:back", "cmd:summon"}},
		{"in the Council's Stewards", "council:rank:porters:high", NavState{Council: TabQuartermaster},
			[]string{"cmd:council", "council:tab:stewards", "council:rank:porters:high"}},
		{"on the Stewards tab already", "council:rank:porters:high", NavState{Overlay: OverlayCouncil, Council: TabStewards}, []string{"council:rank:porters:high"}},
		{"more to summon", "summon:porter:1.0", NavState{Page: PageMore}, []string{"cmd:back", "cmd:summon", "summon:porter:1.0"}},
		{"voyage", "voyage:charter:isle-2", NavState{}, []string{"cmd:build", "cmd:voyage", "voyage:charter:isle-2"}},
		{"stone", "stone:grow", NavState{Tab: TabEvents}, []string{"tab:KUBE", "well:stone", "stone:grow"}},
		{"stone tab shown", "stone:grow", NavState{Tab: TabKube}, []string{"well:stone", "stone:grow"}},
		{"scriptorium", "scr:apply", NavState{Tab: TabEvents}, []string{"tab:YAML", "well:yaml", "scr:apply"}},
		{"contracts", "contract:accept:k1", NavState{Tab: TabEvents}, []string{"tab:TRADE", ContractsOpen, "contract:accept:k1"}},
		{"reforge", "reforge:confirm", NavState{}, []string{"cmd:age", "reforge:open", "reforge:confirm"}},
		{"rebuild back to the tier", "age:advance", NavState{Overlay: OverlayReforge}, []string{"reforge:close", "age:advance"}},
		{"road", "road:loom", NavState{}, []string{"cmd:age", "age:road", "road:loom"}},
		{"road from the Tier", "road:loom", NavState{Overlay: OverlayAge}, []string{"age:road", "road:loom"}},
		{"rebuild from the Tier", "reforge:confirm", NavState{Overlay: OverlayAge}, []string{"reforge:open", "reforge:confirm"}},
		{"loom", SurfaceLoom, NavState{Overlay: OverlayMenu}, []string{"overlay:close", "cmd:age", SurfaceLoom}},
		{"open a page", "learn:muster", NavState{}, []string{"cmd:grimoire", "grim:muster", "learn:muster"}},
		{"page already open", "learn:muster", NavState{Overlay: OverlayGrimoire, Grimoire: "muster"}, []string{"learn:muster"}},
		{"menu", "menu:save", NavState{Page: PageSummon}, []string{"top:menu", "menu:save"}},
		{"map behind a window", "pod:default/porter-a", NavState{Overlay: OverlayCouncil}, []string{"overlay:close", "pod:default/porter-a"}},
		{"map", "keep:lone-tower", NavState{Page: PageBuild}, []string{"keep:lone-tower"}},
		{"spike modal", "cmd:build", NavState{Overlay: OverlaySiege}, []string{"siege:close", "cmd:build"}},
		{"a meter under the Ledger", MeterGold, NavState{Overlay: OverlayLedger}, []string{"overlay:close", MeterGold}},
		{"stone surface", SurfaceStone, NavState{Tab: TabEvents}, []string{"tab:KUBE", "well:stone"}},
		{"stone surface on its tab", SurfaceStone, NavState{Tab: TabKube}, []string{"well:stone"}},
		{"the ledger surface", SurfaceLedger, NavState{Overlay: OverlayGrimoire}, []string{"overlay:close", LedgerButton}},
		{"tidings", "tiding:2", NavState{Tab: TabKube}, []string{"tab:EVT", "tiding:2"}},
		{"arrived", "cmd:grimoire", NavState{Overlay: OverlayGrimoire}, nil},
		{"arrived at the stone", SurfaceStone, NavState{Overlay: OverlayStone}, nil},
		{"nothing", "", NavState{}, nil},
		{"a berth from the realm", BerthButton, NavState{}, []string{GateHaven, BerthButton}},
		{"a berth on the harbor card", SurfaceBerths, NavState{Harbor: true}, []string{BerthButton}},
		{"a berth behind a window", SurfaceBerths, NavState{Overlay: OverlayGrimoire}, []string{"overlay:close", GateHaven, BerthButton}},
	}
	for _, c := range cases {
		if got := NavPath(c.target, c.st); !slices.Equal(got, c.want) {
			t.Errorf("%s: NavPath(%q, %+v) = %v, want %v", c.name, c.target, c.st, got, c.want)
		}
	}
}

func navFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Reveals = append(slices.Clone(f.State.Reveals), "council", "contracts", "keeps", "ageup")
	f.State.Road.Ages = []view.AgeStep{{ID: "dark", Name: "Dark"}, {ID: "feudal", Name: "Feudal"}}
	f.State.Contracts = []view.ContractView{{ID: "k1", Merchant: "Hanse", Title: "Parcels", State: "offered", Box: "parcel"}}
	f.State.Unlocks["charters"] = true
	return f
}

var navTargets = []string{
	"cmd:build", "build:keep", "cmd:summon", "summon:porter:1.0",
	"cmd:grimoire", "grim:muster", "learn:muster", "cmd:soulvault", "vault:porter:1.1", "cmd:chronicles", "chron:c1:a",
	"cmd:age", "age:road", "road:loom", "loom:royal-treasury", SurfaceLoom, "reforge:open", "reforge:confirm", SurfaceReforge,
	"cmd:council", "council:tab:augur", "council:rank:porters:high", "cmd:voyage", "top:menu", "menu:save", SurfaceLedger,
	SurfaceStone, "stone:grow", SurfaceScriptorium, SurfaceContracts, "contract:accept:k1", MeterGold, SurfaceTidings, "tab:EVT",
}

type navStart struct {
	overlay, page, tab, council string
}

func navStarts() []navStart {
	out := []navStart{}
	for _, o := range []string{OverlayNone, OverlayGrimoire, OverlaySoulvault, OverlayChronicles, OverlayAge, OverlayReforge,
		OverlayMenu, OverlayRoad, OverlayVoyage, OverlayCouncil, OverlayLedger, OverlayStone, OverlayScriptorium, OverlayContracts, OverlaySiege} {
		out = append(out, navStart{overlay: o, page: PageMain, tab: TabEvents})
	}
	for _, p := range []string{PageBuild, PageSummon, PageMore} {
		out = append(out, navStart{page: p, tab: TabTech})
	}
	for _, tab := range []string{TabKube, TabYAML, TabTrade} {
		out = append(out, navStart{page: PageMain, tab: tab})
	}
	for _, c := range []string{TabStewards, TabAugur, TabTreasurer} {
		out = append(out, navStart{overlay: OverlayCouncil, page: PageMain, tab: TabEvents, council: c})
	}
	return out
}

func startHUD(s navStart) *HUD {
	h := New(nil, (&fakeRunner{}).backend())
	h.OverlayID, h.PageID, h.TabID, h.CouncilTabID = s.overlay, s.page, s.tab, s.council
	return h
}

func walkTo(t *testing.T, h *HUD, f Frame, target string) error {
	t.Helper()
	for range MaxNavDepth + 2 {
		path := NavPath(target, h.NavState(f))
		if len(path) == 0 {
			if Arrived(target, h.NavState(f)) || Arrived(NavAliases[target], h.NavState(f)) {
				return nil
			}
			return fmt.Errorf("no path from %+v", h.NavState(f))
		}
		r, ok := h.StepRect(path[0], f)
		if !ok {
			return fmt.Errorf("step %s of %v is not on screen at %+v", path[0], path, h.NavState(f))
		}
		if len(path) == 1 {
			return nil
		}
		h.Update(Input{MX: r.X + r.W/2, MY: r.Y + r.H/2, Click: true}, f)
		h.Update(Input{MX: -50, MY: -50}, f)
	}
	return fmt.Errorf("never reached, stuck at %+v", h.NavState(f))
}

func TestEveryTargetIsReachableFromEveryScreen(t *testing.T) {
	withRankReachable(t, true)
	f := navFrame(t)
	for _, s := range navStarts() {
		for _, target := range navTargets {
			if err := walkTo(t, startHUD(s), f, target); err != nil {
				t.Errorf("%s from %+v: %v", target, s, err)
			}
		}
	}
}

func TestStaticTargetsExistInTheFixture(t *testing.T) {
	f := navFrame(t)
	h := New(nil, (&fakeRunner{}).backend())
	if _, ok := h.StepRect("build:keep", f); ok {
		t.Fatal("build:keep should not be on the main page")
	}
	h.PageID = PageBuild
	if _, ok := h.StepRect("build:keep", f); !ok {
		t.Fatal("the fixture offers no Keep to build")
	}
	if _, ok := HitID(h.CommandButtons(f), "cmd:voyage"); !ok {
		t.Fatal("the fixture offers no Voyage")
	}
}

func TestTheGuideReachesCloseBerthThroughTheHarbor(t *testing.T) {
	if got := NavPath(CloseBerthButton, NavState{}); len(got) < 2 || got[len(got)-1] != CloseBerthButton || !slices.Contains(got, GateHaven) {
		t.Errorf("the tide's berth lever is out of the guide's reach: %v", got)
	}
}
