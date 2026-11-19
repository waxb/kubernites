package world

import (
	"image"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

func islandSite(n int) view.Site {
	site := wildSite(n, view.TileWater)
	for y := n / 4; y < n*3/4; y++ {
		for x := n / 4; x < n*3/4; x++ {
			site.Tiles[y*n+x] = view.TileGrass
		}
	}
	return site
}

func cloneSite(site view.Site) view.Site {
	next := site
	next.Tiles = append([]view.TileKind(nil), site.Tiles...)
	next.Wear = append([]uint16(nil), site.Wear...)
	next.Islands = append([]view.Island(nil), site.Islands...)
	return next
}

func TestLandCacheNoticesRefreshes(t *testing.T) {
	c := newLandCache()
	site := islandSite(40)
	if !c.sync(site) || c.sync(site) {
		t.Fatal("sync reports the first site once")
	}
	before := c.digests
	worn := cloneSite(site)
	worn.Wear[20*40+20] = 9000
	if !c.sync(worn) {
		t.Fatal("a refreshed site is noticed")
	}
	changed := 0
	for k, d := range c.digests {
		if before[k] != d {
			changed++
		}
	}
	if changed == 0 || changed > 4 {
		t.Errorf("one worn tile changed %d chunk digests", changed)
	}
	chartered := cloneSite(worn)
	chartered.Tiles = worn.Tiles
	chartered.Wear = worn.Wear
	chartered.Islands[0].Owned = false
	if !c.sync(chartered) {
		t.Error("ownership change is noticed")
	}
}

func waitFlights(c *landCache) {
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) && len(c.Flights) > 0 {
		time.Sleep(5 * time.Millisecond)
		c.collect()
	}
}

func TestChunkImagesRepaintInTheBackground(t *testing.T) {
	c := newLandCache()
	site := islandSite(96)
	c.sync(site)
	keys := []chunkKey{{0, 0}, {2, 2}, {3, 3}}
	first := c.images(keys)
	if first[0] != nil || first[1] == nil {
		t.Fatal("open sea is empty and the island is painted")
	}
	waitFlights(c)
	worn := cloneSite(site)
	worn.Wear[40*96+40] = 9000
	c.sync(worn)
	stale := c.images(keys)
	if stale[1] != first[1] {
		t.Fatal("the stale chunk is shown while it repaints")
	}
	if _, ok := c.Flights[chunkKey{2, 2}]; !ok {
		t.Fatal("the repaint is in flight")
	}
	fresh := c.images(keys)
	if fresh[1] == first[1] || c.chunks[chunkKey{2, 2}].digest != c.digests[chunkKey{2, 2}] {
		t.Fatal("the repaint lands on the next frame")
	}
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) && len(c.Queue)+len(c.Flights) > 0 {
		c.images(keys)
		time.Sleep(5 * time.Millisecond)
	}
	nx, ny := art.Chunks(96, 96)
	if len(c.chunks) != nx*ny {
		t.Errorf("prefetch cached %d of %d chunks", len(c.chunks), nx*ny)
	}
}

func TestChunkCacheEvictsTheOldest(t *testing.T) {
	c := newLandCache()
	c.sync(islandSite(16))
	for i := 0; i < maxChunks+5; i++ {
		c.chunks[chunkKey{100 + i, 0}] = &chunk{used: i}
	}
	c.frame = maxChunks + 10
	c.evict()
	if len(c.chunks) > maxChunks {
		t.Errorf("cache holds %d chunks", len(c.chunks))
	}
	if _, ok := c.chunks[chunkKey{100, 0}]; ok {
		t.Error("the oldest chunk survived")
	}
}

func TestVisibleChunksFollowTheCamera(t *testing.T) {
	c := newLandCache()
	if c.visible(overviewCam(0.5, 64), image.Rect(0, 0, 960, 540)) != nil {
		t.Fatal("no land, nothing visible")
	}
	c.sync(islandSite(64))
	far := c.visible(overviewCam(0.5, 64), image.Rect(0, 0, 960, 540))
	near := c.visible(overviewCam(2, 64), image.Rect(0, 0, 960, 540))
	if len(far) <= len(near) || len(near) == 0 {
		t.Errorf("far %d near %d", len(far), len(near))
	}
	for i := 1; i < len(far); i++ {
		if far[i].cx+far[i].cy < far[i-1].cx+far[i-1].cy {
			t.Fatal("chunks are not painted back to front")
		}
	}
	unknown := wildSite(4, view.TileKind(99))
	if l := LandOf(unknown); l.Ground[0] != art.GroundGrass {
		t.Error("unknown tiles fall back to grass")
	}
}

func TestABlockingLandCacheRepaintsAtOnceAndNeverPaintsInTheBackground(t *testing.T) {
	c := newLandCache()
	c.blocking = true
	site := islandSite(96)
	c.sync(site)
	keys := []chunkKey{{0, 0}, {2, 2}, {3, 3}}
	first := c.images(keys)
	if first[0] != nil || first[1] == nil {
		t.Fatal("open sea is empty and the island is painted")
	}
	worn := cloneSite(site)
	worn.Wear[40*96+40] = 9000
	c.sync(worn)
	fresh := c.images(keys)
	if fresh[1] == first[1] || c.chunks[chunkKey{2, 2}].digest != c.digests[chunkKey{2, 2}] {
		t.Fatal("a stale chunk is repainted in the frame that shows it")
	}
	if len(c.Flights) != 0 {
		t.Fatalf("%d chunks paint in the background", len(c.Flights))
	}
}
