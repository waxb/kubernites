package gfxtest

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func drawSome(n int) {
	dst, src := ebiten.NewImage(16, 16), ebiten.NewImage(4, 4)
	for range n {
		dst.DrawImage(src, nil)
	}
}

func TestDrawsBeforeTheFirstFrameQueueUpAndDropForgetsThem(t *testing.T) {
	Drop()
	drawSome(200)
	if Pending() < 200 {
		t.Fatalf("pending = %d, want at least 200 queued draws", Pending())
	}
	Drop()
	if Pending() != 0 {
		t.Fatalf("pending after Drop = %d, want 0", Pending())
	}
}

func TestTheReaperKeepsTheQueueFromGrowingWithoutBound(t *testing.T) {
	stop := reap(5 * time.Millisecond)
	defer stop()
	drawSome(500)
	deadline := time.Now().Add(2 * time.Second)
	for Pending() > 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if Pending() != 0 {
		t.Fatalf("the reaper left %d queued draws", Pending())
	}
}

func TestStopEndsTheReaper(t *testing.T) {
	stop := reap(time.Millisecond)
	stop()
	Drop()
	drawSome(50)
	time.Sleep(20 * time.Millisecond)
	if Pending() == 0 {
		t.Fatal("the reaper kept running after stop")
	}
	Drop()
}

func TestHoldKeepsTheReaperOffUntilReleased(t *testing.T) {
	stop := reap(time.Millisecond)
	defer stop()
	release := Hold()
	Drop()
	drawSome(50)
	time.Sleep(20 * time.Millisecond)
	if Pending() < 50 {
		t.Fatalf("the reaper dropped the queue while it was held: pending = %d", Pending())
	}
	release()
	release()
	deadline := time.Now().Add(2 * time.Second)
	for Pending() > 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if Pending() != 0 {
		t.Fatalf("after the release the reaper left %d queued draws", Pending())
	}
}
