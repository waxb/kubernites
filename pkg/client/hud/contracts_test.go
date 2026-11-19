package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func contractFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Reveals = append(append([]string(nil), allReveals...), ContractsReveal)
	f.State.RealmTime = 1000
	f.State.Contracts = []view.ContractView{
		{ID: "contract-1", Merchant: "Vendor A", Title: "Deliver 45 crowned parcels fresh within 4 minutes", Kind: "deliver", Box: "parcel",
			Crowned: true, Progress: "on offer for 3m10s more", Total: 45, Deadline: 1190, Reward: view.ContractReward{Gold: 450, Wisdom: 5, Favor: 6}, State: "offered"},
		{ID: "contract-2", Merchant: "the Pearl Factors", Title: "Deliver 30 sealed crates", Kind: "deliver", Box: "crate", Sealed: true,
			Progress: "12/30 delivered, 2m05s left", Done: 12, Total: 30, Deadline: 1125, Reward: view.ContractReward{Gold: 1200, Wisdom: 5, Favor: 6, Boost: 900}, State: "active"},
		{ID: "contract-3", Merchant: "Vendor C", Title: "Hold the rollout", Kind: "festival", Progress: "held 12/30s", Done: 12, Total: 30, Deadline: 1020,
			Reward: view.ContractReward{Gold: 60, Wisdom: 4, Favor: 8}, State: "active"},
		{ID: "contract-0", Merchant: "the Norhold Traders", Title: "Old", State: "met"},
	}
	return f
}

func TestContractRowsListHeldThenOffered(t *testing.T) {
	f := contractFrame(t)
	rows := ContractRows(f)
	if len(rows) != 3 || rows[0].State != StateActive || rows[1].State != StateActive || rows[2].State != StateOffered {
		t.Fatalf("rows = %+v", rows)
	}
	crates := rows[0]
	if crates.Icon != "icon/kind-crate" || !crates.Sealed || crates.Share() != .4 || crates.Clock() != "due in 2:05" || !crates.CanAbandon {
		t.Errorf("crate row = %+v", crates)
	}
	if crates.Reward != "+"+ui.IconGold+"1.2K  +"+ui.IconWisdom+"5  +6 Favor  +900 harbor tier" {
		t.Errorf("reward = %q", crates.Reward)
	}
	if rows[1].Icon != CrownIcon {
		t.Errorf("a contract without a box wears the generic seal: %q", rows[1].Icon)
	}
	offer := rows[2]
	if offer.CanAccept || offer.Clock() != "offer ends in 3:10" {
		t.Errorf("two held: no third accept; %+v", offer)
	}
	if Held(f) != 2 || len(Offered(f)) != 1 {
		t.Errorf("held %d offered %d", Held(f), len(Offered(f)))
	}
	f.State.Contracts[2].State = "met"
	if !ContractRows(f)[1].CanAccept {
		t.Error("one held: the offer can be taken")
	}
	if (ContractRow{}).Share() != 0 || RewardText(view.ContractReward{}) != "" {
		t.Error("empty rows")
	}
}

func TestContractsBoardAcceptsAndAbandonsWithAConfirm(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := contractFrame(t)
	f.State.Contracts[2].State = "met"
	h.Open(OverlayContracts)
	intents := clickOn(t, h, f, "contract:accept:contract-1")
	if it, ok := hasIntent(intents, IntentAction); !ok || it.Action.Action != "acceptContract" || it.Action.ID != "contract-1" {
		t.Fatalf("accept = %+v", intents)
	}
	intents = clickOn(t, h, f, "contract:abandon:contract-2")
	if _, ok := hasIntent(intents, IntentAction); ok {
		t.Fatal("the first click only arms ABANDON")
	}
	if b, _ := HitID(h.Buttons(f), "contract:abandon:contract-2"); b.Label != "SURE?" {
		t.Errorf("armed label = %q", b.Label)
	}
	intents = clickOn(t, h, f, "contract:abandon:contract-2")
	if it, ok := hasIntent(intents, IntentAction); !ok || it.Action.Action != "abandonContract" {
		t.Errorf("second click abandons: %+v", intents)
	}
	if h.PressContract("accept:nope", f) != nil || h.PressContract("fly:contract-1", f) != nil {
		t.Error("unknown contracts do nothing")
	}
	full := contractFrame(t)
	b, _ := HitID(h.Buttons(full), "contract:accept:contract-1")
	if !b.Disabled || b.Tip != FullHands {
		t.Errorf("a third accept is disabled with the reason: %+v", b)
	}
}

func TestContractsBoardLayoutFitsTheWindow(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := contractFrame(t)
	h.Open(OverlayContracts)
	buttons := h.OverlayButtons(f)
	for i := 0; i < ContractsShown; i++ {
		row := ContractRowRect(i)
		if !contains(Overlay, row) {
			t.Errorf("row %d %+v leaves the board", i, row)
		}
		if b := ContractButtonRect(row); !contains(row, b) {
			t.Errorf("button %+v leaves row %d", b, i)
		}
	}
	for i, a := range buttons {
		for _, b := range buttons[i+1:] {
			if Overlaps(a.Rect, b.Rect) {
				t.Errorf("%s overlaps %s", a.ID, b.ID)
			}
		}
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
	empty := fullFrame(t)
	h.Draw(dst, empty)
}

func TestTradeTabTeachesAndOpensTheBoard(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := contractFrame(t)
	f.Guide = Guide{Staged: true, Intro: &Intro{ID: "contracts", Target: SurfaceContracts, Via: ViaOpen}, Introduced: []string{SurfaceTidings}}
	h.SetTaught([]string{TeachTopBar})
	h.Update(Input{MX: -5, MY: -5}, f)
	f = h.SettleStage(Input{MX: -5, MY: -5}, f, settleLimit)
	if r, ok := h.TargetRect(SurfaceContracts, f); !ok || r.W == 0 {
		t.Fatal("the contracts intro points at the TRADE tab")
	}
	intents := clickOn(t, h, f, "tab:"+TabTrade)
	if it, ok := hasIntent(intents, IntentAction); !ok || it.Action.ID != "contracts" {
		t.Errorf("pressing the tab completes the intro: %+v", intents)
	}
	f.Guide = Guide{}
	clickOn(t, h, f, "tab:"+TabTrade)
	intents = clickOn(t, h, f, ContractsOpen)
	if h.Overlay() != OverlayContracts || len(intents) == 0 {
		t.Errorf("the tab body opens the board: %q", h.Overlay())
	}
	h.OverlayID = OverlayNone
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
	if !ContractsKnown(f) {
		t.Error("revealed and shown")
	}
	f.State.Reveals = allReveals
	if ContractsKnown(f) {
		t.Error("contracts wait for their reveal")
	}
	if !strings.Contains(TradeTip, "Contracts") {
		t.Error("tab tip")
	}
}
