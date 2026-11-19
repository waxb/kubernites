package world

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
)

func campers(n int) []model.PodView {
	out := make([]model.PodView, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, model.PodView{Namespace: "default", Name: fmt.Sprintf("camp-%02d", i), Sprite: "porter", Tier: 1, State: model.HearthPending})
	}
	return out
}

func TestPlanCampsCapsTheVisibleCamps(t *testing.T) {
	sc, _ := sceneFor(t, .5)
	pods := append(campers(10), model.PodView{Namespace: "default", Name: "gone", State: model.HearthBanished, Terminating: true})
	plan := PlanCamps(sc.Site, pods, map[string]Vec{})
	if len(plan.Shown) != MaxCampsShown+1 || plan.Hidden != 10-MaxCampsShown {
		t.Fatalf("shown %d hidden %d", len(plan.Shown), plan.Hidden)
	}
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("camp-%02d", i)
		spot, ok := plan.Spots[name]
		switch {
		case !ok:
			t.Fatalf("%s has no spot", name)
		case i < MaxCampsShown && (!plan.Shown[name] || spot != PendingSpot(sc.Site, i)):
			t.Fatalf("%s should camp at spot %d, got %v shown=%v", name, i, spot, plan.Shown[name])
		case i >= MaxCampsShown && (plan.Shown[name] || spot != plan.Post):
			t.Fatalf("%s should wait at the signpost %v, got %v", name, plan.Post, spot)
		}
	}
	if !plan.Shown["gone"] {
		t.Fatal("a fallen hut is not waiting for room and must stay visible")
	}
	for i := 0; i < MaxCampsShown; i++ {
		if Dist(plan.Post, PendingSpot(sc.Site, i)) < 1 {
			t.Fatalf("signpost %v crowds camp %d", plan.Post, i)
		}
	}
}

func TestFewCampsNeedNoSignpost(t *testing.T) {
	sc, _ := sceneFor(t, .5)
	plan := PlanCamps(sc.Site, campers(MaxCampsShown), map[string]Vec{})
	if plan.Hidden != 0 || len(plan.Shown) != MaxCampsShown {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestHearthItemsDrawAtMostFourCampsAndASignpost(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	sc.Pods = append(sc.Pods, campers(12)...)
	r := NewRenderer(gfx.Load(art.Build()))
	items := r.HearthItems(cam, sc)
	camps, posts := 0, 0
	for _, it := range items {
		switch {
		case it.Hit != nil && it.Hit.Kind == HitSignpost:
			posts++
		case it.Hit != nil && it.Hit.Pod.State == model.HearthPending:
			camps++
		}
	}
	if camps != MaxCampsShown || posts != 1 {
		t.Fatalf("drew %d camps and %d signposts", camps, posts)
	}
	if v, ok := r.Home("camp-11"); !ok || v != PlanCamps(sc.Site, campers(12), map[string]Vec{}).Post {
		t.Fatalf("a hidden camper's home = %v, %v", v, ok)
	}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.Draw(dst, cam, sc)
}

func TestSignpostLabelCountsTheHidden(t *testing.T) {
	if got := SignpostLabel(7); got != "+7 waiting for room" {
		t.Fatalf("label = %q", got)
	}
}
