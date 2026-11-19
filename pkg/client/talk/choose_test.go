package talk

import (
	"slices"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
)

var everything = map[string]bool{
	"quay": true, "keep": true, "door": true, "crew": true, "roll": true, "badge": true, "stewards": true, "quartermaster": true,
	"stone": true, "borg": true, "grimoire": true, "pledge": true, "binding": true, "rations": true, "voidtaken": true, "bell": true,
	"shutters": true, "chronicle": true, "keeps": true, "drain": true, "imp": true, "tide": true, "ages": true, "citadel": true, "depot": true,
}

const epoch = 1_788_000_000.0

func burning() Facts {
	return Facts{
		Trade: "Porter", Carrier: true, Now: epoch + 600, Known: everything,
		Worker: model.Kubernite{Name: "porter", Ready: true, State: model.ContainerState{Phase: model.ContainerRunning, Started: epoch + 100},
			Resources: model.Resources{CPURequest: 200, MemRequest: 64 << 20, CPULimit: 500, MemLimit: 128 << 20}},
		Hut: Hut{Name: "porters-abc", Owner: "ReplicaSet/porters-7d9f", Crew: "porters", OnRoll: true, Depot: "parcels", App: "porters",
			Node: "lone-tower", RestartPolicy: "Always", Workers: 1, JoinedAgo: never, LeftAgo: never},
		Use:    Use{Measured: true, Alone: true, CPUMilli: 120, MemBytes: 40 << 20, Served: 0.4},
		Around: Around{ImpAgo: never, AgeAgo: never},
	}
}

func porterOf(tone Tone) Identity {
	for i := 0; ; i++ {
		id := Worker("default", "porters-abc", "porter-"+string(rune('a'+i%26))+string(rune('a'+i/26)))
		if id.Tone() == tone {
			return id
		}
	}
}

func situationOf(text string) Situation {
	for _, l := range Pool() {
		if l.Text == text || l.Then != "" && l.Text+" "+l.Then == text {
			return l.When
		}
	}
	return ""
}

func TestUrgentStateSpeaksFirst(t *testing.T) {
	f := burning()
	f.Use.Throttled = true
	f.Around.Night, f.Around.Tide = true, true
	m := NewMemory()
	id := porterOf(Grumpy)
	for i := range 6 {
		said := m.Click(id, f, float64(i)*10)
		if said.Situation != SitHungry {
			t.Fatalf("click %d: a hungry worker talks about its hunger first, said %q (%s)", i, said.Text, said.Situation)
		}
	}
}

func TestSalienceOrderFallsThroughTiers(t *testing.T) {
	f := burning()
	f.Worker.Restarts, f.Worker.Last = 1, model.ContainerState{Phase: model.ContainerTerminated, Reason: "Error", ExitCode: 1, Finished: f.Now - 30}
	f.Around.Night = true
	id := porterOf(Cheerful)
	if said := NewMemory().Click(id, f, 0); said.Situation != SitRelit {
		t.Fatalf("just relit beats the night: %q (%s)", said.Text, said.Situation)
	}
	f.Worker.Last.Finished = f.Now - 3600
	if said := NewMemory().Click(id, f, 0); tierOf(said.Situation) != TierAround {
		t.Fatalf("with nothing recent, what is around comes next: %q (%s)", said.Text, said.Situation)
	}
	f.Around.Night = false
	if said := NewMemory().Click(id, f, 0); tierOf(said.Situation) != TierWho {
		t.Fatalf("then who it is: %q (%s)", said.Text, said.Situation)
	}
}

func TestOnlyWordsThePlayerHasMet(t *testing.T) {
	f := burning()
	f.Known = map[string]bool{}
	m := NewMemory()
	for i := range 40 {
		said := m.Click(porterOf(Tone(i%int(tones))), f, float64(i)*200)
		for _, l := range Pool() {
			if l.Say(f) == said.Text && len(l.Needs) > 0 {
				t.Fatalf("a line needing %v was said before the player met it: %q", l.Needs, said.Text)
			}
		}
	}
}

func TestNoLineTwiceInARowAndHeardLinesRest(t *testing.T) {
	f := burning()
	m := NewMemory()
	id := porterOf(Dreamy)
	prev := ""
	heard := map[string]float64{}
	for i := range 30 {
		now := float64(i) * 10
		said := m.Click(id, f, now)
		if said.Text == prev {
			t.Fatalf("click %d repeats %q", i, said.Text)
		}
		if at, ok := heard[said.Text]; ok && now-at < RestSeconds && !said.FollowUp {
			fresh := 0
			for _, l := range Pool() {
				if slices.Contains(Situations(f), l.When) && l.speaks(f) && tierOf(l.When) >= tierOf(said.Situation) {
					if at2, ok2 := heard[l.Say(f)]; !ok2 || now-at2 >= RestSeconds {
						fresh++
					}
				}
			}
			if fresh > 0 {
				t.Fatalf("click %d: %q was heard %gs ago while %d fresh lines waited", i, said.Text, now-at, fresh)
			}
		}
		heard[said.Text], prev = now, said.Text
	}
}

func TestAnnoyedLadderAndCalm(t *testing.T) {
	f := burning()
	m := NewMemory()
	id := porterOf(Proud)
	var said Said
	for i := range AnnoyAfter - 1 {
		if said = m.Click(id, f, float64(i)); said.Annoyed != 0 {
			t.Fatalf("click %d is not yet annoying", i+1)
		}
	}
	steps := []string{}
	for i := range 3 {
		said = m.Click(id, f, float64(AnnoyAfter-1+i))
		if said.Annoyed != i+1 || said.Situation != SitAnnoyed {
			t.Fatalf("rapid click %d: annoyed %d %q", AnnoyAfter+i, said.Annoyed, said.Text)
		}
		steps = append(steps, said.Text)
	}
	if steps[0] == steps[1] || steps[1] == steps[2] {
		t.Errorf("the ladder climbs: %q", steps)
	}
	if said = m.Click(id, f, 100); said.Annoyed != 0 {
		t.Errorf("after a calm the worker is not annoyed: %q", said.Text)
	}
}

func TestSecondClickGetsTheFollowUp(t *testing.T) {
	f := burning()
	f.Hut.OnRoll, f.Hut.Owner, f.Trade = false, "", "Porter"
	m := NewMemory()
	id := porterOf(Joker)
	for i := range 20 {
		first := m.Click(id, f, float64(i)*100)
		var line Line
		for _, l := range Pool() {
			if l.Say(f) == first.Text {
				line = l
			}
		}
		if line.Then == "" {
			continue
		}
		next := m.Click(id, f, float64(i)*100+2)
		if next.Text != line.Rest(f) || !next.FollowUp {
			t.Fatalf("second click after %q said %q", first.Text, next.Text)
		}
		return
	}
	t.Fatal("no two-part line was said")
}

func TestToneKeepsAWorkerInCharacter(t *testing.T) {
	f := burning()
	f.Use.Throttled = true
	for tone := range tones {
		said := NewMemory().Click(porterOf(tone), f, 0)
		if said.Tone != tone && said.Tone != AnyTone {
			t.Errorf("a %s worker said a %s line: %q", tone, said.Tone, said.Text)
		}
	}
}

func TestStoryAndNamesPoolsStayApart(t *testing.T) {
	f := burning()
	for _, names := range []bool{false, true} {
		f.Names = names
		m := NewMemory()
		for i := range 60 {
			said := m.Click(porterOf(Tone(i%int(tones))), f, float64(i)*200)
			for _, l := range Pool() {
				if l.Say(f) != said.Text {
					continue
				}
				if names && l.Pool == PoolStory || !names && l.Pool == PoolNames {
					t.Fatalf("names=%v said a line of the other pool: %q", names, said.Text)
				}
			}
		}
	}
}

func TestLinesNeverClaimWhatIsNotTrue(t *testing.T) {
	f := burning()
	f.Names = true
	f.Hut.Static, f.Hut.Owner = false, "ReplicaSet/porters-7d9f"
	for _, l := range Pool() {
		if !l.speaks(f) {
			continue
		}
		text := l.Say(f)
		switch {
		case strings.Contains(text, "static Pod"):
			t.Errorf("a ReplicaSet's worker would say %q", text)
		case strings.Contains(text, "no owner"):
			t.Errorf("an owned Pod's worker would say %q", text)
		case strings.Contains(text, "OOMKilled") && l.When != SitCramped:
			t.Errorf("a worker never OOMKilled would say %q", text)
		}
	}
	f.Hut.Static, f.Hut.Owner = true, "Node/lone-tower"
	for _, l := range Pool() {
		if l.speaks(f) && (strings.Contains(l.Text, "ReplicaSet keeps") || strings.Contains(l.Text, "scheduler picked")) {
			t.Errorf("a static Pod's worker would say %q", l.Text)
		}
	}
}

func TestEveryLineFitsABubble(t *testing.T) {
	for _, l := range Pool() {
		for _, text := range []string{l.Say(burning()), l.Rest(burning())} {
			if len([]rune(text)) > 60 {
				t.Errorf("%q is %d characters", text, len([]rune(text)))
			}
		}
	}
}

func TestVoiceIsTheWorkersOwnAndLeansWithTone(t *testing.T) {
	a, b := porterOf(Grumpy), porterOf(Cheerful)
	if a.Voice() != a.Voice() {
		t.Error("same worker, same voice")
	}
	va, vb := a.Voice(), b.Voice()
	if va.Pitch < PitchLow || va.Pitch > PitchHigh || vb.Pitch < PitchLow || vb.Pitch > PitchHigh {
		t.Errorf("pitch out of 0.55-1.55: %v %v", va.Pitch, vb.Pitch)
	}
	if va.Pitch >= vb.Pitch {
		t.Errorf("grumpy %v speaks lower than cheerful %v", va.Pitch, vb.Pitch)
	}
	for _, v := range []Voice{va, vb} {
		if v.CPS < CPSLow || v.CPS > CPSHigh {
			t.Errorf("typing speed %v out of 26-34", v.CPS)
		}
	}
}

func TestFamiliarsSpeakOnlyTheirOwnLines(t *testing.T) {
	f := burning()
	f.Familiar = "raven"
	id := Pet("default", "porters-abc", "raven", "raven")
	m := NewMemory()
	for i := range 8 {
		said := m.Click(id, f, float64(i))
		if said.Situation != SitPet && said.Text != "..." {
			t.Fatalf("a raven said %q", said.Text)
		}
	}
	if id.Voice().Timbre != TimbreChirp {
		t.Error("a raven chirps")
	}
}

func TestOnlyCarriersTalkAboutBoxes(t *testing.T) {
	f := burning()
	f.Carrier, f.Trade, f.Hut.System, f.Hut.Owner = false, "Quartermaster", true, "Node/lone-tower"
	m := NewMemory()
	sawRole := false
	for i := range 40 {
		said := m.Click(porterOf(Tone(i%int(tones))), f, float64(i)*200)
		if carrierWords.MatchString(said.Text) {
			t.Fatalf("the Quartermaster carries no boxes, yet said %q", said.Text)
		}
		sawRole = sawRole || said.Situation == SitSystem
	}
	if !sawRole {
		t.Error("a Citadel worker speaks of its own work")
	}
}

func TestAFullQuayIsEchoedInEachMode(t *testing.T) {
	f := burning()
	f.Problem = "quayfull"
	id := porterOf(Cheerful)
	if said := NewMemory().Click(id, f, 0); said.Text != "The quay is full. Ships turn boxes away." {
		t.Errorf("story: a porter at a full quay says so, said %q (%s)", said.Text, said.Situation)
	}
	f.Names = true
	if said := NewMemory().Click(id, f, 0); said.Text != "Queue full. New requests are turned away." {
		t.Errorf("NAMES: said %q (%s)", said.Text, said.Situation)
	}
}
