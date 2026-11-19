package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var hearthCalamity = view.CalamityView{ID: "crashloop:default/porters-a1", Kind: view.TagCrashLoop, Object: "pod/default/porters-a1",
	Cause: "porter:2.1 falls on start", Why: "Its class is broken: the kubelet waits longer before each restart (CrashLoopBackOff), and the worker never carries.",
	Remedies: []view.RemedyView{{Label: "Undo the rollout", Command: "kubectl rollout undo deployment porters"}}, Incident: "inc-1"}

func hearthIncident(found ...string) view.IncidentReport {
	inc := view.IncidentReport{ID: "inc-1", Kind: view.IncidentCrashLoop, Name: "The guttering hut", K8s: "CrashLoopBackOff",
		Symptom: "a hut of the porters gutters: it relights and falls again and again", Pod: "default/porters-a1", Crew: "porters",
		Place: "pod/default/porters-a1", State: "open", Guided: true, Step: "hut", StepText: "Find the hut: select it in the world",
		FixText: "undo the rollout, or set a class that does not crash", Timeline: view.IncidentTimeline{StruckAt: 10},
		Clues: []view.IncidentClueView{
			{ID: "hut", Text: "Find the hut: select it in the world", K8s: "Select the Pod"},
			{ID: "logs", Text: "Read what it said before it stopped", K8s: "kubectl logs <pod> --previous"},
			{ID: "describe", Text: "Read its events", K8s: "kubectl describe pod <pod>"},
			{ID: "get-pods", Text: "Ask the Speaking Stone for every hut's state", K8s: "kubectl get pods"},
		}}
	for i, c := range inc.Clues {
		for _, id := range found {
			if c.ID == id {
				inc.Clues[i].Found, inc.Clues[i].FoundAt = true, 12
			}
		}
		if inc.Clues[i].Found && c.ID == "logs" {
			inc.Timeline.CauseAt = 12
		}
	}
	return inc
}

func incidentFrame(t *testing.T, inc view.IncidentReport) Frame {
	f := calamityFrame(t, hearthCalamity)
	f.State.Incidents = []view.IncidentReport{inc}
	return f
}

func TestTheIncidentCardShowsTheSymptomAndHidesTheWhyUntilACauseIsFound(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	cause, why := h.CalamityText(incidentFrame(t, hearthIncident("hut")), hearthCalamity)
	if !strings.HasPrefix(strings.Join(cause, " "), "A hut of the porters gutters") || len(why) != 0 {
		t.Fatalf("before the cause: cause %q why %q", cause, why)
	}
	cause, why = h.CalamityText(incidentFrame(t, hearthIncident("hut", "logs")), hearthCalamity)
	if strings.Join(cause, " ") != hearthCalamity.Cause || !strings.HasPrefix(strings.Join(why, " "), "Its class is broken") {
		t.Fatalf("after the cause: cause %q why %q", cause, why)
	}
	cause, _ = h.CalamityText(calamityFrame(t, hearthCalamity), hearthCalamity)
	if strings.Join(cause, " ") != hearthCalamity.Cause {
		t.Fatalf("a calamity with no incident view keeps its cause: %q", cause)
	}
}

func TestTheIncidentCardListsItsCluesLitAsTheyAreFound(t *testing.T) {
	rows := clueRows(hearthIncident("hut"))
	if len(rows) != 4 || rows[0].icon != art.ClueIconName("hut", true) || rows[1].icon != art.ClueIconName("logs", false) || !rows[0].lit || rows[1].lit {
		t.Fatalf("rows = %+v", rows)
	}
	pending := view.IncidentReport{Clues: []view.IncidentClueView{{ID: "keeps", Text: "Look at what the Keeps have left to pledge", Found: true}}}
	if got := clueRows(pending)[0].icon; got != art.ClueIconName("keep-room", true) {
		t.Fatalf("the Keeps clue draws %q", got)
	}
}

func TestOnlyAGuidedIncidentShowsItsStep(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	guided := hearthIncident()
	guided.Step, guided.StepText = "fix", "Fix it: raze the hut"
	if got := h.stepLines(guided); len(got) == 0 || !strings.Contains(strings.Join(got, " "), "Fix it") {
		t.Fatalf("guided step = %q", got)
	}
	known := hearthIncident()
	known.Guided, known.Step, known.StepText = false, "", ""
	if got := h.stepLines(known); len(got) != 0 {
		t.Fatalf("a known incident shows a step: %q", got)
	}
}

func TestTheIncidentCardMakesRoomForItsCluesAndDraws(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	plain := settled(h, calamityFrame(t, hearthCalamity))
	before, ok := h.CalamityRect(plain)
	if !ok {
		t.Fatal("no calamity card")
	}
	h = New(nil, (&fakeRunner{}).backend())
	f := settled(h, incidentFrame(t, hearthIncident("hut")))
	after, ok := h.CalamityRect(f)
	if !ok || after.H < before.H+4*clueRowH-whyLines*ui.LineHeight {
		t.Fatalf("card height %d, plain %d", after.H, before.H)
	}
	for _, b := range h.CalamityButtons(f) {
		if b.Rect.Y+b.Rect.H > after.Y+after.H {
			t.Errorf("%s falls below the card", b.ID)
		}
	}
	h.Draw(ebiten.NewImage(Width, Height), f)
}

func selectedIDs(buttons []Button) map[string]bool {
	out := map[string]bool{}
	for _, b := range buttons {
		if b.Selected {
			out[b.ID] = true
		}
	}
	return out
}

func TestTheGuidedStepLightsTheControlItNames(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := settled(h, incidentFrame(t, hearthIncident()))
	f.Places = map[string][2]float64{hearthCalamity.Object: {10, 10}}
	if !selectedIDs(h.CalamityButtons(f))[CalamityShow] {
		t.Fatal("at the hut step the card's SHOW is not lit")
	}
	logs := hearthIncident("hut")
	logs.Step = "logs"
	if want := IncidentWants(Frame{State: incidentFrame(t, logs).State, Selection: Selection{Kind: SelectHearth, ID: "default/porters-a1"}}); want != "sel:logs" {
		t.Fatalf("the logs step lights %q", want)
	}
	logs.Step = "describe"
	if want := IncidentWants(Frame{State: incidentFrame(t, logs).State, Selection: Selection{Kind: SelectHearth, ID: "default/porters-a1"}}); want != ActionPrefix+":"+KeyDescribe {
		t.Fatalf("the describe step lights %q", want)
	}
	if want := IncidentWants(Frame{State: incidentFrame(t, logs).State, Selection: Selection{Kind: SelectHearth, ID: "default/other", Pod: model.PodView{Crew: "clerks"}}}); want != "" {
		t.Fatalf("another crew's hut lights %q", want)
	}
	if want := IncidentWants(Frame{State: incidentFrame(t, logs).State, Selection: Selection{Kind: SelectHearth, ID: "default/porters-z9", Pod: model.PodView{Crew: "porters"}}}); want == "" {
		t.Fatal("another hut of the same crew does not light: the realm counts any Pod of the crew")
	}
	logs.Guided = false
	if want := IncidentWants(Frame{State: incidentFrame(t, logs).State, Selection: Selection{Kind: SelectHearth, ID: "default/porters-a1"}}); want != "" {
		t.Fatalf("a known incident lights %q", want)
	}
}
