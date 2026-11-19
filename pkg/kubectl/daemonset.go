package kubectl

import (
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/waxb/kubernites/pkg/sim/podutil"
	"github.com/waxb/kubernites/pkg/sim/store"
)

func daemonSetHeaders(wide bool) []string {
	h := []string{"NAME", "DESIRED", "CURRENT", "READY", "UP-TO-DATE", "AVAILABLE", "NODE SELECTOR", "AGE"}
	if wide {
		h = append(h, "CONTAINERS", "IMAGES", "SELECTOR")
	}
	return h
}

func daemonSetRow(c Cluster, o store.Object, wide bool) []string {
	ds := o.(*appsv1.DaemonSet)
	st := ds.Status
	row := []string{ds.Name, fmt.Sprint(st.DesiredNumberScheduled), fmt.Sprint(st.CurrentNumberScheduled), fmt.Sprint(st.NumberReady),
		fmt.Sprint(st.UpdatedNumberScheduled), fmt.Sprint(st.NumberAvailable), labels.FormatLabels(ds.Spec.Template.Spec.NodeSelector), age(c, ds.CreationTimestamp)}
	if wide {
		names, images := []string{}, []string{}
		for _, ct := range ds.Spec.Template.Spec.Containers {
			names, images = append(names, ct.Name), append(images, ct.Image)
		}
		row = append(row, strings.Join(names, ","), strings.Join(images, ","), metav1.FormatLabelSelector(ds.Spec.Selector))
	}
	return row
}

func daemonSetRolloutStatus(ds *appsv1.DaemonSet) (string, error) {
	if ds.Spec.UpdateStrategy.Type != appsv1.RollingUpdateDaemonSetStrategyType {
		return "", errors.New("rollout status is only available for RollingUpdate strategy type")
	}
	st := ds.Status
	switch {
	case ds.Generation > st.ObservedGeneration:
		return "Waiting for daemon set spec update to be observed...", nil
	case st.UpdatedNumberScheduled < st.DesiredNumberScheduled:
		return fmt.Sprintf("Waiting for daemon set %q rollout to finish: %d out of %d new pods have been updated...", ds.Name, st.UpdatedNumberScheduled, st.DesiredNumberScheduled), nil
	case st.NumberAvailable < st.DesiredNumberScheduled:
		return fmt.Sprintf("Waiting for daemon set %q rollout to finish: %d of %d updated pods are available...", ds.Name, st.NumberAvailable, st.DesiredNumberScheduled), nil
	}
	return fmt.Sprintf("daemon set %q successfully rolled out", ds.Name), nil
}

func describeDaemonSet(c Cluster, ds *appsv1.DaemonSet) string {
	b := &block{}
	selector, _ := metav1.LabelSelectorAsSelector(ds.Spec.Selector)
	b.kv(0, 16, "Name", ds.Name)
	b.kv(0, 16, "Namespace", ds.Namespace)
	b.kv(0, 16, "Selector", selector.String())
	b.kv(0, 16, "Node-Selector", labels.FormatLabels(ds.Spec.Template.Spec.NodeSelector))
	b.kvLines(0, 16, "Labels", labelLines(ds.Labels))
	b.kvLines(0, 16, "Annotations", annotationLines(ds.Annotations))
	st := ds.Status
	b.raw(fmt.Sprintf("Desired Number of Nodes Scheduled: %d", st.DesiredNumberScheduled))
	b.raw(fmt.Sprintf("Current Number of Nodes Scheduled: %d", st.CurrentNumberScheduled))
	b.raw(fmt.Sprintf("Number of Nodes Scheduled with Up-to-date Pods: %d", st.UpdatedNumberScheduled))
	b.raw(fmt.Sprintf("Number of Nodes Scheduled with Available Pods: %d", st.NumberAvailable))
	b.raw(fmt.Sprintf("Number of Nodes Misscheduled: %d", st.NumberMisscheduled))
	running, waiting, succeeded, failed := daemonPodCounts(c, ds, selector)
	b.kv(0, 14, "Pods Status", fmt.Sprintf("%d Running / %d Waiting / %d Succeeded / %d Failed", running, waiting, succeeded, failed))
	b.raw("Pod Template:")
	b.kvLines(2, 9, "Labels", labelLines(ds.Spec.Template.Labels))
	b.raw("  Containers:")
	describeTemplateContainers(b, ds.Spec.Template.Spec.Containers)
	b.raw(eventsSection(c, ds.UID))
	return b.String()
}

func daemonPodCounts(c Cluster, ds *appsv1.DaemonSet, selector labels.Selector) (int, int, int, int) {
	var running, waiting, succeeded, failed int
	for _, p := range store.List[*corev1.Pod](c.Store(), store.KindPod, ds.Namespace) {
		ref := podutil.ControllerRef(p)
		if ref == nil || ref.UID != ds.UID || !selector.Matches(labels.Set(p.Labels)) {
			continue
		}
		switch p.Status.Phase {
		case corev1.PodRunning:
			running++
		case corev1.PodSucceeded:
			succeeded++
		case corev1.PodFailed:
			failed++
		default:
			waiting++
		}
	}
	return running, waiting, succeeded, failed
}
