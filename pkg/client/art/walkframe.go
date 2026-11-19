package art

import (
	"image"
	"runtime"
	"strings"
	"sync"
)

const (
	walkerPadX      = 6
	walkerPadTop    = 4
	walkerPadBottom = 2
	walkerFeetX     = 8
	walkerFeetY     = 22
	walkerMargin    = 1
	walkerFrameW    = WalkerWidth + 2*walkerPadX
	walkerFrameH    = WalkerHeight + walkerPadTop + walkerPadBottom
	walkerFrameAX   = walkerFeetX + walkerPadX
	walkerFrameAY   = walkerFeetY + walkerPadTop
)

func newWalkerBody() *canvas {
	return newFramedCanvas(walkerFrameW, walkerFrameH, walkerPadX, walkerPadTop)
}

func finishWalker(c, body *canvas) {
	shadowless := newCanvas(body.w, body.h)
	for y := 0; y < body.h; y++ {
		for x := 0; x < body.w; x++ {
			if col := body.img.NRGBAAt(x, y); col.A > 200 {
				shadowless.dot(x, y, col)
			}
		}
	}
	shadowless.outline(ink)
	c.paste(body, 0, 0, false)
	c.paste(shadowless, 0, 0, false)
}

func solidBox(img *image.NRGBA) image.Rectangle {
	box := image.Rectangle{}
	for y := 0; y < img.Rect.Dy(); y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			if img.NRGBAAt(x, y).A > 0 {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return box
}

func trimmedEntry(e entry) entry {
	c := newCanvas(e.w, e.h)
	e.paint(c)
	box := solidBox(c.img)
	if box.Empty() {
		return e
	}
	anchor := image.Rect(e.ax, e.ay, e.ax+1, e.ay+1)
	box = box.Inset(-walkerMargin).Union(anchor).Intersect(c.img.Rect)
	e.w, e.h, e.ax, e.ay = box.Dx(), box.Dy(), e.ax-box.Min.X, e.ay-box.Min.Y
	e.paint = func(dst *canvas) { dst.paste(c, -box.Min.X, -box.Min.Y, false) }
	return e
}

func trimWalkers(b *atlasBuilder) {
	jobs := make(chan int, len(b.entries))
	for i, e := range b.entries {
		if strings.HasPrefix(e.name, "walker/") {
			jobs <- i
		}
	}
	close(jobs)
	var wg sync.WaitGroup
	for w := 0; w < runtime.GOMAXPROCS(0); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				b.entries[i] = trimmedEntry(b.entries[i])
			}
		}()
	}
	wg.Wait()
}
