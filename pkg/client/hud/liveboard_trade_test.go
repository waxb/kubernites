package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestTheLiveCrewBoardNamesTheImageATradeAsTheCrewCardDoes(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	card := strings.Join(func() []string {
		var out []string
		for _, l := range DeploymentCard(model.CrewChain{Name: "gate-controller", Image: "registry.example/gate-controller:1.4.2"}).Lines {
			out = append(out, l.Text)
		}
		return out
	}(), "\n")
	for _, chain := range []model.CrewChain{{Image: "registry.example/gate-controller:1.4.2"}, {}} {
		var said []string
		for _, p := range liveImagePieces(chain) {
			said = append(said, p.Text)
		}
		text := strings.Join(said, " ")
		if strings.Contains(strings.ToLower(text), "soulstone") || !strings.Contains(strings.ToLower(text), "trade") {
			t.Errorf("the live crew board says %q; the lexicon and the crew card (%q) call an image a trade", text, card)
		}
	}
}
