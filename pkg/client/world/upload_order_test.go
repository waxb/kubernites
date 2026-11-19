package world

import (
	"crypto/sha256"
	"fmt"
	"image"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
)

func recordUploads(t *testing.T) *[]string {
	t.Helper()
	saved := newImage
	t.Cleanup(func() { newImage = saved })
	got := &[]string{}
	newImage = func(img image.Image) *ebiten.Image {
		nrgba, ok := img.(*image.NRGBA)
		if !ok {
			t.Fatalf("uploaded a %T, want *image.NRGBA", img)
		}
		*got = append(*got, fmt.Sprintf("%v %x", nrgba.Bounds(), sha256.Sum256(nrgba.Pix)))
		return nil
	}
	return got
}

func TestGardenBedsAreUploadedInTheSameOrderEveryTime(t *testing.T) {
	pods := []model.PodView{}
	for _, n := range []string{"ash", "birch", "cedar", "elm", "fir", "hazel", "larch", "oak"} {
		pods = append(pods, householdView("porters-7d9f-"+n, "lone-tower", porterFolk("porter")))
	}
	sc, cam, r := peopleScene(t, 0, pods...)
	bound, kinds := map[string]bool{}, map[uint32]bool{}
	for name, s := range r.hamlet.SpotList {
		if s.Sown && r.hamlet.occAt(s.Garden) == occGarden {
			bound[name], kinds[model.Hash(name)%art.GardenKinds] = true, true
		}
	}
	if len(kinds) < 2 {
		t.Fatalf("%d gardens of %d kinds; the test needs two kinds", len(bound), len(kinds))
	}
	r.livingOf().bound = bound
	got := recordUploads(t)
	var first []string
	for pass := 0; pass < 40; pass++ {
		*got = (*got)[:0]
		r.livingOf().pieces = map[string]livingPiece{}
		r.gardenItems(nil, cam, sc, r.hamlet)
		if len(*got) < 2 {
			t.Fatalf("pass %d uploaded %d garden beds, want two kinds on screen", pass, len(*got))
		}
		if first == nil {
			first = slices.Clone(*got)
			continue
		}
		if !slices.Equal(*got, first) {
			t.Fatalf("pass %d uploaded the garden beds in another order:\n%v\nthe first pass:\n%v", pass, *got, first)
		}
	}
}

func TestThePropsAreUploadedInTheSameOrderEveryTime(t *testing.T) {
	got := recordUploads(t)
	var first []string
	for pass := 0; pass < 40; pass++ {
		*got = (*got)[:0]
		buildProps()
		if len(*got) != len(propArts) {
			t.Fatalf("pass %d uploaded %d props, want %d", pass, len(*got), len(propArts))
		}
		if first == nil {
			first = slices.Clone(*got)
			continue
		}
		if !slices.Equal(*got, first) {
			t.Fatalf("pass %d uploaded the props in another order:\n%v\nthe first pass:\n%v", pass, *got, first)
		}
	}
}
