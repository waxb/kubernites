package world

import (
	"github.com/waxb/kubernites/pkg/client/model"
)

const (
	arriveGap      = .6
	newcomerWindow = 90.0
)

var personAsides = [...]Vec{{0, 0}, {-.32, .16}, {.16, -.3}, {-.3, -.14}}

func firstPerson(p model.PodView) bool {
	return p.Focus == "" || len(p.People) == 0 || p.Focus == p.People[0].Name
}

func PersonSeed(p model.PodView) uint32 {
	if firstPerson(p) {
		return model.Hash(p.Name)
	}
	return model.Hash(p.Name) ^ mix32(model.Hash(p.Focus)+0x9e3779b9)
}

func personName(p model.PodView) string {
	if firstPerson(p) {
		return p.Name
	}
	return p.Name + "/" + p.Focus
}

func (f Figure) seedOf() uint32 {
	if f.Seed == 0 {
		return model.Hash(f.Pod.Name)
	}
	return f.Seed
}

func PersonKey(p model.PodView) string {
	if firstPerson(p) {
		return PodKey(p)
	}
	return PodKey(p) + "/" + p.Focus
}

func PersonAside(i int) Vec {
	if i <= 0 {
		return Vec{}
	}
	a := personAsides[1+(i-1)%(len(personAsides)-1)]
	ring := float64((i - 1) / (len(personAsides) - 1))
	return Vec{a.X * (1 + ring*.5), a.Y * (1 + ring*.5)}
}

func PetsOf(p model.PodView, person int) []string {
	n := p.Persons()
	if n == 1 {
		return p.Familiars
	}
	var out []string
	for j, pet := range p.Familiars {
		if j%n == person {
			out = append(out, pet)
		}
	}
	return out
}

func PetNamesOf(p model.PodView, person int) []string {
	names := p.PetNames
	if len(names) != len(p.Familiars) {
		names = p.Familiars
	}
	n := p.Persons()
	if n == 1 {
		return names
	}
	var out []string
	for j, name := range names {
		if j%n == person {
			out = append(out, name)
		}
	}
	return out
}

func PetKey(p model.PodView, name string) string {
	return PersonKey(p) + "#" + name
}

func healthy(p model.PodView) bool {
	return p.Focus != "" && p.Ready && p.Reason == ""
}

func (r *Renderer) peopleOf(sc Scene, p model.PodView, rt route, out []Figure) []Figure {
	if p.System {
		if f, ok := systemFigure(sc, p); ok {
			f.Seed = PersonSeed(p)
			out = append(out, f)
		}
		return out
	}
	for i := range p.Persons() {
		person := p.Person(i)
		f, ok := r.personFigure(sc, person, i, rt)
		if !ok {
			continue
		}
		f.Person, f.Seed = i, PersonSeed(person)
		out = append(out, f)
	}
	return out
}

func (r *Renderer) personFigure(sc Scene, p model.PodView, i int, rt route) (Figure, bool) {
	if p.State == model.HearthGuttering && healthy(p) {
		home, ok := r.Homes[p.Name]
		if !ok {
			return Figure{}, false
		}
		at := offset(Vec{home.X + .35, home.Y + .35}, PersonAside(i), 1)
		return Figure{Pod: p, role: p.Sprite, Pose: RestPose(sc.Time, PersonSeed(p), at), Mood: MoodRest, home: home}, true
	}
	f, ok := r.FigureFor(sc, p, rt)
	if ok && i > 0 && (settled(f.Mood) || p.Static && f.clock == 0) {
		f.Pose.Pos = offset(f.Pose.Pos, PersonAside(i), 1)
	}
	return f, ok
}

func personIndex(p model.PodView) int {
	for i, k := range p.People {
		if k.Name == p.Focus {
			return i
		}
	}
	return 0
}

func (r *Renderer) Newcomer(name string, now float64) bool {
	if r.hamlet == nil {
		return false
	}
	s, ok := r.hamlet.SpotList[name]
	return ok && s.Born > neverBorn && now-s.Born < newcomerWindow
}

func (r *Renderer) strideOf(sc Scene, p model.PodView, seed uint32, throttled, rest bool) *stride {
	t := r.townOf()
	key := PersonKey(p)
	if st, ok := t.strides[key]; ok {
		return st
	}
	st := t.strideFor(key, sc.Time, seed, throttled, rest)
	if r.Newcomer(p.Name, sc.Time) {
		st.clock = -float64(personIndex(p)) * arriveGap
	}
	return st
}
