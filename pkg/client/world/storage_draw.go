package world

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	MarkStranded = "stranded"
	MarkDeed     = "deed"
	MarkUnbound  = "unbound"
	pagesFront   = .3
)

type StorageMark struct {
	Pod  string
	Kind string
	Rect ui.Rect
	Text string
}

func (r *Renderer) DrawnStorageMarks() []StorageMark { return r.stores }

func (r *Renderer) storageItems(cam render.Camera, sc Scene) []Drawable {
	out := []Drawable{}
	for _, m := range wardenPosts(sc, func() map[string]bool { return r.couriersOf(sc.Pods) }) {
		out = append(out, r.spriteItem(cam, sc, art.WardenPostName(m.State), m.At, 0))
	}
	for vault, keep := range DetachWaits(sc) {
		if s, ok := vaultStructure(sc.Site, vault); ok {
			out = append(out, r.spriteItem(cam, sc, art.VaultDetachName(int(art.NameSeed(keep)%art.VaultDetachBanners)), FootprintCentre(s), .05))
		}
	}
	for i, pod := range LoosePageHuts(sc) {
		if home, ok := r.Homes[pod]; ok {
			out = append(out, r.spriteItem(cam, sc, art.LoosePagesName(i), Vec{home.X + .5 + pagesFront, home.Y + .5 + pagesFront}, .02))
		}
	}
	return out
}

type courierMemo struct {
	first *model.PodView
	n     int
	nodes map[string]bool
}

func (r *Renderer) couriersOf(pods []model.PodView) map[string]bool {
	if len(pods) == 0 {
		return nil
	}
	if m := r.couriers; m.first == &pods[0] && m.n == len(pods) {
		return m.nodes
	}
	r.couriers = courierMemo{first: &pods[0], n: len(pods), nodes: courierNodes(pods)}
	return r.couriers.nodes
}

func (r *Renderer) spriteItem(cam render.Camera, sc Scene, name string, at Vec, lift float64) Drawable {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	return Drawable{Depth: render.DepthKey(at.X, at.Y) + lift, Draw: func(dst *ebiten.Image) {
		if onScreen(sx, sy, zoom) {
			r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{Alpha: 1})
		}
	}}
}

func (r *Renderer) DrawStorageMarks(dst *ebiten.Image, cam render.Camera, sc Scene) {
	r.stores = r.stores[:0]
	for _, l := range StorageLinks(sc, r.Homes) {
		r.drawLink(dst, cam, sc, l)
	}
	stranded, deeds := StrandedHuts(sc), DeedMarks(sc)
	if len(stranded) == 0 && len(deeds) == 0 {
		return
	}
	avoid := append(r.labelObstacles(cam, sc), markRects(r.marks)...)
	for _, pod := range stranded {
		r.drawMark(dst, cam, sc, StorageMark{Pod: pod, Kind: MarkStranded, Text: strandedText()}, art.StrandedTagName(), &avoid)
	}
	for _, d := range deeds {
		m, name := StorageMark{Pod: d.Pod, Kind: MarkDeed, Text: deedText(d.Claim)}, art.DeedTagName()
		if !d.Sealed {
			m.Kind, m.Text, name = MarkUnbound, unboundText(d.Shelves), art.DeedScrollName(false)
		}
		r.drawMark(dst, cam, sc, m, name, &avoid)
	}
}

func strandedText() string {
	return ui.Bridge("Stranded: no post-warden stands on this Keep, so its couriers cannot hand over their satchels", "Stranded: no DaemonSet Pod on this node, so the post Service (internalTrafficPolicy: Local) has no endpoint here")
}

func deedText(claim string) string {
	return ui.Bridge("holds the Deed "+claim+": select it to see its Vault", "mounts PersistentVolumeClaim "+claim)
}

func unboundText(shelves int) string {
	return ui.Bridge(fmt.Sprintf("waiting for a vault of %d shelves", shelves), fmt.Sprintf("PVC Pending: waiting for a %dGi PV", shelves))
}

func markRects(marks []CauseMark) []ui.Rect {
	out := make([]ui.Rect, 0, len(marks))
	for _, m := range marks {
		out = append(out, m.Rect)
	}
	return out
}

func (r *Renderer) drawMark(dst *ebiten.Image, cam render.Camera, sc Scene, m StorageMark, name string, avoid *[]ui.Rect) {
	home, ok := r.Homes[m.Pod]
	_, reg, found := r.Lib.Lookup(name)
	if !ok || !found {
		return
	}
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, Vec{home.X + .5, home.Y + .5})
	if !onScreen(sx, sy, zoom) {
		return
	}
	box := ui.Rect{X: int(sx) - reg.AnchorX, Y: int(sy-causeLift*zoom) - reg.AnchorY, W: reg.W, H: reg.H}
	box, clear := StackLabel(box, int(sx), *avoid, nil)
	if !clear {
		return
	}
	*avoid = append(*avoid, box)
	m.Rect = box
	r.Lib.Draw(dst, name, float64(box.X+reg.AnchorX), float64(box.Y+reg.AnchorY), 1, render.SpriteOptions{Alpha: 1})
	r.stores = append(r.stores, m)
}

func (r *Renderer) drawLink(dst *ebiten.Image, cam render.Camera, sc Scene, l StorageLink) {
	zoom := cam.Scale()
	vx, vy := ScreenPointOf(cam, sc.Site, l.Vault)
	kx, ky := ScreenPointOf(cam, sc.Site, l.Keep)
	if l.Mounted {
		hx, hy := ScreenPointOf(cam, sc.Site, l.Hut)
		r.stampLine(dst, kx, ky, hx, hy, float64(art.MountStep)*zoom, zoom, func(int) string { return art.MountDotName() })
	}
	r.stampLine(dst, vx, vy, kx, ky, float64(art.ChainStep)*zoom, zoom, func(i int) string { return art.ChainLinkName(l.Red, i) })
	mx, my := (vx+kx)/2, (vy+ky)/2
	r.Lib.Draw(dst, art.DeedScrollName(l.Sealed), mx, my, zoom, render.SpriteOptions{Alpha: 1})
	if ui.Naming() {
		var font ui.Font
		w, _ := font.Measure(l.Label, 1)
		box := ui.Rect{X: int(mx) - w/2 - causePadX, Y: int(my+8*zoom) + 2, W: w + 2*causePadX, H: ui.CapHeight + 2*causePadY}
		ui.Fill(dst, box, ui.Palette.Console)
		font.Draw(dst, ui.Settle(l.Label), box.X+causePadX, box.Y+causePadY, 1, ui.Palette.Cream)
	}
}

func (r *Renderer) stampLine(dst *ebiten.Image, x0, y0, x1, y1, step, zoom float64, name func(i int) string) {
	n := int(math.Hypot(x1-x0, y1-y0) / max(step, 1))
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(max(n, 1))
		x, y := x0+(x1-x0)*t, y0+(y1-y0)*t
		if onScreen(x, y, zoom) {
			r.Lib.Draw(dst, name(i), x, y, zoom, render.SpriteOptions{Alpha: 1})
		}
	}
}
