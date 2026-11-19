package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestTheScriptoriumFooterSaysItsLineWholeInEachMode(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, namer := range []func(string) string{nil, names.Kubernetes(64)} {
		ui.SetNamer(namer)
		room := liveEditFootChars()
		for _, line := range []string{readOnlySend(), liveEditSendHint()} {
			if n := len([]rune(line)); n > room {
				t.Errorf("names %v: %q is %d characters, the footer has room for %d", ui.Naming(), line, n, room)
			}
		}
	}
}
