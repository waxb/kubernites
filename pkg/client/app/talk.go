package app

import (
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/talk"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	frozenWindow   = 60.0
	nightDarkness  = .25
	droughtShare   = .97
	livedPerWorker = 12
	tremorReason   = "Tremor"
	lonelyServed   = 0.01
)

type workerLine struct {
	Voice sound.Voice `json:"-"`
	Text  string      `json:"-"`
	Stop  bool        `json:"-"`
}

type Watched struct {
	hungry, cramped, served bool
}

type chatter struct {
	memory *talk.Memory
	Speech *world.Speech `json:"-"`
	say    *workerLine
	Lived  map[string][]talk.Happening `json:"-"`
	Seen   map[string]Watched          `json:"-"`
	age    string
	ageAt  float64
	whole  bool
}

func (c *chatter) Init() {
	if c.memory == nil {
		c.memory = talk.NewMemory()
		c.Lived = map[string][]talk.Happening{}
		c.Seen = map[string]Watched{}
	}
}

func IdentityOf(p model.PodView) talk.Identity {
	return talk.Worker(p.Namespace, p.Name, p.Focus)
}

func (s *Session) realmNow() float64 {
	return float64(s.State.FoundedAt.UnixNano())/1e9 + s.State.RealmTime
}

func (s *Session) Speak(p model.PodView) {
	s.Talk.Init()
	if p.Focus == "" {
		p = p.Person(0)
	}
	id := IdentityOf(p)
	said := s.Talk.memory.Click(id, s.WorkerFacts(p), s.Clock)
	v := id.Voice()
	s.Talk.Speech = &world.Speech{Person: world.PersonKey(p), Text: said.Text, Start: s.Clock, CPS: v.CPS}
	s.Talk.say = &workerLine{Voice: sound.Voice{Pitch: v.Pitch, CPS: v.CPS, Seed: v.Seed, Timbre: v.Timbre}, Text: said.Text}
	if s.Record != nil {
		s.Record("speech", map[string]any{"worker": id.Key, "text": said.Text, "situation": string(said.Situation)})
	}
}

func (s *Session) speakPet(p model.PodView, name string) {
	s.Talk.Init()
	kind := ""
	for i, n := range p.PetNames {
		if n == name && i < len(p.Familiars) {
			kind = p.Familiars[i]
		}
	}
	id := talk.Pet(p.Namespace, p.Name, name, kind)
	f := s.WorkerFacts(p.Person(0))
	f.Familiar, f.Carrier = kind, false
	said := s.Talk.memory.Click(id, f, s.Clock)
	v := id.Voice()
	s.Talk.Speech = &world.Speech{Person: world.PetKey(p.Person(0), name), Text: said.Text, Start: s.Clock, CPS: v.CPS}
	s.Talk.say = &workerLine{Voice: sound.Voice{Pitch: v.Pitch, CPS: v.CPS, Seed: v.Seed, Timbre: v.Timbre}, Text: said.Text}
}

func PetOwner(key string) string {
	owner, _, _ := strings.Cut(key, "#")
	parts := strings.SplitN(owner, "/", 3)
	if len(parts) < 2 {
		return owner
	}
	return parts[0] + "/" + parts[1]
}

func (s *Session) hush() {
	if s.Talk.Speech == nil {
		return
	}
	s.Talk.Speech = nil
	s.Talk.say = &workerLine{Stop: true}
}

func (s *Session) TakeVoice() *workerLine {
	say := s.Talk.say
	s.Talk.say = nil
	return say
}

func (s *Session) speakingTo() string {
	if s.Talk.Speech == nil {
		return ""
	}
	return s.Talk.Speech.Person
}

func (s *Session) KeepSpeech() {
	if s.Talk.Speech == nil {
		return
	}
	if s.focusKey() != s.Talk.Speech.Person {
		s.hush()
		return
	}
	if s.Clock-s.Talk.Speech.Start > s.Talk.Speech.Done()+world.BubbleHold+world.BubbleFade {
		s.Talk.Speech = nil
	}
}

func (s *Session) speechShown() *world.Speech {
	if s.Talk.Speech == nil || !s.Talk.whole {
		return s.Talk.Speech
	}
	sp := *s.Talk.Speech
	sp.Whole = true
	return &sp
}

func (s *Session) focusKey() string {
	if sp := s.Talk.Speech; sp != nil && strings.Contains(sp.Person, "#") && s.Selection.Kind == world.HitHearth && s.Selection.ID == PetOwner(sp.Person) {
		return sp.Person
	}
	if s.Selection.Kind != world.HitWalker {
		return ""
	}
	return world.PersonKey(s.Selection.Pod)
}

func (s *Session) knownConcepts() map[string]bool {
	st := s.State
	rev := func(id string) bool { return slices.Contains(st.Reveals, id) }
	return map[string]bool{
		"quay": rev("serve"), "keep": rev("keep") || rev("build"), "door": rev("door"), "crew": rev("crew"),
		"roll": rev("depot"), "badge": rev("depot"), "depot": rev("depot"), "stewards": rev("found-stewards"),
		"quartermaster": rev("found-quartermaster"), "stone": rev("stone"), "borg": rev(view.BorgIntro),
		"grimoire": rev("grimoire"), "pledge": rev("pledges"), "binding": rev("bindings"), "rations": rev("hands"),
		"voidtaken": rev("oomkilled"), "bell": st.Unlocks[view.BellPage], "shutters": st.Unlocks[view.WatchPage],
		"chronicle": rev("chronicles"), "keeps": rev("keeps"), "drain": st.Age != view.AgeDark, "imp": s.impMet(),
		"tide": rev("siege"), "ages": rev("ageup"), "citadel": rev("found-chancery"),
	}
}

func (s *Session) impMet() bool {
	if s.State.Imp != nil {
		return true
	}
	for _, e := range s.Cache.Events() {
		if e.Reason == tremorReason {
			return true
		}
	}
	return false
}

func (s *Session) WorkerFacts(p model.PodView) talk.Facts {
	w, _ := p.Worker()
	m, measured := s.metrics.Pods[p.Namespace+"/"+p.Name]
	now := s.realmNow()
	f := talk.Facts{
		Names: ui.Naming(), Carrier: gfx.IsTrade(p.Sprite) && !p.System, Trade: p.Profession, Worker: w, Known: s.knownConcepts(), Now: now,
		Hut: talk.Hut{
			Name: p.Name, System: p.System, Rites: p.Terminating, Static: p.Static, Owner: p.Owner, Crew: p.Crew, Depot: p.Depot,
			OnRoll: p.Depot != "" && p.Ready, Claims: len(p.Claims), Node: p.Node, RestartPolicy: p.RestartPolicy,
			Familiars: len(p.Familiars), Workers: len(p.People), JoinedAgo: -1, LeftAgo: -1,
		},
		Use:    talk.Use{Measured: measured, Alone: p.Alone(), CPUMilli: m.CPUMilli, MemBytes: m.MemoryBytes, Served: m.ServedRate, Throttled: m.Throttled},
		Around: talk.Around{Night: world.Darkness(s.State.Pilgrims.DayPhase) > nightDarkness, Tide: s.State.Siege.Phase == "active", Gilded: Gilded(s.State.Golden) != nil, ImpAgo: -1, AgeAgo: -1},
	}
	if p.Depot != "" {
		f.Hut.App = p.Labels["app"]
	}
	if p.ReadySince > 0 && p.Depot != "" {
		if p.Ready {
			f.Hut.JoinedAgo = now - p.ReadySince
		} else {
			f.Hut.LeftAgo = now - p.ReadySince
		}
	}
	if n, ok := s.nodes[p.Node]; ok && p.Node == view.TowerName && slices.Contains(n.Taints, controlPlaneTaint) {
		f.Hut.LoneWard = true
	}
	f.Hut.Rollout, f.Hut.OldClass = s.rolling(p)
	f.Hut.Frozen, f.Hut.ImageReady = s.containerSigns(p, now)
	f.Around.ImpAgo, f.Around.ImpReplaced = s.impNear(p, now)
	if s.Talk.ageAt > 0 {
		f.Around.AgeAgo = now - s.Talk.ageAt
	}
	if mark, ok := s.Renderer.Focus(); ok && mark.Person == world.PersonKey(p) && mark.Seen {
		f.Doing = talk.Doing{Activity: doingOf(mark.Activity), Load: mark.Load}
	}
	f.Problem = s.ProblemFor(p)
	return f
}

const controlPlaneTaint = "node-role.kubernetes.io/control-plane"

func doingOf(activity string) string {
	switch activity {
	case "carrying":
		return talk.DoCarrying
	case "returning":
		return talk.DoReturning
	case "resting":
		return talk.DoResting
	case "serving":
		return talk.DoServing
	case "waiting":
		return talk.DoWaiting
	}
	return ""
}

func (s *Session) rolling(p model.PodView) (bool, bool) {
	rs := hud.PodForeman(p)
	if rs == "" {
		return false, false
	}
	for _, c := range s.Crews {
		for _, f := range c.Foremen {
			if f.ID() != rs {
				continue
			}
			newest, ok := model.Newest(c)
			if !ok {
				return false, false
			}
			out := false
			for _, other := range c.Foremen {
				out = out || !other.Newest && (other.Desired > 0 || other.Current > 0)
			}
			return out, newest.ID() != rs
		}
	}
	return false, false
}

func containerPath(name string) string {
	return "spec.containers" + "{" + name + "}"
}

func (s *Session) podEvents(p model.PodView) []*corev1.Event {
	out := []*corev1.Event{}
	for _, e := range s.Cache.Events() {
		if e.InvolvedObject.Kind == "Pod" && e.InvolvedObject.Namespace == p.Namespace && e.InvolvedObject.Name == p.Name {
			out = append(out, e)
		}
	}
	slices.SortStableFunc(out, func(a, b *corev1.Event) int {
		switch ta, tb := EventTime(a), EventTime(b); {
		case ta < tb:
			return -1
		case ta > tb:
			return 1
		}
		return 0
	})
	return out
}

func (s *Session) containerSigns(p model.PodView, now float64) (bool, bool) {
	frozen, ready := false, false
	path := containerPath(p.Focus)
	for _, e := range s.podEvents(p) {
		if e.InvolvedObject.FieldPath != path {
			continue
		}
		switch {
		case e.Reason == "Pulled" && strings.Contains(e.Message, "already present"):
			ready = true
		case e.Reason == "Unhealthy" && strings.Contains(e.Message, "Liveness"):
			frozen = now-EventTime(e) < frozenWindow
		case e.Reason == "Killing" || e.Reason == "Started":
			frozen = false
		}
	}
	return frozen, ready
}

func (s *Session) impNear(p model.PodView, now float64) (float64, bool) {
	ago, replaced := -1.0, false
	for _, e := range s.Cache.Events() {
		if e.Reason != tremorReason || e.InvolvedObject.Kind != "Pod" {
			continue
		}
		at := EventTime(e)
		if ago >= 0 && now-at >= ago {
			continue
		}
		ago, replaced = now-at, s.shiftOwns(e.InvolvedObject.Namespace, e.InvolvedObject.Name)
	}
	return ago, replaced
}

func (s *Session) shiftOwns(namespace, pod string) bool {
	cut := strings.LastIndex(pod, "-")
	if cut <= 0 {
		return false
	}
	for _, c := range s.Crews {
		for _, f := range c.Foremen {
			if f.ID() == namespace+"/"+pod[:cut] {
				return true
			}
		}
	}
	return false
}

func (s *Session) ProblemFor(p model.PodView) string {
	if p.Depot != "" {
		for _, k := range s.State.Freight.Kinds {
			if k.Depot == p.Depot && k.Losses.QuayFull > 0 {
				return "quayfull"
			}
		}
		for _, k := range s.State.Freight.Kinds {
			if k.Depot == p.Depot && k.SpoiledRate > 0 {
				return "spoiling"
			}
		}
	}
	if p.Crew != "" {
		for _, q := range s.Pods {
			if q.Crew == p.Crew && q.Namespace == p.Namespace && q.State == model.HearthPending {
				return "unhoused"
			}
		}
	}
	if n, ok := s.metrics.Nodes[p.Node]; ok && n.CPUAllocatableMilli > 0 && n.CPUMilli >= droughtShare*float64(n.CPUAllocatableMilli) {
		return "drought"
	}
	return ""
}

func (s *Session) TrackWorkers() {
	s.Talk.Init()
	now := s.realmNow()
	if s.Talk.age != s.State.Age {
		if s.Talk.age != "" {
			s.Talk.ageAt = now
		}
		s.Talk.age = s.State.Age
	}
	for _, p := range s.Pods {
		if p.System || len(p.People) != 1 {
			continue
		}
		w := p.Person(0)
		m, ok := s.metrics.Pods[p.Namespace+"/"+p.Name]
		if !ok {
			continue
		}
		key := world.PersonKey(w)
		k, _ := w.Worker()
		was, seen := s.Talk.Seen[key]
		next := Watched{served: was.served || m.ServedRate > lonelyServed}
		if p.Alone() {
			next.hungry = m.Throttled
			next.cramped = k.Resources.MemLimit > 0 && m.MemoryBytes >= 0.9*float64(k.Resources.MemLimit)
		}
		s.Talk.Seen[key] = next
		if !seen {
			continue
		}
		haps := s.Talk.Lived[key]
		reason := ""
		if k.Resources.CPULimit > 0 && m.CPUMilli >= 0.95*float64(k.Resources.CPULimit) {
			reason = "limit"
		}
		switch {
		case next.hungry && !was.hungry:
			haps = append(haps, talk.Happening{At: now, Kind: talk.HapHungry, Reason: reason})
		case !next.hungry && was.hungry:
			haps = append(haps, talk.Happening{At: now, Kind: talk.HapFed})
		}
		if next.cramped && !was.cramped {
			haps = append(haps, talk.Happening{At: now, Kind: talk.HapCramped})
		}
		if next.served && !was.served {
			haps = append(haps, talk.Happening{At: now, Kind: talk.HapFirstBox, Box: firstBoxWord(p)})
		}
		if len(haps) > livedPerWorker {
			haps = haps[len(haps)-livedPerWorker:]
		}
		s.Talk.Lived[key] = haps
	}
	s.forgetGone()
}

func (s *Session) forgetGone() {
	live := map[string]bool{s.focusKey(): true}
	for _, p := range s.Pods {
		for i := range p.Persons() {
			live[world.PersonKey(p.Person(i))] = true
		}
	}
	for key := range s.Talk.Seen {
		if !live[key] {
			delete(s.Talk.Seen, key)
		}
	}
	for key := range s.Talk.Lived {
		if !live[key] {
			delete(s.Talk.Lived, key)
		}
	}
}

var boxWords = map[string]string{"parcels": "parcel", "ledgers": "ledger", "crates": "crate", "dispatches": "dispatch", "seals": "sealed box"}

func firstBoxWord(p model.PodView) string {
	return boxWords[p.Depot]
}

func (s *Session) happenings(p model.PodView, f talk.Facts) []talk.Happening {
	out := append(talk.FromStatus(f), s.Talk.Lived[world.PersonKey(p)]...)
	path := containerPath(p.Focus)
	for _, e := range s.podEvents(p) {
		at := EventTime(e)
		switch {
		case e.InvolvedObject.FieldPath == path && e.Reason == "BackOff":
			out = append(out, talk.Happening{At: at, Kind: talk.HapBackOff})
		case e.InvolvedObject.FieldPath == path && e.Reason == "Pulled" && strings.Contains(e.Message, "already present"):
			out = append(out, talk.Happening{At: at, Kind: talk.HapPulled})
		case e.InvolvedObject.FieldPath == path && e.Reason == "Killing" && strings.Contains(e.Message, "liveness"):
			out = append(out, talk.Happening{At: at, Kind: talk.HapKilled})
		}
	}
	if f.Hut.JoinedAgo >= 0 {
		out = append(out, talk.Happening{At: p.ReadySince, Kind: talk.HapJoined})
	}
	if f.Hut.LeftAgo >= 0 {
		out = append(out, talk.Happening{At: p.ReadySince, Kind: talk.HapLeft})
	}
	if p.DeletedAt > 0 {
		out = append(out, talk.Happening{At: p.DeletedAt, Kind: talk.HapRites})
	}
	if at, ok := s.rolloutAt(p); ok {
		out = append(out, talk.Happening{At: at, Kind: talk.HapRollout})
	}
	return out
}

func (s *Session) rolloutAt(p model.PodView) (float64, bool) {
	rolling, _ := s.rolling(p)
	if !rolling || p.Crew == "" {
		return 0, false
	}
	at, ok := 0.0, false
	for _, e := range s.Cache.Events() {
		if e.Reason == "ScalingReplicaSet" && e.InvolvedObject.Kind == "Deployment" && e.InvolvedObject.Namespace == p.Namespace && e.InvolvedObject.Name == p.Crew && strings.Contains(e.Message, "Scaled up") {
			at, ok = max(at, EventTime(e)), true
		}
	}
	return at, ok
}
