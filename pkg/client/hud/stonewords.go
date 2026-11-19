package hud

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/waxb/kubernites/pkg/lexicon"
)

var (
	rowLabels = map[string]string{
		"OldReplicaSets": "Old shifts", "NewReplicaSet": "New shift", "StrategyType": "Roll-out style",
		"RollingUpdateStrategy": "Roll-out pace", "Controlled By": "Kept by", "Port": "Door", "TargetPort": "Hut door",
		"Endpoints": "Ready huts", "Allocatable": "For huts", "Requests": "Pledges", "Limits": "Bindings",
	}
	conditionNames = map[string]string{
		"MemoryPressure": "Crowding", "DiskPressure": "Ground full", "PIDPressure": "Throng", "NetworkUnavailable": "Fog",
		"PodScheduled": "Seated", "Initialized": "Squires done", "ContainersReady": "Workers ready",
		"PodReadyToStartContainers": "Ready to start", "DisruptionTarget": "Marked for removal", "ReplicaFailure": "Shift failing",
	}
	conditionReasons = map[string]string{
		"MinimumReplicasAvailable": "Enough huts ready", "MinimumReplicasUnavailable": "Too few huts ready",
		"NewReplicaSetAvailable": "New shift ready", "ReplicaSetUpdated": "Shift swapping in", "FoundNewReplicaSet": "New shift found",
		"ProgressDeadlineExceeded": "Roll-out stalled", "NewReplicaSetCreated": "New shift raised", "ReplicaSetCreateError": "Shift not raised",
		"FailedCreate": "Hut not raised",
	}
	eventReasons = map[string]string{
		"ScalingReplicaSet": "ScalingShift", "SuccessfulCreate": "HutRaised", "SuccessfulDelete": "HutSentHome",
		"FailedCreate": "HutNotRaised", "Unhealthy": "ShuttersClosed", "ProbeWarning": "ShuttersWarned", "NodeNotReady": "KeepNotReady",
	}
	apiNouns = map[string]string{
		"pods": "huts", "nodes": "keeps", "deployments": "crews", "daemonsets": "shrines", "controllerrevisions": "records",
		"replicasets": "shifts", "services": "depots", "endpointslices": "rolls", "events": "tidings", "namespaces": "fiefs",
		"configmaps": "scrolls", "limitranges": "defaults", "horizontalpodautoscalers": "horns", "priorityclasses": "ranks",
		"persistentvolumes": "vaults", "persistentvolumeclaims": "deeds", "storageclasses": "vault kinds", "statefulsets": "orders",
		"jobs": "quests", "cronjobs": "rituals", "secrets": "sealed scrolls",
	}
	conditionMessages = strings.NewReplacer("has sufficient PID available", "has room in the throng", "has no disk pressure", "has room for cellars", "has no PID pressure", "has no throng")
	wardEffect        = regexp.MustCompile(`(\S*):(NoSchedule|NoExecute|PreferNoSchedule)(?: op=\w+)?(?: for (\d+)s)?`)
	ownerRef          = regexp.MustCompile(`^([A-Za-z]+)/(\S+)$`)
	hostAddress       = regexp.MustCompile(`/\d+\.\d+\.\d+\.\d+$`)
	portProto         = regexp.MustCompile(`/(?:TCP|UDP|SCTP)\b`)
	effectWords       = map[string]string{"NoSchedule": "bars new huts", "NoExecute": "turns huts out", "PreferNoSchedule": "prefers other Keeps"}
)

func wardEffects(text string) string {
	return wardEffect.ReplaceAllStringFunc(text, func(m string) string {
		parts := wardEffect.FindStringSubmatch(m)
		note := effectWords[parts[2]]
		if parts[1] == "" {
			return "every Ward that " + note
		}
		if parts[3] != "" {
			note += ", for " + parts[3] + "s"
		}
		return parts[1] + " (" + note + ")"
	})
}

func rowValue(label, value string) string {
	switch label {
	case "Controlled By":
		if m := ownerRef.FindStringSubmatch(value); m != nil {
			return strings.ToLower(lexicon.StoryText(m[1])) + " " + m[2]
		}
	case "Node":
		return hostAddress.ReplaceAllString(value, "")
	case "Endpoints":
		return readyHuts(value)
	case "Port", "TargetPort":
		return strings.Join(strings.Fields(portProto.ReplaceAllString(value, "")), " ")
	}
	return value
}

var moreAddresses = regexp.MustCompile(`\+ (\d+) more`)

func readyHuts(value string) string {
	if value == "" || value == "<none>" || value == "<unset>" {
		return "none: boxes wait at the door"
	}
	head, _, _ := strings.Cut(value, " + ")
	n := len(strings.Split(head, ","))
	if m := moreAddresses.FindStringSubmatch(value); m != nil {
		extra, _ := strconv.Atoi(m[1])
		n += extra
	}
	return strconv.Itoa(n)
}

func hasColumn(header []string, name string) bool {
	for _, h := range header {
		if strings.EqualFold(h, name) {
			return true
		}
	}
	return false
}

func conditionTable(header []string) bool {
	return hasColumn(header, "Type") && hasColumn(header, "Status")
}

func eventTable(header []string) bool {
	return hasColumn(header, "Reason") && hasColumn(header, "Message") && !hasColumn(header, "Status")
}

func conditionsHaveMessage(header []string) bool {
	return hasColumn(header, "Message")
}

func tableCell(header []string, column, cell string) string {
	switch {
	case column == "PORT(S)":
		return doorsOf(cell)
	case conditionTable(header) && strings.EqualFold(column, "Type"):
		if name, ok := conditionNames[cell]; ok {
			return name
		}
	case conditionTable(header) && strings.EqualFold(column, "Message"):
		return lexicon.StoryText(conditionMessages.Replace(cell))
	case conditionTable(header) && strings.EqualFold(column, "Reason"):
		if name, ok := conditionReasons[cell]; ok {
			return name
		}
	case eventTable(header) && strings.EqualFold(column, "Reason"):
		if name, ok := eventReasons[cell]; ok {
			return name
		}
	}
	return ""
}

func doorsOf(cell string) string {
	if cell == "<none>" || cell == "-" {
		return cell
	}
	parts := strings.Split(cell, ",")
	for i, p := range parts {
		p = portProto.ReplaceAllString(p, "")
		if front, node, ok := strings.Cut(p, ":"); ok {
			p = front + " (postern " + node + ")"
		}
		parts[i] = p
	}
	return strings.Join(parts, ", ")
}

var headerNames = map[string]string{"CLUSTER-IP": "ADDRESS", "PORT(S)": "DOOR", "NAMESPACED": "FIEF-BOUND"}

func serviceTable(header []string) bool {
	for _, h := range header {
		if h == "CLUSTER-IP" {
			return true
		}
	}
	return false
}

func tableDropped(header []string) map[string]bool {
	out := map[string]bool{}
	for k, v := range storyDroppedColumns {
		out[k] = v
	}
	if conditionTable(header) && conditionsHaveMessage(header) {
		for _, h := range header {
			if strings.EqualFold(h, "Reason") {
				out[h] = true
			}
		}
	}
	if serviceTable(header) {
		out["TYPE"], out["EXTERNAL-IP"] = true, true
	}
	return out
}

func apiResourceTable(lines []string) bool {
	return len(lines) > 1 && strings.HasPrefix(lines[0], "NAME") && strings.Contains(lines[0], "SHORTNAMES") && strings.Contains(lines[0], "APIVERSION")
}

func storyAPIResources(lines []string) []string {
	starts := []int{}
	for _, m := range regexp.MustCompile(`\S+`).FindAllStringIndex(lines[0], -1) {
		starts = append(starts, m[0])
	}
	cell := func(l string, c int) string {
		if c >= len(starts) || starts[c] >= len(l) {
			return ""
		}
		end := len(l)
		if c+1 < len(starts) {
			end = min(starts[c+1], len(l))
		}
		return strings.TrimSpace(l[starts[c]:end])
	}
	rows := [][]string{{"NAME", "FIEF-BOUND", "KIND"}}
	for _, l := range lines[1:] {
		name := cell(l, 0)
		story, ok := apiNouns[name]
		if !ok {
			story = lexicon.StoryText(name)
		}
		bound := "no"
		if cell(l, 3) == "true" {
			bound = "yes"
		}
		rows = append(rows, []string{story, bound, lexicon.StoryText(cell(l, 4))})
	}
	return alignRows(rows)
}

func alignRows(rows [][]string) []string {
	widths := make([]int, len(rows[0]))
	for _, r := range rows {
		for c, cell := range r {
			widths[c] = max(widths[c], len([]rune(cell)))
		}
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		var b strings.Builder
		for c, cell := range r {
			b.WriteString(cell)
			if c < len(r)-1 {
				b.WriteString(strings.Repeat(" ", widths[c]-len([]rune(cell))+tableGap))
			}
		}
		out = append(out, b.String())
	}
	return out
}
