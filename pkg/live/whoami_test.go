package live

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func fakeConn(t *testing.T, h http.HandlerFunc) *Conn {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := "apiVersion: v1\nkind: Config\nclusters:\n- name: c\n  cluster:\n    server: " + srv.URL + "\ncontexts:\n- name: x\n  context:\n    cluster: c\n    user: u\nusers:\n- name: u\n  user:\n    token: t\n"
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	conn, err := Dial([]string{path}, "x")
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func TestWhoAmIAsksTheFixedQuestionOnce(t *testing.T) {
	var seen []string
	conn := fakeConn(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen = append(seen, r.Method+" "+r.URL.Path+" "+r.Header.Get("Content-Type")+" "+string(body))
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"apiVersion":"authentication.k8s.io/v1","kind":"SelfSubjectReview","status":{"userInfo":{"username":"alice@example.com","uid":"u-1","groups":["viewers","system:authenticated"]}}}`)
	})
	who, err := conn.WhoAmI(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if who.Username != "alice@example.com" || who.UID != "u-1" || len(who.Groups) != 2 {
		t.Errorf("who %+v", who)
	}
	want := "POST " + selfReview + " application/json " + selfReviewBody
	if len(seen) != 1 || seen[0] != want {
		t.Errorf("the server saw %q, want only %q", seen, want)
	}
}

func TestWhoAmISaysWhyTheClusterDidNotAnswer(t *testing.T) {
	conn := fakeConn(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"kind":"Status","status":"Failure","reason":"NotFound","code":404,"message":"the server could not find the requested resource"}`)
	})
	who, err := conn.WhoAmI(context.Background())
	if err == nil || who.Username != "" || who.Unknown == "" {
		t.Errorf("who %+v, err %v", who, err)
	}
}
