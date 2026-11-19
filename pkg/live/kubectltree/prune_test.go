package kubectltree_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
)

func marker(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	mark := filepath.Join(dir, "ran")
	script := "#!/bin/sh\necho \"$0 $*\" >> " + mark + "\n"
	for _, name := range []string{"kubectl-foo", "kubectl-hello", "kubectl_complete-foo", "vi", "vim", "nano", "notepad", "diff", "git", "helm", "editor", "sh-editor"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir, mark
}

func noMarker(t *testing.T, mark string) {
	t.Helper()
	if b, err := os.ReadFile(mark); err == nil {
		t.Fatalf("an outside program ran:\n%s", b)
	}
}

func TestPluginsAreOff(t *testing.T) {
	f := newFake(t)
	bin, mark := marker(t)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	tree := newTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	for _, line := range []string{"foo", "hello world", "plugin list", "plugin", "__complete foo ''", "__completeNoDesc foo ''", "__complete get ''", "completion bash", "help", "--help", "get --help"} {
		res := run(t, tree, line)
		if line == "plugin list" && !strings.Contains(res.Output, kubectltree.ReasonPlugins) {
			t.Errorf("%s answered:\n%s", line, res.Output)
		}
		if strings.HasPrefix(line, "__") && res.Refused == "" {
			t.Errorf("%s was not refused:\n%s", line, res.Output)
		}
		if line == "completion bash" && !strings.Contains(res.Output, kubectltree.ReasonCompletion) {
			t.Errorf("%s answered:\n%s", line, res.Output)
		}
		if line == "foo" && !strings.Contains(res.Output, `unknown command "foo"`) {
			t.Errorf("%s answered:\n%s", line, res.Output)
		}
	}
	noMarker(t, mark)
}

func TestEditOpensTheScriptoriumAndNoEditor(t *testing.T) {
	f := newFake(t)
	bin, mark := marker(t)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("KUBE_EDITOR", filepath.Join(bin, "editor"))
	t.Setenv("EDITOR", filepath.Join(bin, "editor"))
	t.Setenv("VISUAL", filepath.Join(bin, "editor"))
	t.Setenv("SHELL", filepath.Join(bin, "sh-editor"))
	tree := newTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	res := run(t, tree, "edit deployment web")
	if res.Code != 0 || res.Edit == nil {
		t.Fatalf("edit did not open the Scriptorium (%d):\n%s", res.Code, res.Output)
	}
	e := res.Edit
	if e.Kind != "Deployment" || e.Name != "web" || e.Namespace != "shop" || e.Resource != "deployments" || e.Group != "apps" || e.ResourceVersion != "41" || e.Format != "yaml" {
		t.Errorf("the Scriptorium got %+v", e)
	}
	if !strings.Contains(e.Text, "replicas: 2") || strings.Contains(e.Text, "managedFields") || !strings.Contains(e.Text, "kind: Deployment") {
		t.Errorf("the live YAML:\n%s", e.Text)
	}
	if res := run(t, tree, "-n shop edit deployment/web -o json --show-managed-fields"); res.Edit == nil || !strings.Contains(res.Edit.Text, `"managedFields"`) || res.Edit.Format != "json" {
		t.Errorf("edit -o json --show-managed-fields: %+v\n%s", res.Edit, res.Output)
	}
	if res := run(t, tree, "edit deployments"); res.Edit != nil || !strings.Contains(res.Output, "one object at a time") {
		t.Errorf("edit of a list: %+v\n%s", res.Edit, res.Output)
	}
	for _, line := range []string{"edit deployment web --field-manager=other", "edit deployment web --output-patch", "edit deployment web --save-config"} {
		if res := run(t, tree, line); res.Edit != nil || !strings.Contains(res.Output, kubectltree.ReasonManager) {
			t.Errorf("%s: %+v\n%s", line, res.Edit, res.Output)
		}
	}
	manifest := filepath.Join(t.TempDir(), "cm.yaml")
	os.WriteFile(manifest, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: c\n"), 0o600)
	for _, line := range []string{"apply edit-last-applied deployment web", "create --edit -f " + manifest} {
		if res := run(t, tree, line); !strings.Contains(res.Output, kubectltree.ReasonEditor) {
			t.Errorf("%s answered:\n%s", line, res.Output)
		}
	}
	noMarker(t, mark)
}

func TestKubercIsOff(t *testing.T) {
	f := newFake(t)
	rc := filepath.Join(home, ".kube", "kuberc")
	os.MkdirAll(filepath.Dir(rc), 0o700)
	t.Cleanup(func() { os.RemoveAll(filepath.Join(home, ".kube")) })
	os.WriteFile(rc, []byte("apiVersion: kubectl.config.k8s.io/v1beta1\nkind: Preference\naliases:\n- name: getn\n  command: get\n  appendArgs:\n  - namespaces\ndefaults:\n- command: get\n  options:\n  - name: output\n    default: json\n"), 0o600)
	tree := newTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	if os.Getenv("KUBERC") != "off" || os.Getenv("KUBECTL_KUBERC") != "false" {
		t.Fatalf("KUBERC=%q KUBECTL_KUBERC=%q", os.Getenv("KUBERC"), os.Getenv("KUBECTL_KUBERC"))
	}
	if res := run(t, tree, "getn"); !strings.Contains(res.Output, `unknown command "getn"`) {
		t.Errorf("the kuberc alias was read:\n%s", res.Output)
	}
	if res := run(t, tree, "get pods -o name"); strings.Contains(res.Output, "{") {
		t.Errorf("the kuberc default was read:\n%s", res.Output)
	}
	if res := run(t, tree, "--kuberc="+rc+" get pods"); !strings.Contains(res.Output, kubectltree.ReasonConnection) && !strings.Contains(res.Output, "unknown flag: --kuberc") {
		t.Errorf("--kuberc answered:\n%s", res.Output)
	}
	if res := run(t, tree, "kuberc view"); res.Code == 0 {
		t.Errorf("kuberc view answered:\n%s", res.Output)
	}
}

func TestOutsideProgramsAndDoorsAreNotPartOfTheStone(t *testing.T) {
	f := newFake(t)
	bin, mark := marker(t)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("KUBECTL_EXTERNAL_DIFF", filepath.Join(bin, "diff"))
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte("resources:\n- github.com/example/never//base?ref=v1\n"), 0o600)
	os.WriteFile(filepath.Join(dir, "web.yaml"), []byte(deploymentYAML(3)), 0o600)
	tree := newTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	for line, reason := range map[string]string{
		"diff -f " + filepath.Join(dir, "web.yaml"):             kubectltree.ReasonDiff,
		"kustomize " + dir:                                      kubectltree.ReasonKustomize,
		"kustomize --enable-helm --enable-exec " + dir:          kubectltree.ReasonKustomize,
		"apply -k " + dir:                                       kubectltree.ReasonKustomize,
		"get -k " + dir:                                         kubectltree.ReasonKustomize,
		"delete --kustomize=" + dir:                             kubectltree.ReasonKustomize,
		"proxy --port=0":                                        kubectltree.ReasonProxy,
		"--profile=cpu --profile-output=" + dir + "/p get pods": kubectltree.ReasonProfile,
		"cluster-info dump --output-directory=" + dir + "/dump": kubectltree.ReasonDumpFiles,
		"apply -f -":                                            "the Stone has no standard input",
		"replace --force -f -":                                  "the Stone has no standard input",
	} {
		res := run(t, tree, line)
		if res.Code == 0 || !strings.Contains(res.Output, reason) {
			t.Errorf("%s answered %d:\n%s", line, res.Code, res.Output)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "p")); err == nil {
		t.Error("a profile was written")
	}
	if _, err := os.Stat(filepath.Join(dir, "dump")); err == nil {
		t.Error("a dump was written")
	}
	noMarker(t, mark)
}

func TestWebManifestsAreNotFetched(t *testing.T) {
	f := newFake(t)
	web := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { web++; w.Write([]byte(deploymentYAML(3))) }))
	defer srv.Close()
	tree := newTree(t, kubeconfig(t, f.URL, "stone", "shop"), recordingGate(&seen{}))
	for _, line := range []string{"apply -f " + srv.URL + "/web.yaml", "create -f " + srv.URL, "get -f HTTPS://example.invalid/x.yaml", "delete -f a.yaml," + srv.URL + "/b.yaml"} {
		res := run(t, tree, line)
		if res.Code == 0 || !strings.Contains(res.Output, "a web address would be fetched outside Kubernites' gate") {
			t.Errorf("%s answered %d:\n%s", line, res.Code, res.Output)
		}
	}
	if web != 0 {
		t.Fatalf("a web manifest was fetched %d times", web)
	}
}

func TestAWorkloadDoorWithStdinNeedsTheStonesTerminal(t *testing.T) {
	f := newFake(t)
	tree := newTree(t, kubeconfig(t, f.URL, "stone", "shop"), recordingGate(&seen{}))
	for line, verb := range map[string]string{
		"exec -it web-1 -- sh": "exec", "attach web-1 -it": "attach", "debug web-1 -it --image=busybox": "debug",
		"-n shop exec -it web-1 -- sh": "exec", "run tmp -it --image=busybox -- sh": "run", "exec -i web-1 -- cat": "exec",
	} {
		res := run(t, tree, line)
		if res.OK() || !strings.Contains(res.Output, "kubectl "+verb+" -i needs the Stone's terminal") || len(res.Sessions) != 0 {
			t.Errorf("%s: ok %v, sessions %v, said:\n%s", line, res.OK(), res.Sessions, res.Output)
		}
	}
	for _, r := range f.seen.all() {
		if strings.Contains(r, "/exec") || strings.Contains(r, "/attach") || strings.Contains(r, "/portforward") || strings.HasPrefix(r, "POST") {
			t.Fatalf("a door was opened: %s", r)
		}
	}
}

func TestCanINoDoesNotEndTheGame(t *testing.T) {
	f := newFake(t)
	tree := newTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	res := run(t, tree, "auth can-i delete pods")
	if res.Code != 1 || strings.TrimSpace(res.Stdout) != "no" {
		t.Fatalf("can-i no answered %d:\n%s", res.Code, res.Output)
	}
	f.allow = true
	if res := run(t, tree, "auth can-i get pods"); res.Code != 0 || strings.TrimSpace(res.Stdout) != "yes" {
		t.Fatalf("can-i yes answered %d:\n%s", res.Code, res.Output)
	}
	if res := run(t, tree, "auth can-i"); res.Code == 0 || !strings.Contains(res.Output, "you must specify two arguments") {
		t.Fatalf("can-i without arguments answered %d:\n%s", res.Code, res.Output)
	}
}

func TestAConflictNamesItsManager(t *testing.T) {
	f := newFake(t)
	f.conflict = true
	tree := newTree(t, kubeconfig(t, f.URL, "stone", "shop"), recordingGate(&seen{}))
	a := tree.Apply(context.Background(), deploymentYAML(5), false)
	if a.Code == 0 || len(a.Conflicts) != 1 || a.Conflicts[0].Manager != "kubectl-create" || len(a.Conflicts[0].Fields) != 1 || a.Conflicts[0].Fields[0] != ".spec.replicas" {
		t.Fatalf("conflicts %+v:\n%s", a.Conflicts, a.Output)
	}
	var patch string
	for _, r := range f.seen.all() {
		if strings.HasPrefix(r, "PATCH") {
			patch = r
		}
	}
	if patch != "PATCH /apis/apps/v1/namespaces/shop/deployments/web" {
		t.Fatalf("the apply sent %q", patch)
	}
	if strings.Join(a.Command[:4], " ") != "apply --server-side --field-manager=kubernites -f" || len(a.Command) != 5 {
		t.Fatalf("the apply ran %v", a.Command)
	}
	forced := tree.Apply(context.Background(), deploymentYAML(4), true)
	if forced.Command[len(forced.Command)-1] != "--force-conflicts" || forced.Code != 0 {
		t.Fatalf("force ran %v and answered %d:\n%s", forced.Command, forced.Code, forced.Output)
	}
}

func TestConflictsAreReadFromKubectlsWords(t *testing.T) {
	one := "error: Apply failed with 1 conflict: conflict with \"kubectl-create\" using apps/v1: .spec.replicas\nPlease review the fields above--they currently have other managers."
	two := "error: Apply failed with 3 conflicts: conflicts with \"kubectl-client-side-apply\" using apps/v1:\n- .spec.replicas\n- .spec.template.spec.containers[name=\"web\"].image\nconflict with \"helm\" using apps/v1: .metadata.labels.app\nPlease review the fields above"
	if c := kubectltree.Conflicts(one); len(c) != 1 || c[0].Manager != "kubectl-create" || c[0].Fields[0] != ".spec.replicas" {
		t.Errorf("one: %+v", c)
	}
	kind := "error: Apply failed with 1 conflict: conflict with \"kubectl\" with subresource \"scale\" using apps/v1: .spec.replicas\nPlease review the fields above"
	if c := kubectltree.Conflicts(kind); len(c) != 1 || c[0].Manager != "kubectl" || c[0].Subresource != "scale" || len(c[0].Fields) != 1 || c[0].Fields[0] != ".spec.replicas" {
		t.Errorf("kind's: %+v", c)
	}
	timed := "error: Apply failed with 2 conflicts: conflicts with \"helm\" using apps/v1 at 2026-11-16T00:00:00Z:\n- .spec.replicas\n- .metadata.labels.app\n"
	if c := kubectltree.Conflicts(timed); len(c) != 1 || c[0].Manager != "helm" || len(c[0].Fields) != 2 || c[0].Fields[1] != ".metadata.labels.app" {
		t.Errorf("timed: %+v", c)
	}
	c := kubectltree.Conflicts(two)
	if len(c) != 2 || c[0].Manager != "kubectl-client-side-apply" || len(c[0].Fields) != 2 || c[1].Manager != "helm" || c[1].Fields[0] != ".metadata.labels.app" {
		t.Errorf("two: %+v", c)
	}
}

func TestTheSentObjectLeavesTheServersFields(t *testing.T) {
	body, err := kubectltree.Sendable(deploymentYAML(3))
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, gone := range []string{"status:", "uid:", "creationTimestamp", "generation:", "managedFields"} {
		if strings.Contains(s, gone) {
			t.Errorf("%s was sent:\n%s", gone, s)
		}
	}
	if !strings.Contains(s, `resourceVersion: "41"`) || !strings.Contains(s, "replicas: 3") {
		t.Errorf("the object sent:\n%s", s)
	}
	for _, bad := range []string{"::", "kind: [", "apiVersion: v1\nkind: Pod\n", "metadata:\n  name: x\n"} {
		if _, err := kubectltree.Sendable(bad); err == nil {
			t.Errorf("%q was sendable", bad)
		}
	}
}

func TestWords(t *testing.T) {
	for line, want := range map[string][]string{
		`get pods -o jsonpath='{.items[*].metadata.name}'`:  {"get", "pods", "-o", "jsonpath={.items[*].metadata.name}"},
		`patch deploy web -p "{\"spec\":{\"replicas\":3}}"`: {"patch", "deploy", "web", "-p", `{"spec":{"replicas":3}}`},
		`create secret generic s --from-literal=a="b c"`:    {"create", "secret", "generic", "s", "--from-literal=a=b c"},
		`label pods a\ b x=y`:                               {"label", "pods", "a b", "x=y"},
		`get pods --field-selector status.phase!=Running`:   {"get", "pods", "--field-selector", "status.phase!=Running"},
		`annotate pod p "note=a\b"`:                         {"annotate", "pod", "p", `note=a\b`},
		`  get   pods  `:                                    {"get", "pods"},
	} {
		got, err := kubectltree.Words(line)
		if err != nil || strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("%s: %q %v", line, got, err)
		}
	}
	for _, line := range []string{"get pods | grep web", "get pods > x", "get pods; rm -rf /", "get pods && x", "get 'pods", "echo `x`"} {
		if _, err := kubectltree.Words(line); err == nil {
			t.Errorf("%s was split", line)
		}
	}
	if _, err := kubectltree.Words("get pods | grep x"); !errors.Is(err, kubectltree.ErrShell) {
		t.Error("a pipe is not named as a shell's")
	}
}

func TestAStoppedCommandEnds(t *testing.T) {
	stall := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("watch") == "true" || r.URL.Query().Get("watch") == "1" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			select {
			case <-r.Context().Done():
			case <-stall:
			}
			return
		}
		newFakeHandler(w, r)
	}))
	defer srv.Close()
	defer close(stall)
	tree := newTree(t, kubeconfig(t, srv.URL, "stone", "shop"), live.ReadOnlyConfig)
	j := tree.Start("get pods -w -o name", nil)
	deadline := time.Now().Add(10 * time.Second)
	text := ""
	for !strings.Contains(text, "pod/web-1") && time.Now().Before(deadline) {
		chunks, _ := j.Take()
		for _, c := range chunks {
			text += c.Text
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(text, "pod/web-1") {
		t.Fatalf("the first lines did not stream: %q", text)
	}
	if j.Done() {
		t.Fatal("a watch ended on its own")
	}
	j.Stop()
	for !j.Done() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !j.Done() || !j.Result().Stopped {
		t.Fatalf("the stopped watch: done %v, %+v", j.Done(), j.Result())
	}
}

func TestClosingTheRealmStopsARunningCommand(t *testing.T) {
	stall := make(chan struct{})
	defer close(stall)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("watch") != "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			select {
			case <-r.Context().Done():
			case <-stall:
			}
			return
		}
		newFakeHandler(w, r)
	}))
	defer srv.Close()
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{kubeconfig(t, srv.URL, "stone", "shop")}, Context: "stone", Wrap: live.ReadOnlyConfig})
	if err != nil {
		t.Fatal(err)
	}
	j := tree.Start("get pods -w -o name", nil)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		chunks, _ := j.Take()
		if len(chunks) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	closed := make(chan error, 1)
	go func() { closed <- tree.Close() }()
	select {
	case <-closed:
	case <-time.After(10 * time.Second):
		t.Fatal("Close waited for the running watch")
	}
	for !j.Done() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !j.Done() || !j.Result().Stopped {
		t.Fatalf("the watch after Close: done %v, %+v", j.Done(), j.Result())
	}
	if res := tree.Run(context.Background(), "get pods"); res.Refused == "" {
		t.Fatalf("a closed tree ran: %+v", res)
	}
}
