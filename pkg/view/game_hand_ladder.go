package view

import (
	"fmt"
)

const BadgeObjective = "depot-badge"

const BadgesPage = "marks"

type RollFacts struct {
	Hand      bool    `json:"hand,omitempty"`
	LooksFor  bool    `json:"looksFor,omitempty"`
	Live      int     `json:"live,omitempty"`
	HiredOn   int     `json:"hiredOn,omitempty"`
	Gone      int     `json:"gone,omitempty"`
	GoneName  string  `json:"goneName,omitempty"`
	GoneAddr  string  `json:"goneAddr,omitempty"`
	GoneIndex int     `json:"goneIndex,omitempty"`
	Idle      string  `json:"idle,omitempty"`
	IdleAddr  string  `json:"idleAddr,omitempty"`
	IdleCount int     `json:"idleCount,omitempty"`
	Lost      float64 `json:"lost,omitempty"`
	Depot     string  `json:"depot,omitempty"`
	Listed    int     `json:"listed,omitempty"`
}

func HandRollName(service string) string {
	return service + "-by-hand"
}

func parcelsService() string {
	return DepotOf(DepotParcels).Service
}

func rollPatch(body string) string {
	return fmt.Sprintf("kubectl patch endpointslice %s --type=json -p '%s'", HandRollName(parcelsService()), body)
}

func EnrollCommand(addr, pod string) string {
	return rollPatch(fmt.Sprintf(`[{"op":"add","path":"/endpoints/-","value":{"addresses":["%s"],"targetRef":{"kind":"Pod","namespace":"default","name":"%s"}}}]`, addr, pod))
}

func StrikeCommand(index int, addr string) string {
	return rollPatch(fmt.Sprintf(`[{"op":"test","path":"/endpoints/%d/addresses/0","value":"%s"},{"op":"remove","path":"/endpoints/%d"}]`, index, addr, index))
}

func TearUpCommand() string {
	return "kubectl delete endpointslice " + HandRollName(parcelsService())
}

func LookForCommand() string {
	d := DepotOf(DepotParcels)
	return fmt.Sprintf("kubectl set selector service %s app=%s", d.Service, d.Crew)
}
