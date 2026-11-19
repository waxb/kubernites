package view

import (
	"errors"
	"fmt"
	"strings"
)

const ActScaleCrew = "scaleCrew"

const ActCrewImage = "setCrewImage"

const ActCrewResources = "setCrewResources"

const ActRolloutCrew = "rolloutCrew"

const ActAutoscaleCrew = "autoscaleCrew"

const ActSilenceHorn = "silenceCrewHorn"

const ActRetuneHorn = "retuneCrewHorn"

const ActMoveWorker = "moveCrewWorker"

const RolloutRestart = "restart"

const RolloutUndo = "undo"

const RolloutPause = "pause"

const RolloutResume = "resume"

const maxPitch = 100

type CrewOrder struct {
	Container string `json:"container,omitempty"`
	Requests  string `json:"requests,omitempty"`
	Limits    string `json:"limits,omitempty"`
	Revision  int64  `json:"revision,omitempty"`
	Min       int    `json:"min,omitempty"`
	Max       int    `json:"max,omitempty"`
	CPU       int    `json:"cpu,omitempty"`
	Keeper    string `json:"keeper,omitempty"`
	Probe     string `json:"probe,omitempty"`
	Off       bool   `json:"off,omitempty"`
	Period    int    `json:"period,omitempty"`
	Threshold int    `json:"threshold,omitempty"`
	Timeout   int    `json:"timeout,omitempty"`
	Delay     int    `json:"delay,omitempty"`
	Before    string `json:"before,omitempty"`
	Forget    bool   `json:"forget,omitempty"`
}

var CrewCommands = map[string]func(Action) (string, error){
	ActScaleCrew:     scaleCrewCommand,
	ActCrewImage:     crewImageCommand,
	ActCrewResources: crewResourcesCommand,
	ActRolloutCrew:   rolloutCrewCommand,
	ActAutoscaleCrew: autoscaleCrewCommand,
	ActSilenceHorn:   silenceHornCommand,
	ActRetuneHorn:    retuneHornCommand,
	ActMoveWorker:    moveWorkerCommand,
}

func CrewActionCommand(a Action) (string, error) {
	build, ok := CrewCommands[a.Action]
	if !ok {
		return "", fmt.Errorf("%q is not a crew order", a.Action)
	}
	if strings.TrimSpace(a.Name) == "" {
		return "", errors.New("name the crew: its Deployment")
	}
	return build(a)
}

func moveWorkerCommand(a Action) (string, error) {
	if strings.TrimSpace(a.ID) == "" {
		return "", errors.New("name the hut to move")
	}
	return "kubectl delete pod " + a.ID, nil
}

func scaleCrewCommand(a Action) (string, error) {
	if a.Replicas < 0 || a.Replicas > MaxReplicas {
		return "", fmt.Errorf("a crew keeps between 0 and %d huts", MaxReplicas)
	}
	return ScaleCommandOf(a.Crew.Keeper, a.Name, a.Replicas), nil
}

func workloadRef(a Action, sep string) string {
	if a.Crew.Keeper == KeeperReplicaSet {
		return "rs" + sep + a.Name
	}
	return "deployment" + sep + a.Name
}

func crewImageCommand(a Action) (string, error) {
	if a.Crew.Container == "" || a.Image == "" {
		return "", errors.New("name the container and the trade's new class, for example porter=porter:1.1")
	}
	return fmt.Sprintf("kubectl set image %s %s=%s", workloadRef(a, "/"), a.Crew.Container, a.Image), nil
}

func crewResourcesCommand(a Action) (string, error) {
	if a.Crew.Container == "" || a.Crew.Requests == "" && a.Crew.Limits == "" {
		return "", errors.New("name the container and at least one request or limit")
	}
	cmd := fmt.Sprintf("kubectl set resources %s -c=%s", workloadRef(a, " "), a.Crew.Container)
	if a.Crew.Requests != "" {
		cmd += " --requests=" + a.Crew.Requests
	}
	if a.Crew.Limits != "" {
		cmd += " --limits=" + a.Crew.Limits
	}
	return cmd, nil
}

func rolloutCrewCommand(a Action) (string, error) {
	if a.Crew.Keeper == KeeperReplicaSet {
		return "", errForemanRollout
	}
	switch a.ID {
	case RolloutRestart, RolloutPause, RolloutResume:
		return fmt.Sprintf("kubectl rollout %s deployment/%s", a.ID, a.Name), nil
	case RolloutUndo:
		cmd := "kubectl rollout undo deployment/" + a.Name
		if a.Crew.Revision > 0 {
			cmd += fmt.Sprintf(" --to-revision=%d", a.Crew.Revision)
		}
		return cmd, nil
	}
	return "", fmt.Errorf("a rollout may restart, undo, pause or resume, not %q", a.ID)
}

func autoscaleCrewCommand(a Action) (string, error) {
	o := a.Crew
	switch {
	case o.Min < 1 || o.Max < o.Min || o.Max > MaxReplicas:
		return "", fmt.Errorf("a War Horn needs a Floor of at least 1 and a Ceiling between the Floor and %d", MaxReplicas)
	case o.CPU < 1 || o.CPU > maxPitch:
		return "", fmt.Errorf("the Horn's Pitch is a CPU share between 1%% and %d%%", maxPitch)
	}
	return fmt.Sprintf("kubectl autoscale %s --min=%d --max=%d --cpu=%d%%", workloadRef(a, " "), o.Min, o.Max, o.CPU), nil
}

func retuneHornCommand(a Action) (string, error) {
	o := a.Crew
	switch {
	case o.Min < 1 || o.Max < o.Min || o.Max > MaxReplicas:
		return "", fmt.Errorf("a War Horn needs a Floor of at least 1 and a Ceiling between the Floor and %d", MaxReplicas)
	case o.CPU < 0 || o.CPU > maxPitch:
		return "", fmt.Errorf("the Horn's Pitch is a CPU share between 1%% and %d%%", maxPitch)
	}
	spec := fmt.Sprintf(`"minReplicas":%d,"maxReplicas":%d`, o.Min, o.Max)
	if o.CPU > 0 {
		spec += fmt.Sprintf(`,"metrics":[{"type":"Resource","resource":{"name":"cpu","target":{"type":"Utilization","averageUtilization":%d}}}]`, o.CPU)
	}
	meta := ""
	switch {
	case o.Before != "":
		meta = fmt.Sprintf(`"metadata":{"annotations":{"%s":"%s"}},`, HornBeforeRoom, o.Before)
	case o.Forget:
		meta = fmt.Sprintf(`"metadata":{"annotations":{"%s":null}},`, HornBeforeRoom)
	}
	return fmt.Sprintf(`kubectl patch hpa %s -p '{%s"spec":{%s}}'`, a.Name, meta, spec), nil
}

func silenceHornCommand(a Action) (string, error) {
	return "kubectl delete hpa " + a.Name, nil
}
