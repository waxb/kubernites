package app

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
)

const (
	pickStep    = 3
	plotReach   = 24
	buttonLines = 3
	plotPrefix  = "plot:"
	plotGhost   = "plot:suggested"
)

func (d *Driver) Buttons(f hud.Frame, runs []ui.TextRun) []LookButton {
	out := []LookButton{}
	seen := map[string]bool{}
	tips := map[string]string{}
	for _, c := range d.Sess().Hud.TipCards(f) {
		tips[c.ID] = ui.Plain(ui.Name(c.Label))
	}
	all := d.Sess().Hud.Buttons(f)
	for _, b := range all {
		if b.Rect.W <= 0 || b.Rect.H <= 0 || seen[b.ID] {
			continue
		}
		seen[b.ID] = true
		region := regionOfRect(runs, b.Rect)
		label := cmp.Or(ui.Plain(ui.Name(b.Label)), tips[b.ID], noticeLabel(b))
		if raw := ui.Plain(b.Label); raw != label && buttonText(runs, b.Rect, "") == raw {
			label = raw
		}
		text := buttonText(runs, b.Rect, label)
		if label == "" {
			label, text = bareLabel(b.ID, region, text), ""
		}
		_, _, covered := visiblePoint(all, b)
		out = append(out, LookButton{ID: b.ID, Label: label, Text: text, Region: region, Enabled: !b.Disabled, CoveredBy: covered})
	}
	return append(out, minimapButtons(f)...)
}

func noticeLabel(b hud.Button) string {
	if !strings.HasPrefix(b.ID, "notice:") {
		return ""
	}
	title, _, _ := strings.Cut(b.Tip, "\n")
	if title == "" {
		return ""
	}
	return "rail notice: " + ui.Plain(ui.Name(title))
}

func buttonText(runs []ui.TextRun, r ui.Rect, label string) string {
	inside := []ui.TextRun{}
	for _, run := range runs {
		if r.Contains(run.Rect.X+run.Rect.W/2, run.Rect.Y+run.Rect.H/2) {
			inside = append(inside, run)
		}
	}
	lines := readLines(inside)
	text := strings.Join(lines, " / ")
	if len(lines) > buttonLines || text == label {
		return ""
	}
	return text
}

var containers = []string{"overlay:body", "window:", "well:", "panel:"}

func bareLabel(id, region, text string) string {
	for _, c := range containers {
		if strings.HasPrefix(id, c) && region != "" {
			return "(the " + RegionLabel(region) + " itself)"
		}
	}
	if text != "" {
		return "(" + text + ")"
	}
	return "(icon: " + strings.ReplaceAll(id[strings.LastIndex(id, ":")+1:], "-", " ") + ")"
}

func regionOfRect(runs []ui.TextRun, r ui.Rect) string {
	count := map[string]int{}
	best := ""
	for _, run := range runs {
		if r.Contains(run.Rect.X+run.Rect.W/2, run.Rect.Y+run.Rect.H/2) {
			count[run.Region]++
			if count[run.Region] > count[best] {
				best = run.Region
			}
		}
	}
	if best != "" {
		return best
	}
	for _, run := range runs {
		if run.Rect.Contains(r.X+r.W/2, r.Y+r.H/2) {
			return run.Region
		}
	}
	return ""
}

func hitObjectID(h world.Hit) string {
	raw := strings.TrimPrefix(h.ID, h.Kind+":")
	return h.Kind + ":" + strings.ReplaceAll(ui.Plain(ui.Name(raw)), " ", "-")
}

var placeLabels = map[string]string{
	world.HitHaven: "a harbor berth", world.HitStone: "the Speaking Stone", world.HitDormantStone: "the dormant Speaking Stone", world.HitGolden: "a golden gleam",
}

func hitLabel(h world.Hit) string {
	if kind, ok := world.PileKind(h.ID); ok && h.Kind == world.HitPile {
		return ui.Plain(ui.Name(world.KindName(kind) + "s waiting on the quay"))
	}
	switch {
	case h.Kind == world.HitFamiliar:
		return ui.Plain(h.Pet + " of " + h.Pod.Name)
	case h.Structure.Label != "":
		return ui.Plain(ui.Name(h.Structure.Label))
	case h.Pod.Name != "":
		return ui.Plain(ui.Name(h.Pod.Profession + " " + h.Pod.Name))
	case placeLabels[h.Kind] != "":
		return placeLabels[h.Kind]
	}
	return ui.Plain(ui.Name(h.ID))
}

func havenLabel(h world.Hit, f hud.Frame) string {
	if h.Kind != world.HitHaven {
		return hitLabel(h)
	}
	for _, p := range f.State.Ports {
		if p.ID == strings.TrimPrefix(h.ID, h.Kind+":") && p.Name != "" {
			return p.Name + ": " + hitLabel(h)
		}
	}
	return hitLabel(h)
}

func (d *Driver) World(f hud.Frame) []LookObject {
	out := []LookObject{}
	seen := map[string]bool{}
	for _, h := range d.Sess().Renderer.Hits() {
		id := hitObjectID(h)
		if seen[id] {
			continue
		}
		if _, _, ok := d.pickPoint(h, f); ok {
			seen[id] = true
			out = append(out, LookObject{ID: id, Kind: h.Kind, Label: havenLabel(h, f)})
		}
	}
	return append(out, d.plots(f)...)
}

func (d *Driver) pickPoint(h world.Hit, f hud.Frame) (int, int, bool) {
	r := h.Rect
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	for radius := 0; radius <= max(r.W, r.H)/2; radius += pickStep {
		for _, p := range [][2]int{{cx, cy + radius}, {cx, cy - radius}, {cx - radius, cy}, {cx + radius, cy}} {
			if d.pickable(h, p[0], p[1], f) {
				return p[0], p[1], true
			}
		}
	}
	return 0, 0, false
}

func (d *Driver) pickable(h world.Hit, x, y int, f hud.Frame) bool {
	s := d.Sess()
	if x < 0 || y < hud.TopH || x >= hud.Width || y >= hud.Height || s.Hud.Captures(x, y, f) {
		return false
	}
	got, ok := s.Renderer.Pick(x, y)
	return ok && got.Kind == h.Kind && got.ID == h.ID
}

func (d *Driver) plots(f hud.Frame) []LookObject {
	s := d.Sess()
	if s.Placing == "" {
		return nil
	}
	out := []LookObject{}
	kind := s.Placing
	if b, ok := s.buildable(s.Placing); ok && b.Name != "" {
		kind = b.Name
	}
	kind = ui.Plain(ui.Name(kind))
	if g := s.Land.suggested; g != nil {
		label := fmt.Sprintf("the suggested plot for %s at %d,%d", kind, g.X, g.Y)
		if _, _, ok := d.plotPoint(g.X, g.Y, f); !ok {
			label += ", " + d.hiddenPlot(g.X, g.Y)
		}
		out = append(out, LookObject{ID: plotGhost, Kind: "plot", Label: label})
	}
	for i, m := range s.SpotMarks() {
		if _, _, ok := d.plotPoint(m.X, m.Y, f); !ok {
			continue
		}
		label := strings.TrimSpace(fmt.Sprintf("%s at %d,%d %s %s", kind, m.X, m.Y, m.Note, m.Machine))
		out = append(out, LookObject{ID: plotPrefix + strconv.Itoa(i), Kind: "plot", Label: ui.Plain(ui.Name(label))})
	}
	return out
}

func (d *Driver) plotOf(id string) (int, int, bool) {
	s := d.Sess()
	if s.Placing == "" {
		return 0, 0, false
	}
	rest := strings.TrimPrefix(id, plotPrefix)
	if id == plotGhost {
		g := s.Land.suggested
		if g == nil {
			return 0, 0, false
		}
		return g.X, g.Y, true
	}
	if xs, ys, ok := strings.Cut(rest, ","); ok {
		x, errX := strconv.Atoi(xs)
		y, errY := strconv.Atoi(ys)
		return x, y, errX == nil && errY == nil
	}
	i, err := strconv.Atoi(rest)
	marks := s.SpotMarks()
	if err != nil || i < 0 || i >= len(marks) {
		return 0, 0, false
	}
	return marks[i].X, marks[i].Y, true
}

func (d *Driver) plotPoint(px, py int, f hud.Frame) (int, int, bool) {
	s := d.Sess()
	w, h := s.Footprint(s.Placing)
	cx, cy := world.ScreenOf(s.Cam, s.Site, world.Vec{X: float64(px + (w-1)/2), Y: float64(py + (h-1)/2)})
	for radius := 0; radius <= plotReach; radius += pickStep {
		for _, p := range [][2]int{{0, 0}, {0, radius}, {0, -radius}, {-radius, 0}, {radius, 0}} {
			x, y := int(cx)+p[0], int(cy)+p[1]
			if d.groundAt(x, y, f) {
				if g := s.ghostUnder(x, y); g != nil && g.X == px && g.Y == py {
					return x, y, true
				}
			}
		}
	}
	return 0, 0, false
}

func (d *Driver) hiddenPlot(px, py int) string {
	s := d.Sess()
	x, y := world.ScreenOf(s.Cam, s.Site, world.Vec{X: float64(px), Y: float64(py)})
	cx, cy := ViewCenter()
	keys := []string{}
	for _, k := range []struct {
		far  bool
		name string
	}{{x < cx-keyPan, "ArrowLeft"}, {x > cx+keyPan, "ArrowRight"}, {y < cy-keyPan, "ArrowUp"}, {y > cy+keyPan, "ArrowDown"}} {
		if k.far {
			keys = append(keys, k.name)
		}
	}
	if len(keys) == 0 {
		return "hidden under a window: close the window"
	}
	return "hidden: off screen or under a window; press " + strings.Join(keys, " and ") + " to bring it nearer the middle"
}

func (d *Driver) groundAt(x, y int, f hud.Frame) bool {
	return x >= 0 && y >= hud.TopH && x < hud.Width && y < hud.Height && !d.Sess().Hud.Captures(x, y, f)
}

func (d *Driver) Target(id string, f hud.Frame, hover bool) (int, int, error) {
	if x, y, ok, err := d.minimapTarget(id, f); ok || err != nil {
		return x, y, err
	}
	all := d.Sess().Hud.Buttons(f)
	for _, b := range all {
		if b.ID == id && b.Rect.W > 0 {
			if b.Disabled && !hover {
				return 0, 0, fmt.Errorf("%s is shown but cannot be pressed now", id)
			}
			x, y, covered := visiblePoint(all, b)
			if covered != "" && !(covered == hud.KMapCardID && strings.HasPrefix(id, "grim:")) {
				return 0, 0, fmt.Errorf("%s is under %s: close or move that first", id, covered)
			}
			return x, y, nil
		}
	}
	if strings.HasPrefix(id, plotPrefix) {
		px, py, ok := d.plotOf(id)
		if !ok {
			return 0, 0, fmt.Errorf("no plot %q: pick a building in BUILD first, then click a plot from look", id)
		}
		if x, y, ok := d.plotPoint(px, py, f); ok {
			return x, y, nil
		}
		return 0, 0, fmt.Errorf("plot %d,%d is not on screen: move the view first", px, py)
	}
	d.Capture()
	for _, h := range d.Sess().Renderer.Hits() {
		if hitObjectID(h) == id {
			if x, y, ok := d.pickPoint(h, f); ok {
				return x, y, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("nothing called %q is on screen to click: look again", id)
}
