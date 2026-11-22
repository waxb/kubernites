package write

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestATokenRequestAsksItsOwnWarningAfterTheWebhookNoPreviewWarningWasPassed(t *testing.T) {
	for _, level := range []Level{Writes, Seamless} {
		fresh := &fakeServer{}
		var held *Refusal
		first := bound(t, level, fresh).SendConfirmed(context.Background(), Order{
			Request:     Request{Method: http.MethodPost, Path: "/api/v1/namespaces/shop/serviceaccounts/deployer/token", Body: []byte(`{"apiVersion":"authentication.k8s.io/v1","kind":"TokenRequest"}`)},
			ContentType: "application/json",
		})
		if !errors.As(first.Err, &held) || held.Detail != MintsOnDryRun || len(fresh.lines()) != 0 {
			t.Fatalf("at %s a fresh context does not hold a token request for its warning: %+v", level, first)
		}
		f := &fakeServer{}
		g := bound(t, level, f, Unpreviewed)
		o := Order{
			Request:     Request{Method: http.MethodPost, Path: "/api/v1/namespaces/shop/serviceaccounts/deployer/token", Body: []byte(`{"apiVersion":"authentication.k8s.io/v1","kind":"TokenRequest","spec":{"expirationSeconds":86400}}`)},
			ContentType: "application/json",
		}
		ans := g.SendConfirmed(context.Background(), o)
		var refusal *Refusal
		if lines := f.lines(); len(lines) != 0 {
			t.Errorf("at %s, after only the webhook card order's no-preview warning was passed, a ServiceAccount token was minted with no warning of its own: %v", level, lines)
		}
		if !errors.As(ans.Err, &refusal) || refusal.Why != NeedsClass || refusal.Detail != MintsOnDryRun {
			t.Errorf("at %s the token request was not held for its own warning (mints a real token): %+v", level, ans)
		}
	}
}
