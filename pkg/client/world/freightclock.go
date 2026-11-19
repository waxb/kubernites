package world

const (
	ClockAhead = 6.0
	clockRate  = 25.0
)

type FreightClock struct {
	realm, clock, rate, out float64
	primed                  bool
}

func (c *FreightClock) Step(realmTime, clock float64) float64 {
	switch {
	case !c.primed || realmTime < c.realm-resetSlack || clock < c.clock:
		*c = FreightClock{realm: realmTime, clock: clock, out: realmTime, primed: true}
		return realmTime
	case realmTime != c.realm:
		if dt := clock - c.clock; dt > 1e-6 {
			c.rate = min(max((realmTime-c.realm)/dt, 0), clockRate)
		}
		c.realm, c.clock = realmTime, clock
	}
	guess := c.realm + min(clock-c.clock, ClockAhead)*c.rate
	c.out = max(c.out, guess, realmTime)
	return c.out
}
