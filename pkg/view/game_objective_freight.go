package view

import (
	"fmt"
	"github.com/waxb/kubernites/pkg/soulstone"
	"math"
	"strconv"
)

func DepotOf(i int) soulstone.Depot {
	kind := soulstone.KindSealed
	if i < KindCount {
		kind = BoxKinds[i].ID
	}
	d, _ := soulstone.DepotFor(kind)
	return d
}

func ScaleCommand(crew string, replicas int) string {
	return fmt.Sprintf("kubectl scale deployment %s --replicas=%d", crew, replicas)
}

func FreshShare(slo float64) string {
	return strconv.FormatFloat(math.Floor(slo*1000)/10, 'f', 1, 64) + "%"
}
