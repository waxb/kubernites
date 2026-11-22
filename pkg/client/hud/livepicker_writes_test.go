package hud

import (
	"strings"
	"testing"
)

var pickerCtx = LiveContext{Name: "kind-kind", Server: "https://127.0.0.1:6443", User: "kind-kind"}

func confirmPicker(remembered map[string]LiveMemory, forced bool) *LivePicker {
	p := &LivePicker{Remembered: remembered, Forced: forced}
	p.Open([]string{"/tmp/kubeconfig"})
	p.ShowContexts([]LiveContext{pickerCtx}, nil)
	p.Chosen = 0
	p.press(liveChoose)
	return p
}

func pickerButton(p *LivePicker, id string) (Button, bool) {
	for _, b := range p.Buttons() {
		if b.ID == id {
			return b, true
		}
	}
	return Button{}, false
}

func intentKinds(list []Intent) map[string]Intent {
	out := map[string]Intent{}
	for _, it := range list {
		out[it.Kind] = it
	}
	return out
}

func TestTheReadOnlyBoxIsTickedAtEveryConnectWithItsExactTooltip(t *testing.T) {
	p := confirmPicker(map[string]LiveMemory{pickerCtx.Name: {Server: pickerCtx.Server, User: pickerCtx.User, Level: LevelSeamless, Warned: LevelSeamless}}, false)
	b, ok := pickerButton(p, LiveBoxID)
	if !ok || b.Label != "[x] READ ONLY" || b.Disabled {
		t.Fatalf("the box %+v", b)
	}
	if b.Tip != "Ticked by default: Kubernites only reads this cluster. Untick to allow changes. Set before connecting." {
		t.Fatalf("tooltip %q", b.Tip)
	}
	if p.ChosenLevel() != LevelReadOnly {
		t.Fatalf("level %s before unticking", p.ChosenLevel())
	}
	got := intentKinds(p.press(liveConfirm))
	if it := got[IntentLiveOpen]; it.Command != LevelReadOnly || it.Target != pickerCtx.Name {
		t.Fatalf("connect %+v", it)
	}
	p.Step = LiveStepList
	p.press(liveChoose)
	if b, _ := pickerButton(p, LiveBoxID); b.Label != "[x] READ ONLY" {
		t.Fatalf("the box was not ticked again: %+v", b)
	}
}

func TestUntickingANewContextWarnsAndSeamlessWarnsAgain(t *testing.T) {
	p := confirmPicker(nil, false)
	p.press(LiveBoxID)
	if p.Step != LiveStepWrites || p.ChosenLevel() != LevelReadOnly {
		t.Fatalf("unticking skipped the WRITES warning: %s %s", p.Step, p.ChosenLevel())
	}
	p.press(liveAllow)
	if p.Step != LiveStepConfirm || p.ChosenLevel() != LevelWrites {
		t.Fatalf("after the warning: %s %s", p.Step, p.ChosenLevel())
	}
	p.press(liveLevel)
	if p.Step != LiveStepSeam {
		t.Fatalf("SEAMLESS without its own warning: %s", p.Step)
	}
	p.press(liveAllow)
	if p.ChosenLevel() != LevelSeamless {
		t.Fatalf("level %s", p.ChosenLevel())
	}
	if it := intentKinds(p.press(liveConfirm))[IntentLiveOpen]; it.Command != LevelSeamless {
		t.Fatalf("connect %+v", it)
	}
}

func TestBackFromAWarningKeepsReadOnly(t *testing.T) {
	p := confirmPicker(nil, false)
	p.press(LiveBoxID)
	p.press(liveBack)
	if p.Step != LiveStepConfirm || p.ChosenLevel() != LevelReadOnly {
		t.Fatalf("back from the warning: %s %s", p.Step, p.ChosenLevel())
	}
}

func TestUntickingARememberedContextGoesStraightToItsLevel(t *testing.T) {
	p := confirmPicker(map[string]LiveMemory{pickerCtx.Name: {Server: pickerCtx.Server, User: pickerCtx.User, Level: LevelSeamless, Warned: LevelSeamless}}, false)
	p.press(LiveBoxID)
	if p.Step != LiveStepConfirm || p.ChosenLevel() != LevelSeamless {
		t.Fatalf("a remembered context warned again: %s %s", p.Step, p.ChosenLevel())
	}
}

func TestAChangedServerOrUserDropsTheMemoryAndSaysWhat(t *testing.T) {
	for _, changed := range []LiveMemory{
		{Server: "https://10.0.0.1:6443", User: pickerCtx.User, Level: LevelWrites, Warned: LevelWrites},
		{Server: pickerCtx.Server, User: "admin", Level: LevelWrites, Warned: LevelWrites},
	} {
		p := confirmPicker(map[string]LiveMemory{pickerCtx.Name: changed}, false)
		intents := intentKinds(p.press(LiveBoxID))
		if p.Step != LiveStepWrites || p.Changed == "" {
			t.Fatalf("a changed context did not warn: %s %q", p.Step, p.Changed)
		}
		if it, ok := intents[IntentLiveForget]; !ok || it.Target != pickerCtx.Name {
			t.Fatalf("the memory was not dropped: %+v", intents)
		}
		if _, still := p.Remembered[pickerCtx.Name]; still {
			t.Fatal("the picker still holds the memory")
		}
	}
}

func TestReadonlyLocksTheBox(t *testing.T) {
	p := confirmPicker(map[string]LiveMemory{pickerCtx.Name: {Server: pickerCtx.Server, User: pickerCtx.User, Level: LevelSeamless, Warned: LevelSeamless}}, true)
	b, _ := pickerButton(p, LiveBoxID)
	if !b.Disabled || b.Label != "[x] READ ONLY" {
		t.Fatalf("the box %+v", b)
	}
	p.untick()
	if p.ChosenLevel() != LevelReadOnly {
		t.Fatalf("--readonly let the level change to %s", p.ChosenLevel())
	}
}

func TestAnotherCertificateAuthorityOrCredentialDropsTheMemoryAndSaysWhat(t *testing.T) {
	ctx := pickerCtx
	ctx.Identity = "now"
	p := confirmPicker(map[string]LiveMemory{ctx.Name: {Server: ctx.Server, User: ctx.User, Identity: "before", Level: LevelSeamless, Warned: LevelSeamless}}, false)
	p.Contexts[p.Chosen] = ctx
	intents := intentKinds(p.press(LiveBoxID))
	if p.Step != LiveStepWrites || !strings.Contains(p.Changed, "credentials") {
		t.Fatalf("another identity did not warn: %s %q", p.Step, p.Changed)
	}
	if it, ok := intents[IntentLiveForget]; !ok || it.Target != ctx.Name {
		t.Fatalf("the memory was not dropped: %+v", intents)
	}
}
