package app

import (
	"errors"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/waxb/kubernites/pkg/client"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	chapterStartFailed = "The chapter could not be set: "
	chapterRecordLost  = "Your finished chapters could not be saved: "
	chapterSaveNote    = "A chapter is never saved: your realm was saved as it began, and nothing here changes it."
	chapterNamesNote   = "In LEARN KUBERNETES every name is the Kubernetes one; your NAMES setting returns when you leave."
	chapterTrialNote   = "Leave the chapter first: a Trial waits for your realm."
	chapterInTrialNote = "Leave the trial first: LEARN KUBERNETES waits for your realm."
)

type TutorialPlay struct {
	Run       client.ChapterRun `json:"-"`
	Home      *Session          `json:"-"`
	prev      *Session
	fromTitle bool
	Record    view.ChapterRecord `json:"-"`
	Loaded    bool               `json:"-"`
	recorded  bool
	cheered   bool
	shown     int
	glided    int
}

func (g *Game) InChapter() bool { return g.Yard.Run != nil }

func (g *Game) tutorialRecordPath() string {
	if g.Capture != nil {
		return ""
	}
	return campaignHook.ChapterRecordPath(g.Opts.SavePath)
}

func (g *Game) loadTutorialRecord() {
	if g.Yard.Loaded {
		return
	}
	g.Yard.Loaded, g.Yard.Record = true, campaignHook.NewChapterRecord()
	path := g.tutorialRecordPath()
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		g.Log.Warn("tutorial progress ignored", "err", err)
		return
	}
	if rec, err := campaignHook.ParseChapterRecord(data); err != nil {
		g.Log.Warn("tutorial progress ignored", "err", err)
	} else {
		g.Yard.Record = rec
	}
}

func (g *Game) saveTutorialRecord() error {
	path := g.tutorialRecordPath()
	if path == "" {
		return nil
	}
	data, err := campaignHook.MarshalChapterRecord(g.Yard.Record)
	if err != nil {
		return err
	}
	return SaveAtomic(path, data)
}

func (g *Game) openChapterMap() {
	g.loadTutorialRecord()
	g.Chapters.Open(g.Yard.Record)
}

func (g *Game) TutorialIntent(it hud.Intent) {
	run := g.Yard.Run
	switch it.Command {
	case hud.TutorialOpen:
		g.openChapterMap()
	case hud.TutorialStart:
		g.StartChapter(it.Target)
	case hud.TeacherBegin:
		if run != nil {
			run.Begin()
			g.Sess.Paused = false
		}
	case hud.TeacherNext:
		if run != nil {
			run.Next()
		}
	case hud.TeacherWho:
		if run != nil {
			if n, err := strconv.Atoi(it.Target); err == nil {
				run.Pick(n - 1)
			}
		}
	case hud.TeacherShow:
		g.ShowTeacherTargets()
	case hud.TeacherLeave:
		g.leaveChapter(false)
	case hud.TeacherMap:
		g.leaveChapter(true)
	case hud.TutorialNote:
		if run != nil {
			run.Note(it.Target)
		}
	}
	if g.InChapter() && g.Sess != nil {
		g.Sess.Refresh()
	}
	g.syncTeacher()
}

func (g *Game) StartChapter(id string) {
	g.loadTutorialRecord()
	if g.InTrial() {
		g.Sess.Hud.Toast(chapterInTrialNote, true)
		return
	}
	run, err := campaignHook.StartChapter(id)
	if err != nil {
		g.Log.Error("chapter setup failed", "chapter", id, "err", err)
		if g.Sess != nil && g.Scene == SceneRealm {
			g.Sess.Hud.Toast(chapterStartFailed+err.Error(), true)
		}
		return
	}
	play := TutorialPlay{Run: run, Home: g.Yard.Home, prev: g.Yard.prev, fromTitle: g.Yard.fromTitle, Record: g.Yard.Record, Loaded: true}
	if !g.InChapter() {
		play.prev, play.fromTitle = g.Sess, g.Scene == SceneTitle
		if !play.fromTitle {
			play.Home = g.Sess
			g.Save()
		}
	}
	ApplyNames(true)
	s := g.settledSession(run.Realm())
	s.AutoPauseOn, s.TrueNames = false, true
	s.Hud.SetTutorialRecord(play.Record)
	focusChapter(s)
	g.Yard = play
	g.Sess, g.Scene = s, SceneRealm
	g.writeChapter("start", id)
	g.syncTeacher()
	g.Fade, g.FadeDir = 1, 0
}

func (g *Game) leaveChapter(toMap bool) {
	if !g.InChapter() {
		return
	}
	g.writeChapter("leave", g.Yard.Run.Chapter().ID)
	taught := g.Sess.Hud.Taught()
	play := g.Yard
	g.Yard = TutorialPlay{Record: play.Record, Loaded: true}
	ApplyNames(g.Settings.TrueNames)
	g.SaveTick = 0
	g.Fade, g.FadeDir = 1, 0
	if play.fromTitle {
		g.Sess, g.Scene = play.prev, SceneTitle
		if toMap {
			g.Chapters.Open(play.Record)
		}
		return
	}
	g.Sess = play.Home
	g.Sess.Hud.SetTaught(append(g.Sess.Hud.Taught(), taught...))
	g.Sess.Hud.SetTutorialRecord(play.Record)
	if toMap {
		g.Sess.Hud.OpenChapters(play.Record)
	}
}

func (g *Game) StepChapter() []sound.SFX {
	if g.Sess != nil && !g.InChapter() {
		g.loadTutorialRecord()
		g.Sess.Hud.SetTutorialRecord(g.Yard.Record)
	}
	if !g.InChapter() {
		return nil
	}
	g.SaveTick = 0
	run := g.Yard.Run
	var fx []sound.SFX
	if run.Over() && !g.Yard.cheered {
		g.Yard.cheered = true
		g.writeChapter("complete", run.Chapter().ID)
		fx = append(fx, sound.SFXVictory)
	}
	g.syncTeacher()
	if run.Over() && !g.Yard.recorded && g.Sess.Hud.DebriefShown() {
		g.Yard.recorded = true
		g.Yard.Record = campaignHook.FinishChapter(g.Yard.Record, run.Chapter().ID)
		if err := g.saveTutorialRecord(); err != nil {
			g.Log.Warn("tutorial progress not saved", "err", err)
			g.Sess.Hud.Toast(chapterRecordLost+err.Error(), true)
		}
	}
	return fx
}

func (g *Game) syncTeacher() {
	if !g.InChapter() || g.Sess == nil {
		return
	}
	run := g.Yard.Run
	c := run.Chapter()
	lesson, n := run.Lesson()
	v := &hud.TeacherView{Chapter: c, Status: run.Status(), Step: run.Step(), Progress: run.Progress(), Soaking: run.Soaking(),
		Chip: run.Chip(), Lesson: lesson, Lessons: n}
	if at := run.LessonStep(); at < len(c.Steps) {
		v.LessonStep, v.LessonTitle = at, hud.KubernetesOnly(c.Steps[at].Title())
	}
	if next, ok := c.Next(); ok && next.Built() {
		v.Next, v.HasNext = next, true
	}
	if hop, ok := run.ShownHop(); ok && run.Status() == view.ChapterRunning && !run.Soaking() {
		v.Hop = hop
	}
	v.Armed, v.LastHop, v.Verdict = run.Armed(), run.LastHop(), run.Verdict()
	g.Sess.holdSays = run.HoldsWorldEvents()
	v.Question, v.Asking = run.Asking()
	v.Met = run.Met()
	if run.HopAt() != g.Yard.glided {
		g.Yard.glided = run.HopAt()
		g.glideToHop()
	}
	g.Sess.Hud.SetTeacher(v)
	g.Sess.Hud.SetTutorialRecord(g.Yard.Record)
}

func (g *Game) ShowTeacherTargets() {
	s := g.Sess
	spots := []world.Vec{}
	if v, ok := s.SuggestedPlot(); ok {
		spots = append(spots, v)
	}
	for _, t := range hud.TeacherWorldTargets(s.Hud.Teacher()) {
		if v, ok := s.TeacherSpot(t); ok {
			spots = append(spots, v)
		}
	}
	if len(spots) == 0 {
		if v, ok := s.chapterHome(); ok {
			spots = append(spots, v)
		}
	}
	if len(spots) == 0 {
		return
	}
	v := spots[g.Yard.shown%len(spots)]
	g.Yard.shown++
	s.GlideTo(v.X, v.Y)
}

func (g *Game) glideToHop() {
	hop, ok := g.Yard.Run.ShownHop()
	if !ok {
		return
	}
	if v, ok := g.Sess.TeacherSpot(hop.Target); ok {
		g.Sess.GlideTo(v.X, v.Y)
	}
}

func (s *Session) SuggestedPlot() (world.Vec, bool) {
	v := s.Hud.Teacher()
	if v == nil {
		return world.Vec{}, false
	}
	step, ok := v.Current()
	if !ok || !slices.Contains(step.Targets, hud.PlotSuggested) || len(step.Builds) == 0 {
		return world.Vec{}, false
	}
	def, found := view.FindBuildDef(step.Builds[0])
	x, y, placed := s.Realm.SuggestPlacement(step.Builds[0])
	if !found || !placed {
		return world.Vec{}, false
	}
	return world.Vec{X: float64(x) + float64(def.W)/2, Y: float64(y) + float64(def.H)/2}, true
}

func (s *Session) TeacherSpot(target string) (world.Vec, bool) {
	if strings.HasPrefix(target, "stone:") {
		return world.StoneSpot(s.Site)
	}
	return ResolveTarget(target, s.Site, s.Renderer.Home)
}

func (s *Session) teacherBeacon() (world.Vec, bool) {
	if s.Hud.Overlay() != hud.OverlayNone {
		return world.Vec{}, false
	}
	for _, t := range hud.TeacherWorldTargets(s.Hud.Teacher()) {
		if v, ok := s.TeacherSpot(t); ok {
			return v, true
		}
	}
	return world.Vec{}, false
}

func (s *Session) chapterHome() (world.Vec, bool) {
	for _, st := range s.Site.Structures {
		if st.Kind == view.StructKeep {
			return world.Anchor(st), true
		}
	}
	return world.StoneSpot(s.Site)
}

func focusChapter(s *Session) {
	if v, ok := s.chapterHome(); ok {
		s.CenterOn(v.X, v.Y)
	}
}

func (g *Game) writeChapter(event, id string) {
	if g.Dev == nil || g.Sess == nil {
		return
	}
	g.Dev.write("chapter", g.Sess.State.RealmTime, map[string]any{"event": event, "chapter": id})
}
