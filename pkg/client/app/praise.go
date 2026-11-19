package app

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	stepTickLead  = "Done: "
	stepsTickLead = "Steps done: "
	skippedLead   = "Already done: "
)

var wisdomTail = regexp.MustCompile(` \(\+(\d+) wisdom\)$`)

type stepEnd struct {
	title   string
	wisdom  int
	major   bool
	skipped bool
	lost    bool
}

func stepEndOf(p view.Proclamation) (stepEnd, bool) {
	if p.Kind != "info" {
		return stepEnd{}, false
	}
	text := strings.TrimSpace(p.Text)
	var rest string
	var skipped bool
	switch {
	case strings.HasPrefix(text, fulfilledLead):
		rest = strings.TrimPrefix(text, fulfilledLead)
	case strings.HasPrefix(text, view.AlreadyDone):
		rest, skipped = strings.TrimPrefix(text, view.AlreadyDone), true
	default:
		return stepEnd{}, false
	}
	end := stepEnd{major: p.Major, skipped: skipped, lost: strings.Contains(rest, view.TideNotWeathered+":")}
	if m := wisdomTail.FindStringSubmatchIndex(rest); m != nil {
		end.wisdom, _ = strconv.Atoi(rest[m[2]:m[3]])
		rest = rest[:m[0]]
	}
	end.title = strings.TrimSuffix(strings.TrimSpace(rest), ".")
	return end, true
}

func quietSteps(procs []view.Proclamation) map[int]bool {
	folded := stepLessons(procs)
	quiet := map[int]bool{}
	for i, p := range procs {
		if _, ok := stepEndOf(p); !ok {
			continue
		}
		quiet[i] = true
		if lesson, has := folded[i]; has {
			quiet[lesson] = true
		}
	}
	return quiet
}

func TickText(ends []stepEnd) string {
	switch len(ends) {
	case 0:
		return ""
	case 1:
		e := ends[0]
		if e.skipped {
			return skippedLead + e.title
		}
		return stepTickLead + e.title + wisdomText(e.wisdom)
	}
	titles := make([]string, 0, len(ends))
	total := 0
	for _, e := range ends {
		titles = append(titles, e.title)
		total += e.wisdom
	}
	list := strings.Join(titles[:len(titles)-1], ", ") + " and " + titles[len(titles)-1]
	return stepsTickLead + list + wisdomText(total)
}

func wisdomText(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf(" (+%d wisdom)", n)
}

func anyLost(ends []stepEnd) bool {
	for _, e := range ends {
		if e.lost {
			return true
		}
	}
	return false
}
