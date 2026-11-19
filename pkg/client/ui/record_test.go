package ui

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestRecordedTextKnowsItsRegion(t *testing.T) {
	dst := ebiten.NewImage(200, 60)
	defer dst.Deallocate()
	var runs []TextRun
	stop := RecordText(func(r TextRun) { runs = append(runs, r) })
	var f Font
	InRegion("outer", func() {
		f.Draw(dst, "first", 0, 0, 1, Palette.Text)
		InRegion("inner", func() { f.Draw(dst, "second", 0, 12, 1, Palette.Text) })
		f.Draw(dst, "third", 0, 24, 1, Palette.Text)
	})
	f.Draw(dst, "fourth", 0, 36, 1, Palette.Text)
	stop()
	want := map[string]string{"first": "outer", "second": "inner", "third": "outer", "fourth": ""}
	if len(runs) != len(want) {
		t.Fatalf("runs = %+v", runs)
	}
	for _, r := range runs {
		if want[r.Text] != r.Region {
			t.Errorf("%q drawn in %q, want %q", r.Text, r.Region, want[r.Text])
		}
	}
}

func TestTextTheScreenClipsAwayIsNotRecorded(t *testing.T) {
	dst := ebiten.NewImage(200, 60)
	defer dst.Deallocate()
	view := dst.SubImage(image.Rect(0, 0, 100, 30)).(*ebiten.Image)
	var runs []TextRun
	stop := RecordText(func(r TextRun) { runs = append(runs, r) })
	var f Font
	f.Draw(view, "seen", 4, 4, 1, Palette.Text)
	f.Draw(view, "hidden", 4, 40, 1, Palette.Text)
	f.Draw(view, "also hidden", 150, 4, 1, Palette.Text)
	stop()
	if len(runs) != 1 || runs[0].Text != "seen" {
		t.Fatalf("the look reads %+v, want only the text the clip shows", runs)
	}
}
