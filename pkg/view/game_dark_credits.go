package view

import (
	"github.com/waxb/kubernites/pkg/soulstone"
)

func RolloutCredited(image string) bool {
	p, ok := soulstone.Lookup(image)
	return ok && p.Trade == FoundingTrade && soulstone.ClassIndex(p.Tag) == DevTopClass
}
