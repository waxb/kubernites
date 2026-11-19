package gfx

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

var civicStalls = map[string]string{
	view.StructGuildhall: "stall-guildhall",
	view.StructArchive:   "stall-archive",
	view.StructHerald:    "stall-herald",
	view.StructPlaque:    "stall-plaque",
}

var stallFaces = map[string]bool{view.FaceSE: true, view.FaceSW: true, view.FaceNW: true, view.FaceNE: true}

func StallArt(kind string) (string, bool) {
	if k, ok := civicStalls[kind]; ok {
		return k, true
	}
	box, ok := strings.CutPrefix(kind, view.DepotStructure(""))
	if !ok || box == "" {
		return "", false
	}
	return art.StallForBox(box), true
}

func StallFace(s view.Structure) string {
	if stallFaces[s.Variant] {
		return s.Variant
	}
	if s.W < s.H {
		return view.FaceSE
	}
	return view.FaceSW
}

func stallArtKind(s view.Structure) (string, bool) {
	kind, ok := StallArt(s.Kind)
	if !ok {
		return "", false
	}
	return art.StallArtKind(kind, StallFace(s)), true
}

func portraitKind(s view.Structure) string {
	if kind, ok := StallArt(s.Kind); ok {
		return kind
	}
	return ArtKind(s)
}

func buildArt(kind string) string {
	if stall, ok := StallArt(kind); ok {
		return stall
	}
	return kind
}
