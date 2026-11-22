package hud

import "testing"

func TestAWarningPassedForOneAPIServerIsAskedAgainWhenTheContextNamesAnother(t *testing.T) {
	for _, level := range []string{LevelWrites, LevelSeamless} {
		refused := LiveContext{Name: "staging", Server: "https://staging.example:6443", User: "oidc", Identity: "staging-before"}
		prod := LiveContext{Name: "prod", Server: "https://prod-a.example:6443", User: "oidc", Identity: "oidc-public-ca"}
		p := &LivePicker{}
		p.Open([]string{"/tmp/kubeconfig"})
		p.ShowContexts([]LiveContext{prod, refused}, nil)
		p.Chosen = 0
		p.press(liveChoose)
		p.press(LiveBoxID)
		if level == LevelSeamless {
			p.press(liveSeam)
		}
		p.press(liveAllow)
		if p.ChosenLevel() != level {
			t.Fatalf("the warning passed for prod gives %s, want %s", p.ChosenLevel(), level)
		}
		p.press(liveBack)
		p.Chosen = 1
		p.press(liveChoose)
		p.press(LiveBoxID)
		p.press(liveAllow)
		p.press(liveConfirm)
		movedProd := prod
		movedProd.Server = "https://prod-b.example:6443"
		changed := refused
		changed.Identity = "staging-after"
		p.IdentityChanged(refused.Name, []LiveContext{movedProd, changed}, nil, []string{"credentials"})
		p.press(liveBack)
		if p.Step != LiveStepList {
			t.Fatalf("BACK after the refusal: step %s", p.Step)
		}
		p.Chosen = 0
		p.press(liveChoose)
		p.press(LiveBoxID)
		if p.Step != LiveStepWrites || p.ChosenLevel() != LevelReadOnly {
			t.Errorf("prod now names the API server %s, not %s, yet unticking READ ONLY went straight to %s (step %s) on the warning passed for the old server", movedProd.Server, prod.Server, p.ChosenLevel(), p.Step)
		}
	}
}
