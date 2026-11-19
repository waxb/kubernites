package talk

import "slices"

const (
	RestSeconds   = 150.0
	FollowSeconds = 8.0
	RapidGap      = 3.0
	AnnoyAfter    = 5
	CalmSeconds   = 20.0
)

type Said struct {
	Text      string
	Situation Situation
	Tone      Tone
	Annoyed   int
	FollowUp  bool
}

type clickRun struct {
	count int
	at    float64
}

type followUp struct {
	text string
	at   float64
}

type Memory struct {
	heard  map[string]float64
	ever   map[string]bool
	last   map[string]string
	clicks map[string]clickRun
	follow map[string]followUp
}

func NewMemory() *Memory {
	return &Memory{heard: map[string]float64{}, ever: map[string]bool{}, last: map[string]string{}, clicks: map[string]clickRun{}, follow: map[string]followUp{}}
}

func (m *Memory) Clicked(id Identity) bool {
	_, ok := m.clicks[id.Key]
	return ok
}

func (m *Memory) Last(id Identity) string {
	return m.last[id.Key]
}

func (m *Memory) Click(id Identity, f Facts, now float64) Said {
	run, seen := m.clicks[id.Key]
	f.FirstClick = !seen
	switch {
	case !seen || now-run.at > CalmSeconds:
		run.count = 1
	case now-run.at <= RapidGap:
		run.count++
	}
	run.at = now
	m.clicks[id.Key] = run
	said := m.say(id, f, now, run.count)
	m.heard[said.Text] = now
	m.ever[said.Text] = true
	m.last[id.Key] = said.Text
	return said
}

func (m *Memory) say(id Identity, f Facts, now float64, count int) Said {
	if count >= AnnoyAfter && id.Familiar == "" {
		if said, ok := m.annoyed(id, f, count-AnnoyAfter); ok {
			delete(m.follow, id.Key)
			return said
		}
	}
	if fu, ok := m.follow[id.Key]; ok && now-fu.at <= FollowSeconds && fu.text != m.last[id.Key] {
		delete(m.follow, id.Key)
		return Said{Text: fu.text, Situation: SitGeneral, Tone: id.Tone(), FollowUp: true}
	}
	delete(m.follow, id.Key)
	line, ok := m.pick(id, f, now, count)
	if !ok {
		return Said{Text: "...", Situation: SitGeneral, Tone: id.Tone()}
	}
	if line.Then != "" {
		m.follow[id.Key] = followUp{text: line.Rest(f), at: now}
	}
	return Said{Text: line.Say(f), Situation: line.When, Tone: line.Tone}
}

func (m *Memory) annoyed(id Identity, f Facts, step int) (Said, bool) {
	ladder := []Line{}
	for _, l := range annoyedLines {
		if l.speaks(f) {
			ladder = append(ladder, l)
		}
	}
	if len(ladder) == 0 {
		return Said{}, false
	}
	at := min(step, len(ladder)-1)
	if ladder[at].Text == m.last[id.Key] {
		at = (at + 1) % len(ladder)
	}
	return Said{Text: ladder[at].Say(f), Situation: SitAnnoyed, Tone: AnyTone, Annoyed: step + 1}, true
}

type candidate struct {
	line  Line
	tier  int
	score float64
	rest  bool
}

func (m *Memory) pick(id Identity, f Facts, now float64, count int) (Line, bool) {
	cands := m.candidates(id, f, now, count)
	if len(cands) == 0 {
		return Line{}, false
	}
	tier := chosenTier(cands)
	best := -1
	for i, c := range cands {
		if c.tier != tier {
			continue
		}
		if best < 0 || c.score > cands[best].score {
			best = i
		}
	}
	return cands[best].line, true
}

func chosenTier(cands []candidate) int {
	fresh, any := TierGeneral+1, TierGeneral+1
	for _, c := range cands {
		any = min(any, c.tier)
		if !c.rest {
			fresh = min(fresh, c.tier)
		}
	}
	if any == TierUrgent || fresh > TierGeneral {
		return any
	}
	return fresh
}

func (m *Memory) candidates(id Identity, f Facts, now float64, count int) []candidate {
	active := Situations(f)
	if id.Familiar != "" {
		active = []Situation{SitPet}
	}
	tone := id.Tone()
	out := []candidate{}
	for _, l := range Pool() {
		if l.When == SitAnnoyed || !slices.Contains(active, l.When) || !l.speaks(f) {
			continue
		}
		text := l.Say(f)
		if text == m.last[id.Key] {
			continue
		}
		at, heard := m.heard[text]
		c := candidate{line: l, tier: tierOf(l.When), rest: heard && now-at < RestSeconds}
		c.score = toneScore(l.Tone, tone) - 0.05*float64(rankOf(l.When))
		if !m.ever[text] {
			c.score += 3
		}
		if c.rest {
			c.score -= 8
		}
		c.score += float64(mix(id.Seed^hash(text), uint32(count))%1000) / 1e5
		out = append(out, c)
	}
	return out
}

func toneScore(line, worker Tone) float64 {
	switch line {
	case worker:
		return 4
	case AnyTone:
		return 2
	}
	return -2
}

func Current(id Identity, f Facts) (Line, bool) {
	m := NewMemory()
	cands := m.candidates(id, f, 0, 0)
	best := -1
	for i, c := range cands {
		if !thinks(c.line.When) {
			continue
		}
		if best < 0 || c.tier < cands[best].tier || c.tier == cands[best].tier && c.score > cands[best].score {
			best = i
		}
	}
	if best < 0 {
		return Line{}, false
	}
	return cands[best].line, true
}

var thoughtful = []Situation{SitGuttering, SitVoidtaken, SitNever, SitFrozen, SitHungry, SitCramped, SitKindling, SitRites, SitShutters,
	SitRelit, SitWaiting, SitBurning, SitTrade, SitSystem, SitPet}

func thinks(s Situation) bool {
	return slices.Contains(thoughtful, s)
}
