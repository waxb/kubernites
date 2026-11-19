package world

func (sc Scene) RealTime() float64 {
	if sc.Real == 0 {
		return sc.Time
	}
	return sc.Real
}
