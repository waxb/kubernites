package rolloututil

import (
	"encoding/json"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

const (
	RevisionAnnotation        = "deployment.kubernetes.io/revision"
	RevisionHistoryAnnotation = "deployment.kubernetes.io/revision-history"
	DesiredReplicasAnnotation = "deployment.kubernetes.io/desired-replicas"
	MaxReplicasAnnotation     = "deployment.kubernetes.io/max-replicas"
	ReasonTimedOut            = "ProgressDeadlineExceeded"
)

var annotationsToSkip = map[string]bool{
	corev1.LastAppliedConfigAnnotation: true,
	RevisionAnnotation:                 true,
	RevisionHistoryAnnotation:          true,
	DesiredReplicasAnnotation:          true,
	MaxReplicasAnnotation:              true,
	appsv1.DeprecatedRollbackTo:        true,
}

func SkipCopyAnnotation(key string) bool {
	return annotationsToSkip[key]
}

func DeploymentCondition(status appsv1.DeploymentStatus, t appsv1.DeploymentConditionType) *appsv1.DeploymentCondition {
	for i := range status.Conditions {
		if status.Conditions[i].Type == t {
			c := status.Conditions[i]
			return &c
		}
	}
	return nil
}

func labelsEqualIgnoreHash(a, b map[string]string) bool {
	count := func(m map[string]string) int {
		n := len(m)
		if _, ok := m[appsv1.DefaultDeploymentUniqueLabelKey]; ok {
			n--
		}
		return n
	}
	if count(a) != count(b) {
		return false
	}
	for k, v := range a {
		if k == appsv1.DefaultDeploymentUniqueLabelKey {
			continue
		}
		if other, ok := b[k]; !ok || other != v {
			return false
		}
	}
	return true
}

func EqualIgnoreHash(t1, t2 *corev1.PodTemplateSpec) bool {
	if !labelsEqualIgnoreHash(t1.Labels, t2.Labels) {
		return false
	}
	m1, m2 := t1.ObjectMeta, t2.ObjectMeta
	m1.Labels, m2.Labels = nil, nil
	return equality.Semantic.DeepEqual(m1, m2) && equality.Semantic.DeepEqual(t1.Spec, t2.Spec)
}

func ByCreation(rss []*appsv1.ReplicaSet) []*appsv1.ReplicaSet {
	out := append([]*appsv1.ReplicaSet(nil), rss...)
	sort.SliceStable(out, func(i, j int) bool { return CreatedBefore(out[i], out[j]) })
	return out
}

func CreatedBefore(a, b *appsv1.ReplicaSet) bool {
	if a.CreationTimestamp.Equal(&b.CreationTimestamp) {
		return a.Name < b.Name
	}
	return a.CreationTimestamp.Before(&b.CreationTimestamp)
}

func FindNewReplicaSet(d *appsv1.Deployment, rsList []*appsv1.ReplicaSet) *appsv1.ReplicaSet {
	for _, rs := range ByCreation(rsList) {
		if EqualIgnoreHash(&rs.Spec.Template, &d.Spec.Template) {
			return rs
		}
	}
	return nil
}

func FindOldReplicaSets(d *appsv1.Deployment, rsList []*appsv1.ReplicaSet) []*appsv1.ReplicaSet {
	return OldSetsBesides(FindNewReplicaSet(d, rsList), rsList)
}

func OldSetsBesides(newRS *appsv1.ReplicaSet, rsList []*appsv1.ReplicaSet) []*appsv1.ReplicaSet {
	all := []*appsv1.ReplicaSet{}
	for _, rs := range rsList {
		if newRS != nil && rs.UID == newRS.UID {
			continue
		}
		all = append(all, rs)
	}
	return all
}

func SortRevisions(revs []*appsv1.ControllerRevision) {
	sort.SliceStable(revs, func(i, j int) bool {
		if revs[i].Revision == revs[j].Revision {
			if revs[i].CreationTimestamp.Equal(&revs[j].CreationTimestamp) {
				return revs[i].Name < revs[j].Name
			}
			return revs[i].CreationTimestamp.Before(&revs[j].CreationTimestamp)
		}
		return revs[i].Revision < revs[j].Revision
	})
}

func ApplyTemplate(cr *appsv1.ControllerRevision) (corev1.PodTemplateSpec, error) {
	var patch struct {
		Spec struct {
			Template map[string]any `json:"template"`
		} `json:"spec"`
	}
	var tmpl corev1.PodTemplateSpec
	if err := json.Unmarshal(cr.Data.Raw, &patch); err != nil {
		return tmpl, err
	}
	delete(patch.Spec.Template, "$patch")
	data, _ := json.Marshal(patch.Spec.Template)
	err := json.Unmarshal(data, &tmpl)
	return tmpl, err
}
