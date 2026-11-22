package live

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/waxb/kubernites/pkg/live/write"
)

func userinfoKubeconfig(t *testing.T, path, server string) {
	t.Helper()
	text := "apiVersion: v1\nkind: Config\nclusters:\n- name: prod\n  cluster:\n    server: " + server + "\n" +
		"users:\n- name: prod\n  user:\n    token: laptop-token\n" +
		"contexts:\n- name: prod\n  context:\n    cluster: prod\n    user: prod\ncurrent-context: prod\n"
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestASchemelessServersUserinfoIsPartOfTheRememberedKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("Authorization"))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"gitVersion":"v1.37.0"}`))
	}))
	defer srv.Close()
	hostPort := strings.TrimPrefix(srv.URL, "http://")
	path := filepath.Join(t.TempDir(), "config.yaml")

	userinfoKubeconfig(t, path, "alice:alice-pw@"+hostPort)
	list, err := ListContexts([]string{path})
	if err != nil || len(list) != 1 {
		t.Fatalf("contexts %v %v", list, err)
	}
	picked := list[0].Remembered()

	userinfoKubeconfig(t, path, "root:root-pw@"+hostPort)
	again, err := ListContexts([]string{path})
	if err != nil || len(again) != 1 {
		t.Fatalf("contexts %v %v", again, err)
	}
	now := again[0].Remembered()

	gate, err := write.New(write.Choice{Context: picked, Level: write.Seamless, Classes: write.AllClasses})
	if err != nil {
		t.Fatal(err)
	}
	conn, dialErr := DialGated([]string{path}, picked.Name, gate)
	sent := ""
	if dialErr == nil {
		var v map[string]any
		if err := conn.getJSON(context.Background(), "/version", nil, &v); err != nil {
			t.Fatalf("read through the gate: %v", err)
		}
		mu.Lock()
		if len(seen) > 0 {
			if raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(seen[len(seen)-1], "Basic ")); err == nil {
				sent = string(raw)
			}
		}
		mu.Unlock()
	}

	mem := write.Memory{}
	mem.Keep(write.Remembered{Context: picked, Level: write.Seamless, Warned: write.Seamless, Classes: write.AllClasses})
	_, _, recalled := mem.Recall(now)

	if picked == now || recalled || dialErr == nil {
		t.Errorf("the kubeconfig's server changed from alice:alice-pw@%s to root:root-pw@%s (a schemeless server whose userinfo Go's client sends as Basic credentials in place of the token): remembered key unchanged %v (server %q, identity %s), DANGEROUSLY SEAMLESS recalled %v, DialGated for the choice made as alice succeeded %v and the API server was asked as %q",
			hostPort, hostPort, picked == now, now.Server, now.Identity[:12], recalled, dialErr == nil, sent)
	}
}
