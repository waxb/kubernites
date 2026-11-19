package world

import "github.com/waxb/kubernites/pkg/view"

const hutLead = 2

func hutHiddenBy(s view.Structure, x, y int) bool {
	lo, hi := Footprint(s)
	clo, chi := lo.X-hi.Y-1, hi.X-lo.Y+1
	if x-y-1 >= chi || clo >= x-y+1 {
		return false
	}
	diff := hi.X + hi.Y - (x + y)
	return diff > -hutLead && diff < view.StructureLift(s.Kind, s.Variant)
}

func InAnyShadow(site view.Site, x, y int) bool {
	for _, s := range site.Structures {
		if hutHiddenBy(s, x, y) {
			return true
		}
	}
	return false
}
