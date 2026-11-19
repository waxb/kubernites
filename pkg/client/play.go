package client

import (
	"time"

	"github.com/waxb/kubernites/pkg/view"
)

type Campaign interface {
	NewRealm() Realm
	FoundRealm(seed uint64, epoch time.Time) Realm
	ReadSave(raw []byte) (Realm, error)
	DemoRealm() Realm
	LoadRealm(data []byte, savedAt time.Time) (Realm, error)
	Reforge(r Realm) (Realm, error)
	StartChapter(id string) (ChapterRun, error)
	NewChapterRecord() view.ChapterRecord
	ParseChapterRecord(data []byte) (view.ChapterRecord, error)
	MarshalChapterRecord(rec view.ChapterRecord) ([]byte, error)
	FinishChapter(rec view.ChapterRecord, id string) view.ChapterRecord
	ChapterRecordPath(savePath string) string
	StartTrial(id string) (TrialRun, error)
	ParseTrialScores(data []byte) (view.TrialScores, error)
	TrialScoresPath(savePath string) string
}

type ChapterRun interface {
	Realm() Realm
	Chapter() view.TutorialChapter
	Status() view.ChapterStatus
	Step() int
	Progress() view.StepProgress
	Soaking() bool
	Over() bool
	Lesson() (view.StepLesson, int)
	LessonStep() int
	Begin()
	Next()
	Met() bool
	Armed() bool
	Note(kind string)
	Chip() string
	HopAt() int
	ShownHop() (view.Hop, bool)
	HoldsWorldEvents() bool
	Asking() (view.HopQuestion, bool)
	Pick(choice int)
	LastHop() bool
	Verdict() string
}

type TrialRun interface {
	Realm() Realm
	Trial() view.Trial
	Status() view.TrialStatus
	Verdict() view.TrialVerdict
	Elapsed() float64
	Begin()
	Over() bool
	Result() view.TrialResult
	Pace() (int, int)
}
