package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestAChipKeepsTheImageInView(t *testing.T) {
	cmd := "kubectl set image deployment/porters porter=porter:1.1"
	if got := ChipFit(cmd, 60); got != ChipLabel(cmd) {
		t.Fatalf("a chip with room shows the whole command: %q", got)
	}
	got := ChipFit(cmd, 44)
	if !strings.HasSuffix(got, "porter=porter:1.1") || len(got) > 44 {
		t.Fatalf("a narrow chip keeps the image: %q", got)
	}
	if got := ChipFit("kubectl expose deployment clerks --name=ledgers --port=80 --target-port=8080", 44); got != "expose deploy clerks --name=ledgers ..." {
		t.Fatalf("trailing flags go first, the Service name stays: %q", got)
	}
	long := ChipFit("kubectl set image deployment/porters porter=porter:1.1 clerk=clerk:1.0 hauler=hauler:1.0", 40)
	if len([]rune(long)) > 40 || !strings.HasPrefix(long, "set image deploy/") || !strings.HasSuffix(long, "hauler:1.0") {
		t.Fatalf("a chip too long even shortened keeps its head and its tail: %q", long)
	}
}

func TestADrainChipDropsItsShrineFlagBeforeTheRealmsFlags(t *testing.T) {
	defer ui.SetNamer(nil)
	cmd := "kubectl drain stone-keep-1 --ignore-daemonsets --force"
	if got := ChipFit(cmd, 60); got != ChipLabel(cmd) {
		t.Fatalf("a drain chip with room shows the whole command: %q", got)
	}
	if got := ChipFit(cmd, 42); got != "drain stone-keep-1 ... --force" {
		t.Fatalf("a narrow drain chip hides the realm's flag: %q", got)
	}
	both := "kubectl drain stone-keep-1 --ignore-daemonsets --delete-emptydir-data --force"
	if got := ChipFit(both, 42); got != "drain stone-keep-1 ... --force" {
		t.Fatalf("a drain chip with both flags keeps the last that fits: %q", got)
	}
	if got := ChipFit(both, 60); got != "drain stone-keep-1 ... --delete-emptydir-data --force" {
		t.Fatalf("a drain chip with room for both realm flags = %q", got)
	}
	if got := ChipFit("kubectl drain stone-keep-1 --ignore-daemonsets", 30); got != "drain stone-keep-1 ..." {
		t.Fatalf("a plain drain chip keeps the general fit: %q", got)
	}
	ui.SetStoryNamer(lexicon.StoryText)
	if got := ChipFit(cmd, 42); got != "drain stone-keep-1 ... --force" {
		t.Fatalf("a narrow story drain chip hides the realm's flag: %q", got)
	}
}
