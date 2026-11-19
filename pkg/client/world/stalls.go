package world

import "github.com/waxb/kubernites/pkg/view"

const (
	stallDoorAlong = .2
	stallDoorOut   = 1.0
	stallPileAlong = .5
	stallPileOut   = .95
	stallPostAlong = -.55
	stallPostOut   = .9
	stallPostStep  = .75
	stallLampAlong = 1.05
	stallLampOut   = .55
)

type stallFrame struct {
	centre, along, out Vec
}

func stallFrameOf(s view.Structure) (stallFrame, bool) {
	if !view.IsStall(s) || s.W*s.H != 2 {
		return stallFrame{}, false
	}
	c := Vec{X: float64(s.X) + float64(s.W-1)/2, Y: float64(s.Y) + float64(s.H-1)/2}
	if s.W == 1 {
		return stallFrame{centre: c, along: Vec{Y: 1}, out: Vec{X: 1}}, true
	}
	return stallFrame{centre: c, along: Vec{X: 1}, out: Vec{Y: 1}}, true
}

func (f stallFrame) at(along, out float64) Vec {
	return Vec{X: f.centre.X + f.along.X*along + f.out.X*out, Y: f.centre.Y + f.along.Y*along + f.out.Y*out}
}

func (f stallFrame) door() Vec { return f.at(stallDoorAlong, stallDoorOut) }

func (f stallFrame) pile() Vec { return f.at(stallPileAlong, stallPileOut) }

func (f stallFrame) lamp() Vec { return f.at(stallLampAlong, stallLampOut) }

func (f stallFrame) post() (Vec, Vec) {
	return f.at(stallPostAlong, stallPostOut), Vec{X: f.out.X * stallPostStep, Y: f.out.Y * stallPostStep}
}

func IsStallAt(s view.Structure) bool {
	_, ok := stallFrameOf(s)
	return ok
}
