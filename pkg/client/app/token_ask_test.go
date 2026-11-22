package app

import (
	"net/http"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestATokenRequestsWarningAndConfirmSayItMintsARealTokenWithNoPreview(t *testing.T) {
	order := write.Order{Request: write.Request{Method: http.MethodPost, Path: "/api/v1/namespaces/shop/serviceaccounts/deployer/token", Body: []byte(`{"apiVersion":"authentication.k8s.io/v1","kind":"TokenRequest","spec":{}}`)}, ContentType: "application/json", Command: "kubectl create token deployer -n shop"}
	class := live.StoneAsk{Why: write.NeedsClass, Class: write.Tokens, Level: write.Writes, Detail: write.MintsOnDryRun, Command: order.Command}
	confirm := live.StoneAsk{Why: write.NeedsConfirm, Order: order, Unpreviewed: true, Detail: write.MintsOnDryRun, Command: order.Command}
	for _, story := range []bool{false, true} {
		if story {
			ui.SetStoryNamer(lexicon.StoryText)
		} else {
			ApplyNames(true)
		}
		w, c := RenderAsk(class), RenderAsk(confirm)
		warning, confirming := strings.Join(append([]string{w.Title}, w.Lines...), "\n"), strings.Join(append([]string{c.Title}, c.Lines...), "\n")
		for what, text := range map[string]string{"warning": warning, "CONFIRM": confirming} {
			if !strings.Contains(text, "token") || !strings.Contains(text, "until it expires") || strings.Contains(text, "webhook") || strings.Contains(text, "Chancery's preview answered") {
				t.Errorf("story %v: the %s does not say a real token is made with no preview, valid until it expires:\n%s", story, what, text)
			}
			if story {
				if leak := lexicon.KubernetesWords.FindString(text); leak != "" {
					t.Errorf("story mode says %q in the %s:\n%s", leak, what, text)
				}
			}
		}
		if !c.Confirm || strings.Contains(confirming, "Dry run (dryRun=All)") {
			t.Errorf("story %v: the CONFIRM shows a dry run or cannot be confirmed:\n%s", story, confirming)
		}
	}
	ui.SetNamer(nil)
}

func TestTheTokenWarningAndConfirmAreTrueAtEachLevelAndTheStoryConfirmNamesTheWritAndItsFief(t *testing.T) {
	order := write.Order{Request: write.Request{Method: http.MethodPost, Path: "/api/v1/namespaces/shop/serviceaccounts/deployer/token", Body: []byte(`{"apiVersion":"authentication.k8s.io/v1","kind":"TokenRequest","spec":{}}`)}, ContentType: "application/json", Command: "kubectl create token deployer -n shop"}
	defer ui.SetNamer(nil)
	for _, story := range []bool{false, true} {
		if story {
			ui.SetStoryNamer(lexicon.StoryText)
		} else {
			ApplyNames(true)
		}
		joined := func(a hud.StoneAsk) string { return strings.Join(append([]string{a.Title}, a.Lines...), "\n") }
		writes := joined(RenderAsk(live.StoneAsk{Why: write.NeedsClass, Class: write.Tokens, Level: write.Writes, Detail: write.MintsOnDryRun, Command: order.Command}))
		seamless := joined(RenderAsk(live.StoneAsk{Why: write.NeedsClass, Class: write.Tokens, Level: write.Seamless, Detail: write.MintsOnDryRun, Command: order.Command}))
		confirm := joined(RenderAsk(live.StoneAsk{Why: write.NeedsConfirm, Level: write.Writes, Order: order, Unpreviewed: true, Detail: write.MintsOnDryRun, Command: order.Command}))
		if !strings.Contains(writes, "After this warning a CONFIRM asks") || strings.Contains(writes, "now") {
			t.Errorf("story %v: at WRITES the token warning does not say a CONFIRM follows whose Y mints the token:\n%s", story, writes)
		}
		if !strings.Contains(seamless, "no CONFIRM follows") || strings.Contains(seamless, "a CONFIRM asks") {
			t.Errorf("story %v: at DANGEROUSLY SEAMLESS the token warning does not say its own Y mints the token:\n%s", story, seamless)
		}
		if story && !strings.Contains(strings.ToLower(confirm), "writ shop/deployer") {
			t.Errorf("the story CONFIRM names no writ and fief:\n%s", confirm)
		}
		for what, text := range map[string]string{"WRITES warning": writes, "SEAMLESS warning": seamless, "CONFIRM": confirm} {
			if story {
				if leak := lexicon.KubernetesWords.FindString(text); leak != "" {
					t.Errorf("story mode says %q in the %s:\n%s", leak, what, text)
				}
			}
			if strings.Contains(text, "webhook") || strings.Contains(strings.ToLower(text), "side effects") {
				t.Errorf("story %v: the %s speaks of the webhook class:\n%s", story, what, text)
			}
		}
	}
}
