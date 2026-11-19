package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestTheCrewBoardDrawsItsYardAlikeInBothModes(t *testing.T) {
	defer ui.SetNamer(nil)
	tracks := map[string]ui.Rect{}
	for _, mode := range []string{"story", "names"} {
		if mode == "names" {
			ui.SetNamer(names.Kubernetes(64))
		} else {
			ui.SetStoryNamer(lexicon.StoryText)
		}
		f := boardFrame()
		_, v := openBoard(t, f)
		tracks[mode] = v.Track
	}
	if tracks["story"].W != tracks["names"].W {
		t.Fatalf("the slider track is %d px in the story and %d px in NAMES mode, so the yard draws huts differently: %+v", tracks["story"].W, tracks["names"].W, tracks)
	}
}
