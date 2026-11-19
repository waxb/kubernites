package app

import (
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	ForumTipStory = "The Forum. etcd keeps the realm's record; what belongs to no Node stands at its door."
	ForumTipK8s   = "Beside etcd: objects on no Node (Services, HPAs, Pods with no nodeName yet)."
)

func ForumTip() string { return ui.Bridge(ForumTipStory, ForumTipK8s) }

func (s *Session) forumUnder(mx, my int) string {
	x, y := s.Cam.TileAt(mx, my)
	if s.Site.OnForum(x, y) {
		return world.HitForum
	}
	return ""
}

func LessonOf(p view.Proclamation) (string, string) {
	if p.Kind == "lesson" && p.Title == view.ForumTitle {
		return ui.Bridge(view.ForumTitle, view.ForumTitleK8s), ui.Bridge(view.ForumLesson, view.ForumLessonK8s)
	}
	return p.Title, p.Text
}

func ForumPreviewTints(site view.Site, g world.Ghost) []world.Tint {
	p, ok := site.PreviewForum(g.X, g.Y, g.W, g.H)
	if !ok {
		return nil
	}
	return world.MergeTints(world.DiscTints(p.Plan, world.TintOwn), world.DiscTints(p.Base, world.TintForum))
}

const (
	ForumOpens    = "Its door opens on the pale ground beside it."
	ForumOpensK8s = "Services and HPAs go on the pale ground."
)

func ForumNote(site view.Site, kind string, chapter bool) string {
	if kind != view.StructCodex || site.Forum != nil || chapter {
		return ""
	}
	return ui.Bridge(ForumOpens, ForumOpensK8s)
}
