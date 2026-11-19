package live

import (
	"slices"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

var proxyVariables = map[string]bool{"HTTP_PROXY": true, "HTTPS_PROXY": true, "NO_PROXY": true, "ALL_PROXY": true}

func withoutProxyEnv(containers []corev1.Container) []corev1.Container {
	out := slices.Clone(containers)
	for i, c := range out {
		out[i].Env = slices.DeleteFunc(slices.Clone(c.Env), func(e corev1.EnvVar) bool { return proxyVariables[strings.ToUpper(e.Name)] })
	}
	return out
}

func withoutProxySpec(spec corev1.PodSpec) corev1.PodSpec {
	spec.InitContainers = withoutProxyEnv(spec.InitContainers)
	spec.Containers = withoutProxyEnv(spec.Containers)
	return spec
}

func scrubEach[T any](list []T, scrub func(T) T) []T {
	out := make([]T, len(list))
	for i, v := range list {
		out[i] = scrub(v)
	}
	if list == nil {
		return nil
	}
	return out
}

func (s Snapshot) WithoutProxyEnvironment() Snapshot {
	s.Pods = scrubEach(s.Pods, func(p corev1.Pod) corev1.Pod { p.Spec = withoutProxySpec(p.Spec); return p })
	s.Deployments = scrubEach(s.Deployments, func(d appsv1.Deployment) appsv1.Deployment {
		d.Spec.Template.Spec = withoutProxySpec(d.Spec.Template.Spec)
		return d
	})
	s.ReplicaSets = scrubEach(s.ReplicaSets, func(r appsv1.ReplicaSet) appsv1.ReplicaSet {
		r.Spec.Template.Spec = withoutProxySpec(r.Spec.Template.Spec)
		return r
	})
	s.DaemonSets = scrubEach(s.DaemonSets, func(d appsv1.DaemonSet) appsv1.DaemonSet {
		d.Spec.Template.Spec = withoutProxySpec(d.Spec.Template.Spec)
		return d
	})
	s.StatefulSets = scrubEach(s.StatefulSets, func(d appsv1.StatefulSet) appsv1.StatefulSet {
		d.Spec.Template.Spec = withoutProxySpec(d.Spec.Template.Spec)
		return d
	})
	s.Jobs = scrubEach(s.Jobs, func(j batchv1.Job) batchv1.Job {
		j.Spec.Template.Spec = withoutProxySpec(j.Spec.Template.Spec)
		return j
	})
	s.CronJobs = scrubEach(s.CronJobs, func(c batchv1.CronJob) batchv1.CronJob {
		c.Spec.JobTemplate.Spec.Template.Spec = withoutProxySpec(c.Spec.JobTemplate.Spec.Template.Spec)
		return c
	})
	return s
}
