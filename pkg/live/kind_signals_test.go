package live

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	signalsKubeconfigEnv = "KUBERNITES_SIGNALS_KUBECONFIG"
	signalsKubectlEnv    = "KUBERNITES_SIGNALS_KUBECTL"
	signalsReportEnv     = "KUBERNITES_SIGNALS_REPORT"
	signalsOnlyEnv       = "KUBERNITES_SIGNALS_ONLY"
	signalsContext       = "kind-kubernites"
	signalsWorker        = "kubernites-worker"
	signalsOpens         = 5 * time.Minute
	signalsMends         = 4 * time.Minute
	signalsTick          = 200 * time.Millisecond
)

type kindTrouble struct {
	name   string
	kind   string
	id     string
	setup  []string
	break_ []string
	undo   []string
	words  string
}

var kindTroubles = []kindTrouble{
	{name: "crash loop", kind: view.TagCrashLoop, id: "crashloop:signals/Deployment/crasher", break_: []string{"apply -f crasher.yaml"},
		undo: []string{`-n signals patch deployment crasher --type=json -p=[{"op":"replace","path":"/spec/template/spec/containers/0/command","value":["sleep","3600"]}]`}, words: "CrashLoopBackOff: back-off"},
	{name: "OOM kill", kind: view.TagOOMKilled, id: "oomkilled:signals/Deployment/hog", break_: []string{"apply -f hog.yaml"},
		undo: []string{`-n signals patch deployment hog --type=json -p=[{"op":"replace","path":"/spec/template/spec/containers/0/command","value":["sleep","3600"]}]`}, words: "OOMKilled: container eater"},
	{name: "bad image", kind: view.TagImagePull, id: "imagepull:signals/Deployment/lost", break_: []string{"apply -f lost.yaml"},
		undo: []string{"-n signals set image deployment/lost worker=busybox:1.37"}, words: "busybox:0.0-never-published"},
	{name: "pod too big for any node", kind: view.TagUnhoused, id: "unhoused:signals/Deployment/giant", break_: []string{"apply -f giant.yaml"},
		undo: []string{"-n signals set resources deployment giant -c worker --requests=memory=8Mi"}, words: "Insufficient memory"},
	{name: "eviction past an ephemeral-storage limit", kind: view.TagEvicted, id: "evicted:signals/digger", break_: []string{"apply -f digger.yaml"},
		undo: []string{"-n signals delete pod digger --wait=false"}, words: "ephemeral local storage"},
	{name: "roll-out held by unschedulable pods", kind: view.CalamityStalled, id: "stalled:signals/Deployment/roller", setup: []string{"apply -f roller.yaml", "-n signals rollout status deployment/roller --timeout=120s"},
		break_: []string{"-n signals set resources deployment roller -c worker --requests=memory=64Gi"}, undo: []string{"-n signals rollout undo deployment/roller"}, words: "stalled at"},
	{name: "failing probes", kind: view.ReportHut, id: "report-hut:signals/Deployment/shy", break_: []string{"apply -f shy.yaml"},
		undo: []string{`-n signals patch deployment shy --type=json -p=[{"op":"replace","path":"/spec/template/spec/containers/0/readinessProbe/exec/command","value":["true"]}]`}, words: "Readiness probe failed"},
	{name: "failed Job", kind: view.ReportQuest, id: "report-quest:signals/Job/doomed", break_: []string{"apply -f doomed.yaml"},
		undo: []string{"-n signals delete job doomed --wait=false"}, words: "BackoffLimitExceeded"},
	{name: "Pending PVC", kind: view.ReportDeed, id: "report-deed:signals/PersistentVolumeClaim/orphan", break_: []string{"apply -f orphan.yaml"},
		undo: []string{"-n signals delete pvc orphan --wait=false"}, words: "\"nope\" not found"},
	{name: "HPA at its maximum", kind: view.ReportHorn, id: "report-horn:signals/HorizontalPodAutoscaler/busy", break_: []string{"apply -f busy.yaml"},
		undo: []string{"-n signals delete hpa busy --wait=false"}, words: "TooManyReplicas"},
	{name: "cordoned node", kind: view.ReportKeep, id: "report-keep:/Node/" + signalsWorker, break_: []string{"cordon " + signalsWorker},
		undo: []string{"uncordon " + signalsWorker}, words: "SchedulingDisabled"},
	{name: "NotReady node", kind: view.ReportKeep, id: "report-keep:/Node/" + signalsWorker, break_: []string{"docker pause " + signalsWorker},
		undo: []string{"docker unpause " + signalsWorker}, words: "NotReady"},
}

type kindResult struct {
	Name       string
	Kind       string
	ID         string
	Opened     bool
	OpenSecs   float64
	FirstCause string
	WordsSecs  float64
	Quiet      bool
	Cause      string
	Why        string
	Remedies   []view.RemedyView
	Mended     string
	MendSecs   float64
	Chronicle  string
	Error      string
}

type kindRun struct {
	t       *testing.T
	config  string
	kubectl string
}

func (k kindRun) run(line string) error {
	k.t.Helper()
	if rest, ok := strings.CutPrefix(line, "docker "); ok {
		out, err := exec.Command("docker", strings.Fields(rest)...).CombinedOutput()
		if err != nil {
			k.t.Logf("docker %s: %s", rest, out)
		}
		return err
	}
	args := []string{"--kubeconfig", k.config, "--context", signalsContext}
	if strings.Contains(line, " -p=") {
		head, patch, _ := strings.Cut(line, " -p=")
		args = append(append(args, strings.Fields(head)...), "-p="+patch)
	} else {
		args = append(args, strings.Fields(line)...)
	}
	cmd := exec.Command(k.kubectl, args...)
	cmd.Dir = filepath.Join("testdata", "signals")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+k.config)
	out, err := cmd.CombinedOutput()
	k.t.Logf("kubectl %s: %s", line, strings.TrimSpace(string(out)))
	return err
}

func (k kindRun) fresh() {
	k.run("delete namespace signals --ignore-not-found --wait=true --timeout=120s")
	k.run("create namespace signals")
	k.run("uncordon " + signalsWorker)
	exec.Command("docker", "unpause", signalsWorker).Run()
}

func findCalamity(r *Realm, id string) (view.CalamityView, bool) {
	for _, v := range r.GameState().Calamities {
		if v.ID == id {
			return v, true
		}
	}
	return view.CalamityView{}, false
}

func TestEverySignalOnAKindClusterOpensItsCardAndMendsWhenUndone(t *testing.T) {
	config, kubectl := os.Getenv(signalsKubeconfigEnv), os.Getenv(signalsKubectlEnv)
	if config == "" || kubectl == "" {
		t.Skipf("set %s to the throwaway kind cluster's kubeconfig file and %s to a kubectl binary to run this test", signalsKubeconfigEnv, signalsKubectlEnv)
	}
	ctxs, err := ListContexts([]string{config})
	if err != nil || len(ctxs) != 1 || ctxs[0].Name != signalsContext {
		t.Fatalf("the kubeconfig must hold only %s: %+v %v", signalsContext, ctxs, err)
	}
	k := kindRun{t: t, config: config, kubectl: kubectl}
	var results []kindResult
	defer func() {
		if path := os.Getenv(signalsReportEnv); path != "" {
			data, _ := json.MarshalIndent(results, "", "  ")
			os.WriteFile(path, data, 0o644)
		}
	}()
	only := os.Getenv(signalsOnlyEnv)
	for _, tr := range kindTroubles {
		if only != "" && !strings.Contains(tr.name, only) {
			continue
		}
		res := kindResult{Name: tr.name, Kind: tr.kind, ID: tr.id}
		t.Run(tr.name, func(t *testing.T) {
			k.t = t
			defer func() { results = append(results, res) }()
			k.fresh()
			for _, line := range tr.setup {
				if err := k.run(line); err != nil {
					t.Fatalf("setup %s: %v", line, err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			r, err := Connect(ctx, []string{config}, signalsContext, time.Now(), nil)
			cancel()
			if err != nil {
				t.Fatal(err)
			}
			r.StartWatching()
			defer r.Stop()
			if v, ok := findCalamity(r, tr.id); ok {
				t.Fatalf("%s stands before the break: %+v", tr.id, v)
			}
			for _, line := range tr.break_ {
				if err := k.run(line); err != nil {
					t.Fatalf("break %s: %v", line, err)
				}
			}
			start := time.Now()
			var v view.CalamityView
			for {
				r.Pump(signalsTick.Seconds())
				var ok bool
				if v, ok = findCalamity(r, tr.id); ok && v.Mended == "" {
					break
				}
				if time.Since(start) > signalsOpens {
					res.Error = "never opened"
					t.Fatalf("%s never opened; calamities %+v", tr.id, r.GameState().Calamities)
				}
				time.Sleep(signalsTick)
			}
			res.Opened, res.OpenSecs, res.Quiet, res.FirstCause = true, time.Since(start).Seconds(), v.Quiet, v.Cause
			for !strings.Contains(v.Cause+" "+v.Why, tr.words) && time.Since(start) < signalsOpens {
				time.Sleep(signalsTick)
				r.Pump(signalsTick.Seconds())
				v, _ = findCalamity(r, tr.id)
			}
			res.WordsSecs = time.Since(start).Seconds()
			res.Cause, res.Why, res.Remedies = v.Cause, v.Why, v.Remedies
			if v.Quiet || v.Kind != tr.kind || !strings.Contains(v.Cause+" "+v.Why, tr.words) {
				res.Error = "wrong card"
				t.Errorf("card %+v; want kind %s with %q, not quiet", v, tr.kind, tr.words)
			}
			for _, line := range tr.undo {
				if err := k.run(line); err != nil {
					t.Fatalf("undo %s: %v", line, err)
				}
			}
			start = time.Now()
			for {
				r.Pump(signalsTick.Seconds())
				if v, ok := findCalamity(r, tr.id); ok && v.Mended != "" {
					res.Mended = v.Mended
					break
				}
				if time.Since(start) > signalsMends {
					res.Error = "never mended"
					got, _ := findCalamity(r, tr.id)
					t.Fatalf("%s never mended: %+v", tr.id, got)
				}
				time.Sleep(signalsTick)
			}
			res.MendSecs = time.Since(start).Seconds()
			st := r.GameState()
			if len(st.Chronicles) == 0 {
				t.Error("no history entry")
			} else {
				c := st.Chronicles[len(st.Chronicles)-1]
				res.Chronicle = c.Subject + " struck at " + c.Struck + ", mended at " + c.Healed
			}
			t.Logf("%s: opened in %.0fs: %s | %s; mended in %.0fs: %s", tr.name, res.OpenSecs, res.Cause, res.Why, res.MendSecs, res.Mended)
		})
	}
	k.t = t
	k.fresh()
}
