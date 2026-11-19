package hud

import (
	"math"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func foodOf(milli float64) string { return view.Rations(milli) }

func housingOf(bytes float64) string { return view.Rooms(bytes / mebi) }

func foodOfCores(cores float64) string { return view.Rations(cores * view.RationsPerCPU) }

func housingOfGiB(g float64) string { return view.Rooms(g * roomsPerGiB) }

func foodCount(milli float64) string { return view.Grouped(int64(math.Round(milli))) }

func housingCount(bytes float64) string { return view.Grouped(int64(math.Round(bytes / mebi))) }

func shout(s string) string { return strings.ToUpper(s) }

func foodTag(milli float64) string { return ui.IconFood + foodCount(milli) }

func housingTag(bytes float64) string { return ui.IconHousing + housingCount(bytes) }
