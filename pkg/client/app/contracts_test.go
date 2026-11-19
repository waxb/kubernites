package app

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/view"
)

func contractState() view.GameState {
	st := view.GameState{}
	st.Reveals = []string{"contracts"}
	st.RealmTime = 100
	st.Contracts = []view.ContractView{
		{ID: "contract-4", Merchant: "the web crew", Title: "Deliver 45 crowned parcels fresh within 4 minutes", State: "offered", Deadline: 250,
			Reward: view.ContractReward{Gold: 450, Wisdom: 5, Favor: 6}},
		{ID: "contract-5", Merchant: "the db crew", Title: "Hold the rollout", State: "offered", Deadline: 300},
		{ID: "contract-3", Merchant: "the Pearl Factors", Title: "Deliver crates", State: "active", Deadline: 200},
	}
	return st
}

func TestContractOffersJoinTheRailAsDecisions(t *testing.T) {
	f := hud.Frame{State: contractState()}
	n, ok := contractOffers(f)
	if !ok || n.Count != 2 || n.Intent != hud.ContractsOpen || n.Kind != hud.NoticeDecision || n.Icon != decreeIcon {
		t.Fatalf("offer notice = %+v", n)
	}
	if !strings.HasPrefix(n.Detail, "The web crew offers: Deliver 45 crowned parcels") || !strings.Contains(n.Detail, "ends in 2:30") {
		t.Errorf("detail = %q", n.Detail)
	}
	f.State.Reveals = nil
	if _, ok := contractOffers(f); ok {
		t.Error("offers wait for the contracts reveal")
	}
	quiet := hud.Frame{State: view.GameState{Snapshot: view.Snapshot{Reveals: []string{"contracts"}}}}
	if _, ok := contractOffers(quiet); ok {
		t.Error("no offer, no decision")
	}
	found := false
	for _, d := range DecisionNotices(hud.Frame{State: contractState()}) {
		found = found || d.Intent == hud.ContractsOpen
	}
	if !found {
		t.Error("the rail lists contract offers")
	}
	if capitalized("") != "" {
		t.Error("empty merchant")
	}
}

func TestContractNewsMetAndMissedButNotOffers(t *testing.T) {
	procs := []view.Proclamation{
		{T: 10, Kind: "unlock", Title: "Contract delivered", Text: "The web crew is healthy."},
		{T: 11, Kind: "calamity", Title: "Contract missed", Text: "Deliver crates lapsed."},
		{T: 12, Kind: "info", Title: offerTitle, Text: "The db crew offers: rollout."},
		{T: 13, Kind: "unlock", Title: "Learned", Text: "x"},
	}
	news := NewsNotices(procs, 0, 20, view.Site{})
	if len(news) != 2 {
		t.Fatalf("news = %+v", news)
	}
	for _, n := range news {
		if n.Icon != decreeIcon {
			t.Errorf("contract news wears the seal: %+v", n)
		}
	}
	if !ContractNews(procs[0]) || ContractNews(procs[2]) || !OfferNews(procs[2]) {
		t.Error("contract news classification")
	}
}
