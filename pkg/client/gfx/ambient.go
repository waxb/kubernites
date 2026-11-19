package gfx

import "github.com/waxb/kubernites/pkg/client/art"

const (
	FolkLooks   = art.FolkLooks
	CartLoads   = art.CartLoads
	ScrollPiles = art.ScrollPiles
	StallLooks  = art.StallLooks
)

var FolkRoles = append([]string(nil), art.FolkRoles...)

func wrap(v, n int) int {
	return ((v % n) + n) % n
}

func FolkName(role string, look int, action, dir string, frame int) string {
	return art.FolkName(role, wrap(look, FolkLooks), action, dir, wrap(frame, art.WalkerFrames))
}

func CartName(load int, dir string, frame int) string {
	return art.CartName(min(max(load, 0), CartLoads-1), dir, wrap(frame, art.CartFrames))
}

func AmbientName(kind string, variant int) string {
	return art.AmbientName(kind, max(variant, 0))
}

func WispName(frame int) string {
	return EffectName("wisp", frame)
}
