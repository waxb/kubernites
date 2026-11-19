package gfx

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheBellAndTheShuttersDrawTheirOwnPageIcons(t *testing.T) {
	blank := art.PageIconName("blank")
	bell, shutters := PageIconName(view.BellPage), PageIconName(view.WatchPage)
	if bell == blank || shutters == blank || bell == shutters {
		t.Fatalf("the Bell draws %q and the Shutters %q", bell, shutters)
	}
}
