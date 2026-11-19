package ui

import "testing"

func TestAClipLeavesNoPunctuationBeforeItsDots(t *testing.T) {
	for _, c := range []struct {
		text string
		max  int
		want string
	}{
		{"frontendconfigs.networking.gke.io", 19, "frontendconfigs..."},
		{"pods, services, ingresses", 8, "pods..."},
		{"pods, services, ingresses", 9, "pods ..."},
		{"KubeletReady   True    KubeletReady", 18, "KubeletReady   ..."},
		{"Reason   Message", 10, "Reason ..."},
		{"LEDGER: 15:04 SCALE web", 10, "LEDGER..."},
		{"hello world", 5, "he..."},
	} {
		if got := Clip(c.text, c.max); got != c.want {
			t.Errorf("Clip(%q, %d) = %q, want %q", c.text, c.max, got, c.want)
		}
	}
	var f Font
	lines := f.WrapLines("aaaa, bbbb, cccc, dddd, eeee, ffff, gggg", 14*Advance, 1, 2)
	if got := lines[len(lines)-1]; got != "cccc, dddd..." {
		t.Errorf("the last line is %q, want no comma before its dots", got)
	}
}

func TestALongNameWrapsAtACommaThenASlashThenADotHyphenOrColonAndNeverInsideANumber(t *testing.T) {
	var f Font
	for _, c := range []struct {
		text string
		want []string
	}{
		{"podmonitorings.monitoring.googleapis.com,", []string{"podmonitorings.", "monitoring.googleapis.com,"}},
		{"live-ledger-staging-north-5f2c19a07d3e.jsonl", []string{"live-ledger-staging-", "north-5f2c19a07d3e.jsonl"}},
		{"/home/player/.config/kubernites/live-ledger-staging-north-5f2c19a07d3e.jsonl", []string{"/home/player/.config/kubernites/", "live-ledger-staging-", "north-5f2c19a07d3e.jsonl"}},
		{`'[{"op":"add","path":"/endpoints/-","value":{"addresses":["10.244.0.7"],"targetRef":{"kind":"Pod","namespace":"default","name":"porter-2s2p2"}}}]'`,
			[]string{`'[{"op":"add","path":"/endpoints/-",`, `"value":{"addresses":["10.244.0.7"],`, `"targetRef":{"kind":"Pod",`, `"namespace":"default",`, `"name":"porter-2s2p2"}}}]'`}},
		{`["10.244.0.7"],"targetRef":{"kind":"Pod"`, []string{`["10.244.0.7"],`, `"targetRef":{"kind":"Pod"`}},
		{`'{"spec":{"strategy":{"rollingUpdate":{"maxSurge":0,"maxUnavailable":1}}}}'`, []string{`'{"spec":{"strategy":`, `{"rollingUpdate":{"maxSurge":0,`, `"maxUnavailable":1}}}}'`}},
	} {
		got := f.Wrap(c.text, 37*Advance, 1)
		if len(got) != len(c.want) {
			t.Errorf("Wrap(%q) = %q, want %q", c.text, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("Wrap(%q) = %q, want %q", c.text, got, c.want)
				break
			}
		}
	}
}
