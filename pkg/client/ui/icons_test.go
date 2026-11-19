package ui

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func iconed(icon, word string) string { return icon + iconGap + word }

func TestResourceWordsGetTheirIconBeforeThem(t *testing.T) {
	cases := map[string]string{
		"serves 150 cpu":                      "serves 150 " + iconed(IconFood, "cpu"),
		"CPU and CPUs":                        iconed(IconFood, "CPU") + " and " + iconed(IconFood, "CPUs"),
		"memory, 64 rooms.":                   iconed(IconHousing, "memory") + ", 64 " + iconed(IconHousing, "rooms") + ".",
		"food in rations, 1 ration, CPU":      iconed(IconFood, "food") + " in " + iconed(IconFood, "rations") + ", 1 " + iconed(IconFood, "ration") + ", " + iconed(IconFood, "CPU"),
		"(housing, 64 rooms, 1 room, memory)": "(" + iconed(IconHousing, "housing") + ", 64 " + iconed(IconHousing, "rooms") + ", 1 " + iconed(IconHousing, "room") + ", " + iconed(IconHousing, "memory") + ")",
		"Renown and Reputation":               iconed(IconRenown, "Renown") + " and " + iconed(IconRenown, "Reputation"),
	}
	for in, want := range cases {
		if got := Decorate(in); got != want {
			t.Errorf("Decorate(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIdentifiersKeepTheirWords(t *testing.T) {
	for _, in := range []string{
		"requests.cpu", "limits.memory", "--requests=cpu=200m", "kubectl top pods --sort-by=memory", "cpu_usage",
		"golden", "goldsmith", "memory.limit", "a/cpu", "$cpu", "Goldie", "Next plot: memory-xlarge", "the plots of land", "no Keep has room", "NO ROOM", "room for 15 porters",
	} {
		if got := Decorate(in); got != in {
			t.Errorf("Decorate(%q) = %q, want it untouched", in, got)
		}
	}
}

func TestAQuantityWithAUnitKeepsItsWord(t *testing.T) {
	if got, want := Decorate("asks 750m CPU and 64Mi memory"), "asks "+iconed(IconFood, "750m CPU")+" and "+iconed(IconHousing, "64Mi memory"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDecoratingTwiceChangesNothing(t *testing.T) {
	for _, in := range []string{"150 cpu and 2 rooms", "750m CPU", "food: 200 rations"} {
		once := Decorate(in)
		if twice := Decorate(once); twice != once {
			t.Errorf("Decorate twice changed %q into %q", once, twice)
		}
	}
}

func TestIconsCountAsCellsInEveryWidth(t *testing.T) {
	var f Font
	if w, want := f.Width("cpu", 1), cells(Decorate("cpu"))*Advance-1; w != want || w != 5*Advance-1 {
		t.Fatalf("width of cpu = %d, want %d", w, want)
	}
	if w := (Font{Literal: true}).Width("cpu", 1); w != 3*Advance-1 {
		t.Fatalf("a literal font decorated: width %d", w)
	}
}

func TestWrapKeepsAnIconWithItsWord(t *testing.T) {
	var f Font
	for _, line := range f.Wrap("the crew gives some cpu to you", 20*Advance, 1) {
		runes := []rune(line)
		last := runes[len(runes)-1]
		if iconRune(last) {
			t.Fatalf("a line ends on an icon: %q", line)
		}
	}
}

func TestClipNeverSplitsAnIcon(t *testing.T) {
	var f Font
	text := "x " + Decorate("cpu")
	for n := 1; n <= len([]rune(text)); n++ {
		runes := []rune(f.Clip(text, n))
		for i, r := range runes {
			if iconStart(r) && (i+1 >= len(runes) || !iconRune(runes[i+1])) {
				t.Fatalf("clip %d split an icon: %q", n, string(runes))
			}
		}
	}
}

var allIcons = []string{IconGold, IconWisdom, IconFood, IconHousing, IconRenown, IconBox, IconParcel, IconLedger, IconCrate, IconDispatch, IconSealed}

func TestExportedIconsAreTwoCellsWide(t *testing.T) {
	for _, icon := range allIcons {
		if n := len([]rune(icon)); n != 2 {
			t.Fatalf("icon %q has %d runes", icon, n)
		}
		if w := (Font{}).Width(icon+"150", 1); w != 5*Advance-1 {
			t.Fatalf("a cost with its icon measures %d", w)
		}
	}
}

func TestIconsDrawInTheirOwnColours(t *testing.T) {
	dst := ebiten.NewImage(40, 20)
	defer dst.Deallocate()
	drawText(dst, IconGold, 0, 0, 1, color.RGBA{255, 255, 255, 255})
	for _, icon := range allIcons {
		inked := 0
		for y := 0; y < iconRows; y++ {
			for x := 0; x < iconWidth; x++ {
				if c := iconPixel(icon, x, y); c != nil && c != (color.RGBA{255, 255, 255, 255}) {
					inked++
				}
			}
		}
		if inked < 20 {
			t.Fatalf("icon %q has %d coloured pixels", icon, inked)
		}
	}
}

func TestBoxWordsGetTheirBoxBeforeThem(t *testing.T) {
	cases := map[string]string{
		"Carry 5 parcels from the quay":                                   "Carry 5 " + iconed(IconParcel, "parcels") + " from the quay",
		"LEDGERS, crates and dispatches":                                  iconed(IconLedger, "LEDGERS") + ", " + iconed(IconCrate, "crates") + " and " + iconed(IconDispatch, "dispatches"),
		"2 sealed boxes wait; boxes spoil":                                "2 " + iconed(IconSealed, "sealed boxes") + " wait; " + iconed(IconBox, "boxes") + " spoil",
		"fresh boxes":                                                     "fresh " + iconed(IconBox, "boxes"),
		"1 ledger, 1 crate, 1 box, 1 parcel":                              "1 " + iconed(IconLedger, "ledger") + ", 1 " + iconed(IconCrate, "crate") + ", 1 " + iconed(IconBox, "box") + ", 1 " + iconed(IconParcel, "parcel"),
		"Speak it: speakstctl describe depot parcels. Then carry parcels": "Speak it: speakstctl describe depot parcels. Then carry " + iconed(IconParcel, "parcels"),
	}
	for in, want := range cases {
		if got := Decorate(in); got != want {
			t.Errorf("Decorate(%q) =\n%q, want\n%q", in, got, want)
		}
	}
	for _, in := range []string{
		"Open the Ledger", "the ledger shows overhead", "a sealed letter", "$ kubectl expose deployment clerks --name=ledgers --port=80",
		"kubectl get svc parcels", "$ etcdctl get /registry/services/default/crates", "deploy/parcels", "porters-parcels",
	} {
		if got := Decorate(in); got != in {
			t.Errorf("Decorate(%q) = %q, want it untouched", in, got)
		}
	}
}

func TestQuotedTextKeepsItsWords(t *testing.T) {
	got := Decorate(`A request reserves CPU: "cpu limits are enforced by CPU throttling." CPU is compressible.`)
	want := "A request reserves " + iconed(IconFood, "CPU") + `: "cpu limits are enforced by CPU throttling." ` + iconed(IconFood, "CPU") + " is compressible."
	if got != want {
		t.Fatalf("quoted text decorated:\n%q\nwant\n%q", got, want)
	}
}

func TestIconLeadCountsTheIconsCells(t *testing.T) {
	if n := IconLead(Decorate("Food")); n != 2 {
		t.Fatalf("lead %d", n)
	}
	if n := IconLead("Food"); n != 0 {
		t.Fatalf("plain lead %d", n)
	}
	if n := IconLead(IconGold + "150"); n != 2 {
		t.Fatalf("cost lead %d", n)
	}
}

func TestPlainTextDropsIconsAndTheirGaps(t *testing.T) {
	if got := Plain(Decorate("Carry 5 parcels for 150 cpu")); got != "Carry 5 parcels for 150 cpu" {
		t.Fatalf("plain = %q", got)
	}
}

func TestSpelledTextNamesEachIcon(t *testing.T) {
	if got := Spelled(Decorate("Carry 5 parcels for 150 cpu") + " " + IconFood + "200"); got != "Carry 5 [parcel] parcels for 150 [food] cpu [food] 200" {
		t.Fatalf("spelled = %q", got)
	}
}

func TestTheWordSitsRightAfterItsIcon(t *testing.T) {
	dst := ebiten.NewImage(80, 20)
	defer dst.Deallocate()
	var runs []TextRun
	stop := RecordText(func(r TextRun) { runs = append(runs, r) })
	Font{}.Draw(dst, "cpu", 0, 0, 1, Palette.Text)
	stop()
	if len(runs) != 1 || runs[0].Rect.W != 5*Advance-1 {
		t.Fatalf("the icon and its word span %+v", runs)
	}
	for n := 1; n <= 12; n++ {
		clipped := []rune(Clip("ask the cpu now", n))
		for i, r := range clipped {
			if iconStart(r) && (i+2 >= len(clipped) || clipped[i+2] == '.') {
				t.Fatalf("clip %d leaves an icon without its word: %q", n, string(clipped))
			}
		}
	}
}

func TestBarrelsWearTheCellarsIcon(t *testing.T) {
	if got, want := Decorate("2,048 barrels, 1 barrel"), "2,048 "+iconed(IconCellar, "barrels")+", 1 "+iconed(IconCellar, "barrel"); got != want {
		t.Errorf("Decorate = %q, want %q", got, want)
	}
}
