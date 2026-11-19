package names

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/lexicon"
)

var shownTrees = []string{"../../view", "../../kubectl", "../hud", "../app", "../livescenes", "../world"}

func shownLiterals(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	fset := token.NewFileSet()
	for _, root := range shownTrees {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			file, perr := parser.ParseFile(fset, path, nil, 0)
			if perr != nil {
				return perr
			}
			ast.Inspect(file, func(n ast.Node) bool {
				if bridgeCall(n) {
					return false
				}
				if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					if s, uerr := strconv.Unquote(lit.Value); uerr == nil && strings.Contains(s, " ") {
						out[s] = fset.Position(lit.Pos()).String()
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func bridgeCall(n ast.Node) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Bridge" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "ui"
}

var (
	copula  = regexp.MustCompile(`(?i)^(.*?)\b(is|are)\s+(.*)$`)
	letters = regexp.MustCompile(`[A-Za-z][A-Za-z-]*`)
	article = map[string]bool{"a": true, "an": true, "the": true, "its": true, "own": true}
)

func restated(text string) bool {
	for _, clause := range regexp.MustCompile(`[;.:!?(),]`).Split(text, -1) {
		m := copula.FindStringSubmatch(clause)
		if m == nil {
			continue
		}
		before := letters.FindAllString(m[1], -1)
		if len(before) == 0 {
			continue
		}
		head := stem(before[len(before)-1])
		after := letters.FindAllString(m[3], -1)
		for i, taken := 0, 0; i < len(after) && taken < 3; i++ {
			if article[strings.ToLower(after[i])] {
				continue
			}
			taken++
			if stem(after[i]) == head {
				return true
			}
		}
	}
	return false
}

func stem(w string) string {
	return strings.TrimSuffix(strings.ToLower(w), "s")
}

func TestRestatedFindsNonsense(t *testing.T) {
	for text, want := range map[string]bool{
		"Requests are requests. Ships land them.":               true,
		"Across the sea. images are container images":           true,
		"THE CLUSTER IS A CLUSTER":                              true,
		"Every zone is its own zone.":                           true,
		"every request is worth sixteen times a cached request": false,
		"The probe is running at porter-a":                      false,
	} {
		if got := restated(text); got != want {
			t.Errorf("restated(%q) = %v", text, got)
		}
	}
}

func TestNoBridgeSentenceRestatesItself(t *testing.T) {
	name := Kubernetes(64)
	for lit, at := range shownLiterals(t) {
		if got := name(lit); restated(got) && !restated(lit) {
			t.Errorf("%s: %q reads %q with Kubernetes names", at, lit, got)
		}
	}
}

func TestEveryBridgeStillMatchesItsSource(t *testing.T) {
	literals := shownLiterals(t)
	for _, b := range bridges {
		found := false
		for lit := range literals {
			if strings.Contains(lit, b.story) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no shown text contains the bridge %q any more", b.story)
		}
	}
}

func TestBridgesSpeakKubernetes(t *testing.T) {
	for _, b := range bridges {
		if b.story == b.k8s || b.k8s == "" {
			t.Errorf("bridge %q has no Kubernetes form", b.story)
		}
		if got := lexicon.TrueNames(b.k8s); got != b.k8s {
			t.Errorf("bridge %q is not plain Kubernetes: %q", b.k8s, got)
		}
	}
}

func TestKubernetesNamesBridgeThenName(t *testing.T) {
	name := Kubernetes(8)
	got := name("A depot (a Service) sends each box to any ready hut.")
	if got != "A Service sends each request to any ready Pod." {
		t.Errorf("bridged = %q", got)
	}
	if got := name("Hut"); got != "Pod" {
		t.Errorf("named = %q", got)
	}
}

func TestTidingsReadAsKubernetesEvents(t *testing.T) {
	name := Kubernetes(16)
	for story, want := range map[string]string{
		"Hut porter-a is shutting down (Last Rites)":                               "Pod porter-a is Terminating",
		"Hut porter-a is Guttering: its worker stops and is relit again and again": "Pod porter-a is in CrashLoopBackOff, restarting",
		"3 porter huts are Burning: up and ready for boxes":                        "3 porter Pods are Running",
		"2 clerk huts wait for their trade to arrive from overseas":                "2 clerk Pods are pulling their image",
		"Keep stone-keep-1 has gone dark: it hosts no new huts":                    "Node stone-keep-1 is NotReady",
	} {
		if got := name(story); got != want {
			t.Errorf("%q reads %q, want %q", story, got, want)
		}
	}
}

func TestSummoningReadsAsKubectlRun(t *testing.T) {
	if got, want := Kubernetes(8)("Hire a porter"), "Run a porter Pod"; got != want {
		t.Errorf("%q reads %q, want %q", "Hire a porter", got, want)
	}
}

func TestTheForemanReadsAsAReplicaSetNotAPerson(t *testing.T) {
	name := Kubernetes(32)
	for _, story := range []string{
		"Press SHIFT: it adopts the bare porters its badge (a selector) matches, and raises new ones from its template until three stand.",
		"A ReplicaSet never replaces Pods that already run: give the shift a new class and only the huts it raises later get it. A crew adopts the shift as it stands, and nothing restarts.",
		"A node drain evicted porter-a. A bare hut (a Pod) has no keeper, so nothing raises another: raise a shift, and it keeps its count.",
		"Say which trade and how many huts: a shift takes in the bare ones wearing its badge and keeps that many standing.",
		"A shift keeps a count of huts of one trade. It takes in the bare huts its badge matches, and when one of its huts is razed it raises another from its template.",
		"A shift would give the new class only to the huts it raises later; a Deployment rolls it out hut by hut.",
		"The shift's template says porter:2.0 now, but not one of its 3 porters changed: a ReplicaSet replaces no Pod that already runs, so only the huts it raises from now on get it.",
		"Would have raised another for the deleted porter: a shift keeps its count through every eviction.",
		"Raises a shift (a ReplicaSet): it keeps that many huts of one trade. A new class reaches only the huts raised after it",
	} {
		got := " " + name(story) + " "
		for _, word := range []string{" he ", " He ", " his ", " him "} {
			if strings.Contains(got, word) {
				t.Errorf("%q reads %q with Kubernetes names", story, got)
				break
			}
		}
	}
}

func TestTheForemanButtonReadsAsReplicaSet(t *testing.T) {
	name := Kubernetes(8)
	for story, want := range map[string]string{
		"Next: press SHIFT": "Next: press REPLICASET", "Press SHIFT.": "Press REPLICASET.",
	} {
		if got := name(story); got != want {
			t.Errorf("%q reads %q, want %q", story, got, want)
		}
	}
}

func TestTheDeploymentRungReadsAsADeployment(t *testing.T) {
	story := "Raise a crew over the shift for porter:2.0"
	if got, want := Kubernetes(8)(story), "Raise a Deployment over the ReplicaSet for porter:2.0"; got != want {
		t.Errorf("%q reads %q, want %q", story, got, want)
	}
}
