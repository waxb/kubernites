package ui

import (
	"image"
	"strings"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
)

type TextRun struct {
	Text    string
	Spelled string
	Rect    Rect
	Region  string
}

var (
	watcher  atomic.Pointer[func(string)]
	locator  atomic.Pointer[func(string, func() (Rect, bool))]
	recorder atomic.Pointer[func(TextRun)]
	region   atomic.Pointer[string]
)

func InRegion(name string, draw func()) {
	if recorder.Load() == nil {
		draw()
		return
	}
	prev := region.Swap(&name)
	defer region.Store(prev)
	draw()
}

func currentRegion() string {
	if r := region.Load(); r != nil {
		return *r
	}
	return ""
}

func Recording() bool { return recorder.Load() != nil }

func RecordText(f func(TextRun)) func() {
	recorder.Store(&f)
	return func() { recorder.Store(nil) }
}

func WatchText(f func(string)) {
	if f == nil {
		watcher.Store(nil)
		return
	}
	watcher.Store(&f)
}

func WatchTextAt(f func(string, func() (Rect, bool))) {
	if f == nil {
		locator.Store(nil)
		return
	}
	locator.Store(&f)
}

func record(dst *ebiten.Image, text string, x, y, scale int) {
	if w := watcher.Load(); w != nil {
		(*w)(text)
	}
	if l := locator.Load(); l != nil {
		(*l)(text, func() (Rect, bool) {
			r, ok := inkRect(text, x, y, max(scale, 1))
			return r, ok && visibleIn(dst, r)
		})
	}
	rec := recorder.Load()
	if rec == nil {
		return
	}
	if r, ok := inkRect(text, x, y, max(scale, 1)); ok && visibleIn(dst, r) {
		(*rec)(TextRun{Text: strings.TrimSpace(Plain(text)), Spelled: strings.TrimSpace(Spelled(text)), Rect: r, Region: currentRegion()})
	}
}

func Caption(text string, r Rect) {
	rec := recorder.Load()
	if rec == nil || text == "" {
		return
	}
	(*rec)(TextRun{Text: text, Spelled: text, Rect: r, Region: currentRegion()})
}

func inkRect(text string, x, y, scale int) (Rect, bool) {
	first, last, left, right := -1, -1, -1, 0
	for i, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimRight(line, " ")
		lead := cells(trimmed) - cells(strings.TrimLeft(trimmed, " "))
		if lead == cells(trimmed) {
			continue
		}
		if first < 0 {
			first = i
		}
		last = i
		if left < 0 || lead < left {
			left = lead
		}
		right = max(right, cells(trimmed))
	}
	if first < 0 {
		return Rect{}, false
	}
	return Rect{X: x + left*Advance*scale, Y: y + first*LineHeight*scale,
		W: ((right-left)*Advance - 1) * scale, H: ((last-first)*LineHeight + GlyphHeight) * scale}, true
}

func visibleIn(dst *ebiten.Image, r Rect) bool {
	if dst == nil {
		return true
	}
	return image.Pt(r.X+r.W/2, r.Y+r.H/2).In(dst.Bounds())
}
