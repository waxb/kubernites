package art

import "fmt"

const (
	WardenPostW        = 24
	WardenPostH        = 34
	WardenPostAX       = 12
	WardenPostAY       = 29
	MarkTagW           = tagW + 2
	MarkTagH           = tagH + 2
	MarkTagAX          = tagAX + 1
	MarkTagAY          = tagAY + 1
	VaultW             = 36
	VaultH             = 40
	VaultAX            = 18
	VaultAY            = 37
	VaultShelfLevels   = 4
	VaultDetachBanners = 4
	DeedScrollW        = scrollW + 2
	DeedScrollH        = scrollH + 2
	ChainLinkSize      = 7
	ChainPhases        = 2
	ChainStep          = 4
	MountDotSize       = dotSize + 2
	MountStep          = 4
	LoosePageVariants  = 3
	LoosePagesW        = pagesW + 2
	LoosePagesH        = pagesH + 2
	LoosePagesAX       = pagesAX + 1
	LoosePagesAY       = pagesAY + 1
	tagW, tagH         = 15, 14
	tagAX, tagAY       = 7, 13
	scrollW, scrollH   = 14, 10
	dotSize            = 3
	pagesW, pagesH     = 30, 16
	pagesAX, pagesAY   = 15, 11
)

func padded(paint func(c *canvas)) func(c *canvas) {
	return func(c *canvas) {
		inner := newCanvas(c.w-2, c.h-2)
		paint(inner)
		c.paste(inner, 1, 1, false)
	}
}

var WardenPostStates = []string{"running", "starting", "absent"}

var VaultStates = []string{"bound", "available", "released", "terminating"}

var StoragePageIcons = []string{"daemons", "vaults", "stateful-crews", "vault-wrights"}

func WardenPostName(state string) string { return "warden/post/" + state }
func StrandedTagName() string            { return "tag/stranded" }
func DeedTagName() string                { return "tag/deed" }
func VaultName(state string, shelves int) string {
	return fmt.Sprintf("vault/%s/%d", state, min(max(shelves, 0), VaultShelfLevels-1))
}
func VaultDetachName(banner int) string {
	return fmt.Sprintf("vault/detach/%d", ((banner%VaultDetachBanners)+VaultDetachBanners)%VaultDetachBanners)
}
func DeedScrollName(sealed bool) string {
	if sealed {
		return "deed/scroll/sealed"
	}
	return "deed/scroll/unbound"
}
func ChainLinkName(red bool, phase int) string {
	tone := "iron"
	if red {
		tone = "red"
	}
	return fmt.Sprintf("chain/%s/%d", tone, ((phase%ChainPhases)+ChainPhases)%ChainPhases)
}
func MountDotName() string { return "mount/dot" }
func LoosePagesName(v int) string {
	return fmt.Sprintf("pages/loose/%d", ((v%LoosePageVariants)+LoosePageVariants)%LoosePageVariants)
}

func addStorageArt(b *atlasBuilder) {
	for _, st := range WardenPostStates {
		state := st
		b.add(WardenPostName(st), WardenPostW, WardenPostH, WardenPostAX, WardenPostAY, func(c *canvas) { paintWardenPost(c, state) })
	}
	b.add(StrandedTagName(), MarkTagW, MarkTagH, MarkTagAX, MarkTagAY, padded(paintStrandedTag))
	b.add(DeedTagName(), MarkTagW, MarkTagH, MarkTagAX, MarkTagAY, padded(paintDeedTag))
	for _, st := range VaultStates {
		for l := 0; l < VaultShelfLevels; l++ {
			state, level := st, l
			b.add(VaultName(st, l), VaultW, VaultH, VaultAX, VaultAY, func(c *canvas) { paintVault(c, state, level) })
		}
	}
	for i := 0; i < VaultDetachBanners; i++ {
		banner := i
		b.add(VaultDetachName(i), VaultW, VaultH, VaultAX, VaultAY, func(c *canvas) { paintVaultDetach(c, banner) })
	}
	for _, sealed := range []bool{true, false} {
		sl := sealed
		b.add(DeedScrollName(sealed), DeedScrollW, DeedScrollH, DeedScrollW/2, DeedScrollH/2, padded(func(c *canvas) { paintDeedScroll(c, sl) }))
	}
	addLinkPieces(b)
}

func addLinkPieces(b *atlasBuilder) {
	for _, red := range []bool{false, true} {
		for p := 0; p < ChainPhases; p++ {
			rd, phase := red, p
			b.add(ChainLinkName(red, p), ChainLinkSize, ChainLinkSize, ChainLinkSize/2, ChainLinkSize/2, func(c *canvas) { paintChainLink(c, rd, phase) })
		}
	}
	b.add(MountDotName(), MountDotSize, MountDotSize, MountDotSize/2, MountDotSize/2, padded(paintMountDot))
	for v := 0; v < LoosePageVariants; v++ {
		variant := v
		b.add(LoosePagesName(v), LoosePagesW, LoosePagesH, LoosePagesAX, LoosePagesAY, padded(func(c *canvas) { paintLoosePages(c, variant) }))
	}
}
