package world

const routedMin = 3

func (r *Renderer) RoutedStep(sc Scene, e Errand, to Vec, step ErrandStep) ErrandStep {
	if step.Phase != PhaseOut && step.Phase != PhaseBack {
		return step
	}
	path, water := r.HarborOf().errandPath(sc.Site, e.From, to)
	if len(path) < routedMin {
		return step
	}
	frac := step.Progress
	if step.Phase == PhaseBack {
		frac = 1 - frac
	}
	pos, dir := PathAt(path, frac)
	if step.Phase == PhaseBack {
		dir = Vec{-dir.X, -dir.Y}
	}
	step.Pose.Pos, step.Pose.Dir = pos, Facing(Vec{}, dir)
	if afloat(path, water, frac) {
		step.Skiff = ShipHeading(dir)
	}
	return step
}
