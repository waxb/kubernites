package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func openingFrame(t *testing.T, carried bool) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Reveals = []string{"serve", "gold", "build", "stone", "summon", "grimoire", "soulvault", "chronicles"}
	f.State.Objectives = []view.ObjectiveView{{ID: CrewServesObjective, Done: carried}, {ID: "three-hearths"}}
	f.Guide = Guide{Staged: true, Introduced: []string{"cmd:serve", "cmd:build", "cmd:summon", "cmd:grimoire", "cmd:soulvault", "cmd:chronicles"}}
	return f
}

func TestTheMainCardHasNoMusterAndNoMusterPage(t *testing.T) {
	h := New(nil, Backend{})
	f := crowdedFrame(t)
	main, more := mainAndMore(t, h, f)
	for _, b := range append(main, more...) {
		if b.ID == "cmd:muster" || b.ID == "cmd:raise" {
			t.Fatalf("%s is still on the command card", b.ID)
		}
	}
	for _, id := range []string{"cmd:muster", "cmd:raise"} {
		if _, ok := pageButtons[id]; ok {
			t.Errorf("%s still opens a page", id)
		}
	}
}

func TestSummonIsInThePrimeCellsOnlyDuringTheOpening(t *testing.T) {
	h := New(nil, Backend{})
	main, more := mainAndMore(t, h, openingFrame(t, false))
	if _, ok := HitID(main, "cmd:summon"); !ok || len(more) != 0 {
		t.Fatalf("SUMMON is not a prime cell during the opening: main %d, more %d", len(main), len(more))
	}
	main, more = mainAndMore(t, h, openingFrame(t, true))
	if _, ok := HitID(main, "cmd:summon"); ok {
		t.Fatal("SUMMON keeps a prime cell after the crew carries")
	}
	if _, ok := HitID(more, "cmd:summon"); !ok {
		t.Fatalf("SUMMON is not on MORE after the crew carries: %+v", more)
	}
	if _, ok := HitID(main, CmdMore); !ok {
		t.Fatal("no MORE button leads to SUMMON")
	}
}

func loneTowerFrame(t *testing.T, reveals ...string) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Reveals = append([]string{"stone"}, reveals...)
	f.Guide = Guide{}
	f.Selection = Selection{Kind: "structure", ID: "keep-1", Structure: view.Structure{ID: "keep-1", Kind: view.StructKeep, Variant: "tower",
		State: view.StateReady, Object: view.ObjectRef{Kind: "Node", Name: view.TowerName}}}
	return f
}

func TestTheLoneTowerCardSummonsAPorter(t *testing.T) {
	f := loneTowerFrame(t, "summon")
	a, ok := ActionByKey(SelectionActions(f), keySummon)
	if !ok || a.Label != "HIRE" || a.Act.Action != "summon" || a.Act.Soulstone != "porter:1.0" || !strings.Contains(ActionTip(a), "kubectl run") || strings.Contains(ActionTip(a), "Pod") {
		t.Fatalf("summon action = %+v", a)
	}
	h := New(nil, Backend{})
	it, ok := hasIntent(h.PressAction(keySummon, f), IntentAction)
	if !ok || it.Action.Action != "summon" || it.From != FromKeepCard {
		t.Fatalf("pressing SUMMON sends %+v", it)
	}
	if _, ok := ActionByKey(SelectionActions(loneTowerFrame(t)), keySummon); ok {
		t.Fatal("the Tower offers SUMMON before it is revealed")
	}
	stone := loneTowerFrame(t, "summon")
	stone.Selection.Structure.Object.Name = "stone-keep-2"
	if _, ok := ActionByKey(SelectionActions(stone), keySummon); ok {
		t.Fatal("a Stone Keep offers SUMMON")
	}
}

func TestTheStewardsListCrewsAndRankThem(t *testing.T) {
	withRankReachable(t, true)
	f := fullFrame(t)
	c := crewChain()
	c.Rank = model.RankHigh
	f.Crews = []model.CrewChain{c, {Namespace: "default", Bare: true, Desired: 1, Ready: 1, Foremen: []model.Foreman{{Name: "lantern"}}}}
	page := StewardsPage(f)
	s := page.Sections[0]
	if s.Title != "CREWS" || s.Head[0] != "CREW" || len(s.Rows) != 2 || s.Rows[0][0].Text != "porters" || s.Rows[1][0].Text != "lantern" {
		t.Fatalf("stewards page = %+v", page)
	}
	for _, text := range append([]string{page.Line, s.Empty, RankLockTip, RankTip("porters", "high")}, s.Head...) {
		if musterWord(text) {
			t.Errorf("%q still holds a retired word", text)
		}
	}
	names := []string{}
	for _, b := range rankButtons(f) {
		names = append(names, b.ID)
	}
	if !slices.Contains(names, CouncilPrefix+":rank:porters:high") || !slices.Contains(names, CouncilPrefix+":rank:lantern:low") {
		t.Fatalf("rank buttons = %v", names)
	}
}

func musterWord(text string) bool {
	for _, w := range []string{"Muster", "MUSTER", "Levy", "Levies", "LEVIES"} {
		if strings.Contains(text, w) {
			return true
		}
	}
	return false
}

func TestNoHUDTextNamesAMuster(t *testing.T) {
	texts := []string{HornLines(view.Structure{}, nil)[0].Text, lore[view.StructTreasury]}
	for _, g := range WelcomeLegend {
		texts = append(texts, g[0], g[1], g[2])
	}
	for _, a := range Advisors {
		texts = append(texts, a.tip)
	}
	for _, text := range texts {
		if musterWord(text) {
			t.Errorf("%q still holds a retired word", text)
		}
	}
}
