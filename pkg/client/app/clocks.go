package app

type Clocks struct {
	Real  float64
	World float64
	Event float64
}

func realmSpeed(speed float64) float64 {
	if speed <= 0 {
		return 1
	}
	return speed
}

func WorldRate(speed float64, running bool) float64 {
	if !running {
		return 0
	}
	return realmSpeed(speed)
}

func EventRate(speed float64, running bool) float64 {
	if !running {
		return 1
	}
	return realmSpeed(speed)
}

func (c Clocks) Step(dt, speed float64, running bool) Clocks {
	return Clocks{Real: c.Real + dt, World: c.World + dt*WorldRate(speed, running), Event: c.Event + dt*EventRate(speed, running)}
}
