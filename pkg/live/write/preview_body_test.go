package write

import (
	"context"
	"net/http"
	"testing"
)

func TestAPreviewOfABodyThatIsNotAnObjectNeverPanics(t *testing.T) {
	f := &fakeServer{}
	g := bound(t, Writes, f)
	o := Order{
		Request:     Request{Method: http.MethodDelete, Path: "/api/v1/namespaces/shop/pods/web-1", Body: []byte("null"), ContentType: "application/json"},
		ContentType: "application/json",
	}
	var crashed any
	func() {
		defer func() { crashed = recover() }()
		g.Preview(context.Background(), o)
	}()
	if crashed != nil {
		t.Errorf("the CONFIRM's preview of a DELETE whose body is JSON null (kubectl delete --raw PATH -f FILE holding null) panicked instead of answering: %v", crashed)
	}
}

func TestABodyThatIsNotAJSONObjectHasNoPreviewAndSaysWhy(t *testing.T) {
	for _, c := range []struct {
		method, contentType, body string
	}{
		{http.MethodDelete, "application/json", "null"},
		{http.MethodDelete, "application/json", "[1]"},
		{http.MethodDelete, "", `"x"`},
		{http.MethodPatch, "application/merge-patch+json", "null"},
		{http.MethodPatch, "application/strategic-merge-patch+json", "[]"},
		{http.MethodPost, "application/json", "7"},
		{http.MethodPut, "application/json; charset=utf-8", "true"},
	} {
		f := &fakeServer{}
		g := bound(t, Writes, f)
		o := Order{Request: Request{Method: c.method, Path: "/api/v1/namespaces/shop/configmaps/c", Body: []byte(c.body), ContentType: c.contentType}, ContentType: c.contentType}
		ans, _ := g.Preview(context.Background(), o)
		if ans.OK() || ans.Message != NotAnObject {
			t.Errorf("%s %s %s: the preview answered %+v, want no preview with its reason", c.method, c.contentType, c.body, ans)
		}
		if lines := f.lines(); len(lines) != 0 {
			t.Errorf("%s %s %s: sent %v", c.method, c.contentType, c.body, lines)
		}
	}
	f := &fakeServer{answer: func(r *http.Request, body string) (int, string) { return http.StatusOK, `{}` }}
	g := bound(t, Writes, f)
	patch := Order{Request: Request{Method: http.MethodPatch, Path: "/api/v1/namespaces/shop/configmaps/c", Body: []byte(`[{"op":"add","path":"/data/a","value":"b"}]`), ContentType: "application/json-patch+json"}, ContentType: "application/json-patch+json"}
	if ans, _ := g.Preview(context.Background(), patch); !ans.OK() || len(f.lines()) != 1 {
		t.Errorf("a JSON patch, whose body is an array, lost its preview: %+v %v", ans, f.lines())
	}
}
