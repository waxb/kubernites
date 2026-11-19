package app

import (
	"errors"
	"time"

	"github.com/waxb/kubernites/pkg/client"
	"github.com/waxb/kubernites/pkg/view"
)

var errNoCampaign = errors.New("this build has no campaign")

var campaignHook client.Campaign = noCampaign{}

func SetCampaign(c client.Campaign) {
	if c == nil {
		c = noCampaign{}
	}
	campaignHook = c
}

type noCampaign struct{}

func (noCampaign) NewRealm() client.Realm  { return nil }
func (noCampaign) DemoRealm() client.Realm { return nil }
func (noCampaign) FoundRealm(uint64, time.Time) client.Realm {
	return nil
}
func (noCampaign) ReadSave([]byte) (client.Realm, error) { return nil, errNoCampaign }
func (noCampaign) LoadRealm([]byte, time.Time) (client.Realm, error) {
	return nil, errNoCampaign
}
func (noCampaign) Reforge(client.Realm) (client.Realm, error)     { return nil, errNoCampaign }
func (noCampaign) StartChapter(string) (client.ChapterRun, error) { return nil, errNoCampaign }
func (noCampaign) NewChapterRecord() view.ChapterRecord {
	return view.ChapterRecord{Done: map[string]bool{}}
}
func (noCampaign) ParseChapterRecord([]byte) (view.ChapterRecord, error) {
	return view.ChapterRecord{Done: map[string]bool{}}, errNoCampaign
}
func (noCampaign) MarshalChapterRecord(view.ChapterRecord) ([]byte, error) { return nil, errNoCampaign }
func (noCampaign) FinishChapter(rec view.ChapterRecord, _ string) view.ChapterRecord {
	return rec
}
func (noCampaign) ChapterRecordPath(string) string            { return "" }
func (noCampaign) StartTrial(string) (client.TrialRun, error) { return nil, errNoCampaign }
func (noCampaign) ParseTrialScores([]byte) (view.TrialScores, error) {
	return view.NewScores(), errNoCampaign
}
func (noCampaign) TrialScoresPath(string) string { return "" }
