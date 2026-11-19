package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func keepsFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Reveals = append(append([]string(nil), allReveals...), "keeps")
	f.State.Buildables = []view.Buildable{
		{Kind: "keep", Name: "Keep", K8s: "Node (a machine running a kubelet)", Description: "Land and walls.", Cost: 150, W: 3, H: 3, MinPower: 1, Available: true},
		{Kind: "great-keep", Name: "Great Keep", K8s: "Node (instance type great-keep: 8 cpu, 16Gi)", Description: "Eight Hands.", Cost: 4000, W: 3, H: 3, MinPower: 2, Reason: "Read about Great Keeps in the manual"},
		{Kind: "citadel-keep", Name: "Citadel Keep", K8s: "Node (instance type citadel-keep: 16 cpu, 32Gi)", Description: "Sixteen Hands.", Cost: 40000, W: 3, H: 3, MinPower: 3, Reason: "Needs 40000 MiB"},
		{Kind: "codex", Name: "The Codex", K8s: "etcd", Cost: 10, W: 2, H: 2, Reason: "Already raised"},
		{Kind: "chancery", Name: "The Chancery", K8s: "kube-apiserver", Cost: 15, W: 2, H: 2, Reason: "Already raised"},
		{Kind: "archive", Name: "The Archive", K8s: "Service (ledgers)", Cost: 300, W: 3, H: 3, Available: true},
	}
	f.Site.Structures = append(append([]view.Structure(nil), f.Site.Structures...),
		view.Structure{ID: "keep-1", Kind: view.StructKeep, Variant: "tower", X: 20, Y: 20, W: 3, H: 3, State: view.StateReady, Label: "Lone Tower"},
		view.Structure{ID: "codex-1", Kind: view.StructCodex, X: 24, Y: 20, W: 2, H: 2, State: view.StateReady, Label: "The Codex"},
		view.Structure{ID: "chancery-1", Kind: view.StructChancery, X: 24, Y: 23, W: 2, H: 2, State: view.StateReady, Label: "The Chancery"})
	return f
}

func buildIDs(h *HUD, f Frame) []string {
	h.PageID = PageBuild
	out := []string{}
	for _, b := range h.CommandButtons(f) {
		out = append(out, b.ID)
	}
	return out
}

func TestBuildPageOffersKeepSizesAndHidesWhatStands(t *testing.T) {
	h := New(nil, Backend{})
	f := keepsFrame(t)
	ids := strings.Join(buildIDs(h, f), " ")
	for _, want := range []string{"build:keep", "build:great-keep", "build:citadel-keep", "build:archive", "cmd:back"} {
		if !strings.Contains(ids, want) {
			t.Errorf("build page lacks %s: %s", want, ids)
		}
	}
	for _, gone := range []string{"build:codex", "build:chancery"} {
		if strings.Contains(ids, gone) {
			t.Errorf("a raised building is still offered: %s", ids)
		}
	}
	early := keepsFrame(t)
	early.State.Reveals = allReveals
	if ids := strings.Join(buildIDs(h, early), " "); strings.Contains(ids, "great-keep") {
		t.Errorf("keep sizes wait for the keeps reveal: %s", ids)
	}
}

func TestKeepButtonsCarryCostSpecAndPower(t *testing.T) {
	h := New(nil, Backend{})
	f := keepsFrame(t)
	var great, stone Button
	for _, b := range h.BuildButtons(f) {
		switch b.ID {
		case "build:great-keep":
			great = b
		case "build:keep":
			stone = b
		}
	}
	if great.Label != "GREAT KEEP" || great.Cost != "4K" || great.Pips != 0 || !great.Disabled {
		t.Errorf("great keep = %+v", great)
	}
	for _, want := range []string{"8,000 rations", "16,384 rooms", "Read about Great Keeps"} {
		if !strings.Contains(great.Tip, want) {
			t.Errorf("great keep tip %q lacks %q", great.Tip, want)
		}
	}
	if strings.Contains(great.Tip, "ley power") {
		t.Errorf("the retired ley power is still asked: %q", great.Tip)
	}
	if stone.Label != "STONE KEEP" || stone.Icon != gfx.KeepPreview("stone", "", gfx.AgeTier(f.State.Age)).BuildIconName() || stone.Pips != 0 || !strings.Contains(stone.Tip, "4,000 rations") {
		t.Errorf("stone keep after the tower = %+v", stone)
	}
	for _, b := range h.BuildButtons(f) {
		if len(b.Label) > ui.MaxChars(CellW-4, 1) {
			t.Errorf("label %q does not fit a cell", b.Label)
		}
	}
	first := fullFrame(t)
	for _, b := range h.BuildButtons(first) {
		if b.ID == "build:keep" && (b.Label != "KEEP" || b.Icon != gfx.KeepPreview("tower", "", gfx.AgeTier(first.State.Age)).BuildIconName() || b.Pips != 0) {
			t.Errorf("the first keep is the Lone Tower: %+v", b)
		}
	}
}

func TestKeepSpec(t *testing.T) {
	cases := map[string]string{"keep": "4,000 rations, 8,192 rooms", "great-keep": "8,000 rations, 16,384 rooms", "citadel-keep": "16,000 rations, 32,768 rooms", "codex": ""}
	for kind, want := range cases {
		if got := KeepSpec(kind); got != want {
			t.Errorf("KeepSpec(%s) = %q, want %q", kind, got, want)
		}
	}
	if !IsKeepKind("keep") || !IsKeepKind("citadel-keep") || IsKeepKind("codex") {
		t.Error("keep kinds")
	}
}

func TestKeepCellsDraw(t *testing.T) {
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h := New(nil, Backend{})
	f := keepsFrame(t)
	h.PageID = PageBuild
	h.Draw(dst, f)
}

func TestAcademyClassesWrapTheirDescriptions(t *testing.T) {
	width := AcademyRowRect(0).W - AcademyTextPad
	s := view.SoulstoneEntry{Image: "raven:1.0", Name: "Ravens, class of 1.0",
		Description: "A familiar that rides beside a hut as a native sidecar: it carries every served request's tale to the Archive, and it restarts on its own when it falls, and it asks for a little Lore of its own on every Keep it visits, which adds up.", Cursed: true}
	lines := ClassLines(s, width)
	if len(lines) != 3 || !strings.HasPrefix(lines[1], "CURSED: ") || !strings.HasSuffix(lines[2], "...") {
		t.Errorf("lines = %q", lines)
	}
	for _, l := range lines {
		if w, _ := (ui.Font{}).Measure(l, 1); w > width {
			t.Errorf("%q is %dpx, room %d", l, w, width)
		}
	}
	short := ClassLines(view.SoulstoneEntry{Image: "hound:1.0", Name: "Hounds", Description: "A proxy."}, width)
	if len(short) != 2 {
		t.Errorf("short = %q", short)
	}
}
