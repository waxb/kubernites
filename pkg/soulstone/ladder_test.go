package soulstone

import (
	"slices"
	"testing"
)

var ladderPrices = map[string][5]float64{
	"porter":     {0, 150, 3000, 5000, 27000},
	"clerk":      {200, 9000, 27000, 45000, 240000},
	"hauler":     {250, 30000, 90000, 150000, 810000},
	"courier":    {350, 330000, 990000, 1700000, 8900000},
	"sealwright": {500, 3300000, 9900000, 17000000, 89000000},
}

func TestEveryTradeClimbsFiveClassesAndKeepsOneCursedRelease(t *testing.T) {
	for _, trade := range TradeNames() {
		for i, tag := range ClassTags() {
			p, ok := Lookup(trade + ":" + tag)
			if !ok || p.Cursed || p.Trade != trade || p.Tag != tag {
				t.Fatalf("%s:%s = %+v, %v", trade, tag, p, ok)
			}
			if p.Cost != ladderPrices[trade][i] {
				t.Errorf("%s:%s costs %v, want %v", trade, tag, p.Cost, ladderPrices[trade][i])
			}
			if ClassIndex(tag) != i {
				t.Errorf("ClassIndex(%q) = %d, want %d", tag, ClassIndex(tag), i)
			}
		}
		cursed, ok := Lookup(trade + ":" + CursedTag)
		if !ok || !cursed.Cursed || cursed.CurseSeconds <= 0 || cursed.MTBFSeconds != 0 || cursed.Trade != trade {
			t.Fatalf("%s:%s does not lie: %+v", trade, CursedTag, cursed)
		}
	}
}

func TestEachClassDoublesTheBoxesPerLaborAfterTheReliableOne(t *testing.T) {
	for _, trade := range TradeNames() {
		first, _ := Lookup(trade + ":1.0")
		want := []float64{1, 1, 2, 4, 8}
		for i, tag := range ClassTags() {
			p, _ := Lookup(trade + ":" + tag)
			for size := SizeParcel; size < SizeClasses; size++ {
				if got := first.BoxCPU(size) / p.BoxCPU(size); got < want[i]*0.999 || got > want[i]*1.001 {
					t.Fatalf("%s:%s carries x%.3f of 1.0 at size %d, want x%v", trade, tag, got, size, want[i])
				}
			}
			if p.ThreadMilli != first.ThreadMilli || p.MemBaseMi != first.MemBaseMi {
				t.Fatalf("%s:%s changes its land: %+v", trade, tag, p)
			}
		}
		reliable, _ := Lookup(trade + ":1.1")
		if first.MTBFSeconds != baseMTBF || reliable.MTBFSeconds != 5*first.MTBFSeconds {
			t.Fatalf("%s: 1.0 falls every %vs and 1.1 every %vs, want %v and five times that", trade, first.MTBFSeconds, reliable.MTBFSeconds, baseMTBF)
		}
		cursed, _ := Lookup(trade + ":" + CursedTag)
		two, _ := Lookup(trade + ":2.0")
		if cursed.BoxCPU(SizeParcel) != two.BoxCPU(SizeParcel) {
			t.Fatalf("%s:%s does not promise the 2.0 pace", trade, CursedTag)
		}
	}
}

func TestLatestIsOnlyANameForTheFinalClass(t *testing.T) {
	for _, trade := range TradeNames() {
		for _, ref := range []string{trade, trade + ":latest", RegistryHost + "/" + trade + ":latest"} {
			p, ok := Lookup(ref)
			if !ok || p.Image != trade+":latest" || p.Cursed || p.Tag != LatestTag {
				t.Fatalf("Lookup(%q) = %q crash-looping %v, want %s:latest", ref, p.Image, p.Cursed, trade)
			}
		}
	}
}

func TestTagNewerSortsLatestAfterEveryNumberedTag(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"latest", "3.0", true}, {"3.0", "latest", false}, {"latest", "latest", false},
		{"", "3.0", true}, {"2.1", "2.0", true}, {"3.0", "2.1", true}, {"1.10", "1.9", true},
		{"1.0", "1.0", false}, {"2.0", "1.1", true},
	}
	for _, c := range cases {
		if got := TagNewer(c.a, c.b); got != c.want {
			t.Errorf("TagNewer(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
	tags := []string{"latest", "1.0", "2.1", "3.0", "1.1", "2.0"}
	slices.SortFunc(tags, func(a, b string) int {
		switch {
		case TagNewer(a, b):
			return 1
		case TagNewer(b, a):
			return -1
		}
		return 0
	})
	if want := []string{"1.0", "1.1", "2.0", "2.1", "3.0", "latest"}; !slices.Equal(tags, want) {
		t.Fatalf("sorted tags = %v, want %v", tags, want)
	}
}

func TestHigherTradesPledgeTheirOwnLand(t *testing.T) {
	cases := map[string][2]string{
		"porter": {"", ""}, "clerk": {"", ""},
		"hauler": {"1", "1Gi"}, "courier": {"1", "256Mi"}, "sealwright": {"2", "512Mi"},
	}
	for trade, want := range cases {
		for _, tag := range append(ClassTags(), CursedTag) {
			p, _ := Lookup(trade + ":" + tag)
			if p.PledgeCPU != want[0] || p.PledgeMemory != want[1] {
				t.Fatalf("%s:%s pledges %q %q, want %v", trade, tag, p.PledgeCPU, p.PledgeMemory, want)
			}
		}
	}
}

func TestClassOfFindsTheTradesClassByIndex(t *testing.T) {
	p, ok := ClassOf("clerk", 3)
	if !ok || p.Image != "clerk:3.0" {
		t.Fatalf("ClassOf(clerk, 3) = %q, %v", p.Image, ok)
	}
	if _, ok := ClassOf("clerk", 5); ok {
		t.Fatal("there is no sixth class")
	}
	if ClassIndex("2.1") != -1 || ClassIndex("9.9") != -1 {
		t.Fatal("the crash-looping release and unknown tags are not rungs of the ladder")
	}
}
