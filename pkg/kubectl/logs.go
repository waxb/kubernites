package kubectl

import (
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/sim/store"
)

const DefaultContainerAnnotation = "kubectl.kubernetes.io/default-container"

type ContainerLogger interface {
	ContainerLogs(uid types.UID, container string, previous bool) ([]string, bool)
}

func logs(c Cluster, inv Invocation) (string, string, error) {
	if len(inv.Args) == 0 || len(inv.Args) > 2 {
		return "", "", fmt.Errorf("error: expected 'logs [-f] [-p] (POD | TYPE/NAME) [-c CONTAINER]'")
	}
	name := strings.TrimPrefix(strings.TrimPrefix(inv.Args[0], "pod/"), "pods/")
	p, ok := store.Get[*corev1.Pod](c.Store(), store.KindPod, inv.Namespace, name)
	if !ok {
		return "", "pod", fmt.Errorf("Error from server (NotFound): pods %q not found", name)
	}
	if len(p.Spec.Containers) == 0 {
		return "", "pod", fmt.Errorf("Error from server (BadRequest): pod %q has no containers", name)
	}
	want := inv.Flags["container"]
	if len(inv.Args) == 2 {
		want = inv.Args[1]
	}
	container, notice, err := pickContainer(p, want)
	if err != nil {
		return "", "pod", err
	}
	out, err := containerLogs(c, p, container, inv)
	if err != nil {
		return "", "pod", fmt.Errorf("%s%w", notice, err)
	}
	return notice + out, "pod", nil
}

func containerLogs(c Cluster, p *corev1.Pod, container string, inv Invocation) (string, error) {
	previous := inv.Bools["previous"]
	lines, ok := fetchLogs(c, p, container, previous)
	if previous && (!ok || len(lines) == 0) {
		return "", fmt.Errorf("Error from server (BadRequest): previous terminated container %q in pod %q not found", container, p.Name)
	}
	waiting := waitingReason(p, container)
	if !ok || (len(lines) == 0 && waiting != "") {
		return "", fmt.Errorf("Error from server (BadRequest): container %q in pod %q is waiting to start: %s", container, p.Name, orDefault(waiting, "ContainerCreating"))
	}
	if raw := inv.Flags["tail"]; raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return "", fmt.Errorf("error: invalid argument %q for \"--tail\" flag", raw)
		}
		if n >= 0 && n < len(lines) {
			lines = lines[len(lines)-n:]
		}
	}
	return strings.Join(lines, "\n"), nil
}

func fetchLogs(c Cluster, p *corev1.Pod, container string, previous bool) ([]string, bool) {
	if cl, ok := c.(ContainerLogger); ok {
		return cl.ContainerLogs(p.UID, container, previous)
	}
	if container != p.Spec.Containers[0].Name {
		return nil, true
	}
	return c.Logs(p.UID, previous)
}

func pickContainer(p *corev1.Pod, want string) (string, string, error) {
	if want != "" {
		if !hasContainer(p, want) {
			return "", "", fmt.Errorf("error: container %s is not valid for pod %s out of: %s", want, p.Name, allContainerNames(p))
		}
		return want, "", nil
	}
	notice := ""
	if annotated := p.Annotations[DefaultContainerAnnotation]; annotated != "" {
		if hasContainer(p, annotated) {
			return annotated, "", nil
		}
		notice = fmt.Sprintf("Default container name %q not found in pod %s\n", annotated, p.Name)
	}
	first := p.Spec.Containers[0].Name
	if len(p.Spec.Containers) > 1 || len(p.Spec.InitContainers) > 0 {
		notice += fmt.Sprintf("Defaulted container %q out of: %s\n", first, allContainerNames(p))
	}
	return first, notice, nil
}

func hasContainer(p *corev1.Pod, name string) bool {
	for _, list := range [][]corev1.Container{p.Spec.Containers, p.Spec.InitContainers} {
		for _, c := range list {
			if c.Name == name {
				return true
			}
		}
	}
	return false
}

func allContainerNames(p *corev1.Pod) string {
	names := []string{}
	for _, c := range p.Spec.Containers {
		names = append(names, c.Name)
	}
	for _, c := range p.Spec.InitContainers {
		names = append(names, c.Name+" (init)")
	}
	return strings.Join(names, ", ")
}

func waitingReason(p *corev1.Pod, container string) string {
	for _, list := range [][]corev1.ContainerStatus{p.Status.ContainerStatuses, p.Status.InitContainerStatuses} {
		for _, cs := range list {
			if cs.Name == container && cs.State.Waiting != nil {
				return cs.State.Waiting.Reason
			}
		}
	}
	return ""
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
