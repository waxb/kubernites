package talk

type Situation string

const (
	SitGuttering Situation = "guttering"
	SitVoidtaken Situation = "voidtaken"
	SitNever     Situation = "never-arrived"
	SitFrozen    Situation = "frozen"
	SitHungry    Situation = "hungry"
	SitCramped   Situation = "cramped"
	SitKindling  Situation = "kindling"
	SitRites     Situation = "last-rites"
	SitShutters  Situation = "shutters"
	SitRelit     Situation = "relit"
	SitJoined    Situation = "joined-roll"
	SitLeft      Situation = "left-roll"
	SitRollout   Situation = "rollout"
	SitProblem   Situation = "problem"
	SitImp       Situation = "imp"
	SitGilded    Situation = "gilded"
	SitTide      Situation = "rush-tide"
	SitNight     Situation = "night"
	SitWaiting   Situation = "waiting"
	SitDoor      Situation = "door"
	SitHand      Situation = "hand-hired"
	SitCrew      Situation = "crew"
	SitRoll      Situation = "on-roll"
	SitWard      Situation = "royal-ward"
	SitShaken    Situation = "shaken"
	SitTrade     Situation = "trade"
	SitBurning   Situation = "burning"
	SitGeneral   Situation = "general"
	SitPet       Situation = "familiar"
	SitSystem    Situation = "citadel"
	SitAnnoyed   Situation = "annoyed"
)

const (
	TierUrgent = iota + 1
	TierRecent
	TierProblem
	TierAround
	TierWho
	TierGeneral
)

type situation struct {
	id    Situation
	tier  int
	holds func(Facts) bool
}

var situations = []situation{
	{SitGuttering, TierUrgent, Facts.Guttering},
	{SitVoidtaken, TierUrgent, func(f Facts) bool { return f.Voidtaken() && (f.Guttering() || f.Relit()) }},
	{SitNever, TierUrgent, Facts.NeverArrived},
	{SitFrozen, TierUrgent, func(f Facts) bool { return f.Hut.Frozen && f.running() }},
	{SitHungry, TierUrgent, Facts.Hungry},
	{SitCramped, TierUrgent, Facts.Cramped},
	{SitKindling, TierUrgent, func(f Facts) bool { return f.Kindling() && !f.Hut.Rites }},
	{SitRites, TierUrgent, func(f Facts) bool { return f.Hut.Rites }},
	{SitShutters, TierUrgent, Facts.Shuttered},
	{SitRelit, TierRecent, Facts.Relit},
	{SitJoined, TierRecent, func(f Facts) bool { return f.Hut.OnRoll && recent(f.Hut.JoinedAgo) }},
	{SitLeft, TierRecent, func(f Facts) bool { return !f.Hut.OnRoll && recent(f.Hut.LeftAgo) }},
	{SitRollout, TierRecent, func(f Facts) bool { return f.Hut.Rollout && f.ReplicaSet() }},
	{SitProblem, TierProblem, func(f Facts) bool { return f.Problem != "" }},
	{SitImp, TierAround, func(f Facts) bool { return f.Around.ImpAgo >= 0 && f.Around.ImpAgo < impSeconds }},
	{SitGilded, TierAround, func(f Facts) bool { return f.Around.Gilded }},
	{SitTide, TierAround, func(f Facts) bool { return f.Around.Tide }},
	{SitNight, TierAround, func(f Facts) bool { return f.Around.Night }},
	{SitWaiting, TierAround, Facts.Waiting},
	{SitDoor, TierWho, func(f Facts) bool { return f.Hut.Static }},
	{SitHand, TierWho, Facts.HandHired},
	{SitCrew, TierWho, Facts.ReplicaSet},
	{SitRoll, TierWho, func(f Facts) bool { return f.Hut.OnRoll }},
	{SitWard, TierWho, func(f Facts) bool { return f.Hut.LoneWard }},
	{SitShaken, TierWho, func(f Facts) bool { return f.Worker.Restarts > 0 }},
	{SitTrade, TierWho, func(f Facts) bool { return f.Trade != "" }},
	{SitBurning, TierWho, Facts.Burning},
	{SitPet, TierWho, func(f Facts) bool { return f.Familiar != "" }},
	{SitSystem, TierWho, func(f Facts) bool { return f.Hut.System }},
	{SitGeneral, TierGeneral, func(Facts) bool { return true }},
}

func Situations(f Facts) []Situation {
	out := []Situation{}
	for _, s := range situations {
		if s.holds(f) {
			out = append(out, s.id)
		}
	}
	return out
}

func holds(id Situation, f Facts) bool {
	for _, s := range situations {
		if s.id == id {
			return s.holds(f)
		}
	}
	return false
}

func tierOf(id Situation) int {
	for _, s := range situations {
		if s.id == id {
			return s.tier
		}
	}
	return TierGeneral
}

func rankOf(id Situation) int {
	for i, s := range situations {
		if s.id == id {
			return i
		}
	}
	return len(situations)
}
