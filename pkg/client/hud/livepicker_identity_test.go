package hud

import (
	"strings"
	"testing"
)

func TestAfterConnectIsRefusedForAChangedIdentityUntickingWarnsAgain(t *testing.T) {
	for _, level := range []string{LevelWrites, LevelSeamless} {
		before := pickerCtx
		before.Identity = "chosen-with"
		p := &LivePicker{}
		p.Open([]string{"/tmp/kubeconfig"})
		p.ShowContexts([]LiveContext{before}, nil)
		p.Chosen = 0
		p.press(liveChoose)
		p.press(LiveBoxID)
		if p.Step != LiveStepWrites {
			t.Fatalf("a new context did not warn: %s", p.Step)
		}
		if level == LevelSeamless {
			p.press(liveSeam)
		}
		p.press(liveAllow)
		if p.ChosenLevel() != level {
			t.Fatalf("the warning passed gives %s, want %s", p.ChosenLevel(), level)
		}
		p.press(liveConfirm)
		now := before
		now.Identity = "changed-since"
		p.IdentityChanged(before.Name, []LiveContext{now}, nil, []string{"credentials"})
		p.press(liveBack)
		if p.Step != LiveStepList {
			t.Fatalf("BACK after the refusal: step %s", p.Step)
		}
		p.press(liveChoose)
		p.press(LiveBoxID)
		if p.Step != LiveStepWrites || p.ChosenLevel() != LevelReadOnly {
			t.Errorf("after CONNECT was refused for changed credentials, unticking READ ONLY went straight to %s (step %s) with no warning, though the refusal says it asks its warnings again", p.ChosenLevel(), p.Step)
		}
	}
}

func TestTheWarningsPassedInThisWindowHoldOnlyForTheIdentityTheyWerePassedFor(t *testing.T) {
	first := pickerCtx
	first.Identity = "first"
	p := &LivePicker{}
	p.Open([]string{"/tmp/kubeconfig"})
	p.ShowContexts([]LiveContext{first}, nil)
	p.Chosen = 0
	p.press(liveChoose)
	p.press(LiveBoxID)
	p.press(liveSeam)
	p.press(liveAllow)
	p.press(liveBack)
	second := first
	second.Identity = "second"
	p.ShowContexts([]LiveContext{second}, nil)
	p.Chosen = 0
	p.press(liveChoose)
	p.press(LiveBoxID)
	if p.Step != LiveStepWrites {
		t.Errorf("the same name listed again with other credentials kept the warnings passed for the first: step %s, level %s", p.Step, p.ChosenLevel())
	}
	p.ShowContexts([]LiveContext{first}, nil)
	p.Chosen = 0
	p.press(liveChoose)
	p.press(LiveBoxID)
	if p.Step != LiveStepConfirm || p.ChosenLevel() != LevelWrites {
		t.Errorf("the identity the warnings were passed for asks them again: step %s, level %s", p.Step, p.ChosenLevel())
	}
}

func TestTurningServerVerificationOffOrOnIsNamedInPlainWordsAndNotAsAnotherCertificateAuthority(t *testing.T) {
	for facet, want := range map[string]string{
		"server verification off": "turns off the checks of the server's certificate",
		"server verification on":  "turns the checks of the server's certificate back on",
	} {
		ctx := pickerCtx
		ctx.Identity = "before"
		p := &LivePicker{}
		p.Open([]string{"/tmp/kubeconfig"})
		p.ShowContexts([]LiveContext{ctx}, nil)
		p.Chosen = 0
		now := ctx
		now.Identity = "after"
		p.IdentityChanged(ctx.Name, []LiveContext{now}, nil, []string{facet})
		if !strings.Contains(p.Err, want) || strings.Contains(p.Err, "certificate authority") {
			t.Errorf("%s: the picker says %q", facet, p.Err)
		}
	}
}
