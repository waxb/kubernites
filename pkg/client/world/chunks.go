package world

import (
	"hash/fnv"
	"image"
	"math"
	"runtime"
	"sort"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	maxChunks   = 96
	maxLandings = 4
	prefetchers = 3
)

type chunkKey struct{ cx, cy int }

type chunk struct {
	img    *ebiten.Image
	digest uint64
	used   int
}

type landCache struct {
	tiles    *view.TileKind
	wear     *uint16
	owned    string
	w, h     int
	land     *art.Land
	digests  map[chunkKey]uint64
	version  uint64
	chunks   map[chunkKey]*chunk
	frame    int
	view     image.Rectangle
	hb       *habitat
	hbSig    uint64
	flora    map[chunkKey]floraChunk
	blocking bool
	Flights  map[chunkKey]*flight `json:"-"`
	Queue    []chunkKey           `json:"-"`
}

func newLandCache() *landCache {
	return &landCache{digests: map[chunkKey]uint64{}, chunks: map[chunkKey]*chunk{}, flora: map[chunkKey]floraChunk{}, Flights: map[chunkKey]*flight{}}
}

func ownership(site view.Site) string {
	b := make([]byte, len(site.Islands))
	for i, is := range site.Islands {
		b[i] = '0'
		if is.Owned {
			b[i] = '1'
		}
	}
	return string(b)
}

func (c *landCache) sync(site view.Site) bool {
	var tiles *view.TileKind
	var wear *uint16
	if len(site.Tiles) > 0 {
		tiles = &site.Tiles[0]
	}
	if len(site.Wear) > 0 {
		wear = &site.Wear[0]
	}
	owned := ownership(site)
	if c.land != nil && tiles == c.tiles && wear == c.wear && owned == c.owned && site.Width == c.w && site.Height == c.h {
		return false
	}
	c.tiles, c.wear, c.owned, c.w, c.h = tiles, wear, owned, site.Width, site.Height
	c.land = LandOf(site)
	c.digests = chunkDigests(c.land)
	c.version = digestOf(c.digests)
	c.Queue = c.Queue[:0]
	for k := range c.digests {
		c.Queue = append(c.Queue, k)
	}
	sortKeys(c.Queue)
	return true
}

func sortKeys(keys []chunkKey) {
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].cy != keys[j].cy {
			return keys[i].cy < keys[j].cy
		}
		return keys[i].cx < keys[j].cx
	})
}

func digestOf(digests map[chunkKey]uint64) uint64 {
	keys := make([]chunkKey, 0, len(digests))
	for k := range digests {
		keys = append(keys, k)
	}
	sortKeys(keys)
	h := fnv.New64a()
	for _, k := range keys {
		d := digests[k]
		h.Write([]byte{byte(d), byte(d >> 8), byte(d >> 16), byte(d >> 24), byte(d >> 32), byte(d >> 40), byte(d >> 48), byte(d >> 56)})
	}
	return h.Sum64()
}

func chunkDigests(l *art.Land) map[chunkKey]uint64 {
	nx, ny := art.Chunks(l.W, l.H)
	out := make(map[chunkKey]uint64, nx*ny)
	buf := make([]byte, 0, 6*(art.ChunkTiles+2)*(art.ChunkTiles+2))
	for cy := 0; cy < ny; cy++ {
		for cx := 0; cx < nx; cx++ {
			buf = buf[:0]
			a := art.ChunkArea(cx, cy)
			for y := a.Min.Y - 1; y <= a.Max.Y; y++ {
				for x := a.Min.X - 1; x <= a.Max.X; x++ {
					buf = appendTile(buf, l, x, y)
				}
			}
			h := fnv.New64a()
			h.Write(buf)
			out[chunkKey{cx, cy}] = h.Sum64()
		}
	}
	return out
}

func appendTile(buf []byte, l *art.Land, x, y int) []byte {
	if x < 0 || y < 0 || x >= l.W || y >= l.H {
		return append(buf, 255)
	}
	i := y*l.W + x
	wild := byte(0)
	if l.Wild[i] {
		wild = 1
	}
	return append(buf, byte(l.Ground[i]), l.Height[i], l.Road[i], wild)
}

func (c *landCache) visible(cam render.Camera, screen image.Rectangle) []chunkKey {
	if c.land == nil {
		return nil
	}
	world := worldRect(cam, screen).Inset(-art.MaxLift)
	nx, ny := art.Chunks(c.land.W, c.land.H)
	out := []chunkKey{}
	for cy := 0; cy < ny; cy++ {
		for cx := 0; cx < nx; cx++ {
			if art.ChunkBounds(cx, cy).Overlaps(world) {
				out = append(out, chunkKey{cx, cy})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		si, sj := out[i].cx+out[i].cy, out[j].cx+out[j].cy
		if si != sj {
			return si < sj
		}
		return out[i].cx < out[j].cx
	})
	return out
}

func worldRect(cam render.Camera, screen image.Rectangle) image.Rectangle {
	z := cam.Scale()
	x0 := cam.X + float64(screen.Min.X-cam.OriginX)/z
	y0 := cam.Y + float64(screen.Min.Y-cam.OriginY)/z
	x1 := cam.X + float64(screen.Max.X-cam.OriginX)/z
	y1 := cam.Y + float64(screen.Max.Y-cam.OriginY)/z
	return image.Rect(int(math.Floor(x0)), int(math.Floor(y0)), int(math.Ceil(x1)), int(math.Ceil(y1)))
}

type flight struct {
	digest uint64
	frame  int
	done   chan *image.NRGBA
}

func paintOne(l *art.Land, k chunkKey) *image.NRGBA {
	if l.Empty(k.cx, k.cy) {
		return nil
	}
	return l.Chunk(k.cx, k.cy)
}

func (c *landCache) launch(k chunkKey) {
	f := &flight{digest: c.digests[k], frame: c.frame, done: make(chan *image.NRGBA, 1)}
	c.Flights[k] = f
	land := c.land
	go func() { f.done <- paintOne(land, k) }()
}

func (c *landCache) install(k chunkKey, img *image.NRGBA, digest uint64) {
	if old, ok := c.chunks[k]; ok && old.img != nil {
		old.img.Deallocate()
	}
	ch := &chunk{digest: digest, used: c.frame}
	if img != nil {
		ch.img = ebiten.NewImageFromImage(img)
	}
	c.chunks[k] = ch
}

func (c *landCache) collect() {
	landed := 0
	for k, f := range c.Flights {
		if landed >= maxLandings {
			return
		}
		select {
		case img := <-f.done:
			delete(c.Flights, k)
			if f.digest == c.digests[k] {
				c.install(k, img, f.digest)
				landed++
			}
		default:
		}
	}
}

func (c *landCache) settle(k chunkKey) bool {
	f, ok := c.Flights[k]
	if !ok || f.digest != c.digests[k] {
		return false
	}
	delete(c.Flights, k)
	c.install(k, <-f.done, f.digest)
	return true
}

func (c *landCache) images(keys []chunkKey) []*ebiten.Image {
	c.frame++
	c.collect()
	missing := []chunkKey{}
	for _, k := range keys {
		ch, ok := c.chunks[k]
		switch {
		case ok && ch.digest == c.digests[k]:
		case ok:
			c.refresh(k)
		case !c.settle(k):
			missing = append(missing, k)
		}
	}
	for i, img := range paintChunks(c.land, missing) {
		c.install(missing[i], img, c.digests[missing[i]])
	}
	out := make([]*ebiten.Image, len(keys))
	for i, k := range keys {
		ch := c.chunks[k]
		ch.used = c.frame
		out[i] = ch.img
	}
	c.evict()
	if !c.blocking {
		c.prefetch()
	}
	return out
}

func (c *landCache) refresh(k chunkKey) {
	if c.blocking {
		c.install(k, paintOne(c.land, k), c.digests[k])
		return
	}
	f, ok := c.Flights[k]
	switch {
	case !ok || f.digest != c.digests[k]:
		c.launch(k)
	case f.frame < c.frame:
		c.settle(k)
	}
}

func (c *landCache) prefetch() {
	for len(c.Queue) > 0 && len(c.Flights) < prefetchers && len(c.chunks)+len(c.Flights) < maxChunks {
		k := c.Queue[0]
		c.Queue = c.Queue[1:]
		if ch, ok := c.chunks[k]; ok && ch.digest == c.digests[k] {
			continue
		}
		if _, ok := c.Flights[k]; !ok {
			c.launch(k)
		}
	}
}

func paintChunks(l *art.Land, keys []chunkKey) []*image.NRGBA {
	out := make([]*image.NRGBA, len(keys))
	next := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < min(runtime.GOMAXPROCS(0), len(keys)); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				out[i] = paintOne(l, keys[i])
			}
		}()
	}
	for i := range keys {
		next <- i
	}
	close(next)
	wg.Wait()
	return out
}

func (c *landCache) evict() {
	if len(c.chunks) <= maxChunks {
		return
	}
	keys := make([]chunkKey, 0, len(c.chunks))
	for k := range c.chunks {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return c.chunks[keys[i]].used < c.chunks[keys[j]].used })
	for _, k := range keys[:len(keys)-maxChunks] {
		if c.chunks[k].used == c.frame {
			break
		}
		if img := c.chunks[k].img; img != nil {
			img.Deallocate()
		}
		delete(c.chunks, k)
	}
}
