package client

import (
	"time"

	"github.com/waxb/kubernites/pkg/sim/store"
	"github.com/waxb/kubernites/pkg/view"
)

type Realm interface {
	Advance(realSeconds float64)
	Watch(fn func(store.Event))
	WithSnapshot(fn func([]store.Object))
	GameState() view.GameState
	Metrics() view.MetricsSnapshot
	Site() view.Site
	Action(a view.Action) (string, error)
	Perform(a view.Action) (view.Performed, error)
	Kubectl(command string) view.CommandResult
	Inspect(command string) view.CommandResult
	Apply(yaml string) (string, bool)
	Marshal(savedAt time.Time) ([]byte, error)
	PlacementError(kind string, x, y int) error
	PlotSurvey(kind string, x, y int) view.Survey
	SuggestPlacement(kind string) (int, int, bool)
	SuggestKeepFor(kind, crew string) (int, int, bool)
	Bootstrap()
}
