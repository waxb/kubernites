package view

const quarryAt = 4

func (s Site) Paved(tiles []Point, f Forum) Site {
	next := s
	next.Tiles = append([]TileKind(nil), s.Tiles...)
	next.Wear = append([]uint16(nil), s.Wear...)
	for _, p := range tiles {
		i, ok := s.Index(p.X, p.Y)
		if !ok {
			continue
		}
		next.Tiles[i] = TileForum
		if i < len(next.Wear) {
			next.Wear[i] = 0
		}
	}
	next.Forum = &f
	return next
}

func (s Site) QuarryPenalty(def BuildDef, fp Rect) int {
	if KeepDef(def) {
		return 0
	}
	ring := Rect{X0: fp.X0 - 1, Y0: fp.Y0 - 1, X1: fp.X1 + 1, Y1: fp.Y1 + 1}
	return s.CountTiles(ring, func(x, y int) bool { t := s.Tile(x, y); return t == TileForest || t == TileRock }) * quarryAt
}
