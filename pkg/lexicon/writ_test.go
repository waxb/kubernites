package lexicon

import "testing"

func TestTheServiceAccountIsAHutsWrit(t *testing.T) {
	lx, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range lx.Entries {
		found = found || e.K8s == "ServiceAccount" && e.Name == "Writ"
	}
	if !found {
		t.Fatal("the lexicon has no Writ for the ServiceAccount")
	}
	story := map[string]string{
		"kubectl get serviceaccounts": "speakstctl get writs",
		"kubectl get sa -A":           "speakstctl get writs -A",
		`Error from server (Forbidden): pods "hello" is forbidden: error looking up service account default/default: serviceaccount "default" not found`: `Error from server (Forbidden): huts "hello" is forbidden: error looking up writ default: writ "default" not found`,
	}
	for in, want := range story {
		if got := StoryText(in); got != want {
			t.Errorf("StoryText(%q)\n got %q\nwant %q", in, got, want)
		}
	}
	if got := CommandFromStory("speakstctl get writs"); got != "kubectl get serviceaccounts" {
		t.Errorf("CommandFromStory = %q", got)
	}
	if got := TrueNames("The Stewards write each hut its writ."); got != "The controllers write each Pod its ServiceAccount." {
		t.Errorf("TrueNames = %q", got)
	}
	if !KubernetesWords.MatchString("serviceaccount") || !KubernetesWords.MatchString("ServiceAccounts") {
		t.Error("the story leak check does not know the ServiceAccount")
	}
}
