package talk

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
)

func TestStatusSaysOnlyTheContainersState(t *testing.T) {
	f := burning()
	f.Doing = Doing{Activity: DoCarrying, Load: "parcel"}
	if got := Status(f); got != "Carrying a parcel to the parcels depot." {
		t.Errorf("story status = %q", got)
	}
	f.Names = true
	if got := Status(f); got != "Running, started 8m ago, Ready true, Restart Count 0" {
		t.Errorf("names status = %q", got)
	}
	f.Hut.Rites = true
	for _, names := range []bool{false, true} {
		f.Names = names
		got := Status(f)
		if strings.Contains(got, "Terminating") {
			t.Errorf("a container is never Terminating: %q", got)
		}
		if names && got != "Running; stopping after SIGTERM within its grace period" {
			t.Errorf("names status while the Pod is deleted = %q", got)
		}
		if !names && !strings.Contains(got, "our hut is at Last Rites") {
			t.Errorf("story status while the hut is at Last Rites = %q", got)
		}
	}
	f.Hut.Rites = false
	f.Worker.State = model.ContainerState{Phase: model.ContainerWaiting, Reason: "CrashLoopBackOff"}
	f.Names = true
	if got := Status(f); got != "Waiting (Reason: CrashLoopBackOff)" {
		t.Errorf("guttering names status = %q", got)
	}
	f.Names = false
	if got := Status(f); !strings.HasPrefix(got, "Guttering") {
		t.Errorf("guttering story status = %q", got)
	}
	f.Worker.State = model.ContainerState{Phase: model.ContainerTerminated, Reason: "OOMKilled", ExitCode: 137}
	f.Names = true
	if got := Status(f); got != "Terminated (Reason: OOMKilled, Exit Code: 137)" {
		t.Errorf("terminated names status = %q", got)
	}
}

func TestNeedsAreReadingsOfThatContainerOnly(t *testing.T) {
	f := burning()
	f.Use.Throttled = true
	f.Use.CPUMilli = 490
	needs := Needs(f)
	keys := []string{}
	for _, n := range needs {
		keys = append(keys, n.Key)
	}
	if strings.Join(keys, ",") != "food,room,busy,shaken,age" {
		t.Fatalf("needs = %v", keys)
	}
	if needs[0].Mark != "HUNGRY" || needs[0].Value != "490 of 200 rations pledged" {
		t.Errorf("food = %+v", needs[0])
	}
	f.Names = true
	if n := Needs(f)[0]; n.Mark != "THROTTLED" || n.Value != "490m of 200m req, limit 500m" {
		t.Errorf("cpu = %+v", n)
	}
	f.Use.Alone = false
	for _, n := range Needs(f) {
		if n.Key == NeedRoom || n.Key == NeedFood && n.Share >= 0 {
			t.Errorf("a hut with a familiar: its food is not this worker's: %+v", n)
		}
	}
	f.Known = map[string]bool{}
	for _, n := range Needs(f) {
		if n.Key == NeedFood || n.Key == NeedRoom {
			t.Errorf("food is shown only once the hut card shows it: %+v", n)
		}
	}
}

func TestMoodFollowsTheReadings(t *testing.T) {
	f := burning()
	if Mood(f) != "content" {
		t.Errorf("mood = %q", Mood(f))
	}
	f.Use.Throttled = true
	if Mood(f) != "peckish" {
		t.Errorf("hungry mood = %q", Mood(f))
	}
	f.Names = true
	if Mood(f) != "Short of CPU" {
		t.Errorf("names hungry = %q", Mood(f))
	}
	f.Use.CPUMilli = 500
	if Mood(f) != "Throttled" {
		t.Errorf("names at limit = %q", Mood(f))
	}
	for _, mood := range []string{Mood(f), Mood(burning())} {
		for _, tired := range []string{"sleep", "tired", "rest", "nap"} {
			if strings.Contains(strings.ToLower(mood), tired) {
				t.Errorf("containers never tire: %q", mood)
			}
		}
	}
}

func TestThoughtsAreRealHappeningsNewestFirst(t *testing.T) {
	f := burning()
	f.Worker.Restarts = 1
	f.Worker.Last = model.ContainerState{Phase: model.ContainerTerminated, Reason: "Error", ExitCode: 1, Finished: epoch + 90}
	haps := append(FromStatus(f), Happening{At: epoch + 300, Kind: HapFirstBox, Box: "parcel"}, Happening{At: epoch + 400, Kind: HapJoined})
	got := Thoughts(f, haps, MaxThoughts)
	want := []string{"Our hut joined the roll.", "Carried my first parcel!", "Kindled. Here we go.", "Stopped (exit 1), then relit."}
	if len(got) != len(want) {
		t.Fatalf("thoughts = %+v", got)
	}
	for i, w := range want {
		if got[i].Text != w {
			t.Errorf("thought %d = %q, want %q", i, got[i].Text, w)
		}
	}
	f.Known = map[string]bool{}
	for _, th := range Thoughts(f, haps, MaxThoughts) {
		if strings.Contains(th.Text, "roll") {
			t.Errorf("a thought names the roll before the player met it: %q", th.Text)
		}
	}
	f.Names = true
	if got := Thoughts(f, FromStatus(f), 1); got[0].Text != "Started." {
		t.Errorf("names thought = %+v", got)
	}
}

func TestCurrentThoughtComesFromTheStrongestNeed(t *testing.T) {
	f := burning()
	f.Worker.Resources.MemLimit = 64 << 20
	f.Use.MemBytes = 62 << 20
	if got := Thinks(porterOf(Anxious), f); situationOf(got) != SitCramped {
		t.Errorf("cramped worker thinks %q", got)
	}
	if Thinks(porterOf(Anxious), f) != Thinks(porterOf(Anxious), f) {
		t.Error("the thought holds still while the readings do")
	}
}
