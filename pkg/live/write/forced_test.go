package write

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"testing"

	"k8s.io/client-go/rest"
)

func TestAZeroGracePeriodInAnySpellingIsAForcedDelete(t *testing.T) {
	var mu sync.Mutex
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body)
		mu.Lock()
		got = append(got, r.Method+" "+r.URL.RequestURI())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Success"}`))
	}))
	t.Cleanup(srv.Close)
	g, err := New(Choice{Context: Context{Name: "c", Server: srv.URL, User: "u"}, Level: Seamless})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Client(&rest.Config{Host: srv.URL}); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/namespaces/shop/pods/web-1"
	for _, v := range []string{"0", "00", "-0", "+0", "000"} {
		r := Request{Method: http.MethodDelete, Path: path, Query: url.Values{"gracePeriodSeconds": {v}}}
		if !slices.Contains(ClassesOf(r), Destroy) {
			t.Errorf("gracePeriodSeconds=%q is not counted as destroying: %v", v, ClassesOf(r))
		}
		ans := g.Send(context.Background(), Order{Request: r})
		var ref *Refusal
		if !errors.As(ans.Err, &ref) || ref.Why != NeedsClass || ref.Class != Destroy {
			t.Errorf("gracePeriodSeconds=%q at DANGEROUSLY SEAMLESS went without the destroying warning: %v", v, ans.Err)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 0 {
		t.Fatalf("forced deletes reached the server without their warning: %v", got)
	}
}
