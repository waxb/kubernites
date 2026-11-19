package lexicon

import "testing"

func TestTheBellsRelightReadsAsTheKubeletsEventInNamesMode(t *testing.T) {
	story := "The Reeve of stone-keep-1: porters-7db9946f9d-x2kq9 stopped answering the watchman's bell and will be relit"
	want := "The kubelet of stone-keep-1: porters-7db9946f9d-x2kq9 failed liveness probe, will be restarted"
	if got := TrueNames(story); got != want {
		t.Fatalf("NAMES mode reads %q, want %q", got, want)
	}
	if KubernetesWords.MatchString(story) {
		t.Fatalf("its story twin leaks a Kubernetes word: %q", story)
	}
	if got := TrueNames("Hang the watchman's bell on the porters"); got != "Hang the liveness probe on the porters" {
		t.Fatalf("the bell step's title reads %q in NAMES mode", got)
	}
}
