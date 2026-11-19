package art

import (
	"fmt"
	"image/color"
)

var (
	vgLooks7 = []string{"", "t1v1", "t1v2", "t1v3", "t1v4", "t1v5", "t1v6", "t1v7"}
	vgLooks3 = []string{"", "t1v1", "t1v2", "t1v3"}
	vgTrees  = []string{"tree-oak", "tree-pine", "tree-maple", "tree-birch"}
	vgKinds  = []string{"parcel", "ledger", "crate", "dispatch"}
)

func vgWalker(role, look, action, dir string, frame int) string {
	if look == "" {
		return fmt.Sprintf("walker/%s/%s/%s/%d", role, action, dir, ((frame%4)+4)%4)
	}
	return fmt.Sprintf("walker/%s/%s/%s/%s/%d", role, look, action, dir, ((frame%4)+4)%4)
}

func vgHut(tier int, state, qos string) string {
	name := fmt.Sprintf("hearth/%d/%s", min(max(tier, 1), 3), state)
	if qos != "" {
		name += "/" + qos
	}
	return name
}

func vgBox(size int, kind string, variant int, spoiled bool) string {
	name := fmt.Sprintf("box/%d/%s/%d", min(max(size, 0), 3), kind, ((variant%4)+4)%4)
	if spoiled {
		name += "/spoiled"
	}
	return name
}

func vgFx(kind string, frame int) string {
	return fmt.Sprintf("fx/%s/%d", kind, ((frame%6)+6)%6)
}

func vgShip(class string, variant int, dir string, frame int) string {
	return fmt.Sprintf("ship/%s/%d/%s/%d", class, ((variant%8)+8)%8, dir, ((frame%4)+4)%4)
}

func vgDecor(kind string, variant int) string {
	return fmt.Sprintf("decor/%s/%d", kind, ((variant%3)+3)%3)
}

func (p *vgPaint) outdoors(top, low color.NRGBA, far color.NRGBA, ground ramp) {
	p.sky(top, low, 46)
	p.clouds(3, 4, 22, mix(vgCloud, low, 0.2), mix(vgCloudSh, top, 0.25))
	p.hills(44, 9, far)
	p.land(48, ground, mix(far, low, 0.3))
}

func (p *vgPaint) treeLine(y int, tone func(color.NRGBA) color.NRGBA) {
	p.sides(func(x int, _ bool) {
		name := vgDecor(vgTrees[p.bgIntn(len(vgTrees))], p.bgIntn(3))
		dx, dy := p.bgIntn(9)-4, p.bgIntn(5)-2
		if tone != nil {
			p.putTone(name, x+dx, y+dy, tone)
			return
		}
		p.put(name, x+dx, y+dy)
	})
}
