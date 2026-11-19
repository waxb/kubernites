package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func listedOf(c LiveCard) (liveLine, bool) {
	for _, l := range c.lines() {
		if l.id == liveListedID {
			return l, true
		}
	}
	return liveLine{}, false
}

func TestTheLiveCardNamesEachKindReadByListingAndItsPeriod(t *testing.T) {
	c := LiveCard{Context: "kind-kubernites", Who: "pod-lister", Keeps: 2, Huts: 9, Fiefs: 5, Isles: 1, Watching: true,
		Listed: []ListedRead{{List: "pods", Every: 10}}}
	defer ui.SetNamer(nil)
	ui.SetNamer(nil)
	line, ok := listedOf(c)
	if !ok || line.text != "READ EVERY 10 S: huts" || !strings.Contains(line.tip, "pod-lister may read the huts but not watch them") || !strings.Contains(line.tip, "every 10 s") {
		t.Fatalf("story line %+v", line)
	}
	for _, text := range []string{line.text, line.tip} {
		text = strings.ReplaceAll(text, c.Who, "")
		if leak := lexicon.KubernetesWords.FindString(text); leak != "" {
			t.Errorf("story mode names %q in %q", leak, text)
		}
	}
	ui.SetNamer(lexicon.TrueNames)
	line, _ = listedOf(c)
	if line.text != "LISTED EVERY 10 S: pods" || !strings.Contains(line.tip, "pod-lister may list pods but not watch them") || !strings.Contains(line.tip, "NotOlderThan") {
		t.Fatalf("NAMES line %+v", line)
	}
	if !LiveReadButton(liveListedID) {
		t.Error("the listing line's tip is gated in a live realm")
	}
	c.Listed = append(c.Listed, ListedRead{List: "widgets.example.com", Every: 20, Cap: 64})
	line, _ = listedOf(c)
	if !strings.HasPrefix(line.text, "LISTED EVERY 10-20 S: pods, widgets") || !strings.Contains(line.tip, "At most 64 custom resource kinds") || !strings.Contains(line.tip, "widgets.example.com every 20 s") {
		t.Errorf("NAMES line past the cap %+v", line)
	}
	if w := liveFont.Width(line.text, 1); w > CardBox.W-16 {
		t.Errorf("%q is %d pixels wide, the card has %d", line.text, w, CardBox.W-16)
	}
	c.Stale, c.Trouble, c.Who = "12:00:00 UTC", 3, "x"
	c.Refused, c.Namespace, c.Idle = []string{"services"}, "shop", []string{"kube-system/a"}
	c.Quiet = []QuietTrouble{{ID: "a", Kind: "crashloop", Cause: "CrashLoopBackOff"}}
	shown := false
	for _, l := range c.shown() {
		shown = shown || l.id == liveListedID
	}
	if !shown {
		t.Error("a full card drops the listing line")
	}
}
