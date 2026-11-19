package model

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var causeTitles = map[string]string{
	view.TagOOMKilled: "BOX TOO BIG", view.TagUnhoused: "UNHOUSED",
	view.CalamityNoDepot: "NO DEPOT", view.CalamityNoCrew: "NO CREW", view.CalamitySpoiling: "SPOILING",
}

var bridgedTitles = map[string][2]string{
	view.CalamityFallen: {"RAZED", "POD DELETED"}, view.TagThrottled: {"HUNGRY", "THROTTLED"}, view.TagEvicted: {"BANISHED", "EVICTED"},
	view.TagCrashLoop: {"GUTTERING", "CRASHLOOP"}, view.TagImagePull: {"NEVER ARRIVED", "IMAGE PULL"},
	view.CalamityFrozen: {"FROZEN HUT", "HUNG POD"}, view.CalamityWarm: {"UNREADY HUT", "NOT READY"},
	view.CalamityNines: {"NINES FALL", "SLO MISSED"}, view.EagerWatchmanKind: {"EAGER WATCHMAN", "LIVENESS RESTARTS"},
	view.CalamityHornLow: {"HORN HELD LOW", "HPA STILL LOWERED"},
	view.ReportHut:       {"REPORT: HUT", "REPORT: POD"}, view.ReportKeep: {"REPORT: KEEP", "REPORT: NODE"}, view.ReportQuest: {"REPORT: QUEST", "REPORT: JOB"},
	view.ReportDeed: {"REPORT: DEED", "REPORT: PVC"}, view.ReportHorn: {"REPORT: WAR HORN", "REPORT: HPA"},
}

func CauseTitle(kind string) string {
	if t, ok := bridgedTitles[kind]; ok {
		return ui.Bridge(t[0], t[1])
	}
	if t, ok := causeTitles[kind]; ok {
		return t
	}
	return strings.ToUpper(strings.ReplaceAll(kind, "-", " "))
}
