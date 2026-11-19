package livescenes

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	corev1 "k8s.io/api/core/v1"

	app "github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/view"
)

func liveGame(t *testing.T, snap live.Snapshot) (*app.Game, *live.Realm) {
	t.Helper()
	r, err := live.OpenRecorded(snap)
	if err != nil {
		t.Fatal(err)
	}
	g := app.NewGame(nil, app.Options{LiveOnly: true}, nil, nil)
	g.Settings.LiveNames = storyNames()
	g.EnterLive(r)
	DismissIntros(g.Sess)
	return g, r
}

func storyNames() *bool {
	story := false
	return &story
}

func play(s *app.Session, seconds float64) {
	for i := 0; float64(i)*app.TickSeconds < seconds; i++ {
		s.Advance(app.TickSeconds)
		s.Hud.Update(idleInput(), s.Frame(true))
	}
}

func shown(s *app.Session) (view.CalamityView, bool) {
	return s.Hud.ShownCalamity(s.Frame(true))
}

func TestTroubleAtConnectOpensNoCardAndIsListedQuietlyOnTheLiveCard(t *testing.T) {
	g, _ := liveGame(t, KindSnapshot())
	s := g.Sess
	play(s, 12)
	if v, ok := shown(s); ok {
		t.Fatalf("a card opened for trouble already there at connect: %+v", v)
	}
	quiet := 0
	for _, v := range s.State.Calamities {
		if !v.Quiet {
			t.Errorf("trouble at connect is not quiet: %+v", v)
		}
		quiet++
	}
	if quiet < 3 {
		t.Fatalf("the kind realm's crash loop, OOM kill, bad image and Unhoused hut raised %d calamities: %+v", quiet, s.State.Calamities)
	}
	if s.Paused {
		t.Error("the live realm paused")
	}
	PressButton(s, hud.LiveQuietID)
	play(s, .5)
	if v, ok := shown(s); !ok || !v.Quiet {
		t.Fatalf("the quiet line did not open the first quiet card: %+v %v", v, ok)
	}
}

func brokenPod(snap live.Snapshot, ns string) corev1.Pod {
	for _, p := range snap.Pods {
		if p.Namespace == ns && len(p.OwnerReferences) > 0 && len(p.Status.ContainerStatuses) > 0 {
			return *p.DeepCopy()
		}
	}
	panic("no pod in " + ns)
}

func crashOf(p corev1.Pod) corev1.Pod {
	p = *p.DeepCopy()
	cs := &p.Status.ContainerStatuses[0]
	cs.Ready, cs.RestartCount = false, 3
	cs.State = corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff", Message: "back-off 40s restarting failed container=" + cs.Name + " pod=" + p.Name + "_" + p.Namespace}}
	cs.LastTerminationState = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 1}}
	for i := range p.Status.Conditions {
		if p.Status.Conditions[i].Type == corev1.PodReady {
			p.Status.Conditions[i].Status = corev1.ConditionFalse
		}
	}
	return p
}

func pullOf(p corev1.Pod) corev1.Pod {
	p = *p.DeepCopy()
	cs := &p.Status.ContainerStatuses[0]
	cs.Ready = false
	cs.State = corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff", Message: "Back-off pulling image \"" + cs.Image + "-never\""}}
	return p
}

func replay(t *testing.T, r *live.Realm, list, typ string, obj any) {
	t.Helper()
	if err := r.Replay(list, bytes.NewReader(live.WatchEvent(typ, obj))); err != nil {
		t.Fatal(err)
	}
}

func TestNewTroubleOpensOneCardAtATimeNeverOverAWindowAndNeverPauses(t *testing.T) {
	snap := ManagedSnapshot()
	g, r := liveGame(t, snap)
	s := g.Sess
	play(s, 2)
	if len(s.State.Calamities) != 0 {
		t.Fatalf("the managed realm is calm, yet %+v", s.State.Calamities)
	}
	healthy := brokenPod(snap, "market")
	replay(t, r, "pods", "MODIFIED", crashOf(healthy))
	play(s, 1)
	if _, ok := shown(s); ok {
		t.Fatal("the card opened before the trouble settled")
	}
	play(s, hud.CalamitySettle+1)
	v, ok := shown(s)
	if !ok || v.Kind != view.TagCrashLoop || v.Quiet || !strings.Contains(v.Cause, "CrashLoopBackOff: back-off 40s restarting failed container=") {
		t.Fatalf("the crash loop's card: %+v %v", v, ok)
	}
	if s.Paused || s.Hud.Overlay() != hud.OverlayNone {
		t.Fatal("the trouble paused the live realm")
	}
	other := brokenPod(snap, "kube-system")
	replay(t, r, "pods", "MODIFIED", pullOf(other))
	play(s, hud.CalamitySettle+1)
	if v2, ok := shown(s); !ok || v2.ID != v.ID {
		t.Fatalf("a second trouble took the first card's place: %+v", v2)
	}
	_, _, count, _ := s.Hud.CurrentCalamity(s.Frame(true))
	if count != 2 {
		t.Errorf("%d troubles behind the card, want 2", count)
	}
	PressButton(s, "top:menu")
	if _, ok := shown(s); ok {
		t.Error("a calamity card stands over the menu")
	}
	PressButton(s, "menu:resume")
	reads, orders := 0, 0
	for _, b := range s.Hud.Buttons(s.Frame(true)) {
		switch {
		case strings.HasSuffix(b.ID, ":"+hud.SpeakOp) && strings.HasPrefix(b.ID, "line:cal:"):
			if b.Disabled || strings.Contains(b.Label, "kubectl") || strings.Contains(b.Label, "logs ") {
				t.Errorf("the read remedy %q is closed or shows its command with HINTS off", b.Label)
			}
			reads++
		case hud.IsLiveOrder(b.ID):
			if !b.Disabled || strings.Contains(b.Label, "kubectl") || strings.Contains(b.Label, "rollout") {
				t.Errorf("the order remedy %q can be pressed at READ ONLY or shows its command with HINTS off", b.Label)
			}
			orders++
		case strings.HasSuffix(b.ID, ":"+hud.UnwiredOp):
			t.Errorf("an order remedy is still an unwired chip: %+v", b)
		case strings.HasPrefix(b.ID, "calamity:fix"):
			t.Errorf("a live card offers FIX: %+v", b)
		}
	}
	if reads != 1 || orders != 1 {
		t.Errorf("%d read remedies and %d orders on the card", reads, orders)
	}
	replay(t, r, "pods", "MODIFIED", healthy)
	replay(t, r, "pods", "MODIFIED", other)
	play(s, live.MendSettle+2)
	mended := 0
	for _, c := range s.State.Calamities {
		if c.Mended != "" {
			mended++
		}
	}
	if mended != 2 || len(s.State.Chronicles) != 2 {
		t.Fatalf("after both mended: %d mended, chronicles %+v", mended, s.State.Chronicles)
	}
	for _, n := range s.Rail.Notices {
		if n.ID == "decision:cmd:chronicles" {
			t.Fatalf("a live history entry asks for an answer: %+v", n)
		}
	}
	c := s.State.Chronicles[0]
	if hud.LiveIntent(hud.Intent{Kind: hud.IntentAction, Action: view.Action{Action: view.ActScaleCrew}}) {
		t.Fatal("the live gate lets a non-live order through")
	}
	if !c.Plain || !c.Resolved || c.Question != "" || len(c.Options) != 0 || c.Struck == "" || c.Healed == "" {
		t.Fatalf("the history entry %+v", c)
	}
}

type drawnText struct {
	mu    sync.Mutex
	texts []string
}

func (d *drawnText) see(text string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.texts = append(d.texts, text)
}

func chipText(text string, commands []string) bool {
	for _, cmd := range commands {
		story := strings.TrimPrefix(lexicon.StoryText(hud.ChipLabel(lexicon.CommandHead(cmd))), lexicon.StoryCommand+" ")
		head, tail, cut := strings.Cut(text, "...")
		if text == story || cut && strings.HasPrefix(story, head) && strings.HasSuffix(story, tail) {
			return true
		}
	}
	return false
}

func storyLeaks(text string, commands []string) []string {
	if strings.HasPrefix(text, "kubectl ") || strings.HasPrefix(text, ">") || chipText(text, commands) {
		return nil
	}
	return lexicon.KubernetesWords.FindAllString(text, -1)
}

func drawnInStory(t *testing.T, s *app.Session, skip []string) []string {
	t.Helper()
	seen := &drawnText{}
	ui.WatchText(seen.see)
	s.Draw(newScreen(), true)
	ui.WatchText(s.Hud.SeeText)
	var commands []string
	for _, c := range s.State.Calamities {
		for _, r := range c.Remedies {
			commands = append(commands, r.Command)
		}
	}
	var leaks []string
	for _, text := range seen.texts {
		if slices.ContainsFunc(skip, func(k string) bool { return k != "" && strings.Contains(text, k) }) {
			continue
		}
		if found := storyLeaks(text, commands); len(found) > 0 {
			leaks = append(leaks, fmt.Sprintf("%v in %q", found, text))
		}
	}
	return leaks
}

func TestStoryModeCalamityAndReportCardsSpeakNoKubernetes(t *testing.T) {
	defer app.ApplyNames(false)
	snap := ManagedSnapshot()
	skip := []string{snap.Context.Name, snap.Context.Server}
	for _, tr := range Troubles {
		t.Run(tr.Name, func(t *testing.T) {
			app.ApplyNames(false)
			g, r := liveGame(t, snap)
			s := g.Sess
			if err := Play(r, tr.Break(snap)); err != nil {
				t.Fatal(err)
			}
			play(s, hud.CalamitySettle+1)
			v, ok := shown(s)
			if !ok || v.Kind != tr.Kind {
				t.Fatalf("card %+v, want %s", v, tr.Kind)
			}
			for _, leak := range drawnInStory(t, s, skip) {
				t.Errorf("the %s card: %s", tr.Kind, leak)
			}
			if err := Play(r, tr.Mend(snap)); err != nil {
				t.Fatal(err)
			}
			play(s, live.MendSettle+1)
			if v, ok := shown(s); !ok || v.Mended == "" {
				t.Fatalf("no mended card: %+v", v)
			}
			for _, b := range s.Hud.Buttons(s.Frame(true)) {
				if strings.HasPrefix(b.ID, "bridge:") {
					t.Errorf("a live card offers the Almanac's bridge: %+v", b)
				}
			}
			for _, leak := range drawnInStory(t, s, skip) {
				t.Errorf("the mended %s card: %s", tr.Kind, leak)
			}
			PressButton(s, "cmd:chronicles")
			play(s, .2)
			if s.Hud.Overlay() != hud.OverlayChronicles {
				t.Fatalf("the Chronicle did not open: overlay %q", s.Hud.Overlay())
			}
			for _, leak := range drawnInStory(t, s, skip) {
				t.Errorf("the %s Chronicle: %s", tr.Kind, leak)
			}
		})
	}
	t.Run("live card", func(t *testing.T) {
		app.ApplyNames(false)
		kind := KindSnapshot()
		r, err := live.OpenRecorded(kind)
		if err != nil {
			t.Fatal(err)
		}
		if err := r.ReplayStale(kind.TakenAt, "pods: Get \"https://127.0.0.1:41231/api/v1/pods?watch=true\": unexpected EOF"); err != nil {
			t.Fatal(err)
		}
		if err := r.ReplayListed([]live.Listed{{List: "services"}, {List: kind.CRDs[0].List(), PastCap: true}}); err != nil {
			t.Fatal(err)
		}
		g := app.NewGame(nil, app.Options{LiveOnly: true}, nil, nil)
		g.Settings.LiveNames = storyNames()
		g.EnterLive(r)
		DismissIntros(g.Sess)
		s := g.Sess
		play(s, 1)
		card := s.Hud.LiveCard()
		if len(card.Quiet) == 0 || card.Stale == "" {
			t.Fatalf("the card shows no quiet trouble or staleness: %+v", card)
		}
		for _, leak := range drawnInStory(t, s, []string{kind.Context.Name, kind.Context.Server, kind.Identity.Username}) {
			t.Errorf("the live card: %s", leak)
		}
		in := hoverButton(s, hud.LiveQuietID)
		s.Hud.Update(in, s.Frame(true))
		for _, leak := range drawnInStory(t, s, []string{kind.Context.Name, kind.Context.Server, kind.Identity.Username}) {
			t.Errorf("the quiet trouble's tip: %s", leak)
		}
		s.Hud.Update(hoverButton(s, "info:live:stale"), s.Frame(true))
		for _, leak := range drawnInStory(t, s, []string{kind.Context.Name, kind.Context.Server, kind.Identity.Username}) {
			t.Errorf("the stale line's tip: %s", leak)
		}
		if len(card.Listed) != 2 {
			t.Fatalf("the card names no kind read by listing: %+v", card.Listed)
		}
		s.Hud.Update(hoverButton(s, "info:live:listed"), s.Frame(true))
		for _, leak := range drawnInStory(t, s, []string{kind.Context.Name, kind.Context.Server, kind.Identity.Username}) {
			t.Errorf("the listing line's tip: %s", leak)
		}
	})
}

func newScreen() *ebiten.Image {
	return ebiten.NewImage(hud.Width, hud.Height)
}

func TestStoryModeSaysACordonedKeepIsClosedToNewHuts(t *testing.T) {
	defer app.ApplyNames(false)
	app.ApplyNames(false)
	snap := ManagedSnapshot()
	g, r := liveGame(t, snap)
	s := g.Sess
	if err := Play(r, TroubleNamed("report-keep").Break(snap)); err != nil {
		t.Fatal(err)
	}
	play(s, hud.CalamitySettle+1)
	if _, ok := shown(s); !ok {
		t.Fatal("no card")
	}
	seen := &drawnText{}
	ui.WatchText(seen.see)
	s.Draw(newScreen(), true)
	ui.WatchText(s.Hud.SeeText)
	all := strings.Join(seen.texts, " ")
	if !strings.Contains(all, "is now closed to new huts") || strings.Contains(all, "status is now") {
		t.Fatalf("the cordoned Keep's card does not say it is closed to new huts:\n%s", all)
	}
}

func TestNamesModeShowsTheClustersWordsAsTheyAre(t *testing.T) {
	defer app.ApplyNames(false)
	snap := ManagedSnapshot()
	g, r := liveGame(t, snap)
	app.ApplyNames(true)
	g.Sess.TrueNames = true
	s := g.Sess
	p := CrashLooping(snap, false)
	cs := &p.Status.ContainerStatuses[0]
	cs.Name, p.Spec.Containers[0].Name = "worker", "worker"
	cs.State = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 1}}
	if err := Play(r, []Replayed{{"pods", "MODIFIED", p}}); err != nil {
		t.Fatal(err)
	}
	play(s, hud.CalamitySettle+1)
	v, ok := shown(s)
	if !ok {
		t.Fatal("no card")
	}
	seen := &drawnText{}
	ui.WatchText(seen.see)
	s.Draw(newScreen(), true)
	ui.WatchText(nil)
	all := strings.Join(seen.texts, "\n")
	if !strings.Contains(all, "Error: container \"worker\" exited with code 1") || !strings.Contains(all, "Read what worker wrote") {
		t.Fatalf("NAMES mode changed the cluster's words of %q:\n%s", v.Cause, all)
	}
}
