package hud

import (
	"fmt"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	academyRow       = 38
	AcademyTextPad   = 150
	academyLeftW     = 216
	academyCardH     = 44
	academyCardGap   = 3
	academyTop       = 40
	academyListTop   = 58
	academyHead      = 52
	academyBody      = 72
	academyTabH      = 14
	academyColHead   = 12
	academyCmdH      = 20
	academyCmdLabelW = 96
)

func academyRows(f Frame) []view.SoulstoneEntry {
	shown := []view.SoulstoneEntry{}
	for _, s := range f.State.Soulstones {
		if s.Visible || s.Owned {
			shown = append(shown, s)
		}
	}
	return byProfession(shown)
}

func byProfession(rows []view.SoulstoneEntry) []view.SoulstoneEntry {
	rank := map[string]int{}
	for _, s := range rows {
		p, _ := soulstone.ProfessionOf(s.Image)
		if _, ok := rank[p]; !ok {
			rank[p] = len(rank)
		}
	}
	out := slices.Clone(rows)
	slices.SortStableFunc(out, func(a, b view.SoulstoneEntry) int {
		pa, _ := soulstone.ProfessionOf(a.Image)
		pb, _ := soulstone.ProfessionOf(b.Image)
		return rank[pa] - rank[pb]
	})
	return out
}

func (h *HUD) academyStart(f Frame, rows []view.SoulstoneEntry) int {
	start := min(h.list, max(len(rows)-academyVisible(), 0))
	for i, s := range rows {
		if view.VaultPrefix+s.Image != f.Guide.Button {
			continue
		}
		switch {
		case i < start:
			return i
		case i >= start+academyVisible():
			return i - academyVisible() + 1
		}
	}
	return start
}

func GuidedClass(f Frame) string {
	image, _ := strings.CutPrefix(f.Guide.Button, view.VaultPrefix)
	return image
}

func rumoured(s view.SoulstoneEntry) bool {
	return s.Owned && s.Cursed
}

func academyVisible() int {
	return (academyRight().H - academyBody - academyColHead - 2 - academyCmdH - 8) / academyRow
}

func academyLeft() ui.Rect {
	return ui.Rect{X: Overlay.X + 8, Y: Overlay.Y + academyListTop, W: academyLeftW, H: Overlay.H - academyListTop - 8}
}

func academyRight() ui.Rect {
	x := Overlay.X + 8 + academyLeftW + 10
	return ui.Rect{X: x, Y: Overlay.Y + academyTop, W: Overlay.X + Overlay.W - 8 - x, H: Overlay.H - academyTop - 8}
}

func academyReposVisible() int { return academyLeft().H / (academyCardH + academyCardGap) }

func academyCardRect(i int) ui.Rect {
	l := academyLeft()
	return ui.Rect{X: l.X, Y: l.Y + i*(academyCardH+academyCardGap), W: l.W, H: academyCardH}
}

func academyInner() ui.Rect { return academyRight().Inset(8) }

func academyTabRect(i int) ui.Rect {
	in := academyInner()
	return ui.Rect{X: in.X + i*68, Y: in.Y + academyHead, W: 64, H: academyTabH}
}

func AcademyRowRect(i int) ui.Rect {
	in := academyInner()
	return ui.Rect{X: in.X, Y: in.Y + academyBody + academyColHead + 2 + i*academyRow, W: in.W, H: academyRow - 4}
}

func academyCmdRect() ui.Rect {
	in := academyInner()
	return ui.Rect{X: in.X, Y: in.Y + in.H - academyCmdH, W: in.W, H: academyCmdH}
}

func academyChipRect(i int) ui.Rect {
	x := Overlay.X + 8 + 42
	for j := range i {
		x += len(acadFilterLabels[acadFilters[j]])*ui.Advance + 10 + 3
	}
	return ui.Rect{X: x, Y: Overlay.Y + academyTop, W: len(acadFilterLabels[acadFilters[i]])*ui.Advance + 10, H: academyTabH}
}

func (h *HUD) academyClasses(f Frame, v academyView) []view.SoulstoneEntry {
	if repo, ok := v.Selected(); ok {
		return repo.Classes
	}
	return nil
}

func (h *HUD) academyReposStart(v academyView) int {
	top := min(h.academy.repoTop, max(len(v.Repos)-academyReposVisible(), 0))
	switch {
	case v.Sel < 0:
		return 0
	case v.Sel < top:
		return v.Sel
	case v.Sel >= top+academyReposVisible():
		return v.Sel - academyReposVisible() + 1
	}
	return max(top, 0)
}

func (h *HUD) AcademyButtons(f Frame) []Button {
	v := h.academyView(f)
	out := []Button{}
	for i, id := range acadFilters {
		out = append(out, Button{ID: acadPrefix + ":filter:" + id, Rect: academyChipRect(i), Label: acadFilterLabels[id], Selected: v.Filter == id,
			Tip: academyFilterTip(id)})
	}
	top := h.academyReposStart(v)
	for i := range min(academyReposVisible(), max(len(v.Repos)-top, 0)) {
		r := v.Repos[top+i]
		out = append(out, Button{ID: acadPrefix + ":repo:" + r.Profession, Rect: academyCardRect(i), Selected: top+i == v.Sel,
			Tip: ui.Bridge("Open the page of the ", "Open the repository of the ") + strings.ToLower(r.Name) + ": " + soulstone.Tagline(r.Classes[0].Image)})
	}
	out = append(out,
		Button{ID: acadPrefix + ":tab:" + acadTabOverview, Rect: academyTabRect(0), Label: "OVERVIEW", Selected: v.Tab == acadTabOverview, Tip: "What this trade is and how its classes climb"},
		Button{ID: acadPrefix + ":tab:" + acadTabClasses, Rect: academyTabRect(1), Label: "CLASSES", Selected: v.Tab == acadTabClasses, Tip: ui.Bridge("Every class of this trade, with its price", "Every tag of this image, with its price")})
	if v.Tab == acadTabClasses {
		out = append(out, h.classButtons(f, v)...)
	}
	return append(out, closeButton(Overlay))
}

func academyFilterTip(id string) string {
	switch id {
	case acadFilterTrades:
		return "Only the trades: workers that carry boxes"
	case acadFilterFamiliars:
		return ui.Bridge("Only the familiars: they ride beside a worker and carry no boxes", "Only the sidecars: images that run beside a worker and carry no boxes")
	}
	return "Every trade and every familiar"
}

func (h *HUD) classButtons(f Frame, v academyView) []Button {
	classes := h.academyClasses(f, v)
	out := []Button{}
	start := h.academyStart(f, classes)
	for i, s := range classes[start:min(len(classes), start+academyVisible())] {
		r := AcademyRowRect(i)
		out = append(out, Button{ID: acadPrefix + ":pick:" + s.Image, Rect: ui.Rect{X: r.X, Y: r.Y, W: r.W - AcademyTextPad, H: r.H},
			Tip: s.Name + ": " + s.Description + classEffectTip(s)})
		out = append(out, h.endowButton(f, s, r))
	}
	if image, command := h.academyCommand(f, v); command != "" {
		out = append(out, Button{ID: acadPrefix + ":cmd", Rect: academyChipRect2(), Label: command, Tip: commandTip(image) + "\n" + command})
	}
	return out
}

func classEffectTip(s view.SoulstoneEntry) string {
	if s.Effect == "" {
		return ""
	}
	return "\n" + s.Effect
}

func commandTip(image string) string {
	return ui.Bridge("Opens the Speaking Stone with the order that puts "+image+" to work. Press Enter to speak it.",
		"Opens the Speaking Stone with the command that sets "+image+". Press Enter to run it.")
}

func academyChipRect2() ui.Rect {
	r := academyCmdRect()
	label := academyCmdLabelW
	return ui.Rect{X: r.X + label, Y: r.Y + (r.H-ChipH)/2, W: r.W - label, H: ChipH}
}

func (h *HUD) endowButton(f Frame, s view.SoulstoneEntry, r ui.Rect) Button {
	b := Button{ID: "vault:" + s.Image, Rect: ui.Rect{X: r.X + r.W - 68, Y: r.Y + (r.H-18)/2, W: 64, H: 18},
		Label: "ENDOW", Tip: withPayoffLine(ui.Bridge("Endow this class: ", "Buy this image tag: ")+s.Name, AcademyPayoff(f, s)), Disabled: f.State.Gold < s.Cost}
	if b.Disabled && !s.Owned {
		b.Tip = fmt.Sprintf("Not enough gold yet: it costs %s and you have %s.\n%s", Compact(s.Cost), Compact(f.State.Gold), b.Tip)
	}
	switch {
	case rumoured(s):
		b.Label, b.Disabled = ui.Bridge("RUMOUR", "UNVERIFIED"), false
		b.Tip = ui.Bridge("Nobody endowed this class: it came as a rumour. Anyone may hire it free, and nobody examined it.",
			"Nobody bought this image tag: it was published untested. Any Pod may pull it, and nobody verified it.")
		b.ID = "vault:owned"
	case s.Owned:
		b.Label, b.Disabled, b.Selected = ui.Bridge("ENDOWED", "BOUGHT"), false, true
		b.Tip = ui.Bridge("Graduates of this class can be hired", "Your Pods can run this image tag")
		b.ID = "vault:owned"
	}
	return b
}

func (h *HUD) pressAcademy(rest string, f Frame) []Intent {
	click := Intent{Kind: IntentSound, Command: "click"}
	kind, arg, _ := strings.Cut(rest, ":")
	v := h.academyView(f)
	h.takeOver(f)
	switch kind {
	case "filter":
		h.academy.filter, h.academy.repo = arg, ""
		h.list, h.academy.repoTop = 0, 0
	case "repo":
		h.academy.filter, h.academy.repo, h.academy.tab = v.Filter, arg, v.Tab
		h.academy.pick, h.list = "", 0
	case "tab":
		h.academy.tab, h.academy.repo, h.academy.filter = arg, selectedProfession(v), v.Filter
		h.list = 0
	case "pick":
		h.academy.pick = arg
		h.academy.tab, h.academy.repo, h.academy.filter = v.Tab, selectedProfession(v), v.Filter
	case "cmd":
		if _, command := h.academyCommand(f, v); command != "" {
			h.OpenStone(command)
		}
	}
	return []Intent{click}
}

func selectedProfession(v academyView) string {
	if r, ok := v.Selected(); ok {
		return r.Profession
	}
	return ""
}

func (h *HUD) academyWheel(in Input, f Frame, step int) {
	v := h.academyView(f)
	if in.MX < academyRight().X {
		h.academy.repoTop = min(max(h.academy.repoTop+step, 0), max(len(v.Repos)-academyReposVisible(), 0))
		return
	}
	h.list = min(max(h.list+step, 0), max(len(h.academyClasses(f, v))-academyVisible(), 0))
}

func ClassLines(s view.SoulstoneEntry, width int) []string {
	var font ui.Font
	room := ui.MaxChars(width-Literal.Span(s.Image, 2), 1)
	desc := s.Description
	if s.Cursed {
		desc = "CURSED: " + desc
	}
	return append([]string{ui.Clip(strings.ToUpper(s.Name), max(room, 8))}, font.WrapLines(desc, width, 1, 2)...)
}
