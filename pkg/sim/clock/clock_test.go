package clock

import (
	"math"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAdvanceAndNow(t *testing.T) {
	epoch := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	c := New(epoch)
	c.Advance(90.5)
	c.Advance(-5)
	c.Advance(math.NaN())
	c.Advance(math.Inf(1))
	if c.Seconds() != 90.5 {
		t.Fatalf("seconds = %v", c.Seconds())
	}
	if got := c.Now(); !got.Equal(epoch.Add(90500 * time.Millisecond)) {
		t.Fatalf("now = %v", got)
	}
	if got := c.MetaNow(); !got.Time.Equal(epoch.Add(90 * time.Second)) {
		t.Fatalf("meta now = %v", got)
	}
	if c.Epoch() != epoch {
		t.Fatal("epoch changed")
	}
}

func TestSetAndSince(t *testing.T) {
	epoch := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	c := New(epoch)
	c.Set(120)
	c.Set(-1)
	if c.Seconds() != 120 {
		t.Fatalf("seconds = %v", c.Seconds())
	}
	if d := c.Since(metav1.NewTime(epoch.Add(20 * time.Second))); d != 100*time.Second {
		t.Fatalf("since = %v", d)
	}
	if d := c.Since(metav1.Time{}); d != 0 {
		t.Fatalf("since zero = %v", d)
	}
}
