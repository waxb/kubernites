package app

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
)

const lessonStop = ". "

func MergeTidings(ts []hud.Tiding) []hud.Tiding {
	out := make([]hud.Tiding, 0, len(ts))
	count := 0
	for i, t := range ts {
		count++
		if i+1 < len(ts) && sameTiding(t, ts[i+1]) {
			continue
		}
		merged := ts[i-count+1]
		merged.Text = hud.Counted(merged.Text, count)
		out, count = append(out, merged), 0
	}
	return out
}

func sameTiding(a, b hud.Tiding) bool {
	return a.Kind == b.Kind && a.Text == b.Text && a.Title == b.Title && a.Reason == b.Reason
}

func LessonLine(title, text string) string {
	first := strings.TrimSpace(text)
	if i := strings.Index(first, lessonStop); i > 0 {
		first = first[:i+1]
	}
	switch {
	case first != "":
		return lessonLead + first
	case title != "":
		return lessonLead + title
	}
	return lessonLead + "a lesson"
}

func MergeNews(news []hud.Notice) []hud.Notice {
	at := map[string]int{}
	out := make([]hud.Notice, 0, len(news))
	for _, n := range news {
		key := n.Kind + "|" + n.Icon + "|" + n.Title
		i, seen := at[key]
		if !seen {
			at[key], out = len(out), append(out, n)
			continue
		}
		out[i] = joinNews(out[i], n)
	}
	return out
}

func joinNews(old, n hud.Notice) hud.Notice {
	next := n
	next.Count = max(old.Count, 1) + max(n.Count, 1)
	lines := strings.Split(n.Detail, "\n")
	for _, l := range strings.Split(old.Detail, "\n") {
		if len(lines) < listMax && !contains(lines, l) {
			lines = append(lines, l)
		}
	}
	next.Detail = strings.Join(lines, "\n")
	return next
}
