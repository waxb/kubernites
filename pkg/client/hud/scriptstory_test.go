package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestTheScriptoriumWritesDecreesInTheRealmsWords(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	run := &fakeRunner{}
	h := New(nil, run.backend())
	h.OverlayID = OverlayScriptorium
	h.Script.Sync(ui.StoryNaming())
	text := h.Script.Editor.Text()
	if !lexicon.IsStoryManifest(text) || strings.Contains(text, "Deployment") || strings.Contains(text, "replicas") {
		t.Fatalf("the story crew template reads:\n%s", text)
	}
	h.ApplyScript()
	if len(run.applied) != 1 || run.applied[0] != text {
		t.Fatalf("the template was not sent as written: %q", run.applied)
	}
	for _, name := range templateOrder {
		if tip := ui.Name(templateTip(name)); strings.Contains(tip, "(") || strings.Contains(tip, "Pod") || strings.Contains(tip, "Deployment") {
			t.Errorf("%s tip in story mode: %q", name, tip)
		}
	}
	h.Script.Editor.Move(len(h.Script.Editor.Lines), 0)
	h.Script.Editor.End()
	h.Script.Editor.Newline()
	for _, r := range "# mine" {
		h.Script.Editor.Insert(r)
	}
	draft := h.Script.Editor.Text()
	ui.SetNamer(names.Kubernetes(64))
	h.Script.Sync(ui.StoryNaming())
	if got := h.Script.Editor.Text(); got != lexicon.ManifestFromStory(draft) || !strings.HasPrefix(got, "apiVersion:") || !strings.HasSuffix(got, "# mine") {
		t.Fatalf("switching to Kubernetes names left the draft as:\n%s", got)
	}
	h.Script.Load(TemplateHearth)
	if !strings.HasPrefix(h.Script.Editor.Text(), "apiVersion: v1") || templateTip(TemplateHearth) != "A single hut (Pod)" {
		t.Fatalf("the Kubernetes Pod template reads:\n%s", h.Script.Editor.Text())
	}
}
