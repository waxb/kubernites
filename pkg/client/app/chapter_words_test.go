package app

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestADarkKeepReadsNotReadyInKubernetesWordsAndNoForumNoteInAChapter(t *testing.T) {
	keep := view.Structure{Kind: view.StructKeep, State: view.StateDark, Label: "lone-tower"}
	if a, _ := structureAlert(keep, nil); !strings.Contains(a.Text, "has gone dark") {
		t.Errorf("story mode: %q", a.Text)
	}
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	if a, _ := structureAlert(keep, nil); strings.Contains(a.Text, "gone dark") || !strings.Contains(a.Text, "is NotReady") {
		t.Errorf("Kubernetes words: %q", a.Text)
	}
	if note := ForumNote(view.Site{}, view.StructCodex, true); note != "" {
		t.Errorf("the etcd plot with the note off says %q", note)
	}
	if note := ForumNote(view.Site{}, view.StructCodex, false); note == "" {
		t.Error("the Codex plot lost its Forum note")
	}
}
