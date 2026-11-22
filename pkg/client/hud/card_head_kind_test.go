package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestACardHeadTooLongForItsNameKeepsTheWholeKindName(t *testing.T) {
	defer ui.SetNamer(nil)
	s := view.Structure{ID: "foreign-1", Kind: "foreign", Label: "Foreign hall of CiliumEndpoint", State: view.StateReady,
		Object: view.ObjectRef{Kind: "CiliumEndpoint", Resource: "ciliumendpoints.cilium.io"}}
	chars := ui.MaxChars(CardHeadWidth(), 1)
	want := map[string]string{"STORY": "FOREIGN HALL", "KUBERNETES": "CUSTOM RESOURCE KIND"}
	for _, mode := range namesModes() {
		mode.apply()
		c := StructureCard(s, Frame{})
		title, _ := SplitCardHead(c.Title, c.K8s, chars)
		if got := ui.Name(title); got != want[mode.name] {
			t.Errorf("[%s] the card head reads %q, want %q", mode.name, got, want[mode.name])
		}
	}
}
