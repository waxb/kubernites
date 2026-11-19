package gfx

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
)

func TestPortSpritesPaintOnDemandWithMasks(t *testing.T) {
	lib := &Library{index: map[string]entry{}, lazy: newLazySprites()}
	for _, name := range []string{
		art.PortLook{Kind: "house", Level: 3, Age: 2}.Name(),
		art.PortLook{Kind: "deck", Level: 1, Age: 0, Variant: 2}.Name(),
		art.FerryLook{Age: 1, Dir: "sw", Frame: 2}.Name(),
	} {
		if !lib.Has(name) {
			t.Fatalf("%s is not in the library", name)
		}
		if m, ok := lib.Mask(name); !ok || m.Empty() {
			t.Fatalf("%s has no pick mask", name)
		}
	}
	for _, name := range []string{"port/house/9/0/0", "port/ferry/0/up/0", "port/nothing"} {
		if lib.Has(name) {
			t.Fatalf("%s resolved", name)
		}
	}
}

func TestPortSpritesWaitForTheWorkerWhenAsync(t *testing.T) {
	lib := &Library{index: map[string]entry{}, lazy: newLazySprites()}
	lib.lazy.async = true
	lib.lazy.jobs = make(chan string, 1)
	name := art.PortLook{Kind: "lighthouse", Level: 4, Age: 3}.Name()
	if lib.Has(name) {
		t.Fatal("an async library painted on the draw thread")
	}
	select {
	case got := <-lib.lazy.jobs:
		if got != name {
			t.Fatalf("queued %q", got)
		}
	default:
		t.Fatal("nothing was queued for the painter")
	}
}
