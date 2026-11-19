package podutil

import (
	"sort"

	corev1 "k8s.io/api/core/v1"
)

const (
	ConfigSourceKey = "kubernetes.io/config.source"
	ConfigHashKey   = "kubernetes.io/config.hash"
	ConfigSeenKey   = "kubernetes.io/config.seen"
	FileSource      = "file"
	apiSource       = "api"
	wildcardIP      = "0.0.0.0"
)

type HostPort struct {
	IP       string
	Protocol corev1.Protocol
	Port     int32
}

func IsMirrorPod(p *corev1.Pod) bool {
	_, ok := p.Annotations[corev1.MirrorPodAnnotationKey]
	return ok
}

func IsStaticPod(p *corev1.Pod) bool {
	source, ok := p.Annotations[ConfigSourceKey]
	return ok && source != apiSource
}

func HostPorts(p *corev1.Pod) []HostPort {
	var out []HostPort
	for _, c := range p.Spec.Containers {
		for _, port := range c.Ports {
			if port.HostPort <= 0 {
				continue
			}
			hp := HostPort{IP: port.HostIP, Protocol: port.Protocol, Port: port.HostPort}
			if hp.IP == "" {
				hp.IP = wildcardIP
			}
			if hp.Protocol == "" {
				hp.Protocol = corev1.ProtocolTCP
			}
			out = append(out, hp)
		}
	}
	return out
}

func OnNodeAddress(p *corev1.Pod, port int32) bool {
	if HoldsHostPort(p, port) {
		return true
	}
	if !p.Spec.HostNetwork {
		return false
	}
	for _, c := range p.Spec.Containers {
		for _, cp := range c.Ports {
			if cp.ContainerPort == port {
				return true
			}
		}
	}
	return false
}

func HoldsHostPort(p *corev1.Pod, port int32) bool {
	for _, hp := range HostPorts(p) {
		if hp.Port == port {
			return true
		}
	}
	return false
}

func FitsHostPorts(p *corev1.Pod, others []*corev1.Pod) bool {
	want := HostPorts(p)
	if len(want) == 0 {
		return true
	}
	for _, o := range others {
		for _, used := range HostPorts(o) {
			if conflicts(want, used) {
				return false
			}
		}
	}
	return true
}

func conflicts(want []HostPort, used HostPort) bool {
	for _, w := range want {
		sameAddress := w.IP == used.IP || w.IP == wildcardIP || used.IP == wildcardIP
		if w.Port == used.Port && w.Protocol == used.Protocol && sameAddress {
			return true
		}
	}
	return false
}

func World(api, static []*corev1.Pod, namespace string) []*corev1.Pod {
	out := make([]*corev1.Pod, 0, len(api)+len(static))
	for _, p := range api {
		if !IsMirrorPod(p) && inNamespace(p, namespace) {
			out = append(out, p)
		}
	}
	extra := 0
	for _, p := range static {
		if inNamespace(p, namespace) {
			out = append(out, p)
			extra++
		}
	}
	if extra > 0 {
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Namespace != out[j].Namespace {
				return out[i].Namespace < out[j].Namespace
			}
			return out[i].Name < out[j].Name
		})
	}
	return out
}

func inNamespace(p *corev1.Pod, namespace string) bool {
	return namespace == "" || p.Namespace == namespace
}
