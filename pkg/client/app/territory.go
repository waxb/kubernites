package app

import (
	"cmp"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	discReach    = 23
	controlPlane = "control-plane"
	TowerVariant = "tower"
)

type landView struct {
	borders   world.Borders
	tints     []world.Tint
	preview   []world.BorderMark
	spots     []world.SpotMark
	tintKey   tintKey
	tinted    bool
	survey    view.Survey
	SurveyKey surveyKey `json:"-"`
	surveyed  bool
	refusedAt float64
	best      bestPlot
	suggested *world.Ghost
}

type tintKey struct {
	sig      uint64
	placing  string
	x, y     int
	ok       bool
	selected string
}

type surveyKey struct {
	kind string
	X, y int `json:"-"`
	sig  uint64
}

type TintInput struct {
	Placing      string
	Keep         bool
	ControlPlane bool
	Forum        bool
	Ghost        *world.Ghost
	Survey       view.Survey
	Selected     *view.Structure
}

func allAreas(b world.Borders) []int {
	out := make([]int, len(b.Areas))
	for i := range out {
		out[i] = i
	}
	return out
}

func towerAreas(b world.Borders, site view.Site) []int {
	for _, s := range site.Structures {
		if s.Kind == view.StructKeep && s.Variant == TowerVariant {
			if i, ok := b.AreaOf(s.ID); ok {
				return []int{i}
			}
		}
	}
	return nil
}

func areaOfStructure(b world.Borders, s view.Structure) (int, bool) {
	if s.Kind == view.StructKeep {
		return b.AreaOf(s.ID)
	}
	return b.At(s.X+s.W/2, s.Y+s.H/2)
}

func PreviewFor(site view.Site, in TintInput) []world.BorderMark {
	if in.Placing == "" || !in.Keep || in.Ghost == nil {
		return nil
	}
	g := in.Ghost
	disc := site.TerritoryTiles(g.X, g.Y, g.W, g.H, in.Survey.Radius)
	return world.PreviewMarks(site, disc, view.Point{X: g.X + g.W/2, Y: g.Y + g.H/2})
}

func TintsFor(b world.Borders, site view.Site, in TintInput) []world.Tint {
	switch {
	case in.Placing != "" && in.Keep && in.Ghost != nil:
		kind := world.TintGood
		if !in.Survey.OK {
			kind = world.TintPoor
		}
		disc := site.TerritoryTiles(in.Ghost.X, in.Ghost.Y, in.Ghost.W, in.Ghost.H, in.Survey.Radius)
		return world.MergeTints(world.AreaTints(b, allAreas(b), world.TintOwn), world.DiscTints(disc, kind))
	case in.Placing == view.StructCodex && in.Ghost != nil && site.Forum == nil:
		return world.MergeTints(world.AreaTints(b, towerAreas(b, site), world.TintGood), ForumPreviewTints(site, *in.Ghost))
	case in.Placing != "" && in.ControlPlane:
		return world.AreaTints(b, towerAreas(b, site), world.TintGood)
	case in.Placing != "" && in.Forum:
		return world.ForumTints(site, world.TintGood)
	case in.Placing != "":
		return world.AreaTints(b, allAreas(b), world.TintGood)
	case in.Selected != nil && world.OnTheForum(site, *in.Selected):
		return world.ForumTints(site, world.TintOwn)
	case in.Selected != nil:
		if i, ok := areaOfStructure(b, *in.Selected); ok {
			return world.AreaTints(b, []int{i}, world.TintOwn)
		}
	}
	return nil
}

func (s *Session) trackLand() {
	if sig := world.BorderSig(s.Site); sig != s.Land.borders.Sig || s.Land.borders.Owner == nil {
		s.Land.borders = world.BordersOf(s.Site)
		s.Land.spots = s.openSpots()
	}
}

func (s *Session) openSpots() []world.SpotMark {
	if !hasKeep(s.Site) {
		return nil
	}
	w, h := s.Footprint(view.StructKeep)
	out := []world.SpotMark{}
	for _, sp := range s.Site.Spots {
		if s.Realm.PlotSurvey(view.StructKeep, sp.X, sp.Y).OK {
			out = append(out, world.SpotMark{X: sp.X, Y: sp.Y, W: w, H: h, Class: sp.Class})
		}
	}
	return out
}

func (s *Session) SpotMarks() []world.SpotMark {
	if !world.IsKeepKind(s.Placing) {
		return nil
	}
	return s.machineMarks(s.Land.best.marked(s.Land.spots, s.Placing, s.Footprint))
}

func (s *Session) previewMarks() []world.BorderMark {
	s.landTints()
	return s.Land.preview
}

func (s *Session) SurveyAt(kind string, x, y int) view.Survey {
	k := surveyKey{kind: kind, X: x, y: y, sig: s.Land.borders.Sig}
	if s.Land.surveyed && s.Land.SurveyKey == k {
		return s.Land.survey
	}
	sv := s.Realm.PlotSurvey(kind, x, y)
	s.Land.survey, s.Land.SurveyKey, s.Land.surveyed = sv, k, true
	return sv
}

func (s *Session) buildable(kind string) (view.Buildable, bool) {
	for _, b := range s.State.Buildables {
		if b.Kind == kind {
			return b, true
		}
	}
	return view.Buildable{}, false
}

func (s *Session) tintInput() TintInput {
	in := TintInput{Placing: s.Placing, Keep: world.IsKeepKind(s.Placing), Ghost: s.Ghost}
	if b, ok := s.buildable(s.Placing); ok {
		in.ControlPlane, in.Forum = b.Placement == controlPlane, b.Placement == view.PlaceForum
	}
	if s.Ghost != nil {
		in.Survey = s.SurveyAt(s.Ghost.Kind, s.Ghost.X, s.Ghost.Y)
	}
	if s.Selection.Kind == world.HitStructure && s.landShown() {
		sel := s.Selection.Structure
		in.Selected = &sel
	}
	return in
}

func (s *Session) landTints() []world.Tint {
	in := s.tintInput()
	k := tintKey{sig: s.Land.borders.Sig, placing: in.Placing, ok: in.Survey.OK}
	if in.Ghost != nil && (in.Keep || in.Placing == view.StructCodex) {
		k.x, k.y = in.Ghost.X, in.Ghost.Y
	}
	if in.Selected != nil {
		k.selected = in.Selected.ID
	}
	if !s.Land.tinted || k != s.Land.tintKey {
		s.Land.tints, s.Land.tintKey, s.Land.tinted = TintsFor(s.Land.borders, s.Site, in), k, true
		s.Land.preview = PreviewFor(s.Site, in)
	}
	return s.Land.tints
}

func (s *Session) BuildName(kind string) string {
	if name, ok := DepotName(kind); ok {
		return name
	}
	if kind == view.StructKeep && !hasKeep(s.Site) {
		return view.TowerType().Name
	}
	if k, ok := view.KeepTypeOf(kind); ok && world.IsKeepKind(kind) {
		return k.Name
	}
	if b, ok := s.buildable(kind); ok {
		return b.Name
	}
	return kind
}

func hasKeep(site view.Site) bool {
	_, ok := world.FirstOfKind(site, view.StructKeep, false)
	return ok
}

func (s *Session) KeepLabel(id string) string {
	for _, st := range s.Site.Structures {
		if st.ID == id {
			return st.Label
		}
	}
	return ""
}

func (s *Session) SurveyPanel() *hud.SurveyPanel {
	if s.Placing == "" || s.Ghost == nil {
		return nil
	}
	g := *s.Ghost
	sv := s.SurveyAt(g.Kind, g.X, g.Y)
	centre := world.Vec{X: float64(g.X) + float64(g.W-1)/2, Y: float64(g.Y) + float64(g.H-1)/2}
	sx, sy := world.ScreenOf(s.Cam, s.Site, centre)
	reach := int(float64(sv.Radius) * discReach * s.Cam.Scale())
	f := hud.Frame{State: s.State, Site: s.Site}
	return &hud.SurveyPanel{Survey: sv, Name: s.BuildName(g.Kind), Keep: world.IsKeepKind(g.Kind), Land: s.KeepLabel(sv.Keep),
		At: [2]int{int(sx), int(sy)}, Reach: reach, RefusedAt: s.Land.refusedAt, Note: cmp.Or(ForumNote(s.Site, g.Kind, s.State.Tutorial), DepotNote(g.Kind, f)),
		Payoff: hud.PlotPayoffFor(f, g.Kind, s.Land.best.crewFor(g.Kind), sv.CPU, sv.MemoryGiB)}
}

func (s *Session) refuse(reason string) []sound.SFX {
	s.Land.refusedAt = s.Clock
	s.Hud.Toast(reason, true)
	return []sound.SFX{sound.SFXError}
}

func (s *Session) Causes() []view.CauseTag {
	if !hud.CalamitiesShown(hud.Frame{Guide: s.shownGuide()}) {
		return nil
	}
	return s.State.CauseTags
}

func (s *Session) incidentMarks() []world.IncidentMark {
	if !hud.CalamitiesShown(hud.Frame{Guide: s.shownGuide()}) {
		return nil
	}
	return IncidentMarks(s.State.Calamities, s.State.Incidents)
}

func (s *Session) CauseTips() []hud.CauseTip {
	marks := s.Renderer.DrawnCauseMarks()
	out := make([]hud.CauseTip, 0, len(marks))
	for _, m := range marks {
		out = append(out, hud.CauseTip{Rect: m.Rect, Text: s.causeTipText(m.Pod, m.Tag, m.Text)})
	}
	for _, m := range s.Renderer.DrawnStorageMarks() {
		p, _ := s.podNamed(m.Pod)
		out = append(out, hud.CauseTip{Rect: m.Rect, Text: hud.CauseTipText(storageTitle(m.Kind), p.Name, p.Node, m.Text, "")})
	}
	return out
}

func (s *Session) placeOf(object string) (world.Vec, bool) {
	kind, name, _ := strings.Cut(object, "/")
	switch kind {
	case "pod":
		return ResolveTarget("pod:"+name, s.Site, s.Renderer.Home)
	case "node":
		return ResolveTarget("keep:"+name, s.Site, s.Renderer.Home)
	case "kind":
		return DepotPlace(s.Site, name)
	}
	return world.Vec{}, false
}

func DepotPlace(site view.Site, kind string) (world.Vec, bool) {
	if d, ok := soulstone.DepotFor(kind); ok {
		for _, st := range site.Structures {
			if st.Object.Kind == "Service" && st.Object.Name == d.Service {
				return world.Anchor(st), true
			}
		}
	}
	if len(site.Havens) > 0 {
		q := site.Havens[0].Quay
		return world.Vec{X: float64(q.X), Y: float64(q.Y)}, true
	}
	return world.Vec{}, false
}

func (s *Session) Places() map[string][2]float64 {
	out := map[string][2]float64{}
	for _, c := range s.State.Calamities {
		if v, ok := s.placeOf(c.Object); ok {
			out[c.Object] = [2]float64{v.X, v.Y}
		}
	}
	for _, inc := range s.State.Incidents {
		if v, ok := s.placeOf(inc.Place); ok && inc.State == "open" {
			out[inc.Place] = [2]float64{v.X, v.Y}
		}
	}
	return out
}

func storageTitle(kind string) string {
	switch kind {
	case world.MarkStranded:
		return ui.Bridge("STRANDED", "NO DAEMONSET POD")
	case world.MarkDeed:
		return ui.Bridge("DEED", "VOLUME CLAIM")
	}
	return strings.ToUpper(kind)
}
