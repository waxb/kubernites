package kubectl

import (
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/waxb/kubernites/pkg/sim/rolloututil"
	"github.com/waxb/kubernites/pkg/sim/store"
)

const (
	RestartedAtAnnotation = "kubectl.kubernetes.io/restartedAt"
	ChangeCauseAnnotation = "kubernetes.io/change-cause"
)

type RolloutVerb func(Cluster, Invocation, *appsv1.Deployment) (string, error)

var rolloutVerbs = map[string]RolloutVerb{
	"status": rolloutStatus, "history": rolloutHistory,
}

func RegisterRollout(sub string, deployment RolloutVerb, revisioned RevisionedVerb) {
	rolloutVerbs[sub] = deployment
	if revisioned != nil {
		revisionedVerbs[sub] = revisioned
	}
}

func rollout(c Cluster, inv Invocation) (string, string, error) {
	if len(inv.Args) == 0 {
		return "", "", errors.New("error: expected a rollout subcommand: history, pause, restart, resume, status or undo")
	}
	verb, ok := rolloutVerbs[inv.Args[0]]
	if !ok {
		return "", "", fmt.Errorf("error: unknown command %q for \"kubectl rollout\"", inv.Args[0])
	}
	if len(inv.Args) < 2 {
		return "", "deployment", errors.New("error: required resource not specified")
	}
	typeTok, names, err := SplitTypeName(inv.Args[1:])
	if err != nil {
		return "", "deployment", err
	}
	r, err := ResolveType(typeTok)
	if err != nil {
		return "", typeTok, err
	}
	if len(names) == 0 {
		return "", r.Singular, errors.New("error: resource(s) were provided, but no name was specified")
	}
	if r.Kind == store.KindDaemonSet || r.Kind == store.KindStatefulSet {
		return rolloutRevisioned(c, inv, r, names)
	}
	if r.Kind != store.KindDeployment {
		return "", r.Singular, unsupportedRollout(inv.Args[0], r, names[0])
	}
	return rolloutEach(c, inv, verb, names)
}

func rolloutEach(c Cluster, inv Invocation, verb RolloutVerb, names []string) (string, string, error) {
	out := []string{}
	for _, name := range names {
		d, err := GetDeployment(c, inv.Namespace, name)
		if err != nil {
			return strings.Join(out, "\n"), "deployment", err
		}
		line, err := verb(c, inv, d)
		if err != nil {
			return strings.Join(out, "\n"), "deployment", err
		}
		out = append(out, strings.TrimRight(line, "\n"))
	}
	return strings.Join(out, "\n"), "deployment", nil
}

func unsupportedRollout(sub string, r ResourceType, name string) error {
	kind := schema.GroupKind{Group: r.group, Kind: r.Kind}
	switch sub {
	case "status":
		return fmt.Errorf("error: no status viewer has been implemented for %v", kind)
	case "history":
		return fmt.Errorf("error: no history viewer has been implemented for %q", kind.String())
	case "undo":
		return fmt.Errorf("error: no rollbacker has been implemented for %q", kind.String())
	}
	action := map[string]string{"restart": "restarting", "pause": "pausing", "resume": "resuming"}[sub]
	return fmt.Errorf("error: %s %q %s is not supported", qualifiedPlural(r), name, action)
}

func rolloutStatus(_ Cluster, _ Invocation, d *appsv1.Deployment) (string, error) {
	if d.Generation > d.Status.ObservedGeneration {
		return "Waiting for deployment spec update to be observed...", nil
	}
	cond := rolloututil.DeploymentCondition(d.Status, appsv1.DeploymentProgressing)
	if cond != nil && cond.Reason == rolloututil.ReasonTimedOut {
		return "", fmt.Errorf("error: deployment %q exceeded its progress deadline", d.Name)
	}
	st := d.Status
	switch {
	case d.Spec.Replicas != nil && st.UpdatedReplicas < *d.Spec.Replicas:
		return fmt.Sprintf("Waiting for deployment %q rollout to finish: %d out of %d new replicas have been updated...", d.Name, st.UpdatedReplicas, *d.Spec.Replicas), nil
	case st.Replicas > st.UpdatedReplicas:
		return fmt.Sprintf("Waiting for deployment %q rollout to finish: %d old replicas are pending termination...", d.Name, st.Replicas-st.UpdatedReplicas), nil
	case st.AvailableReplicas < st.UpdatedReplicas:
		return fmt.Sprintf("Waiting for deployment %q rollout to finish: %d of %d updated replicas are available...", d.Name, st.AvailableReplicas, st.UpdatedReplicas), nil
	}
	return fmt.Sprintf("deployment %q successfully rolled out", d.Name), nil
}
