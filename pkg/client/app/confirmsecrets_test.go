package app

import (
	"net/http"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestTheConfirmShowsASecretsFieldNamesOnlyAndNoBodyThatIsNotJSON(t *testing.T) {
	protobuf := []byte("k8s\x00\n\x0c\n\x02v1\x12\x06Secret\x12\x2a\n\x0a\n\x02s2\x12\x04shop\x12\x0e\n\x08password\x12\x07hunter2")
	secret := []byte(`{"apiVersion":"v1","kind":"Secret","metadata":{"name":"s2","namespace":"shop","annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"data\":{\"password\":\"aHVudGVyMg==\"}}"}},"data":{"password":"aHVudGVyMg=="},"stringData":{"token":"hunter2"},"type":"Opaque"}`)
	ask := func(method, path, contentType string, body []byte) live.StoneAsk {
		return live.StoneAsk{Why: write.NeedsConfirm, Command: "kubectl x", Preview: write.Answer{Code: 200},
			Order: write.Order{Request: write.Request{Method: method, Path: path, Body: body}, ContentType: contentType}}
	}
	secrets := []live.StoneAsk{
		ask(http.MethodPost, "/api/v1/namespaces/shop/secrets", "application/vnd.kubernetes.protobuf", protobuf),
		ask(http.MethodPost, "/api/v1/namespaces/shop/secrets", "application/json", secret),
		ask(http.MethodPatch, "/api/v1/namespaces/shop/secrets/s", "application/apply-patch+yaml", []byte("data:\n  password: aHVudGVyMg==\nstringData:\n  token: hunter2\n")),
		ask(http.MethodPatch, "/api/v1/namespaces/shop/secrets/s", "application/strategic-merge-patch+json", []byte(`{"data":{"password":"aHVudGVyMg=="}}`)),
		ask(http.MethodPut, "/api/v1/namespaces/shop/configmaps/c", "application/json", secret),
		ask(http.MethodPost, "/api/v1/namespaces/shop/configmaps", "application/cbor", []byte("\xa2dkindiConfigMapdnoteghunter2")),
	}
	defer ui.SetNamer(nil)
	for mode := range 2 {
		if mode == 0 {
			ui.SetStoryNamer(lexicon.StoryText)
		} else {
			ui.SetNamer(names.Kubernetes(64))
		}
		for _, a := range secrets {
			text := strings.Join(RenderAsk(a).Lines, "\n")
			for _, v := range []string{"hunter2", "aHVudGVyMg=="} {
				if strings.Contains(text, v) {
					t.Errorf("mode %d, %s %s (%s): the CONFIRM shows %s:\n%s", mode, a.Order.Request.Method, a.Order.Request.Path, a.Order.ContentType, v, text)
				}
			}
		}
		text := strings.Join(RenderAsk(secrets[1]).Lines, "\n")
		if mode == 1 && (!strings.Contains(text, "data.password") || !strings.Contains(text, "stringData.token")) {
			t.Errorf("the CONFIRM does not name the Secret's fields:\n%s", text)
		}
	}
	plain := ask(http.MethodPatch, "/apis/apps/v1/namespaces/shop/deployments/web", "application/strategic-merge-patch+json", []byte(`{"spec":{"replicas":3}}`))
	ui.SetNamer(names.Kubernetes(64))
	if text := strings.Join(RenderAsk(plain).Lines, "\n"); !strings.Contains(text, `Body: {"spec":{"replicas":3}}`) {
		t.Errorf("the CONFIRM no longer shows a JSON body:\n%s", text)
	}
}

func TestTheCpConfirmNamesTheLocalSideAndAnOverwriteIsAsked(t *testing.T) {
	opens := write.Opening{Channel: "exec", Kind: "Pod", Namespace: "shop", Name: "web-1", Container: "web", Command: []string{"tar", "cf", "-", "/app"}}
	from := live.StoneAsk{Why: write.NeedsConfirm, Connect: true, Command: "kubectl cp web-1:/app app -n shop", Opens: opens, Local: "/Users/me/Downloads/Kubernites/app", Writing: true}
	into := live.StoneAsk{Why: write.NeedsConfirm, Connect: true, Command: "kubectl cp ~/notes.txt web-1:/tmp -n shop", Opens: opens, Local: "/Users/me/notes.txt"}
	over := live.StoneAsk{Overwrite: "/Users/me/Downloads/Kubernites/app/a.txt", Command: "kubectl cp web-1:/app app -n shop", Local: "/Users/me/Downloads/Kubernites/app", Writing: true}
	defer ui.SetNamer(nil)
	for mode := range 2 {
		if mode == 0 {
			ui.SetStoryNamer(lexicon.StoryText)
		} else {
			ui.SetNamer(names.Kubernetes(64))
		}
		for _, c := range []struct {
			a    live.StoneAsk
			want string
		}{{from, "into /Users/me/Downloads/Kubernites/app"}, {into, "from /Users/me/notes.txt"}, {over, "/Users/me/Downloads/Kubernites/app/a.txt"}} {
			r := RenderAsk(c.a)
			text := r.Title + "\n" + strings.Join(r.Lines, "\n")
			if !r.Confirm || !strings.Contains(text, c.want) {
				t.Errorf("mode %d: %q does not say %q", mode, text, c.want)
			}
		}
		r := RenderAsk(over)
		if label, _ := r.GoLabel(); !strings.Contains(r.Title, "OVERWRITE") || !strings.Contains(label, "OVERWRITE") {
			t.Errorf("mode %d: the overwrite question reads %q, %q", mode, r.Title, label)
		}
	}
}

func TestTheStoryConfirmSaysWhatItChanges(t *testing.T) {
	ask := func(method, path string, body []byte) live.StoneAsk {
		return live.StoneAsk{Why: write.NeedsConfirm, Command: "kubectl delete pods --all -n prod", Preview: write.Answer{Code: 200},
			Order: write.Order{Request: write.Request{Method: method, Path: path, Body: body}, ContentType: "application/json"}}
	}
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	for _, c := range []struct {
		a    live.StoneAsk
		want string
	}{
		{ask(http.MethodDelete, "/api/v1/namespaces/prod/pods/web-1", nil), "remove hut prod/web-1"},
		{ask(http.MethodDelete, "/api/v1/namespaces/prod/pods/web-2", nil), "remove hut prod/web-2"},
		{ask(http.MethodPost, "/api/v1/namespaces/prod/pods/web-3/eviction", []byte(`{"metadata":{"name":"web-3"}}`)), "send away hut prod/web-3"},
		{ask(http.MethodPatch, "/apis/apps/v1/namespaces/shop/deployments/web/scale", []byte(`{"spec":{"replicas":3}}`)), "change crew shop/web"},
		{ask(http.MethodPost, "/apis/apps/v1/namespaces/shop/deployments", []byte(`{"metadata":{"name":"api"}}`)), "make crew shop/api"},
		{ask(http.MethodPatch, "/api/v1/namespaces/shop/configmaps/settings", []byte(`{"data":{"a":"b"}}`)), "change Scroll shop/settings"},
	} {
		text := strings.Join(RenderAsk(c.a).Lines, "\n")
		if !strings.Contains(text, c.want) {
			t.Errorf("the story CONFIRM of %s %s does not say %q:\n%s", c.a.Order.Request.Method, c.a.Order.Request.Path, c.want, text)
		}
	}
}

func TestALongDownloadAsksToCopyOnWithWhatItHasWritten(t *testing.T) {
	ask := RenderAsk(live.StoneAsk{More: "1.0 GiB in 312 files", Local: "/home/me/Downloads/Kubernites/logs", Writing: true})
	text := ask.Title + "\n" + strings.Join(ask.Lines, "\n")
	for _, want := range []string{"KEEP COPYING", "1.0 GiB in 312 files", "/home/me/Downloads/Kubernites/logs", "N stops"} {
		if !strings.Contains(text, want) {
			t.Errorf("the ask does not say %q:\n%s", want, text)
		}
	}
	if !ask.Confirm {
		t.Error("the ask takes no answer")
	}
}
