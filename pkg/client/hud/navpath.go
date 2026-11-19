package hud

import (
	"slices"
	"strings"
)

const MaxNavDepth = 6

type NavState struct {
	Overlay  string
	Page     string
	Tab      string
	Council  string
	Grimoire string
	Harbor   bool
	Chain    bool
	Object   bool
	Crew     bool
	Sub      bool
	Mode     string
	Haven    string
}

type navPlace struct {
	overlay string
	page    string
	tab     string
	council string
	pre     string
}

var overlayOpeners = map[string]string{
	OverlayGrimoire: "cmd:grimoire", OverlaySoulvault: "cmd:soulvault",
	OverlayChronicles: "cmd:chronicles", OverlayAge: "cmd:age", OverlayReforge: "reforge:open", OverlayMenu: "top:menu",
	OverlaySiege: "cmd:siege", OverlayRoad: "age:road", OverlayVoyage: "cmd:voyage", OverlayCouncil: "cmd:council",
	OverlayLedger: LedgerButton, OverlayStone: "well:stone", OverlayScriptorium: "well:yaml", OverlayContracts: ContractsOpen,
}

var overlayPrefixes = map[string]string{
	"grim": OverlayGrimoire, "kmap": OverlayGrimoire, "learn": OverlayGrimoire, "vault": OverlaySoulvault, acadPrefix: OverlaySoulvault,
	"chron": OverlayChronicles, "age": OverlayAge, "loom": OverlayAge, "menu": OverlayMenu,
	"siege": OverlaySiege, "road": OverlayRoad, "voyage": OverlayVoyage, "council": OverlayCouncil,
	"stone": OverlayStone, "scr": OverlayScriptorium, "contract": OverlayContracts, "reforge": OverlayReforge,
}

var screenPlaces = map[string]navPlace{
	"reforge:open": {overlay: OverlayAge}, ContractsOpen: {tab: TabTrade}, "cmd:council": {},
	"well:stone": {tab: TabKube}, "well:yaml": {tab: TabYAML}, "well:tech": {tab: TabTech},
	SurfaceLoom: {overlay: OverlayAge}, SurfaceReforge: {overlay: OverlayAge},
}

var councilTabs = map[string]string{"rank": TabStewards, "rebind": TabAugur, "disarm": TabAugur}

var cardPages = []struct{ prefix, page string }{
	{"build:", PageBuild}, {"summon:", PageSummon}, {"cmd:voyage", PageBuild}, {stallPrefix + ":", PageBuild}, {"cmd:", PageMain},
}

var pageParents = map[string]string{PageBuild: PageMain, PageSummon: PageMain, PageMore: PageMain}

var pageOpeners = map[string]string{PageBuild: "cmd:build", PageSummon: "cmd:summon", PageMore: CmdMore}

var closeIDs = map[string]string{OverlayReforge: "reforge:close", OverlaySiege: "siege:close", OverlayWelcome: "welcome:close"}

var closesTo = map[string]string{OverlayReforge: OverlayAge}

var NavAliases = map[string]string{SurfaceBerths: BerthButton, SurfaceStone: overlayOpeners[OverlayStone], SurfaceScriptorium: overlayOpeners[OverlayScriptorium], SurfaceContracts: "tab:" + TabTrade, SurfaceLedger: LedgerButton}

var navArrivals = map[string]string{
	"tab:" + TabKube: OverlayStone, "tab:" + TabYAML: OverlayScriptorium, "tab:" + TabTrade: OverlayContracts, "well:tech": OverlayGrimoire,
	overlayOpeners[OverlayStone]: OverlayStone,
}

func placeOf(target string) navPlace {
	if p, ok := screenPlaces[target]; ok {
		return p
	}
	prefix, rest, _ := strings.Cut(target, ":")
	if o, ok := overlayPrefixes[prefix]; ok {
		p := navPlace{overlay: o}
		if o == OverlayCouncil {
			kind, _, _ := strings.Cut(rest, ":")
			p.council = councilTabs[kind]
		}
		if prefix == "learn" {
			p.pre = "grim:" + rest
		}
		return p
	}
	if prefix == "tiding" {
		return navPlace{tab: TabEvents}
	}
	for _, c := range cardPages {
		if strings.HasPrefix(target, c.prefix) {
			return navPlace{page: c.page}
		}
	}
	return navPlace{}
}

func Arrived(target string, st NavState) bool {
	if st.Overlay == "" {
		return false
	}
	if o, ok := overlayButtons[target]; ok && o == st.Overlay {
		return true
	}
	return navArrivals[target] == st.Overlay
}

func NavPath(target string, st NavState) []string {
	if alias, ok := NavAliases[target]; ok {
		target = alias
	}
	if target == "" || Arrived(target, st) {
		return nil
	}
	if (target == BerthButton || target == CloseBerthButton) && !st.Harbor {
		return append(navSteps(GateHaven, normal(st), MaxNavDepth), target)
	}
	if steps, ok := portSteps(target, st); ok {
		return steps
	}
	if steps, ok := crewSteps(target, st); ok {
		return steps
	}
	if st.Sub && objectTarget(target) {
		return []string{ChainButton, target}
	}
	return navSteps(target, normal(st), MaxNavDepth)
}

func normal(st NavState) NavState {
	next := st
	if next.Page == "" {
		next.Page = PageMain
	}
	return next
}

func navSteps(target string, st NavState, depth int) []string {
	if depth <= 0 {
		return nil
	}
	p := placeOf(target)
	if st.Overlay != "" && st.Overlay != p.overlay && !opensFrom(p.overlay, st.Overlay) {
		closed := st
		closed.Overlay = closesTo[st.Overlay]
		return append([]string{closeID(st.Overlay)}, navSteps(target, closed, depth-1)...)
	}
	if p.overlay != "" && st.Overlay != p.overlay {
		opener := overlayOpeners[p.overlay]
		inside := st
		inside.Overlay = p.overlay
		return append(navSteps(opener, st, depth-1), overlaySteps(p, target, inside)...)
	}
	if p.overlay != "" {
		return overlaySteps(p, target, st)
	}
	return append(screenSteps(p, st), target)
}

func overlaySteps(p navPlace, target string, st NavState) []string {
	out := []string{}
	if p.council != "" && st.Council != p.council {
		out = append(out, CouncilPrefix+":tab:"+p.council)
	}
	if p.pre != "" && st.Grimoire != strings.TrimPrefix(p.pre, "grim:") {
		out = append(out, p.pre)
	}
	return append(out, target)
}

func screenSteps(p navPlace, st NavState) []string {
	switch {
	case p.page != "" && st.Object:
		return append([]string{SelClose}, pageSteps(PageMain, p.page)...)
	case p.page != "":
		return pageSteps(st.Page, p.page)
	case p.tab != "" && st.Tab != p.tab:
		return []string{"tab:" + p.tab}
	}
	return nil
}

func closeID(overlay string) string {
	if id, ok := closeIDs[overlay]; ok {
		return id
	}
	return "overlay:close"
}

func pageLine(page string) []string {
	out := []string{}
	for p := page; p != ""; p = pageParents[p] {
		out = append(out, p)
		if len(out) > len(pageParents)+1 {
			break
		}
	}
	return out
}

func pageSteps(from, to string) []string {
	down := pageLine(to)
	out := []string{}
	for _, p := range pageLine(from) {
		if i := slices.Index(down, p); i >= 0 {
			for j := i - 1; j >= 0; j-- {
				out = append(out, pageOpeners[down[j]])
			}
			return out
		}
		out = append(out, "cmd:back")
	}
	for j := len(down) - 2; j >= 0; j-- {
		out = append(out, pageOpeners[down[j]])
	}
	return out
}

func opensFrom(overlay, from string) bool {
	for range MaxNavDepth {
		if overlay == "" {
			return false
		}
		overlay = placeOf(overlayOpeners[overlay]).overlay
		if overlay == from {
			return true
		}
	}
	return false
}
