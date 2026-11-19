package app

import (
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestEveryNewReachTextSpeaksNoKubernetesInStoryMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	var texts []string
	opens := []write.Opening{
		{Channel: "exec", Kind: "Pod", Namespace: "shop", Name: "web-1", Container: "app", Command: []string{"sh"}, Stdin: true, TTY: true},
		{Channel: "attach", Kind: "Pod", Namespace: "shop", Name: "web-1", Container: "app", Stdin: true, TTY: true},
		{Channel: "portforward", Kind: "Pod", Namespace: "shop", Name: "web-1"},
		{Channel: "proxy", Kind: "Service", Namespace: "shop", Name: "web:80", Path: "/"},
		{Channel: "proxy", Kind: "Node", Name: "keep-1", Path: "/configz"},
	}
	for _, o := range opens {
		a := RenderAsk(live.StoneAsk{Why: write.NeedsConfirm, Connect: true, Opens: o, Command: "kubectl port-forward web-1 8080:80"})
		texts = append(texts, a.Title)
		texts = append(texts, a.Lines...)
	}
	class := RenderAsk(live.StoneAsk{Why: write.NeedsClass, Class: write.Workloads})
	texts = append(append(texts, class.Title), class.Lines...)
	for _, ended := range []string{hud.SessionExited, hud.SessionLeft, hud.SessionClosed, hud.SessionFailed} {
		for _, code := range []int{0, 7} {
			line, _ := hud.SessionEndLine(hud.StoneSession{Kind: "Pod", Namespace: "shop", Name: "web-1", Container: "app", Opened: true, Ended: ended, Code: code, Err: `pods "web-1" is forbidden`})
			texts = append(texts, line)
		}
	}
	for _, ss := range []kubectltree.Session{{Ended: kubectltree.EndedExited, Code: 3}, {Ended: kubectltree.EndedLeft}, {Ended: kubectltree.EndedClosed}, {Ended: kubectltree.EndedFailed, Err: `pods "web-1" is forbidden: cannot create resource "pods/exec"`}} {
		texts = append(texts, sessionEnded(ss))
	}
	for _, verb := range []string{"exec", "attach", "debug", "run", "port-forward", "cp"} {
		texts = append(texts, sessionVerb(verb), hud.ReachAnswer(verb))
	}
	f := kubectltree.Forward{ID: 1, Target: "pod/web-1", Namespace: "shop", Pod: "web-1", Ports: []string{":80"}, Listening: []string{"127.0.0.1:41000 -> 80"}, Connections: 2}
	for _, how := range []kubectltree.Ended{kubectltree.EndedLeft, kubectltree.EndedClosed, kubectltree.EndedFailed} {
		f.Ended, f.Err = how, "error upgrading connection: pods is forbidden"
		texts = append(texts, forwardEndLine(f), forwardEnded(f))
	}
	s := &Session{forwards: []kubectltree.Forward{f}}
	for _, lf := range s.liveForwards() {
		texts = append(texts, lf.Text, lf.Tip)
	}
	at := time.Date(2026, 11, 18, 10, 4, 0, 0, time.UTC)
	for _, verb := range []string{"exec", "port-forward", "cp"} {
		l := ledgerLine{at: at, shownVerb: sessionVerb(verb), target: write.Ref{GVR: write.Pods, Kind: "Pod", Namespace: "shop", Name: "web-1"}, ended: "worker app, " + sessionEnded(kubectltree.Session{Ended: kubectltree.EndedFailed, Err: "pods forbidden"})}
		texts = append(texts, l.text())
	}
	p := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "web-1"}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}}}}
	for _, sp := range hutReachSpecs(p, "") {
		texts = append(texts, sp.label, sp.tip)
	}
	for _, text := range texts {
		if leak := lexicon.KubernetesWords.FindString(text); leak != "" {
			t.Errorf("story mode says %q in %q", leak, text)
		}
		if strings.Contains(text, "later round") {
			t.Errorf("story mode still says a later round: %q", text)
		}
	}
	if len(texts) < 60 {
		t.Fatalf("only %d texts were read", len(texts))
	}
}
