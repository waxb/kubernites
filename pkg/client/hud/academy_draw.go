package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	academySubtitle = "Across the sea. Each trade climbs through its classes: endow a class once, and every hut of that trade can take it."
	academyPriceW   = 74
)

func (h *HUD) DrawAcademy(dst *ebiten.Image, f Frame) {
	ui.DarkPanel(dst, Overlay)
	h.header(dst, Overlay, "THE ACADEMY")
	h.Font.Draw(dst, academySubtitle, Overlay.X+10, Overlay.Y+28, 1, ui.Palette.TextDim)
	v := h.academyView(f)
	h.Font.Draw(dst, "FILTER", Overlay.X+10, Overlay.Y+academyTop+3, 1, ui.Palette.TextDim)
	h.drawRepoList(dst, v)
	h.drawRepoPage(dst, f, v)
	for _, b := range h.AcademyButtons(f) {
		h.drawAcademyButton(dst, b)
	}
}

func (h *HUD) drawAcademyButton(dst *ebiten.Image, b Button) {
	prefix, rest, _ := strings.Cut(b.ID, ":")
	kind, _, _ := strings.Cut(rest, ":")
	switch {
	case prefix == acadPrefix && (kind == "repo" || kind == "pick"):
	case b.ID == acadPrefix+":cmd":
		h.drawCommandChip(dst, b.Rect, b.Label, h.Hovering && h.Hover.ID == b.ID)
	default:
		h.drawButton(dst, b)
	}
}

func (h *HUD) drawRepoList(dst *ebiten.Image, v academyView) {
	if len(v.Repos) == 0 {
		h.Font.Draw(dst, "None met yet.", academyLeft().X+4, academyLeft().Y+4, 1, ui.Palette.TextDim)
		return
	}
	top := h.academyReposStart(v)
	for i := range min(academyReposVisible(), len(v.Repos)-top) {
		h.drawRepoCard(dst, academyCardRect(i), v.Repos[top+i], top+i == v.Sel)
	}
	if len(v.Repos) > academyReposVisible() {
		l := academyLeft()
		more := fmt.Sprintf("%d-%d OF %d", top+1, min(top+academyReposVisible(), len(v.Repos)), len(v.Repos))
		h.Font.Draw(dst, more, l.X+l.W-h.Font.Width(more, 1)-2, Overlay.Y+academyTop+3, 1, ui.Palette.TextDim)
	}
}

func (h *HUD) drawRepoCard(dst *ebiten.Image, r ui.Rect, repo academyRepo, selected bool) {
	ui.Well(dst, r)
	hot := h.Hovering && h.Hover.ID == acadPrefix+":repo:"+repo.Profession
	switch {
	case selected:
		ui.Stroke(dst, r, ui.Palette.Gold)
	case hot:
		ui.Stroke(dst, r, ui.Palette.GoldBright)
	}
	h.drawRepoPortrait(dst, ui.Rect{X: r.X + 4, Y: r.Y + 7, W: 30, H: 30}, repo)
	x := r.X + 40
	h.Font.Draw(dst, repo.Name, x, r.Y+5, 1, ui.Palette.Gold)
	count := countClasses(len(repo.Classes))
	h.Font.Draw(dst, count, r.X+r.W-h.Font.Width(count, 1)-6, r.Y+5, 1, ui.Palette.TextDim)
	h.Font.Draw(dst, ui.Clip(soulstone.Tagline(repo.Classes[0].Image), ui.MaxChars(r.W-46, 1)), x, r.Y+17, 1, ui.Palette.Text)
	if repo.Familiar {
		x += h.drawChipBadge(dst, x, r.Y+29, ui.Bridge("FAMILIAR", "SIDECAR"), ui.Palette.Blue) + 6
	}
	h.Font.Draw(dst, fmt.Sprintf("%d ENDOWED", repo.Endowed), x, r.Y+30, 1, endowedColour(repo))
}

func endowedColour(repo academyRepo) color.RGBA {
	if repo.Endowed > 0 {
		return ui.Palette.Green
	}
	return ui.Palette.TextDim
}

func countClasses(n int) string {
	if n == 1 {
		return "1 CLASS"
	}
	return fmt.Sprintf("%d CLASSES", n)
}

func (h *HUD) drawChipBadge(dst *ebiten.Image, x, y int, text string, clr color.RGBA) int {
	w := Literal.Width(text, 1) + 8
	r := ui.Rect{X: x, Y: y, W: w, H: 11}
	ui.Stroke(dst, r, ui.Palette.ChipBorder)
	Literal.Draw(dst, text, x+4, y+1, 1, clr)
	return w
}

func (h *HUD) drawRepoPortrait(dst *ebiten.Image, r ui.Rect, repo academyRepo) {
	_, sprite := soulstone.ProfessionOf(repo.Classes[0].Image)
	if !h.Lib.Portrait(dst, sprite, r) {
		h.icon(dst, "icon/soulstone", r.X+r.W/2-8, r.Y+r.H/2-8)
	}
}

func (h *HUD) drawRepoPage(dst *ebiten.Image, f Frame, v academyView) {
	ui.Well(dst, academyRight())
	repo, ok := v.Selected()
	if !ok {
		in := academyInner()
		h.Font.Draw(dst, "Nothing here yet: the Academy sends news of its familiars when you learn them.", in.X, in.Y+4, 1, ui.Palette.TextDim)
		return
	}
	in := academyInner()
	h.drawRepoPortrait(dst, ui.Rect{X: in.X, Y: in.Y + 2, W: 40, H: 40}, repo)
	x := in.X + 48
	h.Font.DrawShadow(dst, repo.Name, x, in.Y+2, 2, ui.Palette.Gold)
	Literal.Draw(dst, repo.Repository, x+h.Font.Width(repo.Name, 2)+8, in.Y+6, 1, ui.Palette.Blue)
	kind := ui.Bridge("TRADE", "IMAGE")
	if repo.Familiar {
		kind = ui.Bridge("FAMILIAR", "SIDECAR")
	}
	w := h.drawChipBadge(dst, x, in.Y+22, kind, ui.Palette.Blue)
	h.Font.Draw(dst, fmt.Sprintf("%s, %d ENDOWED", countClasses(len(repo.Classes)), repo.Endowed), x+w+8, in.Y+23, 1, ui.Palette.TextDim)
	h.Font.Draw(dst, ui.Clip(soulstone.Tagline(repo.Classes[0].Image), ui.MaxChars(in.W-48, 1)), x, in.Y+35, 1, ui.Palette.Text)
	ui.Fill(dst, ui.Rect{X: in.X, Y: in.Y + academyHead + academyTabH, W: in.W, H: 1}, ui.Palette.Gold)
	if v.Tab == acadTabOverview {
		h.drawRepoOverview(dst, repo, in)
		return
	}
	h.drawClassTable(dst, f, v, repo, in)
}

func (h *HUD) drawClassTable(dst *ebiten.Image, f Frame, v academyView, repo academyRepo, in ui.Rect) {
	y := in.Y + academyBody
	h.Font.Draw(dst, ui.Bridge("CLASS", "TAG"), in.X+4, y, 1, ui.Palette.TextDim)
	h.Font.Draw(dst, "PRICE", in.X+in.W-68-4-h.Font.Width("PRICE", 1), y, 1, ui.Palette.TextDim)
	start := h.academyStart(f, repo.Classes)
	if len(repo.Classes) > academyVisible() {
		more := fmt.Sprintf("%d-%d OF %d", start+1, min(start+academyVisible(), len(repo.Classes)), len(repo.Classes))
		h.Font.Draw(dst, more, in.X+in.W-h.Font.Width(more, 1)-4, y, 1, ui.Palette.TextDim)
	}
	picked, _ := h.pickedClass(f, repo)
	for i, s := range repo.Classes[start:min(len(repo.Classes), start+academyVisible())] {
		r := AcademyRowRect(i)
		h.drawClass(dst, r, s, AcademyPayoff(f, s))
		if s.Image == picked.Image {
			ui.Stroke(dst, r, ui.Palette.Gold)
		}
	}
	label := ui.Bridge("SET ON A CREW", "SET THE IMAGE")
	if foremenOnly(f) {
		label = ui.Bridge("SET ON A SHIFT", "SET THE IMAGE")
	}
	if repo.Familiar {
		label = ui.Bridge("BIND TO A CREW", "ADD A SIDECAR")
	}
	cmd := academyCmdRect()
	h.Font.Draw(dst, label, cmd.X+4, cmd.Y+(cmd.H-9)/2, 1, ui.Palette.GoldBright)
	if !picked.Owned {
		chip := academyChipRect2()
		h.Font.Draw(dst, ui.Bridge("ENDOW THE CLASS FIRST", "BUY THE IMAGE TAG FIRST"), chip.X+4, chip.Y+(chip.H-9)/2, 1, ui.Palette.TextDim)
	}
}

func (h *HUD) drawClass(dst *ebiten.Image, r ui.Rect, s view.SoulstoneEntry, payoff string) {
	ui.Well(dst, r)
	_, sprite := soulstone.ProfessionOf(s.Image)
	if !h.Lib.Portrait(dst, sprite, ui.Rect{X: r.X + 3, Y: r.Y + 2, W: 30, H: 30}) {
		h.icon(dst, "icon/soulstone", r.X+10, r.Y+9)
	}
	width := r.W - AcademyTextPad
	lines := ClassLines(s, width)
	if payoff != "" {
		lines = lines[:min(len(lines), 2)]
	}
	h.Font.Draw(dst, lines[0], r.X+40, r.Y+3, 1, ui.Palette.Gold)
	Literal.Draw(dst, s.Image, r.X+40+h.Font.Span(lines[0], 2), r.Y+3, 1, ui.Palette.Blue)
	clr := ui.Palette.Text
	if s.Cursed {
		clr = ui.Palette.RedText
	}
	for i, row := range lines[1:] {
		h.Font.Draw(dst, row, r.X+40, r.Y+14+i*ui.LineHeight, 1, clr)
	}
	if payoff != "" {
		drawClipped(dst, payoff, r.X+40, r.Y+14+(len(lines)-1)*ui.LineHeight, width, ui.Palette.Green)
	}
	h.drawPrice(dst, r, s)
}

func (h *HUD) drawPrice(dst *ebiten.Image, r ui.Rect, s view.SoulstoneEntry) {
	text, clr := Gold(s.Cost), ui.Palette.GoldBright
	switch {
	case rumoured(s):
		text, clr = ui.Bridge("FREE", "FREE"), ui.Palette.TextDim
	case s.Owned:
		text, clr = "-", ui.Palette.TextDim
	case s.Cost <= 0:
		text = "FREE"
	}
	x := r.X + r.W - 68 - 4 - academyPriceW
	h.Font.Draw(dst, text, x+academyPriceW-h.Font.Width(text, 1), r.Y+(r.H-9)/2, 1, clr)
}

func (h *HUD) drawRepoOverview(dst *ebiten.Image, repo academyRepo, in ui.Rect) {
	y := in.Y + academyBody
	blurb := repo.Classes[0].Description
	if blurb == "" {
		blurb = soulstone.Tagline(repo.Classes[0].Image)
	}
	for _, row := range h.Font.WrapLines(blurb, in.W-8, 1, 4) {
		h.Font.Draw(dst, row, in.X+4, y, 1, ui.Palette.Text)
		y += ui.LineHeight
	}
	y += 6
	h.Font.DrawShadow(dst, ui.Bridge("THE CLASSES", "THE TAGS"), in.X+4, y, 1, ui.Palette.Gold)
	y += ui.LineHeight + 2
	for _, row := range h.Font.WrapLines(classLadder(repo), in.W-8, 1, 3) {
		Literal.Draw(dst, row, in.X+4, y, 1, ui.Palette.Blue)
		y += ui.LineHeight
	}
	y += 6
	h.Font.DrawShadow(dst, ui.Bridge("HOW IT WORKS", "IN KUBERNETES"), in.X+4, y, 1, ui.Palette.Gold)
	y += ui.LineHeight + 2
	for _, row := range h.Font.WrapLines(repoLesson(repo), in.W-8, 1, 5) {
		h.Font.Draw(dst, row, in.X+4, y, 1, ui.Palette.Text)
		y += ui.LineHeight
	}
}

func classLadder(repo academyRepo) string {
	parts := []string{}
	for _, c := range repo.Classes {
		_, tag, _ := strings.Cut(c.Image, ":")
		state := Compact(c.Cost)
		switch {
		case rumoured(c):
			state = "rumour"
		case c.Owned:
			state = "endowed"
		}
		parts = append(parts, tag+" "+state)
	}
	return strings.Join(parts, "  ")
}

func repoLesson(repo academyRepo) string {
	if repo.Familiar {
		return ui.Bridge("A familiar rides beside a worker and carries no boxes of its own. Bind it to a crew and the crew rolls out new huts with it inside.",
			"A sidecar is an init container with restartPolicy: Always. It starts before the main container and runs beside it for the Pod's whole life.")
	}
	return ui.Bridge("A trade is a kind of worker and a class is its tag. A crew raised on a class works as that class; set a better class on the crew and its huts roll out, a few at a time.",
		repo.Repository+" is an image and its class a tag. A Deployment names image:tag; changing it rolls out new Pods, a few at a time, and each node pulls the image first.")
}
