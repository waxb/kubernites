package app

import (
	"math"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

var ActionSounds = map[string]sound.SFX{
	"serve": sound.SFXCoin, "build": sound.SFXBuild, "summon": sound.SFXPodStart, "raiseMuster": sound.SFXRollout,
	"scaleMuster": sound.SFXRollout, "unlock": sound.SFXUnlock, "buySoulstone": sound.SFXUnlock, "ageUp": sound.SFXVictory,
	"chronicle": sound.SFXChronicle, view.ActCloseBerth: sound.SFXClick, "doorHearth": sound.SFXPodStart,
}

func (s *Session) Update(in hud.Input, dt float64, p *Pointer, muted bool) ([]hud.Intent, []sound.SFX) {
	s.Advance(dt)
	f := s.Frame(muted)
	intents := s.Hud.Update(in, f)
	var rest []hud.Intent
	var sfx []sound.SFX
	for _, it := range intents {
		out, fx := s.Handle(it)
		rest = append(rest, out...)
		sfx = append(sfx, fx...)
	}
	s.lookAtPanel()
	s.KeepSpeech()
	sfx = append(sfx, s.Notices()...)
	s.NoteAct(in)
	s.StepAutoPause(f)
	s.StepAway()
	if !s.Hud.KeyFocus() && !s.Typing {
		s.MoveCamera(in, dt, p)
		s.KeyZoom(in)
	}
	s.followStepHut(p)
	s.uncoverBeacon(p, f)
	blocked := s.Hud.Consumed() || s.Hud.Captures(in.MX, in.MY, f) || in.MY < hud.TopH
	s.HoverPile(in, blocked)
	if blocked {
		s.UpdateGhost(-1, -1)
		return rest, sfx
	}
	if in.Wheel != 0 && s.Hud.Overlay() == hud.OverlayNone {
		s.Zoom(in)
	}
	s.UpdateGhost(in.MX, in.MY)
	if in.Click && s.Hud.HelpMode() {
		return rest, append(sfx, s.HelpWorld(in)...)
	}
	if in.Click {
		s.GrabGround(in, p)
		sfx = append(sfx, s.ClickWorld(in)...)
		rest, s.notes = append(rest, s.notes...), nil
		return rest, sfx
	}
	return rest, sfx
}

func (s *Session) GrabGround(in hud.Input, p *Pointer) {
	if _, hit := s.Renderer.Pick(in.MX, in.MY); !hit && s.Placing == "" {
		p.Grab(in.MX, in.MY)
	}
}

func (s *Session) KeyZoom(in hud.Input) {
	if dir := ZoomKey(in); dir != 0 && s.Hud.Overlay() == hud.OverlayNone {
		cx, cy := ViewCenter()
		s.Zoom(hud.Input{Wheel: float64(dir), MX: int(cx), MY: int(cy)})
	}
}

func Sign(v float64) int {
	if v > 0 {
		return 1
	}
	return -1
}

func (s *Session) MoveCamera(in hud.Input, dt float64, p *Pointer) {
	dx, dy := PanVector(dt)
	ddx, ddy := p.drag(in.MX, in.MY, in.Held)
	ex, ey := p.EdgePan(in.MX, in.MY, dt)
	if dx != 0 || dy != 0 || ddx != 0 || ddy != 0 || ex != 0 || ey != 0 {
		s.Lens.Glide = Glide{}
		s.Cam = s.Cam.Pan(dx+ddx+ex, dy+ddy+ey)
		s.StepCam.MovedAt = s.Clock
	}
	s.Cam = s.Cam.Clamp(max(s.Site.Width, s.Site.Height), hud.Width, hud.ConsoleY)
}

func (s *Session) Handle(it hud.Intent) ([]hud.Intent, []sound.SFX) {
	if out, fx, ok := s.writeIntent(it); ok {
		return out, fx
	}
	switch it.Kind {
	case hud.IntentAction:
		return nil, s.actOrPlace(it)
	case hud.IntentPlace:
		s.PlaceAsked(it.Target)
	case hud.IntentCancel:
		s.Placing, s.Ghost = "", nil
		s.returnHome()
	case hud.IntentPause:
		s.Paused = !s.Paused
	case hud.IntentJump:
		s.Lens.Glide = Glide{}
		s.CenterOn(it.X, it.Y)
	case hud.IntentFocus:
		s.GlideTo(it.X, it.Y)
	case hud.IntentSelect:
		s.Selection = hud.Selection{}
	case hud.IntentHarbor:
		return nil, s.SelectHaven(it.Target)
	case hud.IntentGo:
		return nil, s.goTo(it.Target)
	case hud.IntentChoose:
		s.Choose(it)
	case hud.IntentSound:
		return nil, []sound.SFX{sound.SFX(it.Command)}
	case hud.IntentKubectl:
		s.steps.release()
		s.Poll = pollSeconds
	case hud.IntentNotice:
		s.Rail = s.Rail.Handle(it.Target, it.Command)
	case hud.IntentEcho:
		s.Echoes = Echo(s.Echoes, it, float64(s.State.FoundedAt.UnixNano())/1e9+s.State.RealmTime)
		s.Poll = pollSeconds
	default:
		return []hud.Intent{it}, nil
	}
	return nil, nil
}

func (s *Session) Act(a view.Action) []sound.SFX {
	return s.ActFrom(a, hud.FromButton)
}

func (s *Session) ActFrom(a view.Action, from string) []sound.SFX {
	if s.Live {
		s.Hud.Toast(hud.LiveTip, true)
		return []sound.SFX{sound.SFXError}
	}
	before := s.Realm.GameState()
	s.steps.release()
	done, err := s.Realm.Perform(a)
	s.recordAct(a, from, err)
	msg := done.Message
	s.EchoSpoken(done.Commands, from, msg, err)
	if err == nil {
		s.showDeltas(before)
		s.Hud.WatchGain(a, before)
		s.Hud.NoteAct(a.Action, before.RealmTime)
	}
	if PlacedStructure(a, err) {
		s.glanceOut()
	}
	s.Poll = pollSeconds
	s.ReplyToasts(a, msg, len(done.Commands) > 0, err)
	if err != nil {
		s.Coins.picked = false
		return []sound.SFX{sound.SFXError}
	}
	if a.Action == "serve" {
		after := s.Realm.GameState()
		carried := after.Pilgrims.HandServed > before.Pilgrims.HandServed
		s.Served(carried, after.Gold-before.Gold)
		if !carried && msg != "" {
			s.Hud.Toast(msg, false)
		}
	}
	if fx, ok := ActionSounds[a.Action]; ok {
		return []sound.SFX{fx}
	}
	return []sound.SFX{sound.SFXClick}
}

func (s *Session) ReplyToasts(a view.Action, msg string, echoed bool, err error) {
	switch {
	case err != nil:
		s.Hud.Toast(hud.StoryError(err.Error()), true)
	case msg != "" && a.Action == "claimGolden":
		s.Hud.Toast(msg, false)
		s.Echoes = Chronicled(s.Echoes, msg, s.echoClock())
	case msg != "" && !echoed && !quietReplies[a.Action]:
		s.Echoes = Chronicled(s.Echoes, msg, s.echoClock())
	}
}

var quietReplies = map[string]bool{"ack": true, "serve": true, "ageUp": true, view.ActReadAge: true}

func (s *Session) StartPlacing(kind string) {
	s.StartPlacingFor(kind, s.keepCrew(kind))
}

func (s *Session) PlaceAsked(kind string) {
	for _, b := range s.State.Buildables {
		if b.Kind == kind && b.Locked && b.Reason != "" {
			s.Hud.Toast(b.Reason+".", true)
			return
		}
	}
	s.StartPlacing(kind)
	s.showPlot()
}

func (s *Session) GhostAt(kind string, x, y int) *world.Ghost {
	if g, stall := s.StallGhost(kind, float64(x), float64(y)); stall {
		return g
	}
	w, h := s.Footprint(kind)
	return &world.Ghost{Kind: kind, X: x, Y: y, W: w, H: h, Valid: true, Class: s.PlotClass(kind, x, y)}
}

func (s *Session) PlotClass(kind string, x, y int) string {
	if !world.IsKeepKind(kind) {
		return ""
	}
	return s.SurveyAt(kind, x, y).Class
}

func (s *Session) Footprint(kind string) (int, int) {
	for _, b := range s.State.Buildables {
		if b.Kind == kind {
			return max(b.W, 1), max(b.H, 1)
		}
	}
	return 1, 1
}

func (s *Session) UpdateGhost(mx, my int) {
	if s.Placing == "" {
		s.Ghost = nil
		return
	}
	if mx < 0 {
		return
	}
	s.Ghost = s.ghostUnder(mx, my)
}

func (s *Session) ghostUnder(mx, my int) *world.Ghost {
	w, h := s.Footprint(s.Placing)
	gx, gy := s.Cam.ScreenToGrid(float64(mx), float64(my))
	if g, stall := s.StallGhost(s.Placing, gx, gy); stall {
		return g
	}
	x := int(math.Floor(gx+.5)) - (w-1)/2
	y := int(math.Floor(gy+.5)) - (h-1)/2
	sv := s.SurveyAt(s.Placing, x, y)
	return &world.Ghost{Kind: s.Placing, X: x, Y: y, W: w, H: h, Valid: sv.OK, Class: sv.Class}
}

func (s *Session) ClickWorld(in hud.Input) []sound.SFX {
	if s.Placing != "" && s.Ghost != nil {
		return s.Build(s.Ghost.X, s.Ghost.Y)
	}
	all := s.Renderer.PickAll(in.MX, in.MY)
	if len(all) == 0 {
		s.Selection, s.cycle = hud.Selection{}, PickCycle{}
		s.hush()
		s.QuayKeyHint()
		return nil
	}
	switch all[0].Kind {
	case world.HitPile:
		s.cycle = PickCycle{}
		return s.clickPile(all[0])
	case world.HitGolden:
		s.cycle = PickCycle{}
		return s.ClaimGleam(all[0])
	case world.HitStone:
		s.cycle = PickCycle{}
		return s.clickStone()
	case world.HitDormantStone:
		s.cycle = PickCycle{}
		return s.clickDormantStone()
	}
	return s.SelectPick(s.choosePick(all, in), in)
}

func (s *Session) SelectPick(hit world.Hit, in hud.Input) []sound.SFX {
	s.Selection = SelectionOf(hit, s.Pods)
	s.Inspect(s.Selection)
	switch hit.Kind {
	case world.HitWalker:
		s.Speak(s.Selection.Pod)
	case world.HitFamiliar:
		s.speakPet(hit.Pod, hit.Pet)
	default:
		s.hush()
	}
	if hit.Kind == world.HitHaven {
		s.harborOpened()
	}
	if in.DoubleClick {
		s.Hud.OpenDetails(s.Selection, hud.DetailDescribe)
	}
	return []sound.SFX{sound.SFXClick}
}

func (s *Session) Build(x, y int) []sound.SFX {
	if sv := s.SurveyAt(s.Placing, x, y); !sv.OK {
		return s.refuse(sv.Reason)
	}
	fx := s.Act(view.Action{Action: "build", ID: s.Placing, X: x, Y: y})
	if len(fx) > 0 && fx[0] != sound.SFXError {
		s.Placing, s.Ghost = "", nil
		s.returnHome()
	}
	return fx
}

func (s *Session) showDeltas(before view.GameState) {
	after := s.Realm.GameState()
	for _, d := range hud.MeterDeltas(before, after) {
		s.Hud.Delta(d.Meter, d.Value)
	}
}

var unrecordedActs = map[string]bool{"serve": true, "claimGolden": true}

func (s *Session) recordAct(a view.Action, from string, err error) {
	if s.Record == nil || unrecordedActs[a.Action] {
		return
	}
	fields := map[string]any{"action": a.Action, "from": from, "ok": err == nil}
	for k, v := range map[string]string{"id": a.ID, "name": a.Name, "image": a.Image, "soulstone": a.Soulstone} {
		if v != "" {
			fields[k] = v
		}
	}
	if a.Replicas != 0 {
		fields["replicas"] = a.Replicas
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	s.Record("act", fields)
}
