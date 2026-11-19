package hud

import (
	"fmt"
	"sort"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	OverlayContracts = "contracts"
	SurfaceContracts = "surface:contracts"
	TabTrade         = "TRADE"
	ContractsOpen    = contractPrefix + ":open"
	contractPrefix   = "contract"
	ContractsReveal  = "contracts"
	MaxHeld          = 2
	StateOffered     = "offered"
	StateActive      = "active"
	CrownIcon        = "icon/decree"
	FullHands        = "you already hold 2 contracts: deliver or abandon one first"
)

type ContractRow struct {
	ID         string
	Merchant   string
	Title      string
	Goal       string
	Icon       string
	Sealed     bool
	Crowned    bool
	Progress   string
	Done       float64
	Total      float64
	Left       float64
	Reward     string
	State      string
	CanAccept  bool
	CanAbandon bool
}

func RewardText(r view.ContractReward) string {
	parts := []string{}
	if r.Gold > 0 {
		parts = append(parts, "+"+Gold(r.Gold))
	}
	if r.Wisdom > 0 {
		parts = append(parts, "+"+Wisdom(r.Wisdom))
	}
	if r.Favor > 0 {
		parts = append(parts, fmt.Sprintf("+%.0f Favor", r.Favor))
	}
	if r.Boost > 0 {
		boost := "+" + Compact(r.Boost)
		parts = append(parts, ui.Bridge(boost+" harbor tier", boost+" traffic"))
	}
	return strings.Join(parts, "  ")
}

func Held(f Frame) int {
	n := 0
	for _, c := range f.State.Contracts {
		if c.State == StateActive {
			n++
		}
	}
	return n
}

func contractIcon(c view.ContractView) string {
	if c.Box == "" {
		return CrownIcon
	}
	return "icon/kind-" + c.Box
}

func ContractRows(f Frame) []ContractRow {
	held := Held(f)
	out := []ContractRow{}
	for _, c := range f.State.Contracts {
		if c.State != StateOffered && c.State != StateActive {
			continue
		}
		out = append(out, ContractRow{
			ID: c.ID, Merchant: c.Merchant, Title: c.Title, Goal: c.Goal, Icon: contractIcon(c), Sealed: c.Sealed, Crowned: c.Crowned,
			Progress: c.Progress, Done: c.Done, Total: c.Total, Left: max(c.Deadline-f.State.RealmTime, 0), Reward: RewardText(c.Reward),
			State: c.State, CanAccept: c.State == StateOffered && held < MaxHeld, CanAbandon: c.State == StateActive,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].State == StateActive && out[j].State != StateActive })
	return out
}

func (r ContractRow) Share() float64 {
	if r.Total <= 0 {
		return 0
	}
	return min(max(r.Done/r.Total, 0), 1)
}

func (r ContractRow) Countdown() string { return model.Countdown(r.Left) }

func (r ContractRow) Clock() string {
	if r.State == StateActive {
		return "due in " + model.Countdown(r.Left)
	}
	return "offer ends in " + model.Countdown(r.Left)
}

func Offered(f Frame) []ContractRow {
	out := []ContractRow{}
	for _, r := range ContractRows(f) {
		if r.State == StateOffered {
			out = append(out, r)
		}
	}
	return out
}

func ContractsKnown(f Frame) bool {
	return f.Revealed(ContractsReveal) && f.Shown(SurfaceContracts)
}

func findRow(f Frame, id string) (ContractRow, bool) {
	for _, r := range ContractRows(f) {
		if r.ID == id {
			return r, true
		}
	}
	return ContractRow{}, false
}

func (h *HUD) openContracts(f Frame) []Intent {
	h.OverlayID = OverlayContracts
	out := []Intent{{Kind: IntentSound, Command: "click"}}
	if in := f.Guide.Intro; in != nil && in.Target == SurfaceContracts {
		out = append(out, act(Ack(in.ID)))
	}
	return out
}

func (h *HUD) PressContract(rest string, f Frame) []Intent {
	verb, id, _ := strings.Cut(rest, ":")
	if verb == "open" {
		return h.openContracts(f)
	}
	row, ok := findRow(f, id)
	if !ok {
		return nil
	}
	switch {
	case verb == "accept" && row.CanAccept:
		return []Intent{act(view.Action{Action: "acceptContract", ID: id})}
	case verb == "abandon" && row.CanAbandon && !h.abandonArmed(id):
		h.Ops.abandon = retireArm{id: abandonKey(id), at: h.Now}
		return []Intent{{Kind: IntentSound, Command: "click"}}
	case verb == "abandon" && row.CanAbandon:
		h.Ops.abandon = retireArm{}
		return []Intent{act(view.Action{Action: "abandonContract", ID: id})}
	}
	return nil
}

func abandonKey(id string) string { return "abandon:" + id }

func (h *HUD) abandonArmed(id string) bool {
	a := h.Ops.abandon
	return a.id == abandonKey(id) && h.Now >= a.at && h.Now-a.at <= retireWindow
}
