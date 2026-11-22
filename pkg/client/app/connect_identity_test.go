package app

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestCredentialsChangedBeforeConnectAreRefusedAndThePickerSaysWhatAtReadOnly(t *testing.T) {
	snap := livetest.Managed()
	srv, rec := livetest.ServeAs(t, snap, livetest.Options{User: "admin@example.com"})
	kubeconfig := livetest.Kubeconfig(t, srv.URL)
	dir := t.TempDir()
	g := NewGame(nil, Options{LiveOnly: true, SettingsDir: dir, Live: LiveSource{Files: []string{kubeconfig}}}, nil, nil)
	listed, err := live.ListContexts([]string{kubeconfig})
	if err != nil || len(listed) != 1 {
		t.Fatalf("contexts %+v %v", listed, err)
	}
	g.choices.Keep(write.Remembered{Context: listed[0].Remembered(), Level: write.Seamless, Warned: write.Seamless, Classes: write.AllClasses})
	if err := write.SaveMemory(dir, g.choices); err != nil {
		t.Fatal(err)
	}
	rig := &liveRig{g: g, rec: rec, dir: dir, snap: snap}
	rig.title(hud.Intent{Kind: hud.IntentLive})
	rig.title(hud.Intent{Kind: hud.IntentLiveList})
	rig.press(t, "live:row:0")
	rig.press(t, "live:choose")
	rig.press(t, hud.LiveBoxID)
	if got := g.Picker.ChosenLevel(); got != hud.LevelSeamless {
		t.Fatalf("the remembered level is %s", got)
	}
	text, err := os.ReadFile(kubeconfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(kubeconfig, []byte(strings.Replace(string(text), "token: not-a-real-token", "token: another-users-token", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	before := len(rec.Full())
	rig.press(t, "live:confirm")
	for deadline := time.Now().Add(30 * time.Second); g.Opening != nil; time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("the opening never ended")
		}
		g.StepLive()
	}
	if g.Scene == SceneRealm {
		t.Fatal("a kubeconfig whose token changed after the level was chosen connected")
	}
	if sent := rec.Full()[before:]; len(sent) > 0 {
		t.Errorf("CONNECT with changed credentials sent %v", sent)
	}
	if !strings.Contains(g.Picker.Err, "credentials") || !strings.Contains(g.Picker.Err, "READ ONLY") {
		t.Errorf("the picker does not say what changed: %q", g.Picker.Err)
	}
	if _, _, ok := g.choices.Recall(listed[0].Remembered()); ok {
		t.Error("the choice remembered for the old credentials is kept")
	}
	rig.press(t, "live:back")
	if g.Picker.Step != hud.LiveStepList || g.Picker.ChosenLevel() != hud.LevelReadOnly {
		t.Fatalf("BACK after the refusal: step %s level %s", g.Picker.Step, g.Picker.ChosenLevel())
	}
	rig.press(t, "live:choose")
	rig.press(t, hud.LiveBoxID)
	if g.Picker.Step != hud.LiveStepWrites {
		t.Fatalf("unticking READ ONLY on the changed context did not ask the warnings again: step %s", g.Picker.Step)
	}
}
