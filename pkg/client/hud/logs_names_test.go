package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheLogsPanelReadsTheSameWordsAsTheStone(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	log := "porter:1.0 listening for parcels on :8080\nporter:1.0 shouldering the first parcel\n"
	h := New(nil, Backend{Inspect: func(cmd string) view.CommandResult { return view.CommandResult{Output: log, OK: true} }})
	h.OpenDetails(Selection{Kind: "hearth", ID: "default/hello", Pod: model.PodView{Name: "hello", Namespace: "default"}}, DetailLogs)
	text := strings.Join(h.Details().Lines, "\n")
	if strings.Contains(text, "parcel") || !strings.Contains(text, "listening for web requests on :8080") || !strings.Contains(text, "$ kubectl logs hello") {
		t.Errorf("the LOGS panel in NAMES mode reads\n%s\nwhile the Stone and the event log say 'web request'", text)
	}
}
