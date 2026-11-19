package kubectl

import (
	"errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
)

func statefulSetRolloutStatus(sts *appsv1.StatefulSet) (string, error) {
	if sts.Spec.UpdateStrategy.Type != appsv1.RollingUpdateStatefulSetStrategyType {
		return "", errors.New("error: rollout status is only available for RollingUpdate strategy type")
	}
	st := sts.Status
	if st.ObservedGeneration == 0 || sts.Generation > st.ObservedGeneration {
		return "Waiting for statefulset spec update to be observed...", nil
	}
	if sts.Spec.Replicas != nil && st.ReadyReplicas < *sts.Spec.Replicas {
		return fmt.Sprintf("Waiting for %d pods to be ready...", *sts.Spec.Replicas-st.ReadyReplicas), nil
	}
	if ru := sts.Spec.UpdateStrategy.RollingUpdate; ru != nil {
		if sts.Spec.Replicas != nil && ru.Partition != nil && st.UpdatedReplicas < *sts.Spec.Replicas-*ru.Partition {
			return fmt.Sprintf("Waiting for partitioned roll out to finish: %d out of %d new pods have been updated...", st.UpdatedReplicas, *sts.Spec.Replicas-*ru.Partition), nil
		}
		return fmt.Sprintf("partitioned roll out complete: %d new pods have been updated...", st.UpdatedReplicas), nil
	}
	if st.UpdateRevision != st.CurrentRevision {
		return fmt.Sprintf("waiting for statefulset rolling update to complete %d pods at revision %s...", st.UpdatedReplicas, st.UpdateRevision), nil
	}
	return fmt.Sprintf("statefulset rolling update complete %d pods at revision %s...", st.CurrentReplicas, st.CurrentRevision), nil
}
