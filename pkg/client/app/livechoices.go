package app

import (
	"errors"
	"slices"
	"time"

	"github.com/waxb/kubernites/pkg/client"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live/write"
)

type gated interface {
	Gate() *write.Gate
}

func (g *Game) writerFor(r client.Realm) *liveWriter {
	gr, ok := r.(gated)
	if !ok || gr.Gate() == nil {
		return nil
	}
	gate := gr.Gate()
	ctx := gate.Context()
	w := newLiveWriter(gate, g.LiveDir(), time.Now, func(c write.Class) { g.rememberClass(ctx, c) })
	w.hints = g.Settings.LiveHints
	return w
}

func (g *Game) loadChoices() {
	mem, err := write.LoadMemory(g.LiveDir())
	if err != nil {
		g.Log.Warn("live choices ignored", "err", err)
	}
	g.choices = mem
	g.Picker.Forced = g.Opts.ReadOnly
	g.Picker.Remembered = pickerMemory(mem)
}

func pickerMemory(m write.Memory) map[string]hud.LiveMemory {
	out := map[string]hud.LiveMemory{}
	for _, r := range m.Contexts {
		classes := make([]string, len(r.Classes))
		for i, c := range r.Classes {
			classes[i] = string(c)
		}
		out[r.Context.Name] = hud.LiveMemory{Server: write.RedactServer(r.Context.Server), User: r.Context.User, Identity: r.Context.Identity, Level: r.Level.String(), Warned: r.Warned.String(), Classes: classes}
	}
	return out
}

func (g *Game) saveChoices() {
	if err := write.SaveMemory(g.LiveDir(), g.choices); err != nil {
		g.Log.Warn("could not save live choices", "err", err)
	}
	g.Picker.Remembered = pickerMemory(g.choices)
}

func (g *Game) updateChoices(change func(*write.Memory)) {
	if dir := g.LiveDir(); dir != "" {
		if mem, err := write.LoadMemory(dir); err == nil {
			g.choices = mem
		}
	}
	change(&g.choices)
	g.saveChoices()
}

func (g *Game) forgetChoice(name string) {
	g.updateChoices(func(m *write.Memory) { m.Forget(name) })
	if gate := g.openGate(); gate != nil && gate.Context().Name == name {
		gate.Forget()
	}
}

func (g *Game) forgetAllChoices() {
	g.updateChoices(func(m *write.Memory) { m.ForgetAll() })
	if gate := g.openGate(); gate != nil {
		gate.Forget()
	}
}

func (g *Game) openGate() *write.Gate {
	if g.Sess == nil || !g.Sess.Live {
		return nil
	}
	if gr, ok := g.Sess.Realm.(gated); ok {
		return gr.Gate()
	}
	return nil
}

func (g *Game) pickedContext(name string) (write.Context, bool) {
	for _, c := range g.Picker.Contexts {
		if c.Name == name {
			return write.Context{Name: c.Name, Server: c.Server, User: c.User, Identity: c.Identity, ClusterID: c.ClusterID}, true
		}
	}
	return write.Context{}, false
}

func (g *Game) gateFor(name, level string) (*write.Gate, error) {
	if level == "" || level == hud.LevelReadOnly || g.Opts.ReadOnly {
		return nil, nil
	}
	choice, err := g.confirmedChoice(name, level)
	if err != nil {
		return nil, err
	}
	gate, err := write.New(choice)
	if err != nil {
		return nil, err
	}
	g.rememberChoice(choice)
	return gate, nil
}

func (g *Game) confirmedChoice(name, level string) (write.Choice, error) {
	l, ok := write.ParseLevel(level)
	if !ok || l == write.ReadOnly {
		return write.Choice{}, errors.New("no level was confirmed")
	}
	ctx, ok := g.pickedContext(name)
	if !ok {
		return write.Choice{}, errors.New("the context is no longer in the kubeconfig")
	}
	var classes []write.Class
	if r, _, ok := g.choices.Recall(ctx); ok {
		classes = slices.Clone(r.Classes)
	}
	return write.Choice{Context: ctx, Level: l, Classes: classes}, nil
}

func (g *Game) rememberChoice(c write.Choice) {
	g.updateChoices(func(m *write.Memory) {
		warned := c.Level
		if r, _, ok := m.Recall(c.Context); ok && r.Warned > warned {
			warned = r.Warned
		}
		m.Keep(write.Remembered{Context: c.Context, Level: c.Level, Warned: warned, Classes: c.Classes})
	})
}

func (g *Game) rememberClass(ctx write.Context, c write.Class) {
	g.updateChoices(func(m *write.Memory) {
		r, _, ok := m.Recall(ctx)
		if !ok {
			return
		}
		if !slices.Contains(r.Classes, c) {
			r.Classes = append(r.Classes, c)
		}
		m.Keep(r)
	})
}

func (g *Game) toggleHints() {
	g.Settings.LiveHints = !g.Settings.LiveHints
	if s := g.Sess; s != nil && s.Live {
		if s.Writer != nil {
			s.Writer.hints = g.Settings.LiveHints
		}
		s.hints = g.Settings.LiveHints
		s.refreshLive()
	}
	if err := SaveSettings(g.SettingsPath(), g.Settings); err != nil {
		g.Log.Warn("could not save settings", "err", err)
	}
}

func (g *Game) pushChoices() {
	if g.Sess == nil || !g.Sess.Live {
		return
	}
	if mem, err := write.LoadMemory(g.LiveDir()); err == nil && g.LiveDir() != "" {
		g.choices = mem
	}
	var rows []hud.LiveChoice
	for _, r := range g.choices.Contexts {
		classes := make([]string, len(r.Classes))
		for i, c := range r.Classes {
			classes[i] = string(c)
		}
		rows = append(rows, hud.LiveChoice{Name: r.Context.Name, Server: write.RedactServer(r.Context.Server), Level: r.Level.String(), Classes: classes})
	}
	if gate := g.openGate(); gate != nil && !slices.ContainsFunc(rows, func(c hud.LiveChoice) bool { return c.Name == gate.Context().Name }) {
		var classes []string
		for _, c := range gate.Classes() {
			classes = append(classes, string(c))
		}
		rows = append(rows, hud.LiveChoice{Name: gate.Context().Name, Server: write.RedactServer(gate.Context().Server), Level: gate.Level().String(), Classes: classes})
	}
	g.Sess.Hud.SetLiveChoices(rows)
}
