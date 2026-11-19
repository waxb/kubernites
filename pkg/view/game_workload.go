package view

import (
	"fmt"
	"github.com/waxb/kubernites/pkg/soulstone"
)

const SummonObjective = "summon-porter"

const HandPortersObjective = "enroll-porters"

const TremorObjective = "witness-tremor"

const RaiseForemanObjective = "raise-foreman"

const DeploymentObjective = "deployment"

const RollClassObjective = "roll-class"

const ForemenPage = "foremen"

const DeploymentsPage = "deployments"

const KeeperReplicaSet = "ReplicaSet"

const KeeperDeployment = "Deployment"

const DeployButton = "crew:deploy"

const HandPorterTarget = 3

const NewClass = "porter:1.1"

func ScaleCommandOf(keeper, crew string, replicas int) string {
	if keeper == KeeperReplicaSet {
		return fmt.Sprintf("kubectl scale rs %s --replicas=%d", crew, replicas)
	}
	return ScaleCommand(crew, replicas)
}

func DeployCommand(crew, image string, replicas int) string {
	cmd := fmt.Sprintf("kubectl create deployment %s --image=%s --replicas=%d", crew, image, max(replicas, 1))
	if p, ok := soulstone.Lookup(image); ok && p.Port > 0 {
		cmd += fmt.Sprintf(" --port=%d", p.Port)
	}
	return cmd
}
