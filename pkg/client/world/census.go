package world

import (
	"reflect"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

type podStat struct {
	m        view.PodMetrics
	measured bool
}

type census struct {
	base     *model.PodView
	metrics  uintptr
	n        int
	at       float64
	stats    []podStat
	rate     float64
	declared int
}

func declaredHearth(p model.PodView) bool {
	return !p.System && !p.Static && !p.Terminating && p.State == model.HearthReady
}

func (c *census) of(sc Scene) *census {
	var base *model.PodView
	if len(sc.Pods) > 0 {
		base = &sc.Pods[0]
	}
	metrics := reflect.ValueOf(sc.Metrics.Pods).Pointer()
	if c.stats != nil && c.base == base && c.metrics == metrics && c.n == len(sc.Pods) && c.at == sc.Time {
		return c
	}
	c.base, c.metrics, c.n, c.at, c.rate, c.declared = base, metrics, len(sc.Pods), sc.Time, 0, 0
	c.stats = c.stats[:0]
	for _, p := range sc.Pods {
		m, ok := sc.Metrics.Pods[PodKey(p)]
		c.stats = append(c.stats, podStat{m: m, measured: ok})
		if !p.System {
			c.rate += m.ServedRate
		}
		if declaredHearth(p) {
			c.declared++
		}
	}
	return c
}
