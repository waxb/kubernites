package app

import (
	"fmt"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	offerTitle     = "A merchant offers a contract"
	contractPrefix = "Contract "
)

func ContractNews(p view.Proclamation) bool {
	return strings.HasPrefix(p.Title, contractPrefix)
}

func OfferNews(p view.Proclamation) bool {
	return p.Title == offerTitle
}

func capitalized(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func contractOffers(f hud.Frame) (hud.Notice, bool) {
	offers := hud.Offered(f)
	if len(offers) == 0 || !hud.ContractsKnown(f) {
		return hud.Notice{}, false
	}
	first := offers[0]
	detail := fmt.Sprintf("%s offers: %s. %s. The offer ends in %s.", capitalized(first.Merchant), first.Title, first.Reward, first.Countdown())
	return decide(hud.ContractsOpen, decreeIcon, offerTitle, detail, len(offers)), true
}
