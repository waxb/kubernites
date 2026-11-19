package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
)

func replayedRealm(t *testing.T) (*live.Realm, *livetest.Record) {
	t.Helper()
	rec, err := livetest.ReadRecording("../../live/testdata/stone-kind-replay.json")
	if err != nil {
		t.Fatal(err)
	}
	srv, seen := livetest.ServeRecordingSeen(rec)
	t.Cleanup(srv.Close)
	f, err := os.Open("../../live/testdata/stone-kind-snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	snap, err := live.ReadSnapshot(f)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := "apiVersion: v1\nkind: Config\nclusters:\n- name: r\n  cluster:\n    server: " + srv.URL + "\ncontexts:\n- name: " + rec.Context + "\n  context:\n    cluster: r\n    user: u\nusers:\n- name: u\n  user: {}\n"
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := live.OpenRecordedTree(snap, path, rec.Context, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.Stop)
	return r, seen
}

func speakAndWait(t *testing.T, s *Session, cmd string) string {
	t.Helper()
	s.Hud.OpenStone(cmd)
	s.Update(hud.Input{MX: -10, MY: -10, Keys: []ebiten.Key{ebiten.KeyEnter}}, 0, &Pointer{}, true)
	deadline := time.Now().Add(20 * time.Second)
	for s.Hud.StoneRunning() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		s.Update(hud.Input{MX: -10, MY: -10}, 0, &Pointer{}, true)
	}
	if s.Hud.StoneRunning() {
		t.Fatalf("%s did not finish", cmd)
	}
	var lines []string
	for _, l := range s.Hud.Stone.Output {
		lines = append(lines, l.Text)
	}
	return strings.Join(lines, "\n")
}

func TestALiveSessionsStoneRunsKubectlAndSendsNoWriteAtReadOnly(t *testing.T) {
	r, seen := replayedRealm(t)
	s := NewSession(r, nil)
	s.Live = true
	s.Hud.SetStoneMode(hud.StoneModeKube)
	out := speakAndWait(t, s, "get pods -A")
	if !strings.Contains(out, "NAMESPACE") || !strings.Contains(out, "web-b-") {
		t.Fatalf("get pods -A in the Stone:\n%s", out)
	}
	for _, cmd := range []string{"scale deployment web-b -n parity --replicas=0", "delete deployment web-b -n parity", "label nodes --all x=y", "apply -f /dev/null", "cordon kubernites-worker", "rollout restart deployment web-b -n parity"} {
		out = speakAndWait(t, s, cmd)
	}
	if !strings.Contains(out, "refused before it left Kubernites") {
		t.Errorf("a write at READ ONLY answered:\n%s", out)
	}
	if w := seen.Writes(); len(w) != 0 {
		t.Fatalf("writes reached the cluster: %v", w)
	}
	if len(seen.Reads()) == 0 {
		t.Fatal("the Stone's kubectl read nothing")
	}
}

func TestAnAnswerBelongsToTheAskOnScreen(t *testing.T) {
	j := newTreeJob()
	ctx := context.Background()
	j.Answer(true)
	answered := make(chan bool, 1)
	go func() {
		answered <- j.wait(ctx, live.StoneAsk{Why: write.NeedsConfirm, Preview: write.Answer{Code: 200}})
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, ok := j.Ask(); ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the ask never showed")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case yes := <-answered:
		t.Fatalf("an answer given before the ask confirmed it: %v", yes)
	case <-time.After(50 * time.Millisecond):
	}
	j.Answer(true)
	j.Answer(true)
	if yes := <-answered; !yes {
		t.Fatal("CONFIRM did not confirm")
	}
	go func() {
		answered <- j.wait(ctx, live.StoneAsk{Why: write.NeedsConfirm, Preview: write.Answer{Code: 200}})
	}()
	select {
	case yes := <-answered:
		t.Fatalf("a second click confirmed the next write unseen: %v", yes)
	case <-time.After(100 * time.Millisecond):
	}
	j.Answer(false)
	if yes := <-answered; yes {
		t.Fatal("CANCEL confirmed")
	}
}

func TestOneAskAtATimeAndEachAnswerReleasesTheAskShown(t *testing.T) {
	ui.SetNamer(names.Kubernetes(64))
	defer ui.SetNamer(nil)
	j := newTreeJob()
	ctx := context.Background()
	shown := func() string {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if a, ok := j.Ask(); ok {
				return a.Title
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("no ask showed")
		return ""
	}
	ask := func(cmd string) live.StoneAsk {
		return live.StoneAsk{Why: write.NeedsConfirm, Command: cmd, Preview: write.Answer{Code: 200}}
	}
	first, second := make(chan bool, 1), make(chan bool, 1)
	go func() { first <- j.wait(ctx, ask("kubectl evict p1")) }()
	if got := shown(); !strings.Contains(got, "p1") {
		t.Fatalf("shown %q", got)
	}
	go func() { second <- j.wait(ctx, ask("kubectl evict p2")) }()
	time.Sleep(50 * time.Millisecond)
	if got := shown(); !strings.Contains(got, "p1") {
		t.Fatalf("a second ask replaced the one on screen: %q", got)
	}
	j.Answer(true)
	select {
	case yes := <-first:
		if !yes {
			t.Fatal("the CONFIRM shown did not confirm its own ask")
		}
	case yes := <-second:
		t.Fatalf("the CONFIRM shown for p1 answered p2's ask: %v", yes)
	case <-time.After(5 * time.Second):
		t.Fatal("no ask was answered")
	}
	if got := shown(); !strings.Contains(got, "p2") {
		t.Fatalf("the second ask did not show after the first: %q", got)
	}
	j.Answer(false)
	if yes := <-second; yes {
		t.Fatal("CANCEL confirmed")
	}
}
