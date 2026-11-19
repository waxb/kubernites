package art

import "image/color"

func buildBuildings() Atlas {
	b, memo := &atlasBuilder{}, newAgedMemo()
	for _, k := range BuildingKinds {
		if _, ok := CanonicalKeep(k, 0); ok {
			addCanonicalKeep(b, k)
			continue
		}
		addAged(b, k, memo)
	}
	addHearths(b)
	addQoSHearths(b)
	addRequestPiles(b)
	addMidden(b)
	addCodexBloat(b)
	addHaven(b)
	addFreight(b)
	addBorders(b)
	addForum(b)
	addStalls(b)
	addStorageArt(b)
	addHeraldry(b)
	addSpeakStone(b)
	addProbeArt(b)
	addIncidentArt(b)
	return b.build()
}

func stateName(stage int) string {
	return []string{"construction-0", "construction-1", "construction-2"}[stage]
}

func scaffold(s iso, n int, height float64) {
	f := float64(n)
	top := height * 0.9
	for _, p := range [][2]float64{{0.2, f - 0.2}, {f / 2, f - 0.15}, {f - 0.2, f - 0.2}, {f - 0.15, f / 2}, {f - 0.2, 0.2}} {
		a, b := s.p(p[0], p[1], 0), s.p(p[0], p[1], top)
		s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[2])
	}
	for z := 8.0; z < top; z += 12 {
		a, b := s.p(0.2, f-0.2, z), s.p(f-0.2, f-0.2, z)
		s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[1])
		d := s.p(f-0.2, 0.2, z)
		s.c.line(int(b.x), int(b.y), int(d.x), int(d.y), woodR[0])
	}
}

func stakes(s iso, n int) {
	f := float64(n)
	for _, p := range [][2]float64{{0.1, 0.1}, {f - 0.1, 0.1}, {0.1, f - 0.1}, {f - 0.1, f - 0.1}} {
		a := s.p(p[0], p[1], 0)
		s.c.rect(int(a.x), int(a.y)-5, 1, 5, woodR[2])
		s.c.dot(int(a.x), int(a.y)-5, color.NRGBA{230, 70, 60, 255})
	}
}
