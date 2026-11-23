package hud

import "github.com/waxb/kubernites/pkg/view"

var actionLessons = map[string]func(Frame) bool{
	KeyDelete:   revealedBy("summon"),
	KeyLabel:    revealedBy("depot"),
	KeyPrevious: revealedBy("chronicles"),
	KeyYAML:     revealedBy("scriptorium"),
	KeyTop:      revealedBy("augury"),
	KeyAlarms:   revealedBy("codex-bloat"),
	KeyCompact:  revealedBy("codex-bloat"),
	KeyDefrag:   revealedBy("codex-bloat"),
	KeyDisarm:   revealedBy("codex-bloat"),
	KeyTaint:    inStaging,
	KeyCordon:   inStaging,
	KeyUncordon: inStaging,
	KeyDrain:    inStaging,
}

func revealedBy(id string) func(Frame) bool {
	return func(f Frame) bool { return f.Revealed(id) }
}

func inStaging(f Frame) bool {
	return f.State.Age != view.AgeDark
}

var readsBeforeGetPods = map[string]bool{
	KeyDescribe: true, KeyLogs: true, keyEvents: true, KeyGet: true, KeyStatus: true, keyEndpoints: true,
	keyVersion: true, keyCluster: true, keyResources: true, KeyEtcdVer: true,
}

func askedGetPods(f Frame) bool {
	if !f.Guide.Staged {
		return true
	}
	for _, o := range f.State.Objectives {
		if o.ID == "speak-get" {
			return o.Done
		}
	}
	return false
}

func taughtActions(acts []CardAction, f Frame) []CardAction {
	out := make([]CardAction, 0, len(acts))
	asked := askedGetPods(f)
	for _, a := range acts {
		if readsBeforeGetPods[a.Key] && !asked {
			continue
		}
		if taught, ok := actionLessons[a.Key]; !ok || taught(f) {
			out = append(out, a)
		}
	}
	return out
}

const (
	restartsLesson = "chronicles"
	classLesson    = "soulvault"
	laborLesson    = "hands"
)

func taughtLines(c Card, f Frame) Card {
	next := c
	next.Lines = make([]Line, 0, len(c.Lines))
	for _, l := range c.Lines {
		switch {
		case l.lesson == "" || f.Revealed(l.lesson):
			next.Lines = append(next.Lines, l)
		case l.plain != "":
			plain := l
			plain.Text = l.plain
			next.Lines = append(next.Lines, plain)
		}
	}
	return next
}
