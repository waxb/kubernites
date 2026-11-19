package art

func addTradeHolds(b *atlasBuilder) {
	for _, r := range TradeRoles {
		for tier := 1; tier <= WalkerTiers(r); tier++ {
			for v := 0; v < WalkerLooks; v++ {
				spec := lookSpec(r, v, tier)
				for f := 0; f < WalkerFrames; f++ {
					role, tt, vv, frame := r, tier, v, f
					addWalker(b, func(dir string) string { return tradeHoldName(role, tt, vv, dir, frame) }, func(c *canvas) {
						paintTrade(c, spec, HoldAction, frame)
					})
				}
			}
		}
	}
}

func tradeHoldName(role string, tier, variant int, dir string, frame int) string {
	if tier <= 1 && variant == 0 {
		return walkerName(role, HoldAction, dir, frame)
	}
	return walkerLookName(role, max(tier, 1), variant, HoldAction, dir, frame)
}

func addCarry(b *atlasBuilder) {
	for size := 0; size < BoxSizes; size++ {
		for _, k := range BoxKinds {
			for f := 0; f < WalkerFrames; f++ {
				src := newCanvas(CarryWidth, CarryHeight)
				paintCarry(src, size, k, f)
				b.add(CarryName(size, k, "se", f), CarryWidth, CarryHeight, CarryAnchorX, CarryAnchorY, func(c *canvas) { c.paste(src, 0, 0, false) })
				b.add(CarryName(size, k, "sw", f), CarryWidth, CarryHeight, CarryAnchorX, CarryAnchorY, func(c *canvas) { c.paste(src, 0, 0, true) })
			}
		}
	}
}

func carryBase(size, frame int) (float64, float64) {
	d := carryBoxes[clampSize(size)]
	bob := poseFor("carry", frame).bob
	return CarryAnchorX, float64(CarryAnchorY-WalkerHeight+2+figureTop+1+bob) - d.hh()
}

func paintCarry(c *canvas, size int, kind string, frame int) {
	x, y := carryBase(size, frame)
	drawBox(c, x, y, carryBoxes[clampSize(size)], kind, 0)
	c.outline(ink)
}
