package write

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestATokenRequestNeverLeavesAsADryRunBeforeConfirm(t *testing.T) {
	f := &fakeServer{}
	g := bound(t, Writes, f)
	o := Order{
		Request:     Request{Method: http.MethodPost, Path: "/api/v1/namespaces/shop/serviceaccounts/deployer/token", Body: []byte(`{"apiVersion":"authentication.k8s.io/v1","kind":"TokenRequest","spec":{"expirationSeconds":86400}}`)},
		ContentType: "application/json",
	}
	g.Preview(context.Background(), o)
	if lines := f.lines(); len(lines) != 0 {
		t.Errorf("at WRITES the CONFIRM's preview sent %v: the API server's TokenREST ignores dryRun, so a real ServiceAccount token was minted before CONFIRM", lines)
	}
	raw := o
	raw.Request.Query = url.Values{"dryRun": {"All"}}
	before := len(f.lines())
	g.Send(context.Background(), raw)
	if lines := f.lines(); len(lines) != before {
		t.Errorf("at WRITES a TokenRequest marked dryRun=All left with no ticket: %v", lines[before:])
	}
}
