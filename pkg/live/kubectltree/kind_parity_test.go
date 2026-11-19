package kubectltree_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
)

type parityRow struct {
	Command string
	Same    bool
	Objects string
	Note    string
}

type parityKind struct {
	t       *testing.T
	config  string
	context string
	bin     string
	home    string
}

func (p parityKind) binary(args ...string) (string, string, int) {
	p.t.Helper()
	cmd := exec.Command(p.bin, append([]string{"--kubeconfig", p.config, "--context", p.context}, args...)...)
	cmd.Env = []string{"HOME=" + p.home, "KUBERC=off", "PATH=" + os.Getenv("PATH")}
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		p.t.Fatalf("%v: %v", args, err)
	}
	return out.String(), errOut.String(), code
}

var (
	ageCell   = regexp.MustCompile(`\b\d+(?:\.\d+)?(?:s|m|h|d|y)(?:\d+(?:s|m|h|d))?\b`)
	longTime  = regexp.MustCompile(`\w{3}, \d{2} \w{3} \d{4} \d{2}:\d{2}:\d{2} [+-]\d{4}`)
	isoTime   = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z`)
	version   = regexp.MustCompile(`"?resourceVersion"?: ?"\d+"`)
	usage     = regexp.MustCompile(`\b\d+(?:m|Mi|Ki|%)`)
	spaces    = regexp.MustCompile(`[ \t]+`)
	restarted = regexp.MustCompile(`restartedAt"?: ?"?[^"\n]*"?`)
	podHash   = regexp.MustCompile(`web-[ab]-[a-z0-9]{8,10}-[a-z0-9]{5}`)
	rsHash    = regexp.MustCompile(`web-[ab]-[a-z0-9]{8,10}`)
	jobPod    = regexp.MustCompile(`from-[ab]-[a-z0-9]{5}`)
	hashLabel = regexp.MustCompile(`pod-template-hash[=:] ?"?[a-z0-9]{8,10}"?`)
	uid       = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	rawStderr = regexp.MustCompile(`(?m)^(?:Found \d+ pods, using pod/\S+|Defaulted container .*|Default container name .*)\n?`)
)

func normal(s string, pair bool) string {
	s = longTime.ReplaceAllString(s, "<time>")
	s = isoTime.ReplaceAllString(s, "<time>")
	s = version.ReplaceAllString(s, "resourceVersion: <rv>")
	s = restarted.ReplaceAllString(s, "restartedAt: <time>")
	if pair {
		s = uid.ReplaceAllString(s, "<uid>")
		s = podHash.ReplaceAllString(s, "web-X-<hash>")
		s = rsHash.ReplaceAllString(s, "web-X-<hash>")
		s = jobPod.ReplaceAllString(s, "from-X-<hash>")
		s = hashLabel.ReplaceAllString(s, "pod-template-hash=<hash>")
		s = strings.NewReplacer("web-a", "web-X", "web-b", "web-X", "extra-a", "extra-X", "extra-b", "extra-X", "settings-a", "settings-X", "settings-b", "settings-X", "from-a", "from-X", "from-b", "from-X", "apply-a", "apply-X", "apply-b", "apply-X").Replace(s)
	}
	s = ageCell.ReplaceAllString(s, "<age>")
	s = usage.ReplaceAllString(s, "<n>")
	return strings.TrimSpace(spaces.ReplaceAllString(s, " "))
}

func TestParityWithTheKubectlBinaryOnKind(t *testing.T) {
	cfg, bin := os.Getenv("KUBERNITES_STONE_KUBECONFIG"), os.Getenv("KUBERNITES_STONE_KUBECTL")
	if cfg == "" || bin == "" {
		t.Skip("KUBERNITES_STONE_KUBECONFIG and KUBERNITES_STONE_KUBECTL name a kind cluster this VM made and the kubectl binary of the same version")
	}
	raw, err := clientcmd.LoadFromFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	p := parityKind{t: t, config: cfg, context: raw.CurrentContext, bin: bin, home: t.TempDir()}
	p.setup()
	podOf := func(d string) string {
		out, _, _ := p.binary("get", "pods", "-n", "parity", "-l", "app="+d, "-o", "jsonpath={.items[0].metadata.name}")
		return out
	}
	worker := strings.TrimPrefix(func() string {
		out, _, _ := p.binary("get", "nodes", "-o", "name", "-l", "!node-role.kubernetes.io/control-plane")
		return strings.TrimSpace(out)
	}(), "node/")
	pod := podOf("web-a")
	waitFor(t, 120*time.Second, func() bool {
		out, _, code := p.binary("top", "pods", "-n", "parity")
		return code == 0 && strings.Contains(out, "web-a")
	})

	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{cfg}, Context: p.context, Wrap: live.ReadOnlyConfig})
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Close()
	var rows []parityRow
	for _, cmd := range []string{
		"get pods -n parity", "get pods -A", "get nodes", "get nodes -o wide", "get deployments -n parity -o wide", "get svc -n parity -o yaml",
		"get deployment web-a -n parity -o json", "get pods -n parity -o jsonpath={.items[*].spec.containers[0].image}", "get all -n parity",
		"describe deployment web-a -n parity", "describe pod " + pod + " -n parity", "describe node " + worker, "describe cronjob nightly -n parity",
		"logs " + pod + " -n parity --tail=2", "logs deployment/web-a -n parity --tail=1", "events -n parity", "get events -n parity",
		"top nodes", "top pods -n parity", "explain pods.spec.containers", "explain deployment.spec --recursive", "api-resources", "api-resources --namespaced=false -o name",
		"get pods -n parity --show-labels --sort-by=.metadata.name", "auth can-i list pods -n parity", "auth whoami",
	} {
		rows = append(rows, p.compare(tree, cmd, cmd, false))
	}

	path := cfg
	g, err := write.New(write.Choice{Context: write.Context{Name: p.context, Server: raw.Clusters[raw.Contexts[p.context].Cluster].Server, User: raw.Contexts[p.context].AuthInfo}, Level: write.Seamless, Classes: write.AllClasses})
	if err != nil {
		t.Fatal(err)
	}
	writer, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{path}, Context: p.context, Wrap: g.WrapConfig})
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	writes := [][2]string{
		{"scale deployment web-a -n parity --replicas=3", "deployment web-X"},
		{"label deployment web-a -n parity tier=front", "deployment web-X"},
		{"annotate deployment web-a -n parity note=parity", "deployment web-X"},
		{"apply -f testdata/parity/apply-a.yaml", "configmap settings-X"},
		{"patch deployment web-a -n parity -p {\"spec\":{\"minReadySeconds\":2}}", "deployment web-X"},
		{"rollout restart deployment web-a -n parity", "deployment web-X"},
		{"rollout undo deployment web-a -n parity", "deployment web-X"},
		{"delete configmap extra-a -n parity", ""},
		{"create job from-a --from=cronjob/nightly -n parity", "job from-X"},
	}
	for _, w := range writes {
		if strings.HasPrefix(w[0], "rollout undo") {
			for _, d := range []string{"web-a", "web-b"} {
				p.binary("rollout", "status", "deployment/"+d, "-n", "parity", "--timeout=120s")
			}
		}
		row := p.compare(writer, w[0], strings.NewReplacer("web-a", "web-b", "extra-a", "extra-b", "apply-a", "apply-b", "from-a", "from-b").Replace(w[0]), true)
		if w[1] != "" {
			row.Objects = p.sameObjects(strings.Replace(w[1], "-X", "-a", 1), strings.Replace(w[1], "-X", "-b", 1))
		} else {
			_, _, ca := p.binary("get", "configmap", "extra-a", "-n", "parity")
			_, _, cb := p.binary("get", "configmap", "extra-b", "-n", "parity")
			row.Objects = fmt.Sprintf("both gone: %v", ca != 0 && cb != 0)
		}
		rows = append(rows, row)
	}
	var table strings.Builder
	table.WriteString("| command | same output | same objects | note |\n|---|---|---|---|\n")
	differ := 0
	for _, r := range rows {
		fmt.Fprintf(&table, "| `kubectl %s` | %v | %s | %s |\n", r.Command, r.Same, r.Objects, r.Note)
		if !r.Same {
			differ++
		}
	}
	t.Logf("parity on kind (%s):\n%s", p.context, table.String())
	if out := os.Getenv("KUBERNITES_STONE_PARITY_OUT"); out != "" {
		os.WriteFile(out, []byte(table.String()), 0o644)
	}
	if differ > 0 {
		t.Errorf("%d of %d commands differ from the binary", differ, len(rows))
	}
}

func (p parityKind) setup() {
	t := p.t
	t.Helper()
	if v, _, _ := p.binary("version", "--client"); !strings.Contains(v, kubectltree.ClientVersion()) {
		t.Fatalf("the binary is not kubectl %s:\n%s", kubectltree.ClientVersion(), v)
	}
	p.binary("delete", "namespace", "parity", "--ignore-not-found", "--wait=true")
	files, _ := filepath.Glob("testdata/parity/[0-9]*.yaml")
	for _, f := range files {
		if out, errOut, code := p.binary("apply", "-f", f); code != 0 {
			t.Fatalf("%s: %s%s", f, out, errOut)
		}
	}
	for _, d := range []string{"web-a", "web-b"} {
		if out, errOut, code := p.binary("rollout", "status", "deployment/"+d, "-n", "parity", "--timeout=180s"); code != 0 {
			t.Fatalf("%s did not roll out: %s%s", d, out, errOut)
		}
	}
	time.Sleep(12 * time.Second)
}

func (p parityKind) compare(tree *kubectltree.Tree, mine, theirs string, pair bool) parityRow {
	p.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	args, err := kubectltree.Words(theirs)
	if err != nil {
		p.t.Fatal(err)
	}
	res := tree.Run(ctx, mine)
	out, errOut, code := p.binary(args...)
	row := parityRow{Command: mine, Same: normal(res.Stdout, pair) == normal(out, pair) && normal(res.Stderr, pair) == normal(errOut, pair) && (res.Code == 0) == (code == 0)}
	if !row.Same && rawStderr.MatchString(errOut) && normal(res.Stderr, pair) == normal(rawStderr.ReplaceAllString(errOut, ""), pair) && normal(res.Stdout, pair) == normal(out, pair) {
		row.Same, row.Note = true, "kubectl writes its "+strings.TrimSpace(rawStderr.FindString(errOut))+" hint to os.Stderr, outside its IOStreams (polymorphichelpers/logsforobject.go), so the Stone does not show it"
		return row
	}
	if !row.Same {
		row.Note = fmt.Sprintf("exit %d vs %d", res.Code, code)
		p.t.Logf("%s differs:\n--- tree (%d)\n%s%s\n--- binary (%d)\n%s%s", mine, res.Code, res.Stdout, res.Stderr, code, out, errOut)
	}
	return row
}

func (p parityKind) sameObjects(a, b string) string {
	p.t.Helper()
	ka, kb := strings.Fields(a), strings.Fields(b)
	oa, _, _ := p.binary("get", ka[0], ka[1], "-n", "parity", "-o", "json", "--show-managed-fields")
	ob, _, _ := p.binary("get", kb[0], kb[1], "-n", "parity", "-o", "json", "--show-managed-fields")
	sa, ma := objectShape(oa)
	sb, mb := objectShape(ob)
	same := normal(sa, true) == normal(sb, true)
	note := fmt.Sprintf("%v", same)
	if ma != mb {
		note += fmt.Sprintf(" (managers %s vs %s)", ma, mb)
	}
	if !same {
		p.t.Logf("%s and %s differ:\n%s\n%s", a, b, sa, sb)
	}
	return note
}

func objectShape(text string) (string, string) {
	var obj map[string]any
	if json.Unmarshal([]byte(text), &obj) != nil {
		return text, ""
	}
	meta, _ := obj["metadata"].(map[string]any)
	var managers []string
	if mf, ok := meta["managedFields"].([]any); ok {
		for _, m := range mf {
			e := m.(map[string]any)
			managers = append(managers, fmt.Sprintf("%v/%v", e["manager"], e["operation"]))
		}
	}
	sort.Strings(managers)
	for _, k := range []string{"uid", "resourceVersion", "creationTimestamp", "managedFields", "generation"} {
		delete(meta, k)
	}
	delete(obj, "status")
	data, _ := json.MarshalIndent(obj, "", " ")
	return string(data), strings.Join(managers, ",")
}

func waitFor(t *testing.T, limit time.Duration, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(limit)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(3 * time.Second)
	}
}

func TestRecordTheStoneScenesOnKind(t *testing.T) {
	cfg, bin, dir := os.Getenv("KUBERNITES_STONE_KUBECONFIG"), os.Getenv("KUBERNITES_STONE_KUBECTL"), os.Getenv("KUBERNITES_STONE_RECORD")
	if cfg == "" || bin == "" || dir == "" {
		t.Skip("KUBERNITES_STONE_RECORD names the directory for the Stone's capture fixtures, recorded on a kind cluster this VM made")
	}
	raw, err := clientcmd.LoadFromFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	p := parityKind{t: t, config: cfg, context: raw.CurrentContext, bin: bin, home: t.TempDir()}
	p.setup()
	p.binary("scale", "deployment", "web-b", "-n", "parity", "--replicas=3")
	waitFor(t, 120*time.Second, func() bool {
		out, _, code := p.binary("top", "pods", "-n", "parity")
		return code == 0 && strings.Contains(out, "web-a")
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	realm, err := live.Connect(ctx, []string{cfg}, p.context, time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := json.MarshalIndent(realm.Snapshot().WithoutProxyEnvironment(), "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stone-kind-snapshot.json"), snap, 0o644); err != nil {
		t.Fatal(err)
	}
	rec := &livetest.Recording{Context: p.context}
	recorder := func(c *rest.Config) *rest.Config {
		c = live.ReadOnlyConfig(c)
		c.Wrap(func(rt http.RoundTripper) http.RoundTripper { return &livetest.Recorder{Next: rt, Rec: rec} })
		return c
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{cfg}, Context: p.context, Wrap: recorder})
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Close()
	var edit *kubectltree.Edit
	for _, cmd := range []string{"get pods -A", "get deployments -n parity -o wide", "get nodes", "edit deployment web-b -n parity"} {
		res := tree.Run(ctx, cmd)
		if res.Code != 0 {
			t.Fatalf("%s: %s", cmd, res.Output)
		}
		if res.Edit != nil {
			edit = res.Edit
		}
	}
	server := raw.Clusters[raw.Contexts[p.context].Cluster].Server
	seamless, err := write.New(write.Choice{Context: write.Context{Name: p.context, Server: server}, Level: write.Seamless})
	if err != nil {
		t.Fatal(err)
	}
	sender, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{cfg}, Context: p.context, Wrap: seamless.WrapConfig})
	if err != nil {
		t.Fatal(err)
	}
	defer sender.Close()
	a := sender.Apply(ctx, strings.Replace(edit.Text, "replicas: 3", "replicas: 5", 1), false)
	if len(a.Conflicts) == 0 {
		t.Fatalf("the edit raised no conflict:\n%s", a.Output)
	}
	for _, c := range a.Conflicts {
		rec.Conflicts = append(rec.Conflicts, livetest.Conflict{Manager: c.Manager, Subresource: c.Subresource, Fields: c.Fields})
	}
	writes, err := write.New(write.Choice{Context: write.Context{Name: p.context, Server: server}, Level: write.Writes})
	if err != nil {
		t.Fatal(err)
	}
	rc, err := clientcmd.NewNonInteractiveClientConfig(*raw, p.context, &clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writes.Client(rc); err != nil {
		t.Fatal(err)
	}
	asker, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{cfg}, Context: p.context, Wrap: writes.WrapConfig})
	if err != nil {
		t.Fatal(err)
	}
	defer asker.Close()
	asker.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, writes, server, h, func(s live.StoneAsk) bool {
			r := s.Order.Request
			rec.Ask = &livetest.Ask{Command: s.Command, Method: r.Method, Path: r.Path, Query: r.Query.Encode(), Body: string(r.Body), ContentType: s.Order.ContentType, Code: s.Preview.Code, Message: s.Preview.Text()}
			return false
		})
	})
	asker.Run(ctx, "scale deployment web-b -n parity --replicas=4")
	if rec.Ask == nil || rec.Ask.Code != http.StatusOK {
		t.Fatalf("the dry run was not recorded: %+v", rec.Ask)
	}
	if out, _, _ := p.binary("get", "deployment", "web-b", "-n", "parity", "-o", "jsonpath={.spec.replicas}"); out != "3" {
		t.Fatalf("the cancelled scale changed web-b to %s", out)
	}
	data, err := json.MarshalIndent(rec, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stone-kind-replay.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("recorded %d exchanges, %d conflicts, the ask %s %s", len(rec.Exchanges), len(rec.Conflicts), rec.Ask.Method, rec.Ask.Path)
}
