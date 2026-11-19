package livetest

import (
	"encoding/json"
	"net/http"

	"github.com/waxb/kubernites/pkg/live"
)

const (
	metricsRoot    = "/apis/metrics.k8s.io"
	metricsVersion = "metrics.k8s.io/v1beta1"
)

func serveMetrics(w http.ResponseWriter, r *http.Request, s live.Snapshot) {
	switch r.URL.Path {
	case metricsRoot:
		version := map[string]string{"groupVersion": metricsVersion, "version": "v1beta1"}
		_ = json.NewEncoder(w).Encode(map[string]any{"kind": "APIGroup", "apiVersion": "v1", "name": "metrics.k8s.io",
			"versions": []any{version}, "preferredVersion": version})
	case "/apis/" + metricsVersion + "/nodes":
		items := []any{}
		for _, n := range s.Nodes {
			items = append(items, map[string]any{"metadata": map[string]string{"name": n.Name}, "window": "15s",
				"usage": map[string]string{"cpu": "250m", "memory": "1500Mi"}})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"kind": "NodeMetricsList", "apiVersion": metricsVersion, "metadata": map[string]any{}, "items": items})
	case "/apis/" + metricsVersion + "/pods":
		items := []any{}
		for _, p := range s.Pods {
			containers := []any{}
			for _, c := range p.Spec.Containers {
				containers = append(containers, map[string]any{"name": c.Name, "usage": map[string]string{"cpu": "3m", "memory": "20Mi"}})
			}
			items = append(items, map[string]any{"metadata": map[string]string{"name": p.Name, "namespace": p.Namespace}, "window": "15s", "containers": containers})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"kind": "PodMetricsList", "apiVersion": metricsVersion, "metadata": map[string]any{}, "items": items})
	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"kind":"Status","status":"Failure","reason":"NotFound","code":404}`))
	}
}
