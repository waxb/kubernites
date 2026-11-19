package art

import (
	"image"
	"image/draw"
	"runtime"
	"sort"
	"sync"
)

const (
	atlasWidth = 1024
	padding    = 1
)

type entry struct {
	name   string
	w, h   int
	ax, ay int
	slice  int
	paint  func(c *canvas)
}

type atlasBuilder struct {
	entries []entry
}

func (b *atlasBuilder) add(name string, w, h, ax, ay int, paint func(c *canvas)) {
	b.entries = append(b.entries, entry{name: name, w: w, h: h, ax: ax, ay: ay, paint: paint})
}

func (b *atlasBuilder) addSliced(name string, w, h, slice int, paint func(c *canvas)) {
	b.entries = append(b.entries, entry{name: name, w: w, h: h, ax: w / 2, ay: h / 2, slice: slice, paint: paint})
}

type placement struct {
	x, y int
}

func (b *atlasBuilder) packOrder() []int {
	order := make([]int, len(b.entries))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool { return b.entries[order[i]].h > b.entries[order[j]].h })
	return order
}

func (b *atlasBuilder) layout() ([]placement, int) {
	out := make([]placement, len(b.entries))
	x, y, shelf := padding, padding, 0
	for _, i := range b.packOrder() {
		e := b.entries[i]
		if x+e.w+padding > atlasWidth {
			x, y, shelf = padding, y+shelf+padding, 0
		}
		out[i] = placement{x: x, y: y}
		x += e.w + padding
		shelf = max(shelf, e.h)
	}
	return out, y + shelf + padding
}

func (b *atlasBuilder) build() Atlas {
	places, height := b.layout()
	img := image.NewNRGBA(image.Rect(0, 0, atlasWidth, max(height, 1)))
	flags := b.paintAll(img, places)
	regions := make(map[string]Region, len(b.entries))
	for i, e := range b.entries {
		p := places[i]
		regions[e.name] = Region{X: p.x, Y: p.y, W: e.w, H: e.h, AnchorX: e.ax, AnchorY: e.ay, Slice: e.slice, Flags: flagsOf(e.name, flags[i], e.ax, e.ay)}
	}
	return Atlas{Image: img, Regions: regions}
}

func (b *atlasBuilder) paintAll(img *image.NRGBA, places []placement) [][]FlagMark {
	flags := make([][]FlagMark, len(b.entries))
	jobs := make(chan int, len(b.entries))
	for i := range b.entries {
		jobs <- i
	}
	close(jobs)
	var wg sync.WaitGroup
	for w := 0; w < min(runtime.GOMAXPROCS(0), max(len(b.entries), 1)); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				flags[i] = paintEntry(img, places[i], b.entries[i])
			}
		}()
	}
	wg.Wait()
	return flags
}

func paintEntry(img *image.NRGBA, p placement, e entry) []FlagMark {
	c := newCanvas(e.w, e.h)
	e.paint(c)
	draw.Draw(img, image.Rect(p.x, p.y, p.x+e.w, p.y+e.h), c.img, image.Point{}, draw.Src)
	return c.flags
}
