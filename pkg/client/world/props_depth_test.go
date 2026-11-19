package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func playerPlaza(t *testing.T) (Scene, render.Camera, *Renderer) {
	t.Helper()
	sc, _ := sceneFor(t, .5)
	sc.Pods, sc.Transients, sc.Ghost, sc.Selected = nil, nil, nil, ""
	keep := view.Structure{ID: "keep-3", Kind: view.StructKeep, X: 30, Y: 65, W: 3, H: 3, State: view.StateDark}
	keep.Object.Name = "lone-tower"
	chancery := view.Structure{ID: "chancery-5", Kind: view.StructChancery, X: 34, Y: 65, W: 2, H: 2, State: view.StateReady}
	quartermaster := view.Structure{ID: "quartermaster-6", Kind: view.StructQuartermaster, X: 37, Y: 64, W: 2, H: 2, State: view.StateConstruction, Progress: .47}
	stewards := view.Structure{ID: "stewards-hall-7", Kind: view.StructStewards, X: 33, Y: 68, W: 2, H: 2, State: view.StateReady}
	sc.Site.Structures = []view.Structure{keep, chancery, quartermaster, stewards}
	cam := render.NewCamera(480, 60).CenterOn(35, 66, 960, 540)
	return sc, cam, NewRenderer(nil)
}

func coveringStructure(site view.Site, at Vec) (string, bool) {
	t := TileOf(at)
	for _, s := range site.Structures {
		if Covers(s, t.X, t.Y) {
			return s.ID, true
		}
	}
	return "", false
}

func checkStandsClearAndInFront(t *testing.T, name string, prop Drawable, foot Vec, items []Drawable, sc Scene, zoom float64) {
	t.Helper()
	if id, covered := coveringStructure(sc.Site, foot); covered {
		t.Fatalf("the %s stands at %v, inside %s", name, foot, id)
	}
	all := append(append([]Drawable{}, items...), prop)
	liftWalkers(all, zoom)
	lifted := all[len(all)-1]
	for _, b := range liftBlocks(items) {
		if InFront(foot, b.lo, b.hi) && overlapsFigure(b.rect, prop.stand, zoom) && lifted.Depth <= b.depth {
			t.Fatalf("the %s at %v stands in front of the building %v-%v yet is drawn behind it", name, foot, b.lo, b.hi)
		}
	}
}

func TestTheChancerysBoardAndTheKeepsWardStandClearOfOtherBuildingsAndInFrontOfThem(t *testing.T) {
	sc, cam, r := playerPlaza(t)
	zoom := cam.Scale()
	items := r.structureItems(cam, sc)
	board := r.boardItems(nil, cam, sc, r.livingOf())
	if len(board) != 1 {
		t.Fatalf("the board is not drawn: %d items", len(board))
	}
	foot, ok := boardSpot(sc.Site, r.hamletOf())
	if !ok {
		t.Fatal("the board found no spot beside the Chancery")
	}
	checkStandsClearAndInFront(t, "Chancery's board", board[0], foot, items, sc, zoom)
	keep := sc.Site.Structures[0]
	ward := r.wardItems(nil, cam, sc, map[string]bool{keep.Object.Name: true})
	if len(ward) != 1 {
		t.Fatalf("the ward is not drawn: %d items", len(ward))
	}
	checkStandsClearAndInFront(t, "Keep's ward", ward[0], WardSpot(sc.Site, keep), items, sc, zoom)
}
