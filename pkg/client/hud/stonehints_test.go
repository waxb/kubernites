package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func okRes(out string) view.CommandResult { return view.CommandResult{Output: out, OK: true} }

func TestPlainGetPodsPointsAtKubeSystemUntilSeen(t *testing.T) {
	hints, seen := ConsoleHints("get pods", okRes("NAME AGE\nporter-a 1s\n"), spacedVocabulary, false)
	if !slices.Contains(hints, systemPodsHint) || seen {
		t.Fatalf("first plain listing = %q seen %v", hints, seen)
	}
	for _, cmd := range []string{"get pods -n kube-system", "kubectl get po --namespace=kube-system", "get pods -A", "-n kube-system get pods", "get pod --all-namespaces"} {
		hints, seen := ConsoleHints(cmd, okRes("NAME\netcd-lone-tower\n"), spacedVocabulary, false)
		if len(hints) != 0 || !seen {
			t.Errorf("%q = %q seen %v", cmd, hints, seen)
		}
	}
	if hints, _ := ConsoleHints("get pods", okRes("x\n"), spacedVocabulary, true); len(hints) != 0 {
		t.Errorf("seen stops the hint: %q", hints)
	}
	if hints, _ := ConsoleHints("get pods porter-a", okRes("x\n"), spacedVocabulary, false); len(hints) != 0 {
		t.Errorf("a named Pod needs no hint: %q", hints)
	}
	if hints, _ := ConsoleHints("get nodes", okRes("x\n"), spacedVocabulary, false); len(hints) != 0 {
		t.Errorf("nodes need no hint: %q", hints)
	}
}

func TestUnknownNamespaceNamesTheRealOnes(t *testing.T) {
	hints, _ := ConsoleHints("get pods -n system", okRes("No resources found in system namespace.\n"), spacedVocabulary, false)
	if want := `No namespace "system": did you mean -n kube-system? (get namespaces lists them all)`; !slices.Contains(hints, want) {
		t.Fatalf("hints = %q", hints)
	}
	hints, _ = ConsoleHints("get pods -n tower", okRes("No resources found in tower namespace.\n"), spacedVocabulary, true)
	if want := `No namespace "tower". Namespaces: default, kube-node-lease, kube-public, kube-system`; !slices.Contains(hints, want) {
		t.Fatalf("hints = %q", hints)
	}
	if hints, _ := ConsoleHints("get svc -n kube-system", okRes("No resources found in kube-system namespace.\n"), spacedVocabulary, true); len(hints) != 0 {
		t.Errorf("a real namespace that is empty needs no hint: %q", hints)
	}
	if hints, _ := ConsoleHints("get pods -n x", okRes("No resources found in x namespace.\n"), Vocabulary{}, true); len(hints) != 0 {
		t.Errorf("no namespace list, no hint: %q", hints)
	}
}

func TestNotFoundPointsAtTheRightNamespace(t *testing.T) {
	failed := view.CommandResult{Output: `Error from server (NotFound): pods "etcd-lone-tower" not found` + "\n"}
	hints, _ := ConsoleHints("describe pod etcd-lone-tower", failed, spacedVocabulary, true)
	if !slices.Contains(hints, "etcd-lone-tower lives in the kube-system namespace: add -n kube-system") {
		t.Fatalf("hints = %q", hints)
	}
	ghost := view.CommandResult{Output: `Error from server (NotFound): pods "ghost" not found` + "\n"}
	if hints, _ := ConsoleHints("logs ghost", ghost, spacedVocabulary, true); len(hints) != 0 {
		t.Errorf("an unknown Pod gets no hint: %q", hints)
	}
}

func TestConsoleTextKeepsKubectlTablesLiteral(t *testing.T) {
	ui.SetNamer(strings.NewReplacer("AGE", "PROMOTE", "parcels", "web requests", "Codex", "etcd", "Huts", "Pods", "huts", "Pods").Replace)
	t.Cleanup(func() { ui.SetNamer(nil) })
	cases := []struct {
		line StoneLine
		want string
	}{
		{StoneLine{"NAME     READY   AGE", LineOutput}, "NAME     READY   AGE"},
		{StoneLine{"  Normal   Started   5s   kubelet", LineOutput}, "  Normal   Started   5s   kubelet"},
		{StoneLine{"2026-09-01 The Codex awakens", LineOutput}, "2026-09-01 The etcd awakens"},
		{StoneLine{`Error from server (NotFound): services "parcels" not found`, LineError}, `Error from server (NotFound): services "parcels" not found`},
		{StoneLine{"> kubectl get pods AGE", LineEcho}, "> kubectl get pods AGE"},
		{StoneLine{"Huts live here", LineHint}, "Pods live here"},
		{StoneLine{"   A Huts paragraph", LineOutput}, "   A Pods paragraph"},
	}
	for _, c := range cases {
		if got := ConsoleText(c.line); got != c.want {
			t.Errorf("ConsoleText(%q) = %q, want %q", c.line.Text, got, c.want)
		}
	}
}

func TestSpeakAppendsHintsAndDraws(t *testing.T) {
	fr := &fakeRunner{}
	h := New(nil, fr.backend())
	f := fullFrame(t)
	f.Vocabulary = spacedVocabulary
	h.OpenStone("")
	h.Update(Input{MX: -5, MY: -5, Chars: []rune("get pods")}, f)
	h.Update(Input{MX: -5, MY: -5, Keys: []ebiten.Key{ebiten.KeyEnter}}, f)
	lines := h.StoneLines()
	if last := lines[len(lines)-1]; last.Kind != LineHint || last.Text != systemPodsHint {
		t.Fatalf("last line = %+v", last)
	}
	h.Update(Input{MX: -5, MY: -5, Chars: []rune("get pods -A")}, f)
	h.Update(Input{MX: -5, MY: -5, Keys: []ebiten.Key{ebiten.KeyEnter}}, f)
	h.Update(Input{MX: -5, MY: -5, Chars: []rune("get pods")}, f)
	h.Update(Input{MX: -5, MY: -5, Keys: []ebiten.Key{ebiten.KeyEnter}}, f)
	if last := h.StoneLines()[len(h.StoneLines())-1]; last.Kind == LineHint {
		t.Fatalf("hint repeats after kube-system was listed: %+v", last)
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
	h.drawStoneSummary(dst, InfoWell)
}

func TestNamesModeNeverRewritesAnObjectsNameInKubectlOutput(t *testing.T) {
	ui.SetNamer(strings.NewReplacer("parcels", "web requests", "porters", "porter Pods").Replace)
	t.Cleanup(func() { ui.SetNamer(nil) })
	for _, line := range []string{"  name: parcels", "metadata:", "    app: porters", "  - name: http", "Name: parcels"} {
		if got := ConsoleText(StoneLine{line, LineOutput}); got != line {
			t.Errorf("kubectl output %q reads %q", line, got)
		}
	}
}

func TestTheFirstWritErrorSaysWhoWritesTheDefaultWrit(t *testing.T) {
	refused := view.CommandResult{Output: `Error from server (Forbidden): pods "hut-one" is forbidden: error looking up service account default/default: serviceaccount "default" not found` + "\n"}
	hints, _ := ConsoleHints("run hut-one --image=porter:1.0", refused, spacedVocabulary, false)
	if len(hints) != 1 || !strings.Contains(hints[0], "Stewards' Hall") || !strings.Contains(hints[0], "default writ") {
		t.Fatalf("story 02:00: 'writ \"default\" not found' with no word of who writes it; hints %q", hints)
	}
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	if got := ConsoleText(StoneLine{hints[0], LineHint}); !strings.Contains(got, "kube-controller-manager") || !strings.Contains(got, "ServiceAccount") {
		t.Errorf("NAMES mode hint reads %q", got)
	}
}
