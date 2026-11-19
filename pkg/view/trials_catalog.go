package view

import (
	"fmt"
)

const TrialPledgeID = "pledge"

const TrialRolloutID = "rollout"

const TrialHornID = "horn"

func PaceCommand(crew, surge, gaps string) string {
	return fmt.Sprintf(`kubectl patch deployment %s -p '{"spec":{"strategy":{"rollingUpdate":{"maxSurge":%s,"maxUnavailable":%s}}}}'`, crew, surge, gaps)
}
