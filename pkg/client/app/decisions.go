package app

import (
	"sort"
	"strconv"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/view"
)

var dangerTitles = map[string]string{
	hud.AlertFire: "Huts guttering", hud.AlertGhost: "Graduates lost at sea", hud.AlertFallen: "Huts fallen",
	hud.AlertPending: "Huts Unhoused", hud.AlertKeep: "A Keep in trouble", hud.AlertDark: "A building has gone dark",
	hud.AlertImp: "The Simian Imp runs", hud.AlertSiege: "The Rush Tide", hud.AlertStorm: "A storm over an isle",
}

func DangerNotices(alerts []hud.Alert) []hud.Notice {
	out := []hud.Notice{}
	index := map[string]int{}
	for _, a := range alerts {
		if i, ok := index[a.Kind]; ok {
			out[i].Count++
			continue
		}
		at := a.At
		title, ok := dangerTitles[a.Kind]
		if !ok {
			title = a.Text
		}
		index[a.Kind] = len(out)
		out = append(out, hud.Notice{ID: "danger:" + a.Kind, Kind: hud.NoticeDanger, Icon: hud.AlertIcon(a.Kind), Title: title,
			Detail: a.Text, Count: 1, At: &at})
	}
	for i := range out {
		if out[i].Count > 1 {
			out[i].Detail += " (and " + strconv.Itoa(out[i].Count-1) + " more)"
		}
	}
	return sortBySeverity(out)
}

func sortBySeverity(ns []hud.Notice) []hud.Notice {
	out := append([]hud.Notice(nil), ns...)
	sort.SliceStable(out, func(i, j int) bool {
		return hud.AlertLevel(strings.TrimPrefix(out[i].ID, "danger:")) > hud.AlertLevel(strings.TrimPrefix(out[j].ID, "danger:"))
	})
	return out
}

type decision func(hud.Frame) (hud.Notice, bool)

var decisions = []decision{pendingChronicles, learnablePages, siegeReady, ageReady, reforgeReady, academyClasses, contractOffers, calmOffer}

func DecisionNotices(f hud.Frame) []hud.Notice {
	out := []hud.Notice{}
	seen := map[string]bool{}
	for _, d := range decisions {
		n, ok := d(f)
		if !ok || seen[n.ID] {
			continue
		}
		seen[n.ID] = true
		out = append(out, n)
	}
	return out
}

func decide(intent, icon, title, detail string, count int) hud.Notice {
	return hud.Notice{ID: "decision:" + intent, Kind: hud.NoticeDecision, Icon: icon, Title: title, Detail: detail, Count: count, Intent: intent}
}

func pendingChronicles(f hud.Frame) (hud.Notice, bool) {
	open := []view.ChronicleView{}
	for _, c := range f.State.Chronicles {
		if !c.Resolved {
			open = append(open, c)
		}
	}
	if len(open) == 0 || !f.Revealed("chronicles") {
		return hud.Notice{}, false
	}
	return decide("cmd:chronicles", "icon/chronicle", "A Chronicle of Woe awaits", open[0].Calamity+": "+open[0].Question, len(open)), true
}

func learnablePages(f hud.Frame) (hud.Notice, bool) {
	names := []string{}
	for _, p := range f.State.Grimoire {
		if p.State == "available" && f.State.Gold >= p.Cost.Gold && f.State.Wisdom >= p.Cost.Wisdom {
			names = append(names, p.Name)
		}
	}
	if len(names) == 0 || !f.Revealed("grimoire") {
		return hud.Notice{}, false
	}
	return decide("cmd:grimoire", "icon/tech", "Pages to learn", "You can learn "+listOf(names)+".", len(names)), true
}

func listOf(names []string) string {
	if len(names) > listMax {
		return strings.Join(names[:listMax], ", ") + " and more"
	}
	return strings.Join(names, ", ")
}

func calmOffer(f hud.Frame) (hud.Notice, bool) {
	if !f.State.CalmOffer {
		return hud.Notice{}, false
	}
	return decide("menu:calm", "icon/alert", "A calmer realm?", "Press for no Rush Tide and troubles half as often. The MENU switches it back.", 0), true
}

func siegeReady(f hud.Frame) (hud.Notice, bool) {
	if !f.State.Siege.Available || hud.SiegeActive(f) {
		return hud.Notice{}, false
	}
	return decide("cmd:siege", "icon/siege", "The Rush Tide can be answered", "Open the Rush Tide card to prepare, then answer it.", 0), true
}

func ageReady(f hud.Frame) (hud.Notice, bool) {
	a := f.State.AgeUp
	if a.NextAge == "" || !a.Available {
		return hud.Notice{}, false
	}
	for _, r := range a.Requirements {
		if !r.Met {
			return hud.Notice{}, false
		}
	}
	return decide("cmd:age", "icon/age", "Advance to the "+view.Content.AgeTitle(a.NextAge), "Every requirement is met.", 0), true
}

func reforgeReady(f hud.Frame) (hud.Notice, bool) {
	if !f.State.Reforge.Available || !f.Revealed("reforge") {
		return hud.Notice{}, false
	}
	return decide("reforge:open", "icon/age", "The realm may be Reforged", "Begin again on a new island. Only what is declared survives.", 0), true
}

func academyClasses(f hud.Frame) (hud.Notice, bool) {
	names := []string{}
	for _, s := range f.State.Soulstones {
		if s.Visible && !s.Owned && s.Cost <= f.State.Gold {
			names = append(names, s.Name)
		}
	}
	if len(names) == 0 || !f.Revealed("soulvault") {
		return hud.Notice{}, false
	}
	return decide("cmd:soulvault", "icon/academy", "A new Academy class", "You can endow "+listOf(names)+".", len(names)), true
}
