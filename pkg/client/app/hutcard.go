package app

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

const hutHoverHead = "hut:"

func (s *Session) podNamed(key string) (model.PodView, bool) {
	for _, p := range s.Pods {
		if p.Namespace+"/"+p.Name == key {
			return p, true
		}
	}
	return model.PodView{}, false
}

func TroubleOf(st view.GameState, pod string) (why, fix string) {
	for _, t := range st.CauseTags {
		if t.Pod == pod && t.Why != "" {
			return t.Why, fixOf(st, pod, t.Tag)
		}
		if t.Pod == pod {
			return PlainWhy(t.Text), fixOf(st, pod, t.Tag)
		}
	}
	return "", ""
}

func PlainWhy(text string) string {
	head, rest, ok := strings.Cut(text, ": ")
	if !ok || strings.Contains(head, " ") || rest == "" {
		return text
	}
	if verdict, read := view.Content.SeatVerdictText(rest); read {
		return verdict
	}
	return rest
}

func fixOf(st view.GameState, pod, tag string) string {
	_, name, _ := cutSlash(pod)
	for _, c := range st.Calamities {
		if (c.Object == "pod/"+name || c.Kind == tag) && len(c.Remedies) > 0 {
			return c.Remedies[0].Label
		}
	}
	return ""
}

func cutSlash(s string) (string, string, bool) {
	for i := range s {
		if s[i] == '/' {
			return s[:i], s[i+1:], true
		}
	}
	return "", s, false
}

func (s *Session) hutTip(key string) string {
	p, ok := s.podNamed(key)
	if !ok {
		return ""
	}
	why, fix := TroubleOf(s.State, key)
	return hud.HutTip(p, s.metrics.Pods[key], why, fix)
}

func (s *Session) causeTipText(pod, tag, text string) string {
	p, _ := s.podNamed(pod)
	why, fix := TroubleOf(s.State, pod)
	if why == "" {
		why = PlainWhy(text)
	}
	return hud.CauseTipText(model.CauseTitle(tag), p.Name, p.Node, why, fix)
}
