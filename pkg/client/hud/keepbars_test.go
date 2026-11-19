package hud

import (
	"image/color"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	towerCPU = 1900
	towerMem = 3584 << 20
)

func towerFrame(t *testing.T) (Frame, view.Structure) {
	t.Helper()
	f := fullFrame(t)
	f.Metrics.Nodes = map[string]view.NodeMetrics{view.TowerName: {CPUAllocatableMilli: towerCPU, CPURequestedMilli: 950, CPUMilli: 1140,
		MemoryAllocatableBytes: towerMem, MemoryRequestedBytes: 240 << 20, MemoryBytes: 896 << 20}}
	f.Nodes = map[string]model.NodeView{view.TowerName: {Name: view.TowerName, SystemCPUMilli: 850, SystemMemBytes: 240 << 20}}
	s := view.Structure{Kind: view.StructKeep, Label: "Lone Tower", State: view.StateReady}
	s.Object.Kind, s.Object.Name = "Node", view.TowerName
	return f, s
}

func TestKeepBarsShowTheControlPlaneShare(t *testing.T) {
	f, s := towerFrame(t)
	lines := KeepSupplyLines(s, f)
	if len(lines) != 3 {
		t.Fatalf("lines = %+v", lines)
	}
	cpu, mem, legend := lines[0], lines[1], lines[2]
	if !cpu.Bar || cpu.Text != "FOOD   " || cpu.Suffix != "1,140/1,900" || cpu.Value != 0.6 || cpu.reserved != 850.0/towerCPU {
		t.Errorf("cpu bar = %+v", cpu)
	}
	if !mem.Bar || mem.Text != "HOUSING" || mem.Suffix != "896/3,584" || mem.reserved != float64(240<<20)/towerMem || mem.reserveFill != cpu.reserveFill {
		t.Errorf("memory bar = %+v", mem)
	}
	if legend.Bar || legend.Text != "Citadel holds "+ui.IconFood+"850, "+ui.IconHousing+"240" || legend.Clr != cpu.reserveFill {
		t.Errorf("legend = %+v", legend)
	}
	card := StructureCard(s, f)
	if len(card.Lines) < 4 || card.Lines[1].Text != "FOOD   " || card.Lines[3].Text != legend.Text {
		t.Errorf("the Keep card leads with its bars: %+v", card.Lines)
	}
}

func TestKeepBarsWithoutASystemShareOrMetrics(t *testing.T) {
	f, s := towerFrame(t)
	f.Nodes = map[string]model.NodeView{}
	if lines := KeepSupplyLines(s, f); len(lines) != 2 || lines[0].reserved != 0 {
		t.Errorf("plain Keep = %+v", lines)
	}
	f.Metrics.Nodes = map[string]view.NodeMetrics{}
	if lines := KeepSupplyLines(s, f); len(lines) != 0 {
		t.Errorf("no metrics, no bars: %+v", lines)
	}
	s.State = view.StateConstruction
	f, _ = towerFrame(t)
	if lines := KeepSupplyLines(s, f); len(lines) != 0 {
		t.Errorf("a Keep under construction has no bars: %+v", lines)
	}
}

func TestKeepBarsAlignInBothNames(t *testing.T) {
	f, s := towerFrame(t)
	ui.SetNamer(strings.NewReplacer("FOOD", "CPUS").Replace)
	t.Cleanup(func() { ui.SetNamer(nil) })
	lines := KeepSupplyLines(s, f)
	if lines[0].Text != "CPU   " || lines[1].Text != "MEMORY" {
		t.Fatalf("labels = %q %q", lines[0].Text, lines[1].Text)
	}
	var font ui.Font
	if font.Width(lines[0].Text, 1) != font.Width(lines[1].Text, 1) {
		t.Errorf("bars start at different columns: %q %q", ui.Name(lines[0].Text), ui.Name(lines[1].Text))
	}
}

func TestQuantities(t *testing.T) {
	for in, want := range map[float64]string{0.95: "0.95", 1.9: "1.9", 1: "1.0", 0.234: "0.23", 12.5: "12.5", 0: "0.0", 3.999: "4.0"} {
		if got := Amount(in); got != want {
			t.Errorf("Amount(%v) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[int64]string{240 << 20: "240Mi", 0: "0Mi", 3 << 30: "3Gi", 3584 << 20: "3.5Gi"} {
		if got := MemQuantity(in); got != want {
			t.Errorf("MemQuantity(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestTopBarHandsMatchesDescribeNode(t *testing.T) {
	f, _ := towerFrame(t)
	hands, lore := SupplyOf(f.Metrics.Nodes)
	if got := SupplyCount(Rations(hands)); got != "950/1.9K" {
		t.Errorf("food = %q", got)
	}
	if got := SupplyCount(Rooms(lore)); got != "240/3.6K" {
		t.Errorf("housing = %q", got)
	}
	tip := HandsTip(f)
	if !strings.Contains(tip, "950m of 1900m") || !strings.Contains(tip, "Citadel") || !strings.Contains(tip, "850 rations") {
		t.Errorf("hands tip = %q", tip)
	}
	if tip := LoreTip(f); !strings.Contains(tip, "240Mi of 3584Mi") {
		t.Errorf("lore tip = %q", tip)
	}
}

func TestSplitBarDrawsTheReservedSegment(t *testing.T) {
	h := New(nil, Backend{})
	dst := ebiten.NewImage(120, 20)
	defer dst.Deallocate()
	r := ui.Rect{X: 10, Y: 5, W: 102, H: 7}
	l := Line{Bar: true, Value: .5, Fill: ui.Palette.Green, reserved: .25, reserveFill: ui.Palette.Purple}
	h.drawSplitBar(dst, r, l)
	h.drawSplitBar(dst, r, Line{Bar: true, Value: .5, Fill: ui.Palette.Green})
	if got := reservedRect(r, l.reserved); got != (ui.Rect{X: 11, Y: 6, W: 25, H: 5}) {
		t.Errorf("reserved segment = %+v", got)
	}
	if got := reservedRect(r, 3); got.W != r.W-2 {
		t.Errorf("an overfull share stays inside the bar: %+v", got)
	}
}

func TestPledgeColorsAndRatios(t *testing.T) {
	for share, want := range map[float64]color.RGBA{.5: ui.Palette.Green, .85: ui.Palette.GoldBright, 1.2: ui.Palette.Red} {
		if got := pledgeColor(share); got != want {
			t.Errorf("pledgeColor(%v) = %v", share, got)
		}
	}
	if ratio(5, 0) != 0 || ratio(1, 4) != .25 {
		t.Error("ratio")
	}
}

func TestKeepCardDrawsItsBars(t *testing.T) {
	f, s := towerFrame(t)
	f.Selection = Selection{Kind: "structure", ID: "tower", Structure: s}
	h := New(nil, Backend{})
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
	h.Stone.SetInput("get pods -n ")
	h.OverlayID = OverlayStone
	f.Vocabulary = spacedVocabulary
	h.Draw(dst, f)
	h.Stone.Tab(spacedVocabulary)
	h.Draw(dst, f)
}
