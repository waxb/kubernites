package live_test

import (
	"testing"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
)

func withBackends(ing networkingv1.Ingress, services ...string) networkingv1.Ingress {
	out := *ing.DeepCopy()
	rule := out.Spec.Rules[0].DeepCopy()
	template := rule.HTTP.Paths[0]
	rule.HTTP.Paths = nil
	for _, svc := range services {
		path := *template.DeepCopy()
		path.Path = "/" + svc
		path.Backend.Service.Name = svc
		rule.HTTP.Paths = append(rule.HTTP.Paths, path)
	}
	out.Spec.Rules = []networkingv1.IngressRule{*rule}
	return out
}

func TestASignpostTipNamesItsDepotsInASentence(t *testing.T) {
	s := livetest.Managed()
	s.Ingresses = append([]networkingv1.Ingress(nil), s.Ingresses...)
	byName := map[string]int{}
	for i, ing := range s.Ingresses {
		byName[ing.Name] = i
	}
	s.Ingresses[byName["storefront"]] = withBackends(s.Ingresses[byName["storefront"]])
	s.Ingresses[byName["checkout"]] = withBackends(s.Ingresses[byName["checkout"]], "checkout", "accounts")
	s.Ingresses[byName["catalog"]] = withBackends(s.Ingresses[byName["catalog"]], "catalog", "accounts", "storefront")
	w := build(t, s)
	want := map[string]live.Tip{
		"accounts": {Story: "Signpost accounts of the fief market: accounts.market.staging.example.org, pointing to the accounts depot",
			Names: "Ingress market/accounts (class nginx): accounts.market.staging.example.org -> Service market/accounts"},
		"checkout": {Story: "Signpost checkout of the fief market: checkout.market.staging.example.org, pointing to the checkout and accounts depots",
			Names: "Ingress market/checkout (class nginx): checkout.market.staging.example.org -> Services market/checkout, market/accounts"},
		"catalog": {Story: "Signpost catalog of the fief market: catalog.market.staging.example.org, pointing to the catalog, accounts and storefront depots",
			Names: "Ingress market/catalog (class nginx): catalog.market.staging.example.org -> Services market/catalog, market/accounts, market/storefront"},
		"storefront": {Story: "Signpost storefront of the fief market: storefront.market.staging.example.org, pointing to no depot",
			Names: "Ingress market/storefront (class nginx): storefront.market.staging.example.org -> no Service"},
	}
	seen := 0
	for _, sgn := range structuresOf(w, live.KindSignpost) {
		tip, ok := want[sgn.Object.Name]
		if !ok {
			continue
		}
		seen++
		if got := w.Tips[sgn.ID]; got != tip {
			t.Errorf("signpost %s:\n got %+v\nwant %+v", sgn.Object.Name, got, tip)
		}
	}
	if seen != len(want) {
		t.Fatalf("found %d of the %d signposts", seen, len(want))
	}
}
