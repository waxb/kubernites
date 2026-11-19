package view

import (
	"fmt"
	"slices"
)

type TutorialChapter struct {
	ID       string
	Number   int
	Part     int
	Title    string
	Short    string
	Icon     string
	Concepts []string
	YouWill  []string
	Course   string
	Pages    []string
	Trial    string
	Briefing []string
	Debrief  Debrief
	Reveals  []string
	Steps    []TutorialStep
}

func (c TutorialChapter) Built() bool { return len(c.Steps) > 0 }

func (c TutorialChapter) MapTitle() string {
	if c.Short != "" {
		return c.Short
	}
	return c.Title
}

func (c TutorialChapter) Label() string { return fmt.Sprintf("Chapter %d", c.Number) }

func (c TutorialChapter) Next() (TutorialChapter, bool) { return ChapterNumber(c.Number + 1) }

func (c TutorialChapter) OwnPage() string {
	prev, ok := ChapterNumber(c.Number - 1)
	for _, p := range c.Pages {
		if !ok || !slices.Contains(prev.Pages, p) {
			return p
		}
	}
	if len(c.Pages) > 0 {
		return c.Pages[0]
	}
	return ""
}

type TutorialStep struct {
	ID      string
	Beat    string
	Say     []string
	Warn    string
	Armed   string
	Why     string
	Goal    string
	Done    string
	Try     string
	Targets []string
	Chips   []string
	Page    string
	Level   StepLevel
	Talk    bool
	Watch   bool
	Stone   bool
	Points  string
	Lesson  StepLesson
	Builds  []string
	Reveals []string
	Fade    []int
}

const stepGoalFade = 2

func (s TutorialStep) Lines() []string {
	if s.Level == StepAsk || s.goalOnly() {
		return []string{s.Goal}
	}
	return s.Say
}

func (s TutorialStep) goalOnly() bool {
	if len(s.Chips) == 0 || len(s.Fade) != len(s.Chips) {
		return false
	}
	for _, f := range s.Fade {
		if f < stepGoalFade {
			return false
		}
	}
	return true
}

func (s TutorialStep) Spotlit() bool { return s.Level == StepShow }

func (s TutorialStep) Title() string {
	if s.Done != "" {
		return s.Done
	}
	return s.Goal
}

func (s TutorialStep) Speaks() bool { return s.Stone || len(s.Chips) > 0 }

type ChapterRecord struct {
	Version int
	Done    map[string]bool
}

func (rec ChapterRecord) DoneCount() int { return len(rec.Done) }

func (rec ChapterRecord) DoneIDs() []string {
	out := []string{}
	for id, done := range rec.Done {
		if done {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

func (rec ChapterRecord) Current() TutorialChapter {
	all := Content.Chapters()
	if len(all) == 0 {
		return TutorialChapter{}
	}
	last := all[0]
	for _, c := range all {
		if !c.Built() {
			continue
		}
		if !rec.Done[c.ID] {
			return c
		}
		last = c
	}
	return last
}

func (rec ChapterRecord) Locked(c TutorialChapter) bool {
	if rec.Done[c.ID] || !c.Built() {
		return false
	}
	return c.Number > rec.Current().Number
}

func ChapterNumber(n int) (TutorialChapter, bool) {
	all := Content.Chapters()
	if n < 1 || n > len(all) {
		return TutorialChapter{}, false
	}
	return all[n-1], true
}

func FindChapter(id string) (TutorialChapter, bool) {
	for _, c := range Content.Chapters() {
		if c.ID == id {
			return c, true
		}
	}
	return TutorialChapter{}, false
}

type Trial struct {
	ID     string
	Title  TrialText
	Idea   TrialText
	Brief  []TrialText
	Goal   TrialText
	Hint   TrialText
	Lesson TrialText
	Times  TrialTimes
	Seed   uint64
	Crew   string
	Pace   bool
	Pages  []string
	K8s    TrialKube
}

func (t Trial) Kube() TrialKube { return t.K8s }

func (t Trial) Next() (Trial, bool) {
	all := Content.Trials()
	for i, c := range all {
		if c.ID == t.ID && i+1 < len(all) {
			return all[i+1], true
		}
	}
	return Trial{}, false
}

func FindTrial(id string) (Trial, bool) {
	for _, t := range Content.Trials() {
		if t.ID == id {
			return t, true
		}
	}
	return Trial{}, false
}

func TrialForPage(page string) (Trial, bool) {
	for _, t := range Content.Trials() {
		if slices.Contains(t.Pages, page) {
			return t, true
		}
	}
	return Trial{}, false
}
