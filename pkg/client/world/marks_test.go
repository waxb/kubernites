package world

import (
	"fmt"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
)

func TestOnlyTheNearestMoodsShowTheirMarks(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	cam = cam.CenterOn(24, 24, ViewWidth, ViewHeight)
	figs := []Figure{}
	for i := range 30 {
		mood := []string{MoodRest, MoodSick, MoodStrained, MoodBrisk}[i%4]
		pos := Vec{24 + float64(i)*.4, 24 + float64(i)*.4}
		figs = append(figs, Figure{Pod: model.PodView{Namespace: "default", Name: fmt.Sprintf("p-%02d", i)}, Mood: mood, Pose: Pose{Pos: pos}})
	}
	got := MarkNearest(cam, sc.Site, figs)
	marked, farthest, nearestSkipped := 0, -1, 99
	for i, f := range got {
		switch {
		case f.mark && !markable(f.Mood):
			t.Errorf("%s (%s) shows a mark", f.Pod.Name, f.Mood)
		case f.mark:
			marked++
			farthest = max(farthest, i)
		case markable(f.Mood):
			nearestSkipped = min(nearestSkipped, i)
		}
	}
	if marked != MaxMarks {
		t.Fatalf("marks = %d, want %d", marked, MaxMarks)
	}
	if farthest > nearestSkipped {
		t.Errorf("a far mark (%d) shows while a nearer one (%d) hides", farthest, nearestSkipped)
	}
	if n := len(MarkNearest(cam, sc.Site, nil)); n != 0 {
		t.Errorf("no figures, no marks: %d", n)
	}
}
