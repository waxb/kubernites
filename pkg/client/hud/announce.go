package hud

import (
	"fmt"
	"slices"
)

const (
	NoticeGap = 1.0

	AnnounceBuilt    = "built"
	AnnounceSeasoned = "seasoned"
	AnnounceCalamity = "calamity"
	AnnounceReveal   = "reveal"
	AnnounceGate     = "gate"
)

type announcement struct {
	kind string
	text string
	bad  bool
}

type bannerItem struct {
	text  string
	cause string
	feat  bool
	still func() bool
}

func (b bannerItem) stale() bool {
	return b.still != nil && !b.still()
}

var announcePlurals = map[string]string{
	AnnounceBuilt:    "%d buildings stand complete",
	AnnounceSeasoned: "%d crews are seasoned: they carry faster",
	AnnounceReveal:   "%d new things are revealed",
}

var logOnly = map[string]bool{AnnounceBuilt: true, AnnounceReveal: true, AnnounceCalamity: true, AnnounceSeasoned: true}

func (h *HUD) Announce(kind, text string, bad bool) {
	if logOnly[kind] {
		return
	}
	h.Director.inbox = append(slices.Clone(h.Director.inbox), announcement{kind: kind, text: text, bad: bad})
}

func (h *HUD) Pending() int { return len(h.Director.inbox) }

func (h *HUD) releaseAnnouncement() {
	if len(h.Director.inbox) == 0 || h.Now < h.Director.nextOut {
		return
	}
	kind := h.Director.inbox[0].kind
	same, rest := []announcement{}, []announcement{}
	for _, a := range h.Director.inbox {
		if a.kind == kind {
			same = append(same, a)
		} else {
			rest = append(rest, a)
		}
	}
	h.Director.inbox = rest
	text, bad := MergeAnnouncements(kind, textsOf(same), anyBad(same))
	h.Director.nextOut = h.Now + NoticeGap
	h.queueNews(Toast{Text: text, Bad: bad, News: true})
}

func textsOf(items []announcement) []string {
	out := make([]string, 0, len(items))
	for _, a := range items {
		out = append(out, a.text)
	}
	return out
}

func anyBad(items []announcement) bool {
	return slices.ContainsFunc(items, func(a announcement) bool { return a.bad })
}

func MergeAnnouncements(kind string, texts []string, bad bool) (string, bool) {
	switch {
	case len(texts) == 0:
		return "", bad
	case len(texts) == 1:
		return texts[0], bad
	}
	if plural, ok := announcePlurals[kind]; ok {
		return fmt.Sprintf(plural, len(texts)), bad
	}
	return fmt.Sprintf("%s (and %d more)", texts[0], len(texts)-1), bad
}

func (h *HUD) PushToast(text string, bad bool) {
	h.pushToastAs(Toast{Text: text, Bad: bad})
}

func (h *HUD) pushToastAs(t Toast) {
	next := make([]Toast, 0, len(h.ToastList)+1)
	count := 1
	for _, old := range h.ToastList {
		if old.Text == t.Text {
			count += max(old.Count, 1)
			t.Spot = old.Spot
			continue
		}
		next = append(next, old)
	}
	t.Start, t.Count = h.Now, count
	next = append(next, t)
	h.ToastList = next[max(len(next)-maxToasts, 0):]
	h.Director.nextOut = h.Now + NoticeGap
}

func (h *HUD) News(text string, bad bool) {
	h.queueNews(Toast{Text: text, Bad: bad, News: true})
}

func (h *HUD) Fanfare(text string) {
	t := Toast{Text: text, News: true, Fanfare: true}
	if h.holdFanfare(t) {
		return
	}
	h.pushToastAs(t)
}

func (h *HUD) BannerText() (string, bool) {
	return h.Juice.Banner, h.BannerActive()
}

func (h *HUD) BannerActive() bool {
	_, ok := BannerOffset(h.Now - h.Juice.BannerAt)
	return ok && h.Juice.Banner != ""
}

func (h *HUD) queueBanner(text, cause string, feat bool) {
	h.queueBannerItem(bannerItem{text: text, cause: cause, feat: feat})
}

func (h *HUD) pruneBanners() {
	h.Director.Banners = slices.DeleteFunc(slices.Clone(h.Director.Banners), bannerItem.stale)
}

func (h *HUD) queueBannerItem(item bannerItem) {
	feat := item.feat
	next := slices.Clone(h.Director.Banners)
	at := len(next)
	if feat {
		at = slices.IndexFunc(next, func(b bannerItem) bool { return !b.feat })
		if at < 0 {
			at = len(next)
		}
	}
	h.Director.Banners = slices.Insert(next, at, item)
	if !h.Director.Stage.Primed && !h.BannerActive() {
		h.showBanner()
	}
}

func (h *HUD) showBanner() {
	h.pruneBanners()
	if len(h.Director.Banners) == 0 {
		return
	}
	b := h.Director.Banners[0]
	h.Director.Banners = slices.Clone(h.Director.Banners[1:])
	h.Juice.Banner, h.Juice.Cause, h.Juice.feat, h.Juice.BannerAt = b.text, b.cause, b.feat, h.Now
}
