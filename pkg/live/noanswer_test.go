package live

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestAServerNameWithNoDNSRecordSaysWhatToCheck(t *testing.T) {
	lookup := &net.DNSError{Err: "no such host", Name: "staging-aks-1a2b3c.hcp.westeurope.azmk8s.io", IsNotFound: true}
	err := NoAnswer(&url.Error{Op: "Get", URL: "https://staging-aks-1a2b3c.hcp.westeurope.azmk8s.io:443/version", Err: &net.OpError{Op: "dial", Net: "tcp", Err: lookup}})
	for _, want := range []string{"no DNS record for staging-aks-1a2b3c.hcp.westeurope.azmk8s.io", "stopped, deleted or recreated", "fetch its kubeconfig again", "VPN"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the message lacks %q: %v", want, err)
		}
	}
	if !errors.Is(err, lookup) {
		t.Error("the lookup error is no longer wrapped")
	}
	if other := NoAnswer(fmt.Errorf("connection refused")); !strings.HasPrefix(other.Error(), "the cluster does not answer: ") {
		t.Errorf("any other failure reads %q", other)
	}
}

type lookupFails struct{ name string }

func (l lookupFails) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, &net.OpError{Op: "dial", Net: "tcp", Err: &net.DNSError{Err: "no such host", Name: l.name, IsNotFound: true}}
}

func TestAskingWhoYouAreOfAServerNameWithNoDNSRecordSaysWhatToCheck(t *testing.T) {
	host := "lab-west.example.internal"
	conn := &Conn{client: &http.Client{Transport: lookupFails{name: host}}, server: url.URL{Scheme: "https", Host: host + ":6443"}}
	who, err := whoOver(context.Background(), conn)
	if err == nil {
		t.Fatal("a lookup that fails answers who you are")
	}
	for _, text := range []string{err.Error(), who.Unknown} {
		for _, want := range []string{"no DNS record for " + host, "stopped, deleted or recreated", "VPN"} {
			if !strings.Contains(text, want) {
				t.Errorf("%q lacks %q", text, want)
			}
		}
	}
}
