package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestTheStoneTabShowsItsHintLinesWhole(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, namer := range []func(string) string{nil, names.Kubernetes(64)} {
		ui.SetNamer(namer)
		h := New(nil, Backend{})
		rows := h.stoneSummaryRows(WellBody())
		if len(rows) == 0 {
			t.Fatal("the Stone tab shows nothing")
		}
		joined := ""
		for _, r := range rows {
			if strings.HasSuffix(r.Text, "...") {
				t.Errorf("the Stone tab cuts a hint line: %q (founding-citadel: \"Speak speakstc...\", \"Tab Tab cycles the c...\")", r.Text)
			}
			joined += strings.TrimSpace(r.Text) + " "
		}
		if !strings.Contains(joined, "Tab Tab cycles the choices.") {
			t.Errorf("the Stone tab's last hint is not whole: %q", joined)
		}
	}
}
