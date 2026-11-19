package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestStoryDescribeFoldsItsLongBlocksBehindMore(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	lines := StoryConsole(SplitOutput(describeNodeRaw))
	kept, folded := foldSheet(lines)
	joined := strings.Join(kept, "\n")
	if strings.Contains(joined, "Reeve is posting ready status") || strings.Contains(joined, "KeepReady") {
		t.Fatalf("the conditions and tidings stay open:\n%s", joined)
	}
	if !strings.Contains(joined, "Folded away: Conditions, Tidings. Say  more  to read them.") || !strings.Contains(joined, "Wards:") || !strings.Contains(joined, "Capacity:") {
		t.Fatalf("the fold lost the head of the sheet:\n%s", joined)
	}
	if got := strings.Join(folded, "\n"); !strings.Contains(got, "Reeve is posting ready status") {
		t.Fatalf("more has nothing to unfold:\n%s", got)
	}
	short, none := foldSheet([]string{"Name: x", "Conditions:", "  a", "  b"})
	if none != nil || len(short) != 4 {
		t.Fatalf("a short block stays open: %v %v", short, none)
	}
}

func TestStoryDescribeFoldsStandingAndCharms(t *testing.T) {
	lines := []string{
		"Name:        porters-vbm9v",
		"Status:      Unhoused",
		"Standing Class:  Freeman",
		"Charms:          the Unready Ward (turns huts out, for 300s)",
		"                 the Storm Ward (turns huts out, for 300s)",
		"Tidings:",
		"  Type      Reason",
		"  ----      ------",
		"  Warning   NoKeepHasRoom",
	}
	kept, folded := foldSheet(lines)
	joined := strings.Join(kept, "\n")
	if strings.Contains(joined, "Freeman") || strings.Contains(joined, "Storm Ward") {
		t.Fatalf("Standing Class and Charms stay open:\n%s", joined)
	}
	if !strings.Contains(joined, "Folded away: Standing Class, Charms, Tidings.") {
		t.Fatalf("the fold note = %q", joined)
	}
	if got := strings.Join(folded, "\n"); !strings.Contains(got, "Storm Ward") {
		t.Fatalf("more has no Charms to unfold:\n%s", got)
	}
}

const describeTowerHuts = `Name:               lone-tower
Roles:              control-plane
Taints:             node-role.kubernetes.io/control-plane:NoSchedule
Unschedulable:      false
Capacity:
  cpu:     2
  memory:  4Gi
Allocatable:
  cpu:     1900m
  memory:  3584Mi
Non-terminated Pods:          (3 in total)
  Namespace     Name                       CPU Requests   CPU Limits   Memory Requests   Memory Limits   Age
  ---------     ----                       ------------   ----------   ---------------   -------------   ---
  default       porter-9dqmj               200m (10%)     500m (26%)   64Mi (1%)         1Gi (28%)       105s
  default       porter-9m4vx               200m (10%)     500m (26%)   64Mi (1%)         1Gi (28%)       105s
  kube-system   etcd-lone-tower            100m (5%)      0 (0%)       100Mi (2%)        0 (0%)          115s
Allocated resources:
  (Total limits may be over 100 percent, i.e., overcommitted.)
  Resource            Requests      Limits
  --------            --------      ------
  cpu                 500m (26%)    1000m (52%)
  memory              228Mi (6%)    2Gi (57%)
Events:
  Type    Reason     Age   From     Message
  ----    ------     ----  ----     -------
  Normal  NodeReady  42s   kubelet  Node lone-tower status is now: NodeReady
`

func TestTheTowersDescribeKeepsItsWardsAndFoldsTheLongSheetInStoryMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	kept, folded := foldSheet(StoryConsole(SplitOutput(describeTowerHuts)))
	joined := strings.Join(kept, "\n")
	if !strings.Contains(joined, "Wards:") || !strings.Contains(joined, "Royal Ward") {
		t.Fatalf("the folded sheet lost the Wards the describe step asks to read:\n%s", joined)
	}
	for _, long := range []string{"porter-9dqmj", "1,900", "rations (26%)", "overcommitted"} {
		if strings.Contains(joined, long) {
			t.Errorf("the story sheet still spreads %q before MORE (a spreadsheet-sized DESCRIBE):\n%s", long, joined)
		}
	}
	if len(kept) > 8 {
		t.Errorf("the story sheet keeps %d lines open, want the head of the sheet only:\n%s", len(kept), joined)
	}
	if got := strings.Join(folded, "\n"); !strings.Contains(got, "porter-9dqmj") || !strings.Contains(got, "1,900") {
		t.Fatalf("more has the huts and the room to unfold:\n%s", got)
	}
}
