package hud

import (
	"sort"
	"strings"
)

const (
	NoticeDanger   = "danger"
	NoticeDecision = "decision"
	NoticeNews     = "news"
	NoticeOpen     = "open"
	NoticeDismiss  = "dismiss"
	IntentNotice   = "notice"
	noticeMore     = "more"
	noticePrefix   = "notice:"
)

type Notice struct {
	ID      string
	Kind    string
	Icon    string
	Title   string
	Detail  string
	Count   int
	At      *[2]float64
	Intent  string
	Since   float64
	Expires float64
	Quiet   bool
	Timely  bool
	Told    float64
}

var noticeRanks = map[string]int{NoticeDanger: 0, NoticeDecision: 1, NoticeNews: 2}

func SortNotices(ns []Notice) []Notice {
	out := append([]Notice(nil), ns...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if noticeRanks[a.Kind] != noticeRanks[b.Kind] {
			return noticeRanks[a.Kind] < noticeRanks[b.Kind]
		}
		if a.Since != b.Since {
			return a.Since > b.Since
		}
		return a.ID < b.ID
	})
	return out
}

func ButtonIcon(id string) string {
	for _, e := range mainEntries {
		if e.button.ID == id {
			return e.button.Icon
		}
	}
	switch {
	case id == DoorButton.ID:
		return DoorButton.Icon
	case strings.HasPrefix(id, "build:"):
		return "icon/build"
	}
	return "icon/chronicle"
}

func AlertIcon(kind string) string {
	return alertIcon(kind)
}

func findNotice(ns []Notice, id string) (Notice, bool) {
	for _, n := range ns {
		if n.ID == id {
			return n, true
		}
	}
	return Notice{}, false
}
