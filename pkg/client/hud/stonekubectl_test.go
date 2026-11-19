package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

type fakeJob struct {
	chunks  [][]StoneChunk
	done    bool
	ask     *StoneAsk
	answers []bool
	stopped bool
	end     StoneEnd
}

func (j *fakeJob) Take() ([]StoneChunk, bool) {
	if len(j.chunks) == 0 {
		return nil, j.done
	}
	c := j.chunks[0]
	j.chunks = j.chunks[1:]
	return c, j.done && len(j.chunks) == 0
}

func (j *fakeJob) Ask() (StoneAsk, bool) {
	if j.ask == nil {
		return StoneAsk{}, false
	}
	return *j.ask, true
}

func (j *fakeJob) Answer(yes bool) {
	j.answers = append(j.answers, yes)
	j.ask = nil
}

func (j *fakeJob) Stop()             { j.stopped = true }
func (j *fakeJob) End() StoneEnd     { return j.end }
func (j *fakeJob) finish(e StoneEnd) { j.done, j.end = true, e }

type treeFake struct {
	started []string
	sent    []string
	forced  []bool
	jobs    []*fakeJob
}

func (f *treeFake) start(cmd string) (StoneJob, bool) {
	f.started = append(f.started, cmd)
	j := &fakeJob{}
	f.jobs = append(f.jobs, j)
	return j, true
}

func (f *treeFake) send(e LiveEdit, text string, force bool) (StoneJob, bool) {
	f.sent, f.forced = append(f.sent, text), append(f.forced, force)
	j := &fakeJob{}
	f.jobs = append(f.jobs, j)
	return j, true
}

func liveHUD(t *testing.T, kube bool) (*HUD, *treeFake) {
	t.Helper()
	f := &treeFake{}
	h := New(nil, Backend{Kubectl: func(string) view.CommandResult { return view.CommandResult{Output: "game verbs\n", OK: true} }, Start: f.start, Send: f.send})
	h.SetLive(LiveCard{Context: "kind-kind", Who: "admin"})
	if kube {
		h.SetStoneMode(StoneModeKube)
	}
	h.OverlayID = OverlayStone
	return h, f
}

func stoneText(h *HUD) string {
	var out []string
	for _, l := range h.Stone.Output {
		out = append(out, l.Text)
	}
	return strings.Join(out, "\n")
}

func TestTheStoneRunsKubectlAndStreamsItsOwnWords(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	h, f := liveHUD(t, true)
	h.Stone.SetInput("get pods -n shop")
	h.Speak()
	if len(f.started) != 1 || f.started[0] != "get pods -n shop" || !h.StoneRunning() {
		t.Fatalf("started %v, running %v", f.started, h.StoneRunning())
	}
	j := f.jobs[0]
	j.chunks = [][]StoneChunk{{{Text: "NAME    READY   STATUS\nweb-1   1/1  "}}, {{Text: " Running\n"}, {Text: "Warning: hut\n", Err: true}}}
	h.pumpKubectl()
	if !strings.Contains(stoneText(h), "NAME    READY   STATUS") || strings.Contains(stoneText(h), "web-1") {
		t.Fatalf("a partial line was shown:\n%s", stoneText(h))
	}
	j.done = true
	j.end = StoneEnd{OK: true}
	h.pumpKubectl()
	text := stoneText(h)
	if !strings.Contains(text, "web-1   1/1   Running") || h.StoneRunning() {
		t.Fatalf("after the job:\n%s", text)
	}
	last := h.Stone.Output[len(h.Stone.Output)-1]
	if last.Kind != LineRawError || ConsoleText(last) != "Warning: hut" {
		t.Errorf("stderr line %+v shows %q", last, ConsoleText(last))
	}
	for _, l := range h.Stone.Output {
		if l.Kind == LineRaw && ConsoleText(l) != l.Text {
			t.Errorf("kubectl's words were renamed: %q -> %q", l.Text, ConsoleText(l))
		}
	}
}

func TestStoneStoryKeepsTheGameVerbs(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	h, f := liveHUD(t, false)
	if h.StoneKubectl() {
		t.Fatal("story mode's default is STONE: KUBECTL")
	}
	h.Stone.SetInput("get pods")
	h.Speak()
	if len(f.started) != 0 || !strings.Contains(stoneText(h), "game verbs") {
		t.Fatalf("STONE: STORY ran kubectl %v:\n%s", f.started, stoneText(h))
	}
	ui.SetNamer(names.Kubernetes(64))
	if !h.StoneKubectl() {
		t.Fatal("NAMES mode's default is STONE: STORY")
	}
	h.SetStoneMode(StoneModeStory)
	if h.StoneKubectl() {
		t.Fatal("a remembered STONE: STORY was overridden")
	}
	if NextStoneMode(true) != StoneModeStory || NextStoneMode(false) != StoneModeKube {
		t.Fatal("the switch does not flip")
	}
}

func TestTheMenuHasTheStoneSwitchBesideNamesOnlyInALiveRealm(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	h, _ := liveHUD(t, false)
	buttons := h.MenuButtons(Frame{})
	ids := []string{}
	for _, b := range buttons {
		ids = append(ids, b.ID)
		if b.ID == StoneMenuID && b.Label != "STONE: STORY" {
			t.Errorf("the switch reads %q", b.Label)
		}
		if b.ID == StoneMenuID && lexicon.KubernetesWords.MatchString(b.Tip) {
			t.Errorf("story mode's tip names Kubernetes: %q", b.Tip)
		}
	}
	at := strings.Index(strings.Join(ids, " "), "menu:names "+StoneMenuID)
	if at < 0 {
		t.Fatalf("the MENU in a live realm: %v", ids)
	}
	for _, id := range ids {
		if !LiveReadButton(id) && !strings.HasPrefix(id, volumePrefix) {
			t.Errorf("the live MENU shows %s, which does not apply", id)
		}
	}
	last := buttons[len(buttons)-2]
	if last.Rect.Y+last.Rect.H > MenuBox.Y+MenuBox.H {
		t.Errorf("the live MENU overflows its box: %v", last.Rect)
	}
	h.SetStoneMode(StoneModeKube)
	for _, b := range h.MenuButtons(Frame{}) {
		if b.ID == StoneMenuID && b.Label != "STONE: KUBECTL" {
			t.Errorf("the switch reads %q", b.Label)
		}
	}
	ui.SetNamer(names.Kubernetes(64))
	for _, mode := range []string{StoneModeStory, StoneModeKube} {
		h.SetStoneMode(mode)
		for _, b := range h.MenuButtons(Frame{}) {
			if b.ID == StoneMenuID && !strings.HasPrefix(ui.Name(b.Label), "STONE: ") {
				t.Errorf("NAMES draws the switch as %q", ui.Name(b.Label))
			}
		}
	}
	plain := New(nil, Backend{})
	for _, b := range plain.MenuButtons(Frame{}) {
		if b.ID == StoneMenuID {
			t.Fatal("the plain MENU shows the STONE switch")
		}
	}
	if len(plain.MenuButtons(Frame{})) <= len(buttons) {
		t.Error("the plain MENU lost rows")
	}
}

func TestWorkloadDoorsWithTheStoryStoneSayHowToReachThem(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, cmd := range []string{"exec -it web-1 -- sh", "-n shop attach web-1", "debug web-1 -it --image=busybox", "port-forward web-1 8080:80", "cp web-1:/x y", "run tmp -it --image=busybox"} {
		ui.SetStoryNamer(lexicon.StoryText)
		h, f := liveHUD(t, false)
		h.Stone.SetInput(cmd)
		h.Speak()
		story := stoneText(h)
		if len(f.started) != 0 || !strings.Contains(story, "true tongue") || !strings.Contains(story, "hut card") || strings.Contains(story, "later round") {
			t.Errorf("%s in story mode:\n%s", cmd, story)
		}
		if leak := lexicon.KubernetesWords.FindString(ReachAnswer(ReachVerb(cmd))); leak != "" {
			t.Errorf("%s: story mode names %q", cmd, leak)
		}
		ui.SetNamer(names.Kubernetes(64))
		if got := ReachAnswer(ReachVerb(cmd)); !strings.HasPrefix(got, "kubectl ") || !strings.Contains(got, "runs with STONE: KUBECTL") || strings.Contains(got, "later round") {
			t.Errorf("%s in NAMES mode: %q", cmd, got)
		}
	}
	for _, cmd := range []string{"get pods", "logs web-1", "run tmp --image=busybox", "describe pod exec"} {
		if v := ReachVerb(cmd); v != "" {
			t.Errorf("%s is said to reach into a workload (%s)", cmd, v)
		}
	}
}

func TestAWriteWaitsOnTheStoneForConfirm(t *testing.T) {
	h, f := liveHUD(t, true)
	h.Stone.SetInput("scale deployment web --replicas=3")
	h.Speak()
	j := f.jobs[0]
	j.ask = &StoneAsk{Title: "CONFIRM: kubectl scale deployment web --replicas=3", Lines: []string{"Request: PATCH /apis/apps/v1/namespaces/shop/deployments/web/scale", "Dry run (dryRun=All): OK"}, Confirm: true}
	h.pumpKubectl()
	ids := map[string]Button{}
	for _, b := range h.StoneButtons() {
		ids[b.ID] = b
	}
	if _, ok := ids[StoneConfirmID]; !ok {
		t.Fatalf("no CONFIRM on the Stone: %v", ids)
	}
	if _, ok := ids[StoneCancelID]; !ok {
		t.Fatal("no CANCEL on the Stone")
	}
	h.pressStone("confirm", Frame{})
	if len(j.answers) != 1 || !j.answers[0] {
		t.Fatalf("answers %v", j.answers)
	}
	j.ask = &StoneAsk{Title: "CONFIRM", Confirm: true}
	h.pumpKubectl()
	h.StoneKeys(Input{Keys: []ebiten.Key{ebiten.KeyEscape}})
	if len(j.answers) != 2 || j.answers[1] {
		t.Fatalf("Esc answered %v", j.answers)
	}
	h.StoneKeys(Input{Keys: []ebiten.Key{ebiten.KeyEscape}})
	if !j.stopped {
		t.Fatal("Esc did not stop the running command")
	}
	h.StoneKeys(Input{Chars: []rune("x"), Keys: []ebiten.Key{ebiten.KeyEnter}})
	if len(f.started) != 1 || len(h.Stone.Input) != 0 {
		t.Fatal("typing reached the Stone while kubectl ran")
	}
}

func TestEditOpensTheScriptoriumWithTheLiveYAMLAndForceNeedsItsWarning(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	h, f := liveHUD(t, true)
	h.Stone.SetInput("edit deployment web -n shop")
	h.Speak()
	yaml := "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: web\n  namespace: shop\nspec:\n  replicas: 2\n"
	f.jobs[0].finish(StoneEnd{OK: true, Edit: &LiveEdit{Ref: "deployments.apps/web", Kind: "Deployment", Namespace: "shop", Name: "web", Format: "yaml", Text: yaml}})
	h.pumpKubectl()
	if h.OverlayID != OverlayScriptorium || h.Script.Editor.Text() != yaml {
		t.Fatalf("the Scriptorium shows %q (%s)", h.Script.Editor.Text(), h.OverlayID)
	}
	h.Script.Sync(false)
	h.Script.Sync(true)
	if h.Script.Editor.Text() != yaml {
		t.Fatal("the live YAML was turned into story words")
	}
	h.Script.Editor = NewEditor(strings.Replace(yaml, "replicas: 2", "replicas: 5", 1))
	h.liveEditKeys(Input{Keys: []ebiten.Key{ebiten.KeyEnter}, Ctrl: true})
	if len(f.sent) != 1 || f.forced[0] || !strings.Contains(f.sent[0], "replicas: 5") {
		t.Fatalf("sent %v forced %v", f.sent, f.forced)
	}
	f.jobs[1].finish(StoneEnd{Sent: StoneEditEnd{Conflicts: []EditConflict{{Manager: "kubectl-create", Fields: []string{".spec.replicas"}}}}})
	h.pumpKubectl()
	buttons := map[string]bool{}
	for _, b := range h.scriptButtons() {
		buttons[b.ID] = true
	}
	if !buttons[EditForceID] || buttons[EditForceYesID] || !strings.Contains(strings.Join(h.liveEdit.result, " "), "kubectl-create") {
		t.Fatalf("after a conflict: %v %v", buttons, h.liveEdit.result)
	}
	h.pressScript("live:force", Frame{})
	buttons = map[string]bool{}
	for _, b := range h.scriptButtons() {
		buttons[b.ID] = true
		if b.ID == EditForceYesID && !strings.Contains(b.Tip, "kubectl-create") {
			t.Errorf("the force warning does not name the manager: %q", b.Tip)
		}
	}
	if !buttons[EditForceYesID] || !buttons[EditForceNoID] || len(f.sent) != 1 {
		t.Fatalf("the force warning: %v, sent %d", buttons, len(f.sent))
	}
	if w := ForceWarning(h.liveEdit.conflicts); lexicon.KubernetesWords.MatchString(strings.ReplaceAll(w, "kubectl-create", "")) {
		t.Errorf("story mode's force warning names Kubernetes: %q", w)
	}
	h.pressScript("live:forceyes", Frame{})
	if len(f.sent) != 2 || !f.forced[1] {
		t.Fatalf("force sent %v %v", f.sent, f.forced)
	}
	f.jobs[2].finish(StoneEnd{Sent: StoneEditEnd{OK: true, Output: "deployment.apps/web serverside-applied"}})
	h.pumpKubectl()
	if len(f.started) != 2 || f.started[1] != "edit deployments.apps/web -n shop" {
		t.Fatalf("after a send the live YAML is not read again: %v", f.started)
	}
	h.OverlayID = OverlayNone
	h.syncScript()
	if h.liveEdit != nil || h.Script.Raw {
		t.Fatal("closing the Scriptorium kept the live edit")
	}
}

func TestTheScriptoriumSaysReadOnlyInStoryWords(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	h, f := liveHUD(t, true)
	h.OpenLiveEdit(LiveEdit{Ref: "deployments.apps/web", Kind: "Deployment", Name: "web", Namespace: "shop", Text: "kind: Deployment\n"})
	h.sendLiveEdit(false)
	f.jobs[0].finish(StoneEnd{Sent: StoneEditEnd{Refused: true, Output: "error: refused before it left Kubernites"}})
	h.pumpKubectl()
	got := strings.Join(h.liveEdit.result, " ")
	if !strings.Contains(got, "read-only") || lexicon.KubernetesWords.MatchString(got) {
		t.Fatalf("story mode's refusal: %q", got)
	}
}

func TestTheDetailsCardHasEditOnlyInALiveRealmWithTheTree(t *testing.T) {
	h, f := liveHUD(t, true)
	h.DetailsCard = Details{Sel: Selection{Kind: "hearth", Pod: model.PodView{Namespace: "shop", Name: "web-1"}}, Tab: DetailDescribe}
	found := false
	for _, b := range h.detailButtons() {
		found = found || b.ID == "det:"+EditTab
	}
	if !found {
		t.Fatal("no EDIT on a live card")
	}
	h.pressDetails(EditTab, Frame{})
	if len(f.started) != 1 || f.started[0] != "edit pod/web-1 -n shop" {
		t.Fatalf("EDIT ran %v", f.started)
	}
	plain := New(nil, Backend{})
	plain.SetLive(LiveCard{})
	plain.DetailsCard = h.DetailsCard
	for _, b := range plain.detailButtons() {
		if b.ID == "det:"+EditTab {
			t.Fatal("a recorded realm shows EDIT")
		}
	}
}

func TestAHiddenValueIsRevealedKeyByKey(t *testing.T) {
	yaml := "apiVersion: v1\nkind: Secret\ndata:\n  password: " + MaskedValue + "\n  tls.crt: " + MaskedValue + "\nmetadata:\n  name: s\n"
	got, ok := RevealLine(yaml, "tls.crt", "Y2VydA==")
	if !ok || !strings.Contains(got, "  tls.crt: Y2VydA==") || !strings.Contains(got, "  password: "+MaskedValue) {
		t.Fatalf("yaml:\n%s", got)
	}
	json := "{\n  \"data\": {\n    \"password\": \"" + MaskedValue + "\"\n  }\n}"
	if got, ok := RevealLine(json, "password", "aHVudGVyMg=="); !ok || !strings.Contains(got, `"password": "aHVudGVyMg=="`) {
		t.Fatalf("json:\n%s", got)
	}
	if _, ok := RevealLine(yaml, "missing", "x"); ok {
		t.Fatal("a missing key was revealed")
	}
	h, _ := liveHUD(t, true)
	revealed := []string{}
	h.backend.Reveal = func(e LiveEdit, key string) (string, bool) { revealed = append(revealed, key); return "Y2VydA==", true }
	h.OpenLiveEdit(LiveEdit{Kind: "Secret", Name: "s", Namespace: "shop", Text: yaml, Hidden: []string{"password", "tls.crt"}})
	ids := map[string]bool{}
	for _, b := range h.scriptButtons() {
		ids[b.ID] = true
	}
	if !ids[EditRevealID+"password"] || !ids[EditRevealID+"tls.crt"] {
		t.Fatalf("no REVEAL per key: %v", ids)
	}
	h.pressScript("live:reveal:tls.crt", Frame{})
	if len(revealed) != 1 || !strings.Contains(h.Script.Editor.Text(), "tls.crt: Y2VydA==") || !strings.Contains(h.Script.Editor.Text(), "password: "+MaskedValue) {
		t.Fatalf("after one REVEAL (%v):\n%s", revealed, h.Script.Editor.Text())
	}
	if e, _ := h.LiveEditing(); len(e.Hidden) != 1 || e.Hidden[0] != "password" {
		t.Fatalf("still hidden: %v", e.Hidden)
	}
}

func TestYConfirmsOnlyAnAskAlreadyOnScreen(t *testing.T) {
	h, f := liveHUD(t, true)
	h.Stone.SetInput("get secrets")
	h.Speak()
	j := f.jobs[0]
	j.ask = &StoneAsk{Title: "! SECRETS !", Confirm: true, Go: "I UNDERSTAND [Y]"}
	h.pumpKubectl()
	for _, b := range h.StoneButtons() {
		if b.ID == StoneConfirmID && b.Label != "I UNDERSTAND [Y]" {
			t.Fatalf("the warning's button reads %q", b.Label)
		}
	}
	h.StoneKeys(Input{Keys: []ebiten.Key{ebiten.KeyY}})
	if len(j.answers) != 0 {
		t.Fatalf("Y answered an ask that was never drawn: %v", j.answers)
	}
	for range AskSeenFrames - 1 {
		h.pumpKubectl()
	}
	h.StoneKeys(Input{Keys: []ebiten.Key{ebiten.KeyY}})
	if len(j.answers) != 1 || !j.answers[0] {
		t.Fatalf("Y on a drawn ask answered %v", j.answers)
	}
	j.ask = &StoneAsk{Title: "CONFIRM", Lines: []string{"Nothing can be sent: the dry run failed."}}
	for range AskSeenFrames {
		h.pumpKubectl()
	}
	h.StoneKeys(Input{Keys: []ebiten.Key{ebiten.KeyY}})
	if len(j.answers) != 1 {
		t.Fatalf("Y confirmed a failed dry run: %v", j.answers)
	}
}

func TestAnEditThatAsksFirstMovesToTheStone(t *testing.T) {
	h, f := liveHUD(t, true)
	h.DetailsCard = Details{Sel: Selection{Kind: "hearth", Pod: model.PodView{Namespace: "shop", Name: "web-1"}}, Tab: DetailDescribe}
	h.OverlayID = OverlayDetails
	h.pressDetails(EditTab, Frame{})
	j := f.jobs[0]
	j.ask = &StoneAsk{Title: "! SECRETS !", Confirm: true, Go: "I UNDERSTAND [Y]"}
	h.pumpKubectl()
	if h.OverlayID != OverlayStone || !h.StoneRunning() || !strings.Contains(stoneText(h), "edit pod/web-1 -n shop") {
		t.Fatalf("the warning did not move to the Stone: overlay %s running %v\n%s", h.OverlayID, h.StoneRunning(), stoneText(h))
	}
	if _, ok := h.StoneAsking(); !ok {
		t.Fatal("the Stone does not show the warning")
	}
	h.pressStone("confirm", Frame{})
	if len(j.answers) != 1 || !j.answers[0] {
		t.Fatalf("answers %v", j.answers)
	}
	j.finish(StoneEnd{OK: true, Edit: &LiveEdit{Ref: "pods/web-1", Kind: "Pod", Namespace: "shop", Name: "web-1", Format: "yaml", Text: "kind: Pod\n"}})
	h.pumpKubectl()
	if h.OverlayID != OverlayScriptorium {
		t.Fatalf("the Scriptorium did not open after the warning: %s", h.OverlayID)
	}
	h.DetailsCard = Details{Sel: Selection{Kind: "hearth", Pod: model.PodView{Namespace: "shop", Name: "web-1"}}, Tab: DetailDescribe}
	h.OverlayID = OverlayDetails
	h.Stone.SetInput("get pods -w")
	h.Speak()
	h.pressDetails(EditTab, Frame{})
	busy := f.jobs[len(f.jobs)-1]
	busy.ask = &StoneAsk{Title: "! SECRETS !", Confirm: true}
	h.pumpKubectl()
	if len(busy.answers) != 1 || busy.answers[0] {
		t.Fatalf("with the Stone busy, the EDIT's warning answered %v", busy.answers)
	}
}

func TestAtReadOnlyTheScriptoriumReadsAndSendsNothing(t *testing.T) {
	h, f := liveHUD(t, true)
	h.SetLive(LiveCard{Context: "kind-kind", Who: "admin", Level: LevelReadOnly})
	h.OpenLiveEdit(LiveEdit{Ref: "deployments.apps/web", Kind: "Deployment", Name: "web", Namespace: "shop", Text: "kind: Deployment\n"})
	for _, b := range h.scriptButtons() {
		if b.ID == EditSendID && (!b.Disabled || !strings.Contains(b.Tip, "READ ONLY") && !strings.Contains(b.Tip, "Read only")) {
			t.Fatalf("SEND at READ ONLY: %+v", b)
		}
	}
	h.sendLiveEdit(false)
	h.liveEditKeys(Input{Ctrl: true, Keys: []ebiten.Key{ebiten.KeyEnter}})
	if len(f.sent) != 0 {
		t.Fatalf("READ ONLY sent %v", f.sent)
	}
	h.SetLive(LiveCard{Context: "kind-kind", Who: "admin", Level: LevelWrites})
	h.sendLiveEdit(false)
	if len(f.sent) != 1 {
		t.Fatal("WRITES did not send")
	}
}

func TestAControlChordNeverPassesAWarningOrAConfirmAndAnAskIsSeenAQuarterSecondFirst(t *testing.T) {
	h, f := liveHUD(t, true)
	h.Stone.SetInput("exec -it web-1 -- sh")
	h.Speak()
	j := f.jobs[0]
	j.ask = &StoneAsk{ID: 1, Title: "! REACHING INTO WORKLOADS !", Confirm: true, Go: "I UNDERSTAND [Y]"}
	for range 60 {
		h.pumpKubectl()
	}
	h.StoneKeys(Input{Keys: []ebiten.Key{ebiten.KeyY}, Control: true})
	if len(j.answers) != 0 {
		t.Fatalf("ctrl-y passed the warning: %v", j.answers)
	}
	j.ask = &StoneAsk{ID: 2, Title: "CONFIRM: kubectl exec -it web-1 -- sh", Confirm: true}
	for range 14 {
		h.pumpKubectl()
	}
	h.StoneKeys(Input{Keys: []ebiten.Key{ebiten.KeyY}})
	if len(j.answers) != 0 {
		t.Fatalf("Y answered a CONFIRM on screen for 14 frames: %v", j.answers)
	}
	h.pumpKubectl()
	h.StoneKeys(Input{Keys: []ebiten.Key{ebiten.KeyY}})
	if len(j.answers) != 1 || !j.answers[0] {
		t.Fatalf("Y on a CONFIRM seen for a quarter second answered %v", j.answers)
	}
}

func TestAControlChordNeverPassesThePickersWarning(t *testing.T) {
	p := confirmPicker(nil, false)
	p.press(LiveBoxID)
	if p.Step != LiveStepWrites {
		t.Fatalf("step %s", p.Step)
	}
	p.Update(Input{MX: -10, MY: -10, Keys: []ebiten.Key{ebiten.KeyY}, Control: true})
	if p.Step != LiveStepWrites || p.ChosenLevel() != LevelReadOnly {
		t.Fatalf("ctrl-y passed the WRITES warning: %s %s", p.Step, p.ChosenLevel())
	}
	p.Update(Input{MX: -10, MY: -10, Keys: []ebiten.Key{ebiten.KeyY}})
	if p.Step != LiveStepConfirm {
		t.Fatalf("Y did not pass the warning: %s", p.Step)
	}
}

func TestALineWithoutAnEndIsCutIntoBoundedLines(t *testing.T) {
	h, f := liveHUD(t, true)
	h.Stone.SetInput("exec web-1 -- cat /dev/zero")
	h.Speak()
	j := f.jobs[0]
	for range 50 {
		j.chunks = append(j.chunks, []StoneChunk{{Text: strings.Repeat("x", 4000)}})
	}
	for range 60 {
		h.pumpKubectl()
	}
	r := h.kube
	if r == nil {
		t.Fatal("the job ended")
	}
	if n := len(r.partial[0]); n > MaxRawLine {
		t.Fatalf("an unended line is held at %d bytes", n)
	}
	for _, l := range h.Stone.Output {
		if len(l.Text) > MaxRawLine {
			t.Fatalf("a Stone line holds %d bytes", len(l.Text))
		}
	}
}

func TestTheQuietTipClipsACauseAtAWord(t *testing.T) {
	cause := `Back-off pulling trade "busybox:0.0-never-published": the graduate never arrived somewhere far away`
	c := LiveCard{Quiet: []QuietTrouble{{Kind: "ImagePull", Cause: cause}}}
	line, ok := c.quietLine()
	if !ok {
		t.Fatal("no quiet line")
	}
	for _, l := range strings.Split(line.tip, "\n") {
		head, clipped := strings.CutSuffix(l, "...")
		if !clipped {
			continue
		}
		words := strings.Fields(head)
		if last := words[len(words)-1]; !strings.Contains(cause, last+" ") {
			t.Errorf("the tip cuts a word: %q", l)
		}
	}
}

func TestTheDoneCardLeavesNoWordAloneOnItsLastLine(t *testing.T) {
	defer ui.SetNamer(nil)
	h := New(nil, Backend{})
	o := view.ObjectiveView{Text: "Raise a Keep: too many parcels while only you carry", Why: "so that workers can carry parcels beside you"}
	for mode := range 2 {
		if mode == 0 {
			ui.SetStoryNamer(lexicon.StoryText)
		} else {
			ui.SetNamer(names.Kubernetes(64))
		}
		var rows []string
		for _, l := range h.doneLines(Frame{}, o) {
			if l.Clr == ui.Palette.TextDim {
				rows = append(rows, l.Text)
			}
		}
		if len(rows) > 1 && !strings.Contains(strings.TrimSpace(rows[len(rows)-1]), " ") {
			t.Errorf("mode %d: the DONE line leaves %q alone: %q", mode, rows[len(rows)-1], rows)
		}
	}
}
