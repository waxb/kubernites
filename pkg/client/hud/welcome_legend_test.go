package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestTheWhatIsWhatLegendExplainsEachNameInNamesMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	for _, row := range WelcomeLegend {
		term, meaning := ui.Name(row[0]), ui.Name(legendMeaning(row))
		if strings.Contains(strings.ToLower(meaning), strings.ToLower(term)) {
			t.Errorf("%s reads %q: it only repeats the name", term, meaning)
		}
	}
}
