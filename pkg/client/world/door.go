package world

import (
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	doorWorkSide = .6
	doorWorkBack = .35
	doorPace     = .6
)

var doorCandidates = [][2]int{{2, 0}, {2, -1}, {0, 2}, {-1, 2}, {2, 1}, {1, 2}, {3, 0}, {0, 3}}

func DoorHut(site view.Site, keep view.Structure) (Point, bool) {
	_, hi := Footprint(keep)
	for _, c := range doorCandidates {
		x, y := hi.X+c[0], hi.Y+c[1]
		if Homeable(site.Tile(x, y)) && !NearAnyStructure(site, x, y) && !InAnyShadow(site, x, y) {
			return Point{x, y}, true
		}
	}
	return Point{}, false
}

func DoorWorkSpot(keep view.Structure) Vec {
	porch := Porch(keep)
	return Vec{porch.X + doorWorkSide, porch.Y - doorWorkBack}
}

func (r *Renderer) DoorFigure(sc Scene, p model.PodView) (Figure, bool) {
	keep, ok := KeepForNode(sc.Site, p.Node)
	if !ok {
		return Figure{}, false
	}
	spot, porch := DoorWorkSpot(keep), Porch(keep)
	rate, working, throttled := Workload(sc, p, true)
	pose := Pose{Pos: spot, Action: "idle", Dir: Facing(spot, porch), Frame: Frame(sc.Time, 2)}
	mood := MoodSteady
	if working {
		pose.Action, pose.Frame = "work", Frame(sc.Time*(doorPace+rate), 6)
	}
	if throttled {
		mood = MoodStrained
	}
	return Figure{Pod: p, role: p.Sprite, Pose: pose, Mood: mood, home: r.Homes[p.Name]}, true
}
