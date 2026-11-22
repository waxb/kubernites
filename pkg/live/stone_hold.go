package live

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

type StoneAsk struct {
	Why         write.Why
	Class       write.Class
	Level       write.Level
	Order       write.Order
	Preview     write.Answer
	Opens       write.Opening
	Connect     bool
	Command     string
	Local       string
	Writing     bool
	Overwrite   string
	More        string
	Detail      string
	Unpreviewed bool
}

func (a StoneAsk) Confirm() bool {
	return a.Why == write.NeedsConfirm
}

func HoldStoneWrite(ctx context.Context, gate *write.Gate, server string, held kubectltree.Held, ask func(StoneAsk) bool) bool {
	command := strings.Join(append([]string{"kubectl"}, held.Command...), " ")
	if held.Overwrite != "" || held.More != "" {
		a := StoneAsk{Overwrite: held.Overwrite, More: held.More, Command: command, Local: held.Local, Writing: held.Writing}
		if gate != nil {
			a.Level = gate.Level()
		}
		return ask(a)
	}
	var refusal *write.Refusal
	if gate == nil || !errors.As(held.Err, &refusal) {
		return false
	}
	switch refusal.Why {
	case write.NeedsClass:
		if !ask(StoneAsk{Why: write.NeedsClass, Class: refusal.Class, Level: gate.Level(), Command: command, Detail: refusal.Detail}) {
			return false
		}
		gate.Pass(refusal.Class)
		return true
	case write.NeedsConfirm:
		req, ok := stoneRequest(server, held)
		if !ok {
			return false
		}
		order := write.Order{Verb: strings.Join(held.Command, " "), Request: req, ContentType: held.Header.Get("Content-Type"), Classes: write.ClassesOf(req), Command: command}
		if opens, ok := write.Opens(req); ok {
			if !ask(StoneAsk{Why: write.NeedsConfirm, Level: gate.Level(), Order: order, Opens: opens, Connect: true, Command: command, Local: held.Local, Writing: held.Writing}) {
				return false
			}
			ticket := gate.Issue(req)
			context.AfterFunc(ctx, func() { gate.Revoke(ticket) })
			return true
		}
		if write.IgnoresDryRun(req) {
			if !ask(StoneAsk{Why: write.NeedsConfirm, Level: gate.Level(), Order: order, Unpreviewed: true, Detail: write.MintsOnDryRun, Command: command}) {
				return false
			}
			ticket := gate.Issue(req)
			context.AfterFunc(ctx, func() { gate.Revoke(ticket) })
			return true
		}
		preview, _ := gate.Preview(ctx, order)
		confirmed := ask(StoneAsk{Why: write.NeedsConfirm, Level: gate.Level(), Order: order, Preview: preview, Command: command})
		if !confirmed || !preview.OK() {
			return false
		}
		ticket := gate.Issue(req)
		context.AfterFunc(ctx, func() { gate.Revoke(ticket) })
		return true
	}
	return false
}

func stoneRequest(server string, held kubectltree.Held) (write.Request, bool) {
	u, err := url.Parse(server)
	if err != nil || held.URL == nil {
		return write.Request{}, false
	}
	query, err := url.ParseQuery(held.URL.RawQuery)
	if err != nil {
		return write.Request{}, false
	}
	path := held.URL.EscapedPath()
	base := strings.TrimRight(u.Path, "/")
	if base != "" && !strings.HasPrefix(path, base+"/") {
		return write.Request{}, false
	}
	return write.Request{Method: held.Method, Path: strings.TrimPrefix(path, base), Query: query, Body: held.Body, ContentType: held.Header.Get("Content-Type")}, true
}
