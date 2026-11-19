package hud

import (
	"fmt"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	acadPrefix          = "acad"
	acadFilterAll       = "all"
	acadFilterTrades    = "trades"
	acadFilterFamiliars = "familiars"
	acadTabOverview     = "overview"
	acadTabClasses      = "classes"
	acadCrewHole        = "<crew>"
)

var acadFilters = []string{acadFilterAll, acadFilterTrades, acadFilterFamiliars}

var acadFilterLabels = map[string]string{acadFilterAll: "ALL", acadFilterTrades: "TRADES", acadFilterFamiliars: "FAMILIARS"}

type academyState struct {
	repo    string
	tab     string
	filter  string
	taken   string
	pick    string
	pinned  bool
	repoTop int
}

type academyRepo struct {
	Profession string
	Repository string
	Name       string
	Familiar   bool
	Classes    []view.SoulstoneEntry
	Endowed    int
}

type academyView struct {
	Repos  []academyRepo
	Sel    int
	Tab    string
	Filter string
}

func (v academyView) Selected() (academyRepo, bool) {
	if v.Sel < 0 || v.Sel >= len(v.Repos) {
		return academyRepo{}, false
	}
	return v.Repos[v.Sel], true
}

func repoName(s view.SoulstoneEntry, profession string) string {
	if name, _, ok := strings.Cut(s.Name, ", class"); ok && name != "" {
		return strings.ToUpper(name)
	}
	return strings.ToUpper(profession) + "S"
}

func academyRepos(f Frame) []academyRepo {
	out := []academyRepo{}
	index := map[string]int{}
	for _, s := range academyRows(f) {
		profession, _ := soulstone.ProfessionOf(s.Image)
		p, _ := soulstone.Lookup(s.Image)
		i, ok := index[profession]
		if !ok {
			i = len(out)
			index[profession] = i
			repo := p.Repository
			if repo == "" {
				repo = strings.ToLower(profession)
			}
			out = append(out, academyRepo{Profession: profession, Repository: repo, Name: repoName(s, profession), Familiar: p.Familiar})
		}
		out[i].Classes = append(out[i].Classes, s)
		if s.Owned && !rumoured(s) {
			out[i].Endowed++
		}
	}
	for i := range out {
		slices.SortStableFunc(out[i].Classes, byTag)
	}
	slices.SortStableFunc(out, func(a, b academyRepo) int { return boolRank(a.Familiar) - boolRank(b.Familiar) })
	return out
}

func boolRank(b bool) int {
	if b {
		return 1
	}
	return 0
}

func filterRepos(repos []academyRepo, filter string) []academyRepo {
	if filter == acadFilterAll || filter == "" {
		return repos
	}
	out := []academyRepo{}
	for _, r := range repos {
		if r.Familiar == (filter == acadFilterFamiliars) {
			out = append(out, r)
		}
	}
	return out
}

func repoIndex(repos []academyRepo, profession string) int {
	for i, r := range repos {
		if r.Profession == profession {
			return i
		}
	}
	return -1
}

func repoOf(repos []academyRepo, image string) int {
	for i, r := range repos {
		for _, c := range r.Classes {
			if c.Image == image {
				return i
			}
		}
	}
	return -1
}

func firstWithOffer(repos []academyRepo) int {
	for i, r := range repos {
		for _, c := range r.Classes {
			if !c.Owned {
				return i
			}
		}
	}
	return 0
}

func (h *HUD) following(f Frame, repos []academyRepo) bool {
	guided := GuidedClass(f)
	return !h.academy.pinned && guided != "" && h.academy.taken != guided && repoOf(repos, guided) >= 0
}

func (h *HUD) academyView(f Frame) academyView {
	all := academyRepos(f)
	if h.following(f, all) {
		return academyView{Repos: all, Sel: repoOf(all, GuidedClass(f)), Tab: acadTabClasses, Filter: acadFilterAll}
	}
	filter := h.academy.filter
	if filter == "" {
		filter = acadFilterAll
	}
	repos := filterRepos(all, filter)
	tab := h.academy.tab
	if tab == "" {
		tab = acadTabClasses
	}
	sel := repoIndex(repos, h.academy.repo)
	if sel < 0 {
		sel = firstWithOffer(repos)
	}
	if len(repos) == 0 {
		sel = -1
	}
	return academyView{Repos: repos, Sel: sel, Tab: tab, Filter: filter}
}

func (h *HUD) takeOver(f Frame) {
	if guided := GuidedClass(f); guided != "" {
		h.academy.taken = guided
	}
}

func (h *HUD) pickedClass(f Frame, repo academyRepo) (view.SoulstoneEntry, bool) {
	for _, c := range repo.Classes {
		if c.Image == h.academy.pick {
			return c, true
		}
	}
	if guided := GuidedClass(f); guided != "" {
		for _, c := range repo.Classes {
			if c.Image == guided {
				return c, true
			}
		}
	}
	for _, c := range repo.Classes {
		if !c.Owned {
			return c, true
		}
	}
	if n := len(repo.Classes); n > 0 {
		return repo.Classes[n-1], true
	}
	return view.SoulstoneEntry{}, false
}

func crewNameFor(st view.GameState, image string) string {
	p, ok := soulstone.Lookup(image)
	if !ok {
		return acadCrewHole
	}
	if c, _, found := crewOfTrade(st, p); found {
		return c.Name
	}
	return acadCrewHole
}

func ClassCommand(st view.GameState, image string) string {
	p, ok := soulstone.Lookup(image)
	if !ok || p.Cursed {
		return ""
	}
	crew := crewNameFor(st, image)
	if p.Familiar && len(st.Crews) > 0 {
		crew = st.Crews[0].Name
	}
	if p.Familiar {
		return view.FamiliarCommand(crew, p.Repository, image)
	}
	if st.RaiseKeeper == view.KeeperReplicaSet && !st.Unlocks[view.DeploymentsPage] {
		return fmt.Sprintf("kubectl set image replicaset/%s %s=%s", crew, p.Trade, image)
	}
	return fmt.Sprintf("kubectl set image deployment/%s %s=%s", crew, p.Trade, image)
}

func (h *HUD) academyCommand(f Frame, v academyView) (string, string) {
	repo, ok := v.Selected()
	if !ok {
		return "", ""
	}
	c, ok := h.pickedClass(f, repo)
	if !ok || !c.Owned {
		return "", ""
	}
	return c.Image, ClassCommand(f.State, c.Image)
}

const (
	AcademyOverview  = acadTabOverview
	AcademyFamiliars = acadFilterFamiliars
)

func (h *HUD) OpenAcademy(tab, filter string) {
	h.OpenAcademyAt("", tab, filter)
}

func (h *HUD) OpenAcademyAt(profession, tab, filter string) {
	h.OverlayID = OverlaySoulvault
	h.academy = academyState{repo: profession, tab: tab, filter: filter, pinned: profession != ""}
}

func tagOf(image string) string {
	_, tag, _ := strings.Cut(image, ":")
	return tag
}

func byTag(a, b view.SoulstoneEntry) int {
	ta, tb := tagOf(a.Image), tagOf(b.Image)
	switch {
	case soulstone.TagNewer(ta, tb):
		return 1
	case soulstone.TagNewer(tb, ta):
		return -1
	}
	return 0
}
