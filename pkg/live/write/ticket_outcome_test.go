package write

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func tickets(g *Gate) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.tickets)
}

func TestAConfirmedSendLeavesNoTicketWhateverItsOutcome(t *testing.T) {
	seen := 0
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"kind":"Status","status":"Success"}`))
	}))
	t.Cleanup(srv.Close)
	ca := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	g, err := New(Choice{Context: Context{Name: "prod", Server: srv.URL, User: "admin"}, Level: Writes})
	if err != nil {
		t.Fatal(err)
	}
	cfg := &rest.Config{Host: srv.URL, TLSClientConfig: rest.TLSClientConfig{CAData: ca}, ExecProvider: &clientcmdapi.ExecConfig{APIVersion: "client.authentication.k8s.io/v1", Command: "false", InteractiveMode: clientcmdapi.NeverExecInteractiveMode}}
	if _, err := g.Client(cfg); err != nil {
		t.Fatal(err)
	}
	o := Order{Request: Request{Method: http.MethodDelete, Path: "/api/v1/namespaces/shop/pods/web-1"}}
	if a := g.SendConfirmed(context.Background(), o); a.OK() || a.Err == nil {
		t.Fatalf("a send whose credential plugin fails answered %+v", a)
	}
	if seen != 0 {
		t.Fatalf("the server saw %d requests", seen)
	}
	if n := tickets(g); n != 0 {
		t.Fatalf("a send that failed before the gate left %d tickets: a later identical request would pass with no CONFIRM", n)
	}
}

func TestATicketHoldsOnlyForTheContentTypeItWasIssuedFor(t *testing.T) {
	f := &fakeServer{}
	g := bound(t, Writes, f)
	body := []byte(`{"spec":{"replicas":3}}`)
	issued := Order{Request: Request{Method: http.MethodPatch, Path: "/apis/apps/v1/namespaces/shop/deployments/web", Body: body}, ContentType: MergePatch}
	g.Issue(issued.request())
	sent := issued
	sent.ContentType = StrategicPatch
	if why := refusedWhy(t, g.Send(context.Background(), sent)); why != NeedsConfirm {
		t.Fatalf("a ticket issued for %s let the same body go as %s (%v): %v", MergePatch, StrategicPatch, why, f.lines())
	}
	if a := g.SendConfirmed(context.Background(), issued); !a.OK() {
		t.Fatalf("the confirmed send: %v", a.Text())
	}
	if n := tickets(g); n != 0 {
		t.Fatalf("%d tickets left", n)
	}
}
