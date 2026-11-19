package app

import (
	"fmt"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

var sessionVerbs = map[string][2]string{
	"exec": {"SHELL", "EXEC"}, "attach": {"ATTACH", "ATTACH"}, "debug": {"DEBUG", "DEBUG"}, "run": {"RUN", "RUN"},
	"port-forward": {"PORT-FORWARD", "PORT-FORWARD"}, "cp": {"COPY", "CP"},
}

func sessionVerb(verb string) string {
	v, ok := sessionVerbs[verb]
	if !ok {
		return strings.ToUpper(verb)
	}
	return ui.Bridge(v[0], v[1])
}

func sessionEnded(ss kubectltree.Session) string {
	switch ss.Ended {
	case kubectltree.EndedExited:
		if ss.Code != 0 {
			return fmt.Sprintf("exited with code %d", ss.Code)
		}
		return "exited"
	case kubectltree.EndedLeft:
		return "left with ctrl-p ctrl-q"
	case kubectltree.EndedClosed:
		return "closed with the realm"
	case kubectltree.EndedFailed:
		return ui.Bridge("broke off", "failed: "+write.Mask(ss.Err))
	}
	return "open"
}

func forwardEnded(f kubectltree.Forward) string {
	switch f.Ended {
	case kubectltree.EndedLeft:
		return "stopped"
	case kubectltree.EndedClosed:
		return "closed with the realm"
	case kubectltree.EndedFailed:
		return ui.Bridge("lost", "lost: "+write.Mask(f.Err))
	}
	return "ended"
}

func (s *Session) recordSession(order string, ss kubectltree.Session) {
	if !ss.Opened || ss.Channel == "portforward" {
		return
	}
	s.recordReach(order, ss.Verb, ss.Namespace, ss.Pod, ss.Container, strings.Join(ss.Command, " "), sessionEnded(ss))
}

func (s *Session) recordReach(order, verb, namespace, pod, container, command, ended string) {
	w := s.Writer
	if w == nil {
		return
	}
	target := write.Ref{GVR: write.Pods, Kind: "Pod", Namespace: namespace, Name: pod}
	e := write.Entry{At: w.now().UTC(), Context: w.gate.Context().Name, Level: w.level.String(), Order: order, Object: target.String(), Answer: ended, Container: container, Command: command, Ended: ended}
	if err := w.ledger.Append(e); err != nil {
		s.Hud.Toast("The ledger file could not be written: "+err.Error(), true)
	}
	shown := ended
	if container != "" {
		shown = ui.Bridge("worker ", "container ") + container + ", " + ended
	}
	line := ledgerLine{at: e.At, shownVerb: sessionVerb(verb), target: target, ended: shown}
	w.entries = append(w.entries, line)
	if len(w.entries) > ledgerShown {
		w.entries = w.entries[len(w.entries)-ledgerShown:]
	}
	s.Echoes = Chronicled(s.Echoes, "WRITE "+e.Level+" "+line.text(), s.echoClock())
	s.refreshLive()
}

func (s *Session) CaptureForwards(list []kubectltree.Forward) {
	s.captured = list
	s.forwards = list
	s.refreshLive()
}

func (s *Session) stepForwards() {
	t, ok := liveTreeOf(s.Realm)
	if !ok || !s.Live || s.captured != nil {
		return
	}
	tree := t.Tree()
	open := tree.Forwards()
	for _, f := range open {
		if !slices.Contains(s.forwardsSeen, f.ID) && len(f.Listening) > 0 {
			s.forwardsSeen = append(s.forwardsSeen, f.ID)
			s.recordReach(forwardOrder(f), "port-forward", f.Namespace, f.Pod, "", strings.Join(f.Ports, " "), "open on "+strings.Join(f.Listening, ", "))
		}
	}
	for _, f := range tree.TakeEndedForwards() {
		if !slices.Contains(s.forwardsSeen, f.ID) && len(f.Listening) > 0 {
			s.forwardsSeen = append(s.forwardsSeen, f.ID)
			s.recordReach(forwardOrder(f), "port-forward", f.Namespace, f.Pod, "", strings.Join(f.Ports, " "), "open on "+strings.Join(f.Listening, ", "))
		}
		if slices.Contains(s.forwardsSeen, f.ID) {
			s.recordReach(forwardOrder(f), "port-forward", f.Namespace, f.Pod, "", strings.Join(f.Ports, " "), forwardEnded(f))
			s.Hud.Stone.Append(hud.LineHint, forwardEndLine(f))
		}
	}
	if !slices.EqualFunc(open, s.forwards, func(a, b kubectltree.Forward) bool {
		return a.ID == b.ID && slices.Equal(a.Listening, b.Listening) && a.Connections == b.Connections
	}) {
		s.forwards = open
		s.refreshLive()
	}
}

func forwardOrder(f kubectltree.Forward) string {
	return "kubectl port-forward " + f.Target + " -n " + f.Namespace + " " + strings.Join(f.Ports, " ")
}

func forwardEndLine(f kubectltree.Forward) string {
	where := f.Namespace + "/" + f.Pod
	ports := strings.Join(f.Listening, ", ")
	switch f.Ended {
	case kubectltree.EndedLeft:
		return ui.Bridge("The port-forward to hut "+where+" ("+ports+") is stopped.", "The port-forward to pod "+where+" ("+ports+") is stopped.")
	case kubectltree.EndedClosed:
		return ui.Bridge("The port-forward to hut "+where+" closed with the realm.", "The port-forward to pod "+where+" closed with the realm.")
	}
	return ui.Bridge("The port-forward to hut "+where+" was lost.", "The port-forward to pod "+where+" was lost: "+write.Mask(f.Err))
}

func (s *Session) liveForwards() []hud.LiveForward {
	var out []hud.LiveForward
	for _, f := range s.forwards {
		where := f.Namespace + "/" + f.Pod
		ports := strings.Join(f.Listening, ", ")
		if ports == "" {
			ports = strings.Join(f.Ports, " ")
		}
		text := "FORWARD " + forwardShort(f) + " -> " + where
		tip := ui.Bridge(fmt.Sprintf("A port-forward from this computer to hut %s: %s, on 127.0.0.1 only. %d connections so far. STOP closes it; it closes with the realm too.", where, ports, f.Connections),
			fmt.Sprintf("kubectl port-forward %s -n %s: %s, listening on 127.0.0.1 only. %d connections so far. STOP closes the listener; it closes with the realm too.", f.Target, f.Namespace, ports, f.Connections))
		out = append(out, hud.LiveForward{ID: f.ID, Text: text, Tip: tip})
	}
	return out
}

func forwardShort(f kubectltree.Forward) string {
	var local []string
	for _, l := range f.Listening {
		addr, _, _ := strings.Cut(l, " -> ")
		local = append(local, strings.TrimPrefix(addr, kubectltree.ForwardAddress))
	}
	if len(local) == 0 {
		return strings.Join(f.Ports, ",")
	}
	return strings.Join(local, ",")
}

func (s *Session) stopForward(target string) {
	t, ok := liveTreeOf(s.Realm)
	if !ok {
		return
	}
	var id int
	if _, err := fmt.Sscan(target, &id); err != nil {
		return
	}
	t.Tree().StopForward(id)
	s.stepForwards()
}
