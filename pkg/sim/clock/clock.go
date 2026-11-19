package clock

import (
	"math"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Clock struct {
	epoch   time.Time
	seconds float64
}

func New(epoch time.Time) *Clock {
	return &Clock{epoch: epoch.UTC().Truncate(time.Second)}
}

func (c *Clock) Advance(dt float64) {
	if dt > 0 && !math.IsInf(dt, 0) && !math.IsNaN(dt) {
		c.seconds += dt
	}
}

func (c *Clock) Seconds() float64 {
	return c.seconds
}

func (c *Clock) Set(seconds float64) {
	if seconds >= 0 {
		c.seconds = seconds
	}
}

func (c *Clock) Epoch() time.Time {
	return c.epoch
}

func (c *Clock) Now() time.Time {
	return c.epoch.Add(time.Duration(c.seconds * float64(time.Second)))
}

func (c *Clock) MetaNow() metav1.Time {
	return metav1.NewTime(c.Now().Truncate(time.Second))
}

func (c *Clock) Since(t metav1.Time) time.Duration {
	if t.IsZero() {
		return 0
	}
	return c.Now().Sub(t.Time)
}
