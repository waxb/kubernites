package soulstone

import "testing"

func TestEveryTradeWarmsUpAndOnlyNewerClassesFreeze(t *testing.T) {
	for _, trade := range TradeNames() {
		for i, tag := range ClassTags() {
			p, ok := ClassOf(trade, i)
			if !ok {
				t.Fatalf("%s:%s missing", trade, tag)
			}
			if p.WarmupSeconds <= 0 {
				t.Fatalf("%s:%s has no warm-up", trade, tag)
			}
			freezes := p.FreezeMTBFSeconds > 0
			if want := i >= ClassIndex("2.0"); freezes != want {
				t.Fatalf("%s:%s freezes=%v (every %vs), want %v", trade, tag, freezes, p.FreezeMTBFSeconds, want)
			}
		}
		cursed, _ := Lookup(trade + ":" + CursedTag)
		if cursed.FreezeMTBFSeconds != 0 {
			t.Fatalf("%s:%s is crash-looping and should crash, not freeze", trade, CursedTag)
		}
	}
	if clerk, _ := ClassOf("clerk", 0); clerk.WarmupSeconds <= mustClass(t, "porter", 0).WarmupSeconds {
		t.Fatal("clerks warm their ledgers longer than porters")
	}
	for _, image := range []string{ArchivistImage, "reeve:1.0"} {
		if p, ok := Lookup(image); ok && (p.WarmupSeconds != 0 || p.FreezeMTBFSeconds != 0) {
			t.Fatalf("%s is a system class and should not warm or freeze", image)
		}
	}
}

func TestNewerClassesFreezeMoreOften(t *testing.T) {
	two, three, latest := mustClass(t, "porter", ClassIndex("2.0")), mustClass(t, "porter", ClassIndex("3.0")), mustClass(t, "porter", ClassIndex(LatestTag))
	if !(two.FreezeMTBFSeconds > three.FreezeMTBFSeconds && three.FreezeMTBFSeconds > latest.FreezeMTBFSeconds) {
		t.Fatalf("freeze MTBF 2.0=%v 3.0=%v latest=%v", two.FreezeMTBFSeconds, three.FreezeMTBFSeconds, latest.FreezeMTBFSeconds)
	}
}

func mustClass(t *testing.T, trade string, class int) Profile {
	t.Helper()
	p, ok := ClassOf(trade, class)
	if !ok {
		t.Fatalf("%s class %d missing", trade, class)
	}
	return p
}
