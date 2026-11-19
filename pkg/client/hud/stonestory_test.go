package hud

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheStoneSpeaksStoryInStoryMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	r := &fakeRunner{}
	h := New(nil, r.backend())
	h.OpenStone("kubectl get pods")
	got := string(h.Stone.Input)
	if lexicon.KubernetesWords.MatchString(got) || got == "" {
		t.Fatalf("story mode prefilled %q", got)
	}
	if p := StonePrompt(got); strings.Contains(p, "kubectl") {
		t.Fatalf("story prompt %q", p)
	}
	h.Update(Input{MX: -5, MY: -5, Keys: []ebiten.Key{ebiten.KeyEnter}}, fullFrame(t))
	if len(r.commands) == 0 || r.commands[len(r.commands)-1] != "get pods" {
		t.Fatalf("the story line ran as %q", r.commands)
	}
}

func TestTheStoneKeepsKubectlInKubernetesMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	h := New(nil, (&fakeRunner{}).backend())
	h.OpenStone("kubectl get pods")
	if got := string(h.Stone.Input); got != "get pods" {
		t.Fatalf("Kubernetes mode prefilled %q", got)
	}
}

func TestStoryModeStoneOutputSpeaksGameWords(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	lines := []StoneLine{{Text: "> get pods", Kind: LineEcho}}
	for _, out := range StoryConsole([]string{"NAME        READY   STATUS    RESTARTS", "pod/porter-1 is Pending on node lone-tower"}) {
		lines = append(lines, StoneLine{Text: out, Kind: LineOutput})
	}
	for _, l := range lines {
		if got := ConsoleText(l); lexicon.KubernetesWords.MatchString(got) {
			t.Errorf("story output %q keeps Kubernetes words", got)
		}
	}
}

const describeServiceK8s = `Name:              parcels
Namespace:         default
Labels:            <none>
Annotations:       <none>
Selector:          app=porters
Type:              ClusterIP
IP:                10.96.0.11
Port:              http  80/TCP
TargetPort:        8080/TCP
Endpoints:         10.244.0.5:8080,10.244.0.6:8080
Session Affinity:  None
Events:            <none>`

func valueColumn(line string) int {
	m := describeField.FindStringSubmatchIndex(line)
	if m == nil || m[8] < 0 {
		return -1
	}
	return len([]rune(line[:m[8]]))
}

func TestStoryDescribeOutputSpeaksGameWordsInOneColumn(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	out := StoryConsole(SplitOutput(describeServiceK8s))
	col := -1
	for _, l := range out {
		if lexicon.KubernetesWords.MatchString(l) || strings.Contains(l, "ClusterIP") || strings.Contains(l, "TCP") {
			t.Errorf("story describe keeps %q", l)
		}
		if c := valueColumn(l); c >= 0 {
			if col >= 0 && c != col {
				t.Errorf("value column %d, want %d: %q", c, col, l)
			}
			col = c
		}
	}
	if len(out) < 3 {
		t.Fatalf("story describe dropped too much: %q", out)
	}
	ui.SetNamer(nil)
	if got := StoryConsole(SplitOutput(describeServiceK8s)); len(got) != len(SplitOutput(describeServiceK8s)) {
		t.Fatal("outside story mode the output changed")
	}
}

func TestStoryTablesStayAligned(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	out := StoryConsole([]string{"NAME           READY   STATUS    RESTARTS   AGE", "porters-7d9f   1/1     Running   0          2m", "clerks-1       0/1     Pending   0          1m"})
	starts := func(l string) []int {
		idx := []int{}
		for _, m := range tableCells.FindAllStringIndex(l, -1) {
			idx = append(idx, len([]rune(l[:m[1]])))
		}
		return idx
	}
	first := starts(out[0])
	for _, l := range out[1:] {
		if fmt.Sprint(starts(l)) != fmt.Sprint(first) {
			t.Errorf("columns drift: %q vs %q", l, out[0])
		}
		if lexicon.KubernetesWords.MatchString(l) {
			t.Errorf("story table keeps %q", l)
		}
	}
}

func TestAStoryErrorKeepsItsOwnSpacing(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	got := StoryConsole([]string{`error: unknown command "foo" for "kubectl"`})
	if len(got) != 1 || strings.Contains(got[0], ":  ") || strings.Contains(got[0], "kubectl") {
		t.Fatalf("story error %q", got)
	}
}

func TestStoryChipsFitInStoryWords(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	for _, width := range []int{12, 20, 30, 60} {
		if got := ChipFit("kubectl scale deployment archive --replicas=2", width); lexicon.KubernetesWords.MatchString(got) || strings.Contains(got, "deploy") || len([]rune(got)) > width {
			t.Errorf("story chip at %d: %q", width, got)
		}
	}
}

func TestAButtonsEchoedAnswerSpeaksTheRealmsWords(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	h := New(nil, (&fakeRunner{}).backend())
	h.EchoAnswer("deployment.apps/porters image updated\n", true)
	for _, l := range h.Stone.Output {
		if strings.Contains(l.Text, "deployment") {
			t.Fatalf("the echoed answer reads %q", l.Text)
		}
	}
}

func TestStoryTablesKeepAgesAsTimesAndDropTheLawVersion(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	out := StoryConsole([]string{"NAME         STATUS   ROLES     AGE   VERSION", "lone-tower   Ready    Citadel   33m   v1.37.0"})
	if strings.Contains(strings.Join(out, "\n"), "v1.37") || strings.Contains(strings.Join(out, "\n"), "VERSION") {
		t.Errorf("story keeps the version column: %q", out)
	}
	if len(out) != 2 || !strings.Contains(out[1], "33min") || strings.Contains(out[1], "ration") {
		t.Errorf("story age reads %q", out)
	}
	tidings := StoryConsole([]string{"  Type     Reason                    Age   From      Message", "  Normal   NodeHasSufficientMemory   2m5s  kubelet   Node lone-tower status is now: NodeHasSufficientMemory"})
	if len(tidings) != 2 || !strings.Contains(tidings[1], "2min 5s") || !strings.Contains(tidings[1], "KeepHasRoom") {
		t.Errorf("story tidings read %q", tidings)
	}
}

func TestTheStoneGivesItsStorySummaryAndNeverAnswersWithSilence(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	run := func(cmd string) view.CommandResult {
		if strings.HasPrefix(cmd, "describe") {
			return view.CommandResult{OK: true, Output: "Name: lone-tower", Summary: "Lone Tower is hungry: 2,484 of 1,000 rations pledged."}
		}
		return view.CommandResult{OK: false, Output: "error: unknown field \"spec.containers[0].resources.requests.memory\""}
	}
	h := New(nil, Backend{Kubectl: run, Inspect: run})
	h.Stone.SetInput("describe keep lone-tower")
	h.Speak()
	text := ""
	for _, l := range h.Stone.Output {
		text += l.Text + "\n"
	}
	if !strings.Contains(text, "Lone Tower is hungry") {
		t.Errorf("the story summary is missing:\n%s", text)
	}
	h.Stone.SetInput("set resources crew/porters --pledge=housing=64")
	h.Speak()
	last := h.Stone.Output[len(h.Stone.Output)-1]
	if last.Text != stoneUnsaid {
		t.Errorf("a refused command answered %q", last.Text)
	}
}
