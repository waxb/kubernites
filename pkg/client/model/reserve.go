package model

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/sim/podutil"
)

type reserve struct {
	cpu int64
	mem int64
}

func systemShares(pods []*corev1.Pod) map[string]reserve {
	out := map[string]reserve{}
	for _, p := range pods {
		if p.Namespace != systemNamespace || p.Spec.NodeName == "" || podutil.IsTerminal(p) {
			continue
		}
		cpu, mem := podutil.Requests(p)
		s := out[p.Spec.NodeName]
		out[p.Spec.NodeName] = reserve{cpu: s.cpu + cpu, mem: s.mem + mem}
	}
	return out
}

func (c *Cache) SpacedNames(kind string) map[string][]string {
	out := map[string][]string{}
	for _, k := range c.keys(kind) {
		out[k.Namespace] = append(out[k.Namespace], k.Name)
	}
	return out
}
