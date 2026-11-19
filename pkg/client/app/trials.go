package app

import (
	"errors"
	"os"
	"strconv"

	"github.com/waxb/kubernites/pkg/client"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	trialPauseRefused = "A trial's clock does not pause: RETRY starts it afresh, LEAVE returns to your realm."
	trialStartFailed  = "The trial could not be set: "
	trialScoresLost   = "Your trial medals could not be saved: "
	trialSaveNote     = "Your realm was saved as the trial began, and nothing in a trial changes it."
)

type TrialPlay struct {
	Run     client.TrialRun `json:"-"`
	home    *Session
	Scores  view.TrialScores `json:"-"`
	shown   bool
	newBest bool
	best    view.TrialScore
	hasBest bool
	Loaded  bool `json:"-"`
}

func (g *Game) InTrial() bool { return g.Trial.Run != nil && g.Trial.home != nil }

func (g *Game) realmSession() *Session {
	if g.InChapter() {
		return g.Yard.Home
	}
	if g.InTrial() {
		return g.Trial.home
	}
	return g.Sess
}

func (g *Game) trialScoresPath() string {
	if g.Capture != nil {
		return ""
	}
	return campaignHook.TrialScoresPath(g.Opts.SavePath)
}

func (g *Game) loadTrialScores() {
	if g.Trial.Loaded {
		return
	}
	g.Trial.Loaded, g.Trial.Scores = true, view.NewScores()
	path := g.trialScoresPath()
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		g.Log.Warn("trial scores ignored", "err", err)
		return
	}
	if s, err := campaignHook.ParseTrialScores(data); err != nil {
		g.Log.Warn("trial scores ignored", "err", err)
	} else {
		g.Trial.Scores = s
	}
}

func (g *Game) saveTrialScores() error {
	path := g.trialScoresPath()
	if path == "" {
		return nil
	}
	data, err := g.Trial.Scores.Marshal()
	if err != nil {
		return err
	}
	return SaveAtomic(path, data)
}

func (g *Game) trialIntent(it hud.Intent) {
	switch it.Command {
	case hud.TrialStart:
		g.StartTrial(it.Target)
	case hud.TrialRetry:
		if g.InTrial() {
			g.StartTrial(g.Trial.Run.Trial().ID)
		}
	case hud.TrialBegin:
		g.BeginTrial()
	case hud.TrialLeave:
		g.leaveTrial()
	case hud.TrialPace:
		g.SetTrialPace(it.Target)
	}
}

func (g *Game) StartTrial(id string) {
	g.loadTrialScores()
	home := g.realmSession()
	if home == nil {
		return
	}
	run, err := campaignHook.StartTrial(id)
	if err != nil {
		g.Log.Error("trial setup failed", "trial", id, "err", err)
		g.Sess.Hud.Toast(trialStartFailed+err.Error(), true)
		return
	}
	if !g.InTrial() {
		g.Save()
	}
	g.writeTrial("start", id, nil)
	s := g.settledSession(run.Realm())
	s.AutoPauseOn = false
	g.Trial = TrialPlay{Run: run, home: home, Scores: g.Trial.Scores, Loaded: true}
	g.Trial.best, g.Trial.hasBest = g.Trial.Scores.Of(id)
	focusTrial(s)
	g.Sess = s
	s.Hud.SetTrialScores(g.Trial.Scores)
	g.syncTrialView()
	s.Hud.OpenTrialBrief()
	g.Fade, g.FadeDir = 1, 0
}

func (g *Game) BeginTrial() {
	if !g.InTrial() {
		return
	}
	g.Trial.Run.Begin()
	g.Sess.Paused = false
	g.writeTrial("begin", g.Trial.Run.Trial().ID, nil)
}

func (g *Game) leaveTrial() {
	if !g.InTrial() {
		return
	}
	g.writeTrial("leave", g.Trial.Run.Trial().ID, nil)
	taught := g.Sess.Hud.Taught()
	g.Sess = g.Trial.home
	g.Sess.Hud.SetTaught(append(g.Sess.Hud.Taught(), taught...))
	g.Trial = TrialPlay{Scores: g.Trial.Scores, Loaded: true}
	g.Sess.Hud.SetTrialScores(g.Trial.Scores)
	g.SaveTick = 0
	g.Fade, g.FadeDir = 1, 0
}

func (g *Game) SetTrialPace(target string) {
	surge, gaps, ok := hud.PaceTarget(target)
	if !g.InTrial() || !ok || g.Trial.Run.Status() != view.TrialRunning {
		return
	}
	cmd := view.PaceCommand(g.Trial.Run.Trial().Crew, strconv.Itoa(surge), strconv.Itoa(gaps))
	res := g.Sess.Realm.Kubectl(cmd)
	var err error
	if !res.OK {
		err = errors.New(res.Output)
	}
	g.Sess.EchoSpoken([]string{cmd}, hud.FromButton, res.Output, err)
	if err != nil {
		g.Sess.Hud.Toast(res.Output, true)
	}
	g.Sess.Poll = pollSeconds
}

func (g *Game) StepTrial() {
	if g.Sess != nil {
		g.loadTrialScores()
		g.Sess.Hud.SetTrialScores(g.Trial.Scores)
	}
	if !g.InTrial() {
		return
	}
	g.SaveTick = 0
	s, run := g.Sess, g.Trial.Run
	if run.Status() == view.TrialRunning && s.Paused {
		s.Paused = false
		s.Hud.Toast(trialPauseRefused, true)
	}
	if run.Over() && !g.Trial.shown {
		g.finishTrial()
	}
	g.syncTrialView()
}

func (g *Game) finishTrial() {
	run := g.Trial.Run
	res := run.Result()
	scores, better := g.Trial.Scores.Record(res)
	g.Trial.Scores, g.Trial.shown, g.Trial.newBest = scores, true, better
	if err := g.saveTrialScores(); err != nil {
		g.Log.Warn("trial scores not saved", "err", err)
		g.Sess.Hud.Toast(trialScoresLost+err.Error(), true)
	}
	g.writeTrial("end", res.Trial, map[string]any{"won": res.Won, "seconds": res.Seconds, "medal": res.Medal.Name()})
	g.syncTrialView()
	g.Sess.Hud.OpenTrialResult()
}

func (g *Game) syncTrialView() {
	run := g.Trial.Run
	surge, gaps := run.Pace()
	g.Sess.Hud.SetTrial(&hud.TrialView{
		Trial: run.Trial(), Status: run.Status(), Elapsed: run.Elapsed(), Verdict: run.Verdict(), Result: run.Result(),
		Best: g.Trial.best, HasBest: g.Trial.hasBest, NewBest: g.Trial.newBest, Surge: surge, Gaps: gaps,
	})
}

func (g *Game) writeTrial(event, id string, fields map[string]any) {
	if g.Dev == nil || g.Sess == nil {
		return
	}
	line := map[string]any{"event": event, "trial": id}
	for k, v := range fields {
		line[k] = v
	}
	g.Dev.write("trial", g.Sess.State.RealmTime, line)
}

func focusTrial(s *Session) {
	sum, n := world.Vec{}, 0.0
	for _, st := range s.Site.Structures {
		if st.Kind == view.StructKeep && st.Object.Name != view.TowerName {
			a := world.Anchor(st)
			sum, n = world.Vec{X: sum.X + a.X, Y: sum.Y + a.Y}, n+1
		}
	}
	if n > 0 {
		s.CenterOn(sum.X/n, sum.Y/n)
	}
}
