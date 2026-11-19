package view

import (
	"github.com/waxb/kubernites/pkg/soulstone"
)

const KindCount = 4

const SizeClasses = soulstone.SizeClasses

const MaxTier = 4

type BoxKind struct {
	ID      string
	Name    string
	Opens   int
	Staged  bool
	TTL     float64
	Rate    float64
	Weights [MaxTier]float64
	Loose   [MaxTier]float64
	Sizes   [MaxTier][SizeClasses]float64
}

var BoxKinds = [KindCount]BoxKind{
	{ID: soulstone.KindParcel, Name: "parcels", Opens: 1, TTL: 28, Rate: 1,
		Weights: [MaxTier]float64{100, 30, 15, 10}, Loose: [MaxTier]float64{100, 40, 15, 10},
		Sizes: [MaxTier][SizeClasses]float64{{1}, {0.85, 0.15}, {0.5, 0.35, 0.15}, {0.4, 0.3, 0.2, 0.1}}},
	{ID: soulstone.KindLedger, Name: "ledgers", Opens: 2, TTL: 36, Rate: 12,
		Weights: [MaxTier]float64{0, 40, 20, 15}, Loose: [MaxTier]float64{0, 60, 20, 15},
		Sizes: [MaxTier][SizeClasses]float64{{1}, {0.9, 0.1}, {0.6, 0.3, 0.1}, {0.5, 0.3, 0.15, 0.05}}},
	{ID: soulstone.KindCrate, Name: "crates", Opens: 2, Staged: true, TTL: 45, Rate: 90,
		Weights: [MaxTier]float64{0, 30, 25, 20}, Loose: [MaxTier]float64{0, 30, 25, 20},
		Sizes: [MaxTier][SizeClasses]float64{{1}, {0.4, 0.5, 0.1}, {0.2, 0.4, 0.3, 0.1}, {0.1, 0.3, 0.35, 0.25}}},
	{ID: soulstone.KindDispatch, Name: "dispatches", Opens: 3, TTL: 120, Rate: 1350,
		Weights: [MaxTier]float64{0, 0, 40, 25}, Loose: [MaxTier]float64{0, 0, 40, 25},
		Sizes: [MaxTier][SizeClasses]float64{{1}, {1}, {0.5, 0.4, 0.1}, {0.4, 0.35, 0.2, 0.05}}},
}

func (k BoxKind) Arrives(tier int, firm bool) bool {
	return k.Opens <= tier && (firm || !k.Staged)
}

func (k BoxKind) WeightAt(tier int, firm bool) float64 {
	if firm {
		return k.Weights[tier-1]
	}
	return k.Loose[tier-1]
}
