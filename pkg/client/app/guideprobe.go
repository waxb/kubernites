package app

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/sound"
)

const probeTicks = 4

type HeldRealm struct{ client.Realm }

func (HeldRealm) Advance(float64) {}

const probeRead = 4.0

var probeAnswers = []string{"lesson:ack", "calamity:close", "quest:ack"}

type GuideProbe struct {
	S      *Session `json:"-"`
	ptr    Pointer
	dst    *ebiten.Image
	reads  bool
	upFrom map[string]float64
	pauses []PauseMark
}

type PauseMark struct {
	Realm float64
	Keys  []string
}

func (p *GuideProbe) Reads() { p.reads = true }

func (p *GuideProbe) CountPauses() {
	p.S.pauseTap = func(asks []hud.PauseAsk) {
		keys := make([]string, 0, len(asks))
		for _, a := range asks {
			keys = append(keys, a.Key)
		}
		p.pauses = append(p.pauses, PauseMark{Realm: p.S.State.RealmTime, Keys: keys})
	}
}

func (p *GuideProbe) Pauses() []PauseMark { return append([]PauseMark(nil), p.pauses...) }

func (p *GuideProbe) Beats() []hud.Beat { return p.S.Hud.Beats() }

func (p *GuideProbe) WatchAttention() { p.S.Hud.SetAttention(true) }

func (p *GuideProbe) Attention() hud.AttnLog {
	return p.S.Hud.Attention().Closed(p.S.Clock, p.S.State.RealmTime)
}

func (p *GuideProbe) answer() {
	if !p.reads {
		return
	}
	if p.upFrom == nil {
		p.upFrom = map[string]float64{}
	}
	buttons := p.S.Hud.Buttons(p.S.Frame(true))
	for _, id := range probeAnswers {
		b, ok := ButtonByID(buttons, id)
		if !ok {
			delete(p.upFrom, id)
			continue
		}
		since, seen := p.upFrom[id]
		if !seen {
			p.upFrom[id] = p.S.Clock
			continue
		}
		if p.S.Clock-since >= probeRead {
			delete(p.upFrom, id)
			p.S.Update(hud.Input{MX: b.Rect.X + b.Rect.W/2, MY: b.Rect.Y + b.Rect.H/2, Click: true}, 0, &p.ptr, true)
			return
		}
	}
}

func ButtonByID(buttons []hud.Button, id string) (hud.Button, bool) {
	for _, b := range buttons {
		if b.ID == id {
			return b, true
		}
	}
	return hud.Button{}, false
}

type GuideCheck struct {
	Time    float64
	Walk    hud.GuideWalk
	Ready   hud.GuideWalk
	Voices  []string
	Context string
	Cells   map[string]int
	Sounds  []sound.SFX
	Rang    []string
}

func NewGuideProbe(r client.Realm) *GuideProbe {
	return &GuideProbe{S: NewSession(HeldRealm{r}, nil), dst: ebiten.NewImage(1, 1)}
}

func (p *GuideProbe) Second() GuideCheck {
	dt := 1 / realmSpeed(p.S.State.Speed) / probeTicks
	p.S.Poll = pollSeconds
	var sounds []sound.SFX
	before := p.S.Rail.Alarms.Rung()
	for range probeTicks {
		_, fx := p.S.Update(hud.Input{MX: -10, MY: -10}, dt, &p.ptr, true)
		sounds = append(sounds, fx...)
	}
	p.answer()
	rang := NewlyRung(before, p.S.Rail.Alarms.Rung())
	if strings.HasPrefix(p.S.Guide.Target, "pod:") {
		p.S.Renderer.Draw(p.dst, p.S.Cam, p.S.Scene())
	}
	f := p.S.Frame(true)
	return GuideCheck{Time: p.S.State.RealmTime, Walk: p.S.Hud.WalkGuide(f), Ready: p.S.Hud.WalkReady(f), Voices: p.voices(f), Context: probeContext(f), Cells: p.S.Hud.MainCells(f), Sounds: sounds, Rang: rang}
}

func probeContext(f hud.Frame) string {
	intro := ""
	if in := f.Guide.Intro; in != nil {
		intro = in.ID
	}
	for _, o := range f.State.Objectives {
		if !o.Done {
			return fmt.Sprintf("intro %q, objective %s (%s) target %q build %q", intro, o.ID, o.Progress, o.Target, o.Build)
		}
	}
	return fmt.Sprintf("intro %q, no objective", intro)
}

func (p *GuideProbe) voices(f hud.Frame) []string {
	out := p.S.Hud.Voices(f)
	if !f.Introducing() {
		for _, c := range p.S.callouts() {
			out = append(out, "callout:"+c.Text)
		}
	}
	for _, n := range p.S.Rail.Notices {
		out = append(out, fmt.Sprintf("rail:%s:%s", n.Kind, n.Title))
	}
	if ship := p.S.State.Golden.Ship; ship != nil && len(ship.Boxes) > 0 {
		out = append(out, "golden:ship")
	}
	return out
}
