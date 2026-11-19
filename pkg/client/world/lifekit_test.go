package world

import (
	"sync"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
)

var (
	lifeLibOnce sync.Once
	lifeLib     *gfx.Library
)

func testLib() *gfx.Library {
	lifeLibOnce.Do(func() { lifeLib = gfx.Load(art.Build()) })
	return lifeLib
}
