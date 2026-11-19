package hud

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	IntentTutorial  = "tutorial"
	OverlayChapters = "chapters"
	TutorialButton  = "menu:tutorial"

	TutorialStart   = "start"
	TutorialBack    = "back"
	TutorialBegin   = "begin"
	TutorialNext    = "next"
	TutorialLeave   = "leave"
	TutorialShow    = "show"
	TutorialMapCmd  = "map"
	TutorialAlmanac = "almanac"
	TutorialOpen    = "open"
	TutorialNote    = "note"

	tutorialPrefix = "tutorial"
	mapStopsPerRow = 5
	mapStopGap     = 10
	mapRowGap      = 6
	mapFirstCol    = 8
	mapTopRow      = 3
	mapLeftBend    = 3
	mapRightBend   = 53
	MapHarborX     = 2
	mapTilesW      = 56
	mapTilesH      = 28
	mapRibbonH     = 44
	MapTitleW      = 152
	mapTitleRows   = 3
	MapCardW       = 300
	MapCardPad     = 8
	mapMedal       = 30
)

var (
	mapOrigin = [2]int{KmapInner.X + (KmapInner.W-mapTilesW*KMapTile)/2, KmapInner.Y + mapRibbonH + 3}
	mapBack   = ui.Rect{X: KmapInner.X + KmapInner.W - 100, Y: KmapInner.Y + 8, W: 56, H: 18}
)

type MapStop struct {
	Chapter view.TutorialChapter
	X, Y    int
	State   string
}

const (
	StopDone    = "done"
	StopCurrent = "current"
	StopLocked  = "locked"
	StopLater   = "later"
)

type TutorialMap struct {
	Record  view.ChapterRecord
	Almanac bool
	Reduce  bool
	focus   int
	card    bool
	now     float64
}

func (m *TutorialMap) Open(rec view.ChapterRecord) {
	m.Record, m.card = rec, false
	m.focus = rec.Current().Number
}

func (m *TutorialMap) Focus() int      { return m.focus }
func (m *TutorialMap) Card() bool      { return m.card }
func (m *TutorialMap) Tick(dt float64) { m.now += dt }

func stopCol(row, i int) int {
	if row%2 == 0 {
		return mapFirstCol + i*mapStopGap
	}
	return mapFirstCol + (mapStopsPerRow-1-i)*mapStopGap
}

func StopRow(row int) int { return mapTopRow + row*mapRowGap }

func (m *TutorialMap) Stops() []MapStop {
	out := []MapStop{}
	for i, c := range view.Content.Chapters() {
		row, col := i/mapStopsPerRow, i%mapStopsPerRow
		out = append(out, MapStop{Chapter: c, X: stopCol(row, col), Y: StopRow(row), State: m.stopState(c)})
	}
	return out
}

func (m *TutorialMap) stopState(c view.TutorialChapter) string {
	switch {
	case !c.Built():
		return StopLater
	case m.Record.Done[c.ID]:
		return StopDone
	case m.Record.Locked(c):
		return StopLocked
	}
	return StopCurrent
}

func tileCenter(x, y int) (int, int) {
	return mapOrigin[0] + x*KMapTile + KMapTile/2, mapOrigin[1] + y*KMapTile + KMapTile/2
}

func StopRect(s MapStop) ui.Rect {
	cx, cy := tileCenter(s.X, s.Y)
	return ui.Rect{X: cx - MapTitleW/2, Y: cy - mapMedal/2 - 4, W: MapTitleW, H: mapMedal + 4 + len(StopTitle(s))*ui.LineHeight + ui.LineHeight + 4}
}

func StopTitle(s MapStop) []string {
	return Literal.WrapLines(s.Chapter.MapTitle(), MapTitleW, 1, mapTitleRows)
}

func MedalRect(s MapStop) ui.Rect {
	cx, cy := tileCenter(s.X, s.Y)
	return ui.Rect{X: cx - mapMedal/2, Y: cy - mapMedal/2, W: mapMedal, H: mapMedal}
}

type roadPoint struct {
	X, Y, stop int `json:"-"`
}

func MapRoadPath(n int) []roadPoint {
	rows := (n + mapStopsPerRow - 1) / mapStopsPerRow
	out := []roadPoint{}
	add := func(x, y int) {
		if len(out) > 0 && out[len(out)-1].X == x && out[len(out)-1].Y == y {
			return
		}
		out = append(out, roadPoint{X: x, Y: y})
	}
	line := func(x0, y0, x1, y1 int) {
		dx, dy := sign(x1-x0), sign(y1-y0)
		for x, y := x0, y0; ; x, y = x+dx, y+dy {
			add(x, y)
			if x == x1 && y == y1 {
				return
			}
		}
	}
	x, y := MapHarborX, StopRow(0)
	for row := range rows {
		y = StopRow(row)
		last := min(n-row*mapStopsPerRow, mapStopsPerRow) - 1
		end := stopCol(row, last)
		if row < rows-1 {
			end = mapRightBend
			if row%2 == 1 {
				end = mapLeftBend
			}
		}
		line(x, y, end, y)
		x = end
		if row < rows-1 {
			line(x, y, x, StopRow(row+1))
		}
	}
	stop := 0
	for i := range out {
		for k := stop; k < n; k++ {
			row, col := k/mapStopsPerRow, k%mapStopsPerRow
			if out[i].X == stopCol(row, col) && out[i].Y == StopRow(row) {
				stop = k + 1
				break
			}
		}
		out[i].stop = stop
	}
	return out
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

var mapRoadStates = map[string]string{StopDone: "learned", StopCurrent: "open", StopLocked: "locked", StopLater: "locked"}

func (m *TutorialMap) Roads() []KMapRoadTile {
	stops := m.Stops()
	path := MapRoadPath(len(stops))
	out := make([]KMapRoadTile, len(path))
	for i, p := range path {
		next := min(p.stop, len(stops)-1)
		if i > 0 && path[i-1].stop < p.stop {
			next = p.stop - 1
		}
		t := KMapRoadTile{X: p.X, Y: p.Y, State: mapRoadStates[stops[next].State]}
		for _, j := range []int{i - 1, i + 1} {
			if j < 0 || j >= len(path) {
				continue
			}
			q := path[j]
			t.N = t.N || q.X == p.X && q.Y == p.Y-1
			t.S = t.S || q.X == p.X && q.Y == p.Y+1
			t.E = t.E || q.Y == p.Y && q.X == p.X+1
			t.W = t.W || q.Y == p.Y && q.X == p.X-1
		}
		out[i] = t
	}
	return out
}

func (m *TutorialMap) stop(n int) (MapStop, bool) {
	stops := m.Stops()
	if n < 1 || n > len(stops) {
		return MapStop{}, false
	}
	return stops[n-1], true
}

func (m *TutorialMap) cardRect() ui.Rect {
	s, ok := m.stop(m.focus)
	if !ok {
		return ui.Rect{}
	}
	cx, cy := tileCenter(s.X, s.Y)
	h := m.cardHeight(s)
	x := cx + mapMedal/2 + 10
	if x+MapCardW > KmapInner.X+KmapInner.W-8 {
		x = cx - mapMedal/2 - 10 - MapCardW
	}
	y := min(max(cy-mapMedal/2, KmapInner.Y+mapRibbonH), KmapInner.Y+KmapInner.H-h-6)
	return ui.Rect{X: x, Y: y, W: MapCardW, H: h}
}

type cardRow struct {
	label string
	lines []string
	ink   bool
}

func (m *TutorialMap) cardRows(s MapStop) []cardRow {
	w := MapCardW - 2*MapCardPad
	c := s.Chapter
	rows := []cardRow{
		{lines: Literal.WrapLines(c.Title, w, 1, 2), ink: true},
		{lines: Literal.WrapLines(strings.Join(c.Concepts, "; "), w, 1, 3)},
		{label: "YOU WILL", lines: Literal.WrapLines(strings.Join(c.YouWill, " "), w, 1, 5), ink: true},
		{label: "COURSE", lines: Literal.WrapLines(c.Course, w, 1, 2)},
	}
	return append(rows, cardRow{lines: []string{m.cardNote(s)}})
}

func (m *TutorialMap) cardNote(s MapStop) string {
	switch s.State {
	case StopLater:
		return "COMING LATER: this chapter is not written yet"
	case StopLocked:
		return fmt.Sprintf("Needs chapter %d", m.Record.Current().Number)
	case StopDone:
		return fmt.Sprintf("DONE: %d steps", len(s.Chapter.Steps))
	}
	return fmt.Sprintf("%d steps", len(s.Chapter.Steps))
}

func (m *TutorialMap) cardHeight(s MapStop) int {
	h := MapCardPad + ui.LineHeight + 2
	for _, r := range m.cardRows(s) {
		if r.label != "" {
			h += ui.LineHeight
		}
		h += len(r.lines)*ui.LineHeight + 3
	}
	return h + TrialBtnH + MapCardPad
}

func chapterButton(id, label, tip string, x, y int) Button {
	return Button{ID: id, Label: label, Tip: tip, Rect: ui.Rect{X: x, Y: y, W: Literal.Width(label, 1) + 2*TrialBtnPad, H: TrialBtnH}}
}

func (m *TutorialMap) Buttons() []Button {
	out := []Button{}
	for _, s := range m.Stops() {
		out = append(out, Button{ID: fmt.Sprintf("%s:stop:%d", tutorialPrefix, s.Chapter.Number), Rect: StopRect(s), Tip: StructTip(TipTitle(ui.Settle(s.Chapter.Title)))})
	}
	out = append(out, Button{ID: tutorialPrefix + ":" + TutorialBack, Label: "BACK", Rect: mapBack, Tip: "Close the map [Esc]"},
		closeButtonAs(KnowledgeBox, tutorialPrefix+":"+TutorialBack+":close", "Close the map [Esc]"))
	if m.card {
		out = append(out, m.cardButtons()...)
	}
	return out
}

func (m *TutorialMap) cardButtons() []Button {
	s, ok := m.stop(m.focus)
	if !ok {
		return nil
	}
	r := m.cardRect()
	out := []Button{{ID: tutorialPrefix + ":card", Rect: r}}
	x, y := r.X+MapCardPad, r.Y+r.H-MapCardPad-TrialBtnH
	c := s.Chapter
	if start, ok := startLabel(s.State); ok {
		b := chapterButton(tutorialPrefix+":"+TutorialStart+":"+c.ID, start, "Begin "+c.Label()+" afresh: a chapter is never saved [Enter]", x, y)
		out = append(out, b)
		x += b.Rect.W + 4
	}
	if page := c.OwnPage(); m.Almanac && page != "" {
		b := chapterButton(tutorialPrefix+":"+TutorialAlmanac+":"+page, "ALMANAC", "Read this chapter's own page in the Almanac", x, y)
		out = append(out, b)
		x += b.Rect.W + 4
	}
	return append(out, chapterButton(tutorialPrefix+":close", "CLOSE", "Close this card [Esc]", x, y))
}

func startLabel(state string) (string, bool) {
	switch state {
	case StopDone:
		return "REPLAY", true
	case StopCurrent:
		return "START", true
	case StopLocked:
		return "START ANYWAY", true
	}
	return "", false
}

func TutorialIntent(cmd, target string) Intent {
	return Intent{Kind: IntentTutorial, Command: cmd, Target: target}
}

func (m *TutorialMap) Press(id string) []Intent {
	click := Intent{Kind: IntentSound, Command: "click"}
	rest, ok := strings.CutPrefix(id, tutorialPrefix+":")
	if !ok {
		return nil
	}
	verb, arg, _ := strings.Cut(rest, ":")
	switch verb {
	case "stop":
		if n, err := strconv.Atoi(arg); err == nil {
			m.focus, m.card = n, true
		}
		return []Intent{click}
	case TutorialStart:
		m.card = false
		return []Intent{click, TutorialIntent(TutorialStart, arg)}
	case TutorialAlmanac:
		return []Intent{click, TutorialIntent(TutorialAlmanac, arg)}
	case "close":
		m.card = false
		return []Intent{click}
	case TutorialBack:
		m.card = false
		return []Intent{click, TutorialIntent(TutorialBack, "")}
	}
	return nil
}

func (m *TutorialMap) Keys(in Input) []Intent {
	last := len(view.Content.Chapters())
	switch {
	case in.Pressed(ebiten.KeyRight) || in.Pressed(ebiten.KeyDown):
		m.focus = min(m.focus+1, last)
	case in.Pressed(ebiten.KeyLeft) || in.Pressed(ebiten.KeyUp):
		m.focus = max(m.focus-1, 1)
	case in.Pressed(ebiten.KeyHome):
		m.focus = m.Record.Current().Number
	case in.Pressed(ebiten.KeyEnter) && !m.card:
		m.card = true
	case in.Pressed(ebiten.KeyEnter):
		s, ok := m.stop(m.focus)
		if _, startable := startLabel(s.State); ok && startable {
			return m.Press(tutorialPrefix + ":" + TutorialStart + ":" + s.Chapter.ID)
		}
	case in.Pressed(ebiten.KeyEscape) && m.card:
		m.card = false
	case in.Pressed(ebiten.KeyEscape):
		return []Intent{TutorialIntent(TutorialBack, "")}
	default:
		return nil
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (m *TutorialMap) Clicked(in Input, b Button, hit bool) []Intent {
	if !in.Click {
		return nil
	}
	if !hit || b.ID == "overlay:body" {
		if m.card {
			m.card = false
		}
		return nil
	}
	if b.ID == tutorialPrefix+":card" {
		return nil
	}
	return m.Press(b.ID)
}

type TitleMap struct {
	TutorialMap
	active  bool
	hover   Button
	hovered bool
}

func (t *TitleMap) Open(rec view.ChapterRecord) {
	t.TutorialMap.Open(rec)
	t.Almanac, t.active = false, true
}

func (t *TitleMap) Close()       { t.active, t.card = false, false }
func (t *TitleMap) Active() bool { return t.active }

func (t *TitleMap) TitleButtons() []Button {
	return append([]Button{{ID: "overlay:body", Rect: KnowledgeBox}}, t.Buttons()...)
}

func (t *TitleMap) Update(in Input, dt float64) []Intent {
	t.Tick(dt)
	t.hover, t.hovered = HitButton(t.TitleButtons(), in.MX, in.MY)
	intents := t.Keys(in)
	if intents == nil {
		intents = t.Clicked(in, t.hover, t.hovered)
	}
	for _, it := range intents {
		if it.Kind == IntentTutorial && (it.Command == TutorialBack || it.Command == TutorialStart) {
			t.Close()
		}
	}
	return intents
}

func (t *TitleMap) Draw(dst *ebiten.Image, lib *gfx.Library) {
	if !t.active {
		return
	}
	hover := ""
	if t.hovered {
		hover = t.hover.ID
	}
	t.TutorialMap.Draw(dst, lib, hover)
}
