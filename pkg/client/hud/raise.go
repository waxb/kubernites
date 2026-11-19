package hud

import (
	"fmt"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	OverlayRaise    = "raise"
	raiseKey        = "raise"
	RaiseConfirm    = raiseKey + ":confirm"
	FromChooser     = "the chooser"
	raiseW          = 344
	raiseHead       = 30
	raiseRowH       = 14
	raiseRowStep    = 16
	raisePad        = 8
	raiseMaxRows    = 5
	raiseDetailRows = 6
	raiseBtnW       = 18
	raiseConfirmW   = 72
	defaultWorkers  = 2
)

type RaiseRow struct {
	Kind      string
	Name      string
	Crew      string
	Depot     string
	Images    []string
	Standing  bool
	Replicas  int
	DepotOpen bool
	Door      bool
	Keeper    string
	Foreman   bool
}

type raiseState struct {
	Pick    string `json:"-"`
	image   map[string]string
	workers map[string]int
}

func RaiseRows(f Frame) []RaiseRow {
	out := []RaiseRow{}
	sealed := false
	for _, k := range f.State.Freight.Kinds {
		sealed = sealed || k.Open && k.Sealed > 0
		if d, ok := soulstone.DepotFor(k.Kind); ok && k.Open {
			out = append(out, raiseRow(f, d, k.Name, k.DepotOpen, k.Door))
		}
	}
	if d, ok := soulstone.DepotFor(soulstone.KindSealed); ok && sealed {
		out = append(out, raiseRow(f, d, "sealed boxes", f.State.Freight.Seals.DepotOpen, false))
	}
	return out[:min(len(out), raiseMaxRows)]
}

func raiseRow(f Frame, d soulstone.Depot, name string, open, door bool) RaiseRow {
	row := RaiseRow{Kind: d.Kind, Name: name, Crew: d.Crew, Depot: d.Service, Images: OwnedClasses(f, d.Image), DepotOpen: open, Door: door,
		Keeper: crewKeeper(f, d.Crew), Foreman: raisesForemen(f)}
	for _, c := range f.State.Crews {
		if c.Name == d.Crew {
			row.Standing, row.Replicas = true, c.Replicas
		}
	}
	return row
}

func OwnedClasses(f Frame, image string) []string {
	base, ok := soulstone.Lookup(image)
	if !ok {
		return nil
	}
	out := []string{}
	for _, s := range f.State.Soulstones {
		if p, found := soulstone.Lookup(s.Image); found && s.Owned && !s.Cursed && p.Trade == base.Trade && !p.Familiar {
			out = append(out, p.Image)
		}
	}
	slices.SortStableFunc(out, func(a, b string) int { return newestFirst(a, b) })
	return out
}

func newestFirst(a, b string) int {
	switch {
	case soulstone.TagNewer(ImageTag(a), ImageTag(b)):
		return -1
	case soulstone.TagNewer(ImageTag(b), ImageTag(a)):
		return 1
	}
	return 0
}

func DefaultClass(images []string) string {
	for _, image := range images {
		if p, ok := soulstone.Lookup(image); ok && !p.Cursed {
			return image
		}
	}
	if len(images) > 0 {
		return images[0]
	}
	return ""
}

func (s raiseState) row(rows []RaiseRow) (RaiseRow, bool) {
	for _, r := range rows {
		if r.Crew == s.Pick {
			return r, true
		}
	}
	return RaiseRow{}, false
}

func (s raiseState) imageOf(r RaiseRow) string {
	if image, ok := s.image[r.Crew]; ok && slices.Contains(r.Images, image) {
		return image
	}
	return DefaultClass(r.Images)
}

func (s raiseState) workersOf(r RaiseRow) int {
	if n, ok := s.workers[r.Crew]; ok {
		return n
	}
	if r.Foreman {
		return foremanWorkers
	}
	return defaultWorkers
}

func (s raiseState) with(pick string, image map[string]string, workers map[string]int) raiseState {
	next := raiseState{Pick: pick, image: map[string]string{}, workers: map[string]int{}}
	for k, v := range s.image {
		next.image[k] = v
	}
	for k, v := range s.workers {
		next.workers[k] = v
	}
	for k, v := range image {
		next.image[k] = v
	}
	for k, v := range workers {
		next.workers[k] = v
	}
	return next
}

func guidedCrew(f Frame, rows []RaiseRow) string {
	line, ok := CrewLine(f)
	for _, r := range rows {
		if ok && line.Action != nil && (line.Action.Name == r.Crew || line.Action.Kind == r.Kind) {
			return r.Crew
		}
	}
	for _, r := range rows {
		if !r.Standing {
			return r.Crew
		}
	}
	if len(rows) > 0 {
		return rows[0].Crew
	}
	return ""
}

func (h *HUD) OpenRaise(f Frame) []Intent {
	click := []Intent{{Kind: IntentSound, Command: "click"}}
	if h.OverlayID == OverlayRaise {
		h.OverlayID = OverlayNone
		return click
	}
	rows := RaiseRows(f)
	if standingCrews(f) || slices.ContainsFunc(rows, func(r RaiseRow) bool { return r.Standing }) {
		return h.toggleCrews(f)
	}
	h.Ops.Raise = raiseState{Pick: guidedCrew(f, rows)}
	h.OverlayID = OverlayRaise
	return click
}

func RaiseAction(r RaiseRow, image string, workers int) (view.Action, bool) {
	if r.Foreman {
		return foremanAction(r, image, workers), image != ""
	}
	return view.Action{Action: "raiseCrew", Name: r.Crew, Image: image, Replicas: workers}, image != ""
}

func DepotAction(r RaiseRow) (view.Action, bool) {
	return view.Action{Action: "openDepot", Kind: r.Kind}, !r.DepotOpen && (r.Standing || r.Door)
}

func (h *HUD) pressRaise(rest string, f Frame) []Intent {
	rows := creatableRows(RaiseRows(f))
	s := h.Ops.Raise
	click := []Intent{{Kind: IntentSound, Command: "click"}}
	verb, arg := cutArg(rest)
	switch verb {
	case "pick":
		h.Ops.Raise = s.with(arg, nil, nil)
		return click
	case "image", "workers":
		h.Ops.Raise = h.stepRaise(s, rows, verb, arg == "up")
		return click
	case "confirm":
		return h.confirmRaise(s, rows)
	}
	return nil
}

func (h *HUD) stepRaise(s raiseState, rows []RaiseRow, verb string, up bool) raiseState {
	r, ok := s.row(rows)
	if !ok {
		return s
	}
	if verb == "workers" {
		n := s.workersOf(r) - 1
		if up {
			n += 2
		}
		return s.with(s.Pick, nil, map[string]int{r.Crew: min(max(n, 1), MaxCrewSize)})
	}
	i := slices.Index(r.Images, s.imageOf(r))
	if len(r.Images) == 0 {
		return s
	}
	if up {
		i = (i + 1) % len(r.Images)
	} else {
		i = (i + len(r.Images) - 1) % len(r.Images)
	}
	return s.with(s.Pick, map[string]string{r.Crew: r.Images[i]}, nil)
}

func (h *HUD) confirmRaise(s raiseState, rows []RaiseRow) []Intent {
	r, ok := s.row(rows)
	if !ok {
		return nil
	}
	a, ok := RaiseAction(r, s.imageOf(r), s.workersOf(r))
	if !ok {
		return nil
	}
	h.OverlayID = OverlayNone
	return []Intent{ActFrom(a, FromChooser)}
}

func cutArg(rest string) (string, string) {
	verb, arg, _ := strings.Cut(rest, ":")
	return verb, arg
}

func CrewChooserLine(r RaiseRow, workers int) string {
	if r.Foreman && !r.Standing {
		return foremanChooserLine(r, workers)
	}
	return ui.Bridge(fmt.Sprintf("A crew (Deployment) keeps %d %s running: the shift replaces any that fall.", workers, r.Crew),
		fmt.Sprintf("A Deployment keeps %d %s Pods running: its ReplicaSet replaces any that fail.", workers, singularOf(r.Crew)))
}

func RaiseCommand(r RaiseRow, image string, workers int) string {
	if image == "" || r.Foreman {
		return ""
	}
	return view.CrewCommand(r.Crew, image, workers)
}

func DepotCommandOf(r RaiseRow, f Frame) string {
	if door, ok := doorPod(f); ok && !r.Standing && r.Door {
		return view.DoorExposeCommand(r.Kind, door)
	}
	return view.DepotCommand(r.Kind)
}

func doorPod(f Frame) (string, bool) {
	for _, p := range f.Pods {
		if p.Static && p.Node == view.TowerName && !p.Terminating {
			return p.Name, true
		}
	}
	return "", false
}

func init() {
	prefixHandlers[raiseKey] = (*HUD).pressRaise
	Overlays[OverlayRaise] = OverlaySpec{(*HUD).raiseButtons, (*HUD).drawRaise}
	overlayPrefixes[raiseKey] = OverlayRaise
	overlayOpeners[OverlayRaise] = CmdCrew
}
