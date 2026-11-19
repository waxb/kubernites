package app

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	DevSnapEvery   = 10.0
	DevMarkMuch    = "too-much"
	DevMarkSlow    = "too-slow"
	DevNoteKind    = "note"
	devStripY      = hud.ConsoleY - 18
	devButtonW     = 96
	devButtonH     = 14
	devNoteW       = 520
	devNoteMax     = 400
	devNoteLines   = 6
	devTextPad     = 6
	DevShotsFolder = "shots"
)

var devButtons = []struct{ kind, label string }{
	{DevMarkMuch, "TOO MUCH [F8]"}, {DevMarkSlow, "TOO SLOW [F9]"}, {DevNoteKind, "NOTE [F10]"},
}

type devNote struct {
	Text      []rune `json:"-"`
	realm     float64
	shot      string
	wasPaused bool
}

type devTrack struct {
	sess      *Session
	attn      devAttn
	primed    bool
	nextSnap  float64
	lastObj   string
	lastIntro string
	lastPanel string
	procAfter float64
	calam     map[string]bool
	alerts    map[string]bool
	lastRealm float64
}

type DevRecorder struct {
	Dir     string   `json:"-"`
	Out     *os.File `json:"-"`
	enc     *json.Encoder
	log     *slog.Logger
	started time.Time
	track   devTrack
	Note    *devNote `json:"-"`
	ShotReq []string `json:"-"`
	shots   int
	saving  sync.WaitGroup
}

func OpenDevRecorder(root, version string, log *slog.Logger) (*DevRecorder, error) {
	start := time.Now()
	dir := filepath.Join(root, start.Format("20060102-150405"))
	if err := os.MkdirAll(filepath.Join(dir, DevShotsFolder), 0o755); err != nil {
		return nil, fmt.Errorf("create the playtest folder: %w", err)
	}
	out, err := os.Create(filepath.Join(dir, "log.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("create the playtest log: %w", err)
	}
	d := &DevRecorder{Dir: dir, Out: out, enc: json.NewEncoder(out), log: log, started: start, track: devTrack{calam: map[string]bool{}}}
	d.write("start", 0, map[string]any{"version": version, "at": start.Format(time.RFC3339)})
	log.Info("recording the playtest", "folder", dir)
	return d, nil
}

func (d *DevRecorder) write(kind string, realmTime float64, fields map[string]any) {
	if d.Out == nil {
		return
	}
	line := map[string]any{"kind": kind, "wall": round1(time.Since(d.started).Seconds()), "realm": round1(realmTime)}
	for k, v := range fields {
		line[k] = v
	}
	if err := d.enc.Encode(line); err != nil {
		d.log.Warn("playtest log write failed", "err", err)
	}
}

func round1(v float64) float64 { return float64(int(v*10)) / 10 }

func (d *DevRecorder) Close() {
	if d == nil || d.Out == nil {
		return
	}
	d.saving.Wait()
	d.write("stop", d.track.lastRealm, nil)
	if err := d.Out.Close(); err != nil {
		d.log.Warn("close the playtest log", "err", err)
	}
	d.Out, d.enc, d.ShotReq = nil, nil, nil
}

func (d *DevRecorder) attach(s *Session) {
	if d.track.sess == s {
		return
	}
	if d.track.sess != nil {
		d.write("new-realm", s.State.RealmTime, map[string]any{"after": d.track.lastRealm})
	}
	d.track = devTrack{sess: s, calam: map[string]bool{}, attn: devAttn{open: map[int]bool{}}}
	s.Hud.SetAttention(true)
	s.Record = func(kind string, fields map[string]any) { d.write(kind, s.State.RealmTime, fields) }
}

func (d *DevRecorder) intercept(in hud.Input, s *Session) hud.Input {
	d.attach(s)
	if d.Note != nil {
		d.typeNote(in, s)
		s.Typing = d.Note != nil
		return hud.Input{MX: in.MX, MY: in.MY}
	}
	kind := ""
	switch {
	case in.Pressed(ebiten.KeyF8):
		kind = DevMarkMuch
	case in.Pressed(ebiten.KeyF9):
		kind = DevMarkSlow
	case in.Pressed(ebiten.KeyF10):
		kind = DevNoteKind
	}
	if in.Click {
		for i, b := range devButtons {
			if DevButtonRect(i).Contains(in.MX, in.MY) {
				kind, in.Click = b.kind, false
			}
		}
	}
	switch kind {
	case DevMarkMuch, DevMarkSlow:
		shot := d.RequestShot(kind, s.State.RealmTime)
		d.write("mark", s.State.RealmTime, merged(d.snapshot(s), map[string]any{"mark": kind, "shot": shot}))
		s.Hud.Toast("Marked for the record: "+strings.ReplaceAll(kind, "-", " "), false)
	case DevNoteKind:
		d.Note = &devNote{realm: s.State.RealmTime, shot: d.RequestShot("note", s.State.RealmTime), wasPaused: s.Paused}
		s.Paused, s.Typing = true, true
	}
	return in
}

func (d *DevRecorder) typeNote(in hud.Input, s *Session) {
	n := d.Note
	switch {
	case in.Pressed(ebiten.KeyEscape):
		d.endNote(s, false)
		return
	case in.Pressed(ebiten.KeyEnter) || in.Pressed(ebiten.KeyNumpadEnter):
		d.endNote(s, true)
		return
	case in.Pressed(ebiten.KeyBackspace) && len(n.Text) > 0:
		n.Text = n.Text[:len(n.Text)-1]
	}
	for _, r := range in.Chars {
		if r >= ' ' && len(n.Text) < devNoteMax {
			n.Text = append(n.Text, r)
		}
	}
}

func (d *DevRecorder) endNote(s *Session, keep bool) {
	n := d.Note
	d.Note = nil
	s.Paused = n.wasPaused
	if !keep || strings.TrimSpace(string(n.Text)) == "" {
		d.write("note-cancelled", n.realm, map[string]any{"shot": n.shot})
		return
	}
	d.write("note", n.realm, merged(d.snapshot(s), map[string]any{"note": string(n.Text), "shot": n.shot}))
	s.Hud.Toast("Note saved to the playtest record", false)
}

func (d *DevRecorder) RequestShot(kind string, realmTime float64) string {
	d.shots++
	name := fmt.Sprintf("%03d-%s-%05.0f.png", d.shots, kind, realmTime)
	d.ShotReq = append(slices.Clone(d.ShotReq), name)
	return filepath.Join(DevShotsFolder, name)
}

func (d *DevRecorder) capture(screen *ebiten.Image) {
	if s := d.track.sess; s != nil {
		s.Hud.NoteDrawn(worldSurfaces(s.Renderer.Drawn()), s.State.RealmTime)
	}
	if len(d.ShotReq) == 0 {
		return
	}
	img := image.NewRGBA(screen.Bounds())
	screen.ReadPixels(img.Pix)
	d.FlushShots(img)
}

func (d *DevRecorder) FlushShots(img image.Image) {
	names := d.ShotReq
	d.ShotReq = nil
	for _, name := range names {
		d.saving.Go(func() { d.SaveShot(name, img) })
	}
}

func (d *DevRecorder) SaveShot(name string, img image.Image) {
	f, err := os.Create(filepath.Join(d.Dir, DevShotsFolder, name))
	if err != nil {
		d.log.Warn("playtest screenshot", "err", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		d.log.Warn("playtest screenshot", "err", err)
	}
}

func (d *DevRecorder) Observe(s *Session) {
	gs := s.State
	now := gs.RealmTime
	t := d.track
	t.lastRealm = now
	if !t.primed {
		t.primed, t.procAfter = true, latestProclamationAt(gs)
		d.write("resume", now, d.snapshot(s))
	}
	if now >= t.nextSnap {
		d.write("snap", now, d.snapshot(s))
		t.nextSnap = now + DevSnapEvery
	}
	if o, ok := shownObjectiveOf(gs); ok && o.ID != t.lastObj {
		d.write("objective", now, map[string]any{"id": o.ID, "text": o.Text, "progress": o.Progress})
		t.lastObj = o.ID
	}
	t.lastIntro = d.noteIntro(gs, t.lastIntro, now)
	if panel := s.Hud.Overlay(); panel != t.lastPanel {
		d.write("panel", now, map[string]any{"open": panel})
		t.lastPanel = panel
	}
	t.procAfter = d.noteProclamations(gs, t.procAfter)
	t.calam = d.noteCalamities(gs, t.calam, now)
	t.alerts = d.noteAlerts(s.Rail.Notices, t.alerts, now)
	t.attn = d.noteAttention(s, t.attn)
	d.track = t
}

func (d *DevRecorder) noteIntro(gs view.GameState, last string, now float64) string {
	intro := ""
	if gs.Intro != nil {
		intro = gs.Intro.ID
	}
	if intro != last && intro != "" {
		d.write("card", now, map[string]any{"id": intro, "title": gs.Intro.Title})
	}
	return intro
}

func (d *DevRecorder) noteProclamations(gs view.GameState, after float64) float64 {
	for _, p := range gs.Proclamations {
		if p.T > after {
			d.write("notice", p.T, map[string]any{"type": p.Kind, "title": p.Title, "text": p.Text})
		}
	}
	return max(after, latestProclamationAt(gs))
}

func (d *DevRecorder) noteCalamities(gs view.GameState, seen map[string]bool, now float64) map[string]bool {
	live := map[string]bool{}
	for _, c := range gs.Calamities {
		live[c.ID] = true
		if !seen[c.ID] {
			d.write("calamity", now, map[string]any{"id": c.ID, "type": c.Kind, "cause": c.Cause})
		}
	}
	return live
}

func (d *DevRecorder) noteAlerts(notices []hud.Notice, open map[string]bool, now float64) map[string]bool {
	live := map[string]bool{}
	for _, n := range notices {
		if n.Kind != hud.NoticeDanger {
			continue
		}
		live[n.Title] = true
		if !open[n.Title] {
			d.write("alert", now, map[string]any{"id": n.ID, "text": n.Title, "detail": n.Detail})
		}
	}
	return live
}

func openAlerts(notices []hud.Notice) int {
	n := 0
	for _, notice := range notices {
		if notice.Kind == hud.NoticeDanger {
			n++
		}
	}
	return n
}

func latestProclamationAt(gs view.GameState) float64 {
	last := 0.0
	for _, p := range gs.Proclamations {
		last = max(last, p.T)
	}
	return last
}

func shownObjectiveOf(gs view.GameState) (struct{ ID, Text, Progress string }, bool) {
	for _, o := range gs.Objectives {
		if !o.Done {
			return struct{ ID, Text, Progress string }{o.ID, o.Text, o.Progress}, true
		}
	}
	return struct{ ID, Text, Progress string }{}, false
}

func (d *DevRecorder) snapshot(s *Session) map[string]any {
	gs := s.State
	crews := []map[string]any{}
	for _, c := range gs.Crews {
		crews = append(crews, map[string]any{"name": c.Name, "replicas": c.Replicas, "ready": c.Ready, "pending": c.Pending})
	}
	calamities := []string{}
	for _, c := range gs.Calamities {
		calamities = append(calamities, c.ID)
	}
	slices.Sort(calamities)
	out := map[string]any{
		"clock": hud.Clock(gs.RealmTime, gs.Pilgrims.DayPhase), "speed": gs.Speed, "paused": s.Paused, "age": gs.Age,
		"gold": round1(gs.Gold), "wisdom": round1(gs.Wisdom), "income": round1(gs.Economy.Income),
		"arriving": round1(gs.Economy.Arriving), "carried": round1(gs.Economy.Carried), "bottleneck": gs.Economy.Bottleneck,
		"waiting": round1(gs.Freight.Waiting), "crews": crews, "calamities": calamities, "panel": s.Hud.Overlay(),
		"alerts": openAlerts(s.Rail.Notices),
	}
	if o, ok := shownObjectiveOf(gs); ok {
		out["objective"], out["progress"] = o.ID, o.Progress
	}
	if gs.Intro != nil {
		out["card"] = gs.Intro.ID
	}
	return out
}

func merged(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func DevButtonRect(i int) ui.Rect {
	return ui.Rect{X: 8 + i*(devButtonW+4), Y: devStripY, W: devButtonW, H: devButtonH}
}

func (d *DevRecorder) draw(dst *ebiten.Image) {
	var font ui.Font
	for i, b := range devButtons {
		r := DevButtonRect(i)
		ui.Fill(dst, r, ui.Tint(ui.Palette.ConsoleEdge, 220))
		ui.Stroke(dst, r, ui.Palette.Gold)
		font.Draw(dst, b.label, r.X+4, r.Y+3, 1, ui.Palette.Text)
	}
	dot := DevButtonRect(len(devButtons))
	font.Draw(dst, "REC", dot.X+2, dot.Y+3, 1, ui.Palette.RedText)
	if d.Note != nil {
		d.drawNote(dst, font)
	}
}

func (d *DevRecorder) drawNote(dst *ebiten.Image, font ui.Font) {
	text := string(d.Note.Text) + "_"
	lines := font.WrapLines(text, devNoteW-2*devTextPad, 1, devNoteLines)
	h := 3*ui.LineHeight + len(lines)*ui.LineHeight + 2*devTextPad
	r := ui.Rect{X: hud.Width/2 - devNoteW/2, Y: hud.Height/2 - h/2, W: devNoteW, H: h}
	ui.Fill(dst, r, ui.Palette.Console)
	ui.Stroke(dst, r, ui.Palette.GoldBright)
	x, y := r.X+devTextPad, r.Y+devTextPad
	font.Draw(dst, "NOTE FOR THE PLAYTEST RECORD (the realm is paused)", x, y, 1, ui.Palette.GoldBright)
	y += ui.LineHeight
	font.Draw(dst, "Enter saves it with the time and a screenshot. Esc cancels.", x, y, 1, ui.Palette.TextDim)
	y += 2 * ui.LineHeight
	for _, l := range lines {
		font.Draw(dst, l, x, y, 1, ui.Palette.Cream)
		y += ui.LineHeight
	}
}
