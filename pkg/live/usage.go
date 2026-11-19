package live

import (
	"context"
	"net/url"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StepMetrics = "metrics"
	StepDrawing = "drawing"
)

var usageLimit = 15 * time.Second

type UsageOf struct {
	CPUMilli float64
	MemBytes float64
}

type Usage struct {
	Available bool
	Pods      map[string]UsageOf
	Nodes     map[string]UsageOf
}

type metricsItem struct {
	Metadata   metav1.ObjectMeta   `json:"metadata"`
	Usage      corev1.ResourceList `json:"usage"`
	Containers []struct {
		Name  string              `json:"name"`
		Usage corev1.ResourceList `json:"usage"`
	} `json:"containers"`
}

func usageOf(l corev1.ResourceList) UsageOf {
	return UsageOf{CPUMilli: float64(l.Cpu().MilliValue()), MemBytes: float64(l.Memory().Value())}
}

func (c *Conn) Usage(ctx context.Context) Usage {
	ctx, cancel := context.WithTimeout(ctx, usageLimit)
	defer cancel()
	return readUsage(ctx, c)
}

func readUsage(ctx context.Context, l lister) Usage {
	var group metav1.APIGroup
	if err := l.getJSON(ctx, "/apis/metrics.k8s.io", nil, &group); err != nil || group.PreferredVersion.GroupVersion == "" {
		return Usage{}
	}
	gv := group.PreferredVersion.GroupVersion
	nodes, err := listAll[metricsItem](ctx, l.getJSON, "/apis/"+gv+"/nodes", nil)
	if err != nil {
		return Usage{}
	}
	pods, err := listAll[metricsItem](ctx, l.getJSON, "/apis/"+gv+"/pods", nil)
	if err != nil {
		return Usage{}
	}
	out := Usage{Available: true, Pods: map[string]UsageOf{}, Nodes: map[string]UsageOf{}}
	for _, n := range nodes {
		out.Nodes[n.Metadata.Name] = usageOf(n.Usage)
	}
	for _, p := range pods {
		sum := UsageOf{}
		for _, c := range p.Containers {
			u := usageOf(c.Usage)
			sum.CPUMilli += u.CPUMilli
			sum.MemBytes += u.MemBytes
		}
		out.Pods[p.Metadata.Namespace+"/"+p.Metadata.Name] = sum
	}
	return out
}

func (c *Conn) podLogs(ctx context.Context, namespace, pod, container string, previous bool, tail int) (string, error) {
	q := url.Values{"tailLines": {strconv.Itoa(tail)}}
	if container != "" {
		q.Set("container", container)
	}
	if previous {
		q.Set("previous", "true")
	}
	return c.getText(ctx, "/api/v1/namespaces/"+url.PathEscape(namespace)+"/pods/"+url.PathEscape(pod)+"/log", q)
}
