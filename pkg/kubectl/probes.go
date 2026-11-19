package kubectl

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

type probeLine struct {
	key   string
	probe *corev1.Probe
}

func containerProbes(c corev1.Container) []probeLine {
	out := []probeLine{}
	for _, l := range []probeLine{{"Liveness", c.LivenessProbe}, {"Readiness", c.ReadinessProbe}, {"Startup", c.StartupProbe}} {
		if l.probe != nil {
			out = append(out, l)
		}
	}
	return out
}

func describeProbe(p *corev1.Probe) string {
	attrs := fmt.Sprintf("delay=%ds timeout=%ds period=%ds successThreshold=%d failureThreshold=%d", p.InitialDelaySeconds, p.TimeoutSeconds, p.PeriodSeconds, p.SuccessThreshold, p.FailureThreshold)
	switch {
	case p.Exec != nil:
		return fmt.Sprintf("exec %v %s", p.Exec.Command, attrs)
	case p.HTTPGet != nil:
		u := &url.URL{Scheme: strings.ToLower(string(p.HTTPGet.Scheme)), Host: p.HTTPGet.Host, Path: p.HTTPGet.Path}
		if port := p.HTTPGet.Port.String(); len(port) > 0 {
			u.Host = net.JoinHostPort(p.HTTPGet.Host, port)
		}
		return fmt.Sprintf("http-get %s %s", u.String(), attrs)
	case p.TCPSocket != nil:
		return fmt.Sprintf("tcp-socket %s:%s %s", p.TCPSocket.Host, p.TCPSocket.Port.String(), attrs)
	case p.GRPC != nil:
		service := ""
		if p.GRPC.Service != nil {
			service = *p.GRPC.Service
		}
		return fmt.Sprintf("grpc <pod>:%d %s %s", p.GRPC.Port, service, attrs)
	}
	return fmt.Sprintf("unknown %s", attrs)
}
