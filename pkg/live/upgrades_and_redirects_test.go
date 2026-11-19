package live

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestAWebsocketProtocolHeaderAloneIsRefusedAtReadOnly(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://h/apis/subresources.kubevirt.io/v1/namespaces/x/virtualmachineinstances/vm/console", nil)
	req.Header["Sec-WebSocket-Protocol"] = []string{"plain.kubevirt.io"}
	if err := Check(req, ""); err == nil {
		t.Fatal("a websocket upgrade marked only by Sec-WebSocket-Protocol passed the READ ONLY check")
	}
}

func TestARedirectNeverCarriesTheCredentialsToAnotherHost(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	record := func(who string, r *http.Request) {
		mu.Lock()
		seen = append(seen, who+" "+r.Host+r.URL.Path+" Authorization="+r.Header.Get("Authorization"))
		mu.Unlock()
	}
	elsewhere := func(w http.ResponseWriter, r *http.Request) {
		record("elsewhere", r)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"kind":"NamespaceList","apiVersion":"v1","items":[]}`)
	}
	plain := httptest.NewServer(http.HandlerFunc(elsewhere))
	t.Cleanup(plain.Close)
	tlsOther := httptest.NewTLSServer(http.HandlerFunc(elsewhere))
	t.Cleanup(tlsOther.Close)
	cluster := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record("cluster", r)
		target := plain.URL
		if r.URL.Path == "/apis/metrics.k8s.io/v1beta1/pods" {
			target = tlsOther.URL
		}
		http.Redirect(w, r, target+"/api/v1/namespaces", http.StatusFound)
	}))
	t.Cleanup(cluster.Close)
	ca := base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cluster.Certificate().Raw}))
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := fmt.Sprintf("apiVersion: v1\nkind: Config\nclusters:\n- name: c\n  cluster:\n    server: %s\n    certificate-authority-data: %s\ncontexts:\n- name: c\n  context:\n    cluster: c\n    user: u\nusers:\n- name: u\n  user:\n    token: not-a-real-token\n", cluster.URL, ca)
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	conn, err := Dial([]string{path}, "c")
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	err1 := conn.getJSON(context.Background(), "/apis/metrics.k8s.io/v1beta1/nodes", nil, &out)
	err2 := conn.getJSON(context.Background(), "/apis/metrics.k8s.io/v1beta1/pods", nil, &out)
	mu.Lock()
	defer mu.Unlock()
	for _, s := range seen {
		if strings.HasPrefix(s, "elsewhere ") {
			t.Errorf("a redirect reached another host: %s", s)
		}
	}
	if err1 == nil || err2 == nil {
		t.Errorf("a redirect to another host was followed: %v, %v", err1, err2)
	}
}
