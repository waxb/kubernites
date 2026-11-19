package world

import (
	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/view"
)

func (r *Renderer) keepVent(k view.Structure) vent {
	age, _ := r.agesOf().phase(k.ID)
	dx, dy := art.KeepVent(gfx.KeepLookOf(k, age))
	return vent{dx: dx, dy: dy}
}
