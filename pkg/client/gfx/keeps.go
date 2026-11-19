package gfx

import (
	"image"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	keepJobs      = 64
	keepSprites   = "building/keep-"
	keepPortraits = "portrait/keep-"
	keepIcons     = "buildicon/keep-"
)

type painted struct {
	img *image.NRGBA
	reg art.Region
}

type lazySprites struct {
	mu      sync.Mutex
	index   map[string]entry
	painted map[string]painted
	pending map[string]bool
	async   bool
	jobs    chan string
}

func newLazySprites() *lazySprites {
	return &lazySprites{index: map[string]entry{}, painted: map[string]painted{}, pending: map[string]bool{}}
}

func (l *Library) Async() bool {
	if l == nil || l.lazy == nil {
		return false
	}
	l.lazy.mu.Lock()
	defer l.lazy.mu.Unlock()
	return l.lazy.async
}

func (l *Library) SetAsync(on bool) {
	if l == nil || l.lazy == nil {
		return
	}
	l.lazy.mu.Lock()
	defer l.lazy.mu.Unlock()
	l.lazy.async = on
	if on && l.lazy.jobs == nil {
		l.lazy.jobs = make(chan string, keepJobs)
		go l.lazy.work()
	}
}

func (z *lazySprites) work() {
	for name := range z.jobs {
		paint, _ := lazyPainter(name)
		img, reg := paint()
		z.mu.Lock()
		z.painted[name] = painted{img: img, reg: reg}
		delete(z.pending, name)
		z.mu.Unlock()
	}
}

func lazyPainter(name string) (func() (*image.NRGBA, art.Region), bool) {
	switch {
	case strings.HasPrefix(name, keepSprites):
		k, state, ok := art.ParseKeepName(name)
		return func() (*image.NRGBA, art.Region) { return art.KeepSprite(k, state) }, ok
	case strings.HasPrefix(name, keepPortraits):
		k, ok := art.ParseKeepPortraitName(name)
		return func() (*image.NRGBA, art.Region) { return art.KeepPortrait(k) }, ok
	case strings.HasPrefix(name, keepIcons):
		k, ok := art.ParseKeepIconName(name)
		return func() (*image.NRGBA, art.Region) { return art.KeepBuildIcon(k) }, ok
	case strings.HasPrefix(name, art.PortPrefix):
		return portPainter(name)
	}
	return nil, false
}

func lazyName(name string) bool {
	return strings.HasPrefix(name, keepSprites) || strings.HasPrefix(name, keepPortraits) || strings.HasPrefix(name, keepIcons) || strings.HasPrefix(name, art.PortPrefix)
}

func (l *Library) lazyEntry(name string) (entry, bool) {
	if l.lazy == nil || !lazyName(name) {
		return entry{}, false
	}
	z := l.lazy
	z.mu.Lock()
	defer z.mu.Unlock()
	if e, ok := z.index[name]; ok {
		return e, true
	}
	paint, ok := lazyPainter(name)
	if !ok {
		return entry{}, false
	}
	p, ok := z.painted[name]
	switch {
	case ok:
		delete(z.painted, name)
	case z.async:
		z.request(name)
		return entry{}, false
	default:
		p.img, p.reg = paint()
	}
	e := newEntry(ebiten.NewImageFromImage(p.img), p.img, p.reg)
	z.index[name] = e
	return e, true
}

func (z *lazySprites) request(name string) {
	if z.pending[name] {
		return
	}
	select {
	case z.jobs <- name:
		z.pending[name] = true
	default:
	}
}

func KeepLookOf(s view.Structure, age int) art.KeepLook {
	return art.KeepLookOf(s.Variant, s.Class, s.Object.Name, age)
}

func KeepNames(s view.Structure, age int) []string {
	state := BuildingState(s)
	return append([]string{KeepLookOf(s, age).Name(state)}, BuildingLookNames(ArtKind(s), state, age, 0)...)
}

func KeepPreview(size, class string, age int) art.KeepLook {
	return art.KeepLookOf(size, class, art.KeepPreviewNode(size), age)
}

func KeepPreviewNode(size string) string { return art.KeepPreviewNode(size) }

func KeepFallback(name string) (string, bool) {
	if k, ok := art.ParseKeepPortraitName(name); ok {
		return "portrait/" + art.AgedPortrait("keep-"+k.Size, k.Age), true
	}
	if k, ok := art.ParseKeepIconName(name); ok {
		return art.BuildIconName("keep-"+k.Size, k.Age), true
	}
	return "", false
}
