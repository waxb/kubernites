package world

import (
	"github.com/waxb/kubernites/pkg/client/ui"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func TestCauseLabelsSayWhatFelledTheWorker(t *testing.T) {
	cases := map[string]string{
		view.TagOOMKilled: "BOX TOO BIG", view.TagThrottled: "HUNGRY", view.TagEvicted: "BANISHED",
		view.TagCrashLoop: "GUTTERING", view.TagImagePull: "NEVER ARRIVED", view.TagUnhoused: "UNHOUSED",
	}
	for tag, want := range cases {
		if got := CauseLabel(tag); got != want {
			t.Errorf("CauseLabel(%s) = %q, want %q", tag, got, want)
		}
	}
	if CauseLabel("strange") != "STRANGE" {
		t.Error("unknown tags speak their own name")
	}
}

func TestCauseRectSitsAboveTheHut(t *testing.T) {
	r := CauseRect("BOX TOO BIG", 200, 300, 1)
	if r.Y+r.H > 300-causeLift || r.X+r.W/2 < 195 || r.X+r.W/2 > 205 {
		t.Errorf("tag %+v not centred above the hut", r)
	}
	big := CauseRect("BOX TOO BIG", 200, 300, 2)
	if big.Y >= r.Y {
		t.Error("the tag rises with the zoom")
	}
}

func causeRenderer() *Renderer {
	r := NewRenderer(nil)
	r.Homes = map[string]Vec{"porters-a": {12, 12}, "porters-b": {12.2, 12}, "clerks-c": {20, 14}}
	return r
}

func TestCauseMarksFollowTheirWorkersAndNeverStack(t *testing.T) {
	r := causeRenderer()
	cam := render.NewCamera(480, 60).CenterOn(15, 13, 960, 480)
	sc := Scene{Site: landSite(), Causes: []view.CauseTag{
		{Pod: "default/porters-a", Tag: view.TagOOMKilled, Text: "OOMKilled: a large box"},
		{Pod: "default/porters-b", Tag: view.TagOOMKilled, Text: "OOMKilled: again"},
		{Pod: "default/clerks-c", Tag: view.TagThrottled, Node: "lone-tower", Text: "Throttled"},
		{Pod: "default/porters-a", Tag: view.TagThrottled, Node: "lone-tower", Text: "Throttled too"},
		{Pod: "default/missing", Tag: view.TagEvicted, Text: "Evicted"},
	}}
	marks := r.CauseMarks(cam, sc)
	if len(marks) != 2 {
		t.Fatalf("marks = %+v", marks)
	}
	for i, a := range marks {
		for _, b := range marks[i+1:] {
			if Overlaps(a.Rect, b.Rect) {
				t.Errorf("tags %s and %s overlap", a.Pod, b.Pod)
			}
		}
	}
	if marks[0].Label != "BOX TOO BIG" || marks[0].Text == "" || marks[1].Tag != view.TagThrottled || !marks[1].Strain || marks[0].Strain {
		t.Errorf("marks = %+v", marks)
	}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.DrawCauseTags(dst, cam, sc)
	far := cam.Pan(5000, 0)
	if len(r.CauseMarks(far, sc)) != 0 {
		t.Error("off-screen workers carry no tag")
	}
}

func TestACauseTagsTailKeepsClearOfThePanelsToo(t *testing.T) {
	r := causeRenderer()
	cam := render.NewCamera(480, 60).CenterOn(15, 13, 960, 480)
	sc := Scene{Site: landSite(), Causes: []view.CauseTag{{Pod: "default/porters-a", Tag: view.TagOOMKilled, Text: "OOMKilled: a large box"}}}
	free := r.CauseMarks(cam, sc)
	if len(free) != 1 {
		t.Fatalf("no tag to move: %+v", free)
	}
	b := free[0].Rect
	sc.Panels = []ui.Rect{{X: 0, Y: b.Y + b.H + 1, W: 960, H: 200}}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.DrawCauseTags(dst, cam, sc)
	drawn := 0
	for _, d := range r.Drawn() {
		if d.Kind != DrawnTag {
			continue
		}
		drawn++
		for _, p := range sc.Panels {
			if Overlaps(d.Rect, p) {
				t.Fatalf("the tag %+v, its tail included, reaches into the panel %+v", d.Rect, p)
			}
		}
	}
	sc.Panels = []ui.Rect{{X: 0, Y: b.Y + b.H + causeTail + 1, W: 960, H: 200}}
	r.DrawCauseTags(dst, cam, sc)
	if len(r.DrawnCauseMarks()) != 1 {
		t.Fatalf("a tag whose whole shape fits above the panel is not drawn (it waited %d times before)", drawn)
	}
}
