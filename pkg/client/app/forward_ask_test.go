package app

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestForwardsAskAtSeamlessNamesThePodItsPortsAndTheStoneCommand(t *testing.T) {
	addr, _, _ := shellFront(t)
	rig := connectStoneTo(t, hud.LevelSeamless, addr)
	s := rig.s
	sel, _ := hutOf(t, rig)
	rig.select_(sel)
	rig.do(hud.Intent{Kind: hud.IntentLiveOrder, Target: "port-forward"})
	form, open := s.Hud.OrderOpen()
	if !open {
		t.Fatal("FORWARD at DANGEROUSLY SEAMLESS did not ask")
	}
	text := strings.Join(form.Lines, "\n")
	if form.Title != "FORWARD" || form.Object == "" || !strings.Contains(text, "kubectl port-forward") || !strings.Contains(text, "127.0.0.1") {
		t.Errorf("FORWARD asks with title %q, object %q and lines:\n%s", form.Title, form.Object, text)
	}
	p := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "web-1"}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Ports: []corev1.ContainerPort{{ContainerPort: 8080}, {ContainerPort: 9090}}}}}}
	defer ui.SetNamer(nil)
	for _, story := range []bool{false, true} {
		if story {
			ui.SetStoryNamer(lexicon.StoryText)
		} else {
			ApplyNames(true)
		}
		for _, sp := range hutReachSpecs(p, "") {
			if sp.key != "port-forward" {
				continue
			}
			lines := strings.Join(sp.lines, "\n")
			if sp.title != "FORWARD" || !strings.Contains(sp.object, "web-1") || !strings.Contains(lines, "8080") || !strings.Contains(lines, "9090") {
				t.Errorf("story %v: FORWARD's ask: title %q object %q lines:\n%s", story, sp.title, sp.object, lines)
			}
			if !story && !strings.Contains(lines, sp.stone) {
				t.Errorf("FORWARD's ask does not show its Stone command %q:\n%s", sp.stone, lines)
			}
			if story {
				if leak := lexicon.KubernetesWords.FindString(lines + sp.object); leak != "" {
					t.Errorf("story mode says %q in FORWARD's ask:\n%s", leak, lines)
				}
			}
		}
	}
}
