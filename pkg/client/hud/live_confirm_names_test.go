package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestTheOpenThisClusterDialogNamesKubectlInNamesMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	text := strings.Join(liveConfirmBody(), " ")
	if strings.Contains(text, "Speaking Stone") || !strings.Contains(text, "kubectl") {
		t.Fatalf("the dialog reads %q", text)
	}
}
