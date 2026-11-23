package hud

import (
	"fmt"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	livePendingRows  = 4
	liveShortRows    = 12
	liveUnservedRows = 12
	liveCellarRows   = 6
	liveCouncilTip   = "The Council: what the realm's servants report, read-only [C]"
)

var liveCouncilTabs = []string{TabQuartermaster, TabStewards, TabWayfinder, TabAugur}

func liveAdvisorTip(id string) string {
	switch id {
	case TabStewards:
		return ui.Bridge("The Stewards: crews, stateful crews and shrines short of huts Fit for Duty", "Controllers: Deployments, StatefulSets and DaemonSets below their desired replicas")
	case TabWayfinder:
		return ui.Bridge("The Wayfinder: depots with no hut Fit for Duty on their Guild Roll", "kube-proxy: Services with no ready endpoint")
	case TabAugur:
		return ui.Bridge("The Augur: the busiest huts, and the cellars pledged on every Keep", "metrics-server: the busiest Pods, and the ephemeral storage requested on every node")
	}
	return ui.Bridge("The Quartermaster: food and housing pledged on every Keep, and the Unhoused", "kube-scheduler: CPU and memory requested on every node, and the Pending Pods")
}

func councilAdvisors(f Frame) []advisor {
	if !f.Live {
		return Advisors
	}
	out := []advisor{}
	for _, a := range Advisors {
		if slices.Contains(liveCouncilTabs, a.Id) {
			out = append(out, a)
		}
	}
	return out
}

func liveUnread(f Frame, lists ...string) bool {
	for _, l := range lists {
		if slices.Contains(f.LiveCouncil.Unread, l) {
			return true
		}
	}
	return false
}

func share(n, of int) float64 {
	if of <= 0 {
		return 0
	}
	return 100 * float64(n) / float64(of)
}

func LiveDials(f Frame) []Dial {
	c := f.LiveCouncil
	all := []struct {
		dial  Dial
		n, of int
		lists []string
	}{
		{Dial{Label: ui.Bridge("KEEPS", "NODES"), Icon: "icon/rating-resilience",
			Tip: ui.Bridge("Keeps standing: the Keeps whose Reeve says the Keep is ready, of every Keep", "Nodes Ready: the nodes whose Ready condition is True, of every node")}, c.NodesReady, c.Nodes, []string{"nodes"}},
		{Dial{Label: ui.Bridge("HUTS", "PODS"), Icon: "icon/rating-reliability",
			Tip: ui.Bridge("Huts Fit for Duty, of the huts neither finished nor fallen", "Pods Ready: the Pods whose Ready condition is True, of the Pods that are neither Succeeded nor Failed")}, c.PodsReady, c.PodsActive, []string{"pods"}},
		{Dial{Label: ui.Bridge("CREWS", "WORKLOADS"), Icon: "icon/rating-efficiency",
			Tip: ui.Bridge("Crews, stateful crews and shrines with every hut they want Fit for Duty, of all of them",
				"Workloads at their desired replicas: Deployments and StatefulSets whose readyReplicas reach spec.replicas, and DaemonSets whose numberReady reaches desiredNumberScheduled, of all of them")},
			c.WorkloadsFull, c.Workloads, []string{"deployments", "statefulsets", "daemonsets"}},
		{Dial{Label: ui.Bridge("DEPOTS", "SERVICES"), Icon: "icon/rating-favor",
			Tip: ui.Bridge("Depots with at least one hut Fit for Duty on their Guild Roll, of the depots that look for a badge",
				"Services with at least one ready endpoint in their EndpointSlices, of the Services with a selector")}, c.ServicesServed, c.Services, []string{"services", "endpointslices"}},
	}
	out := []Dial{}
	for _, d := range all {
		if d.of <= 0 || liveUnread(f, d.lists...) {
			continue
		}
		d.dial.Value, d.dial.Known, d.dial.Text = share(d.n, d.of), true, fmt.Sprintf("%d/%d", d.n, d.of)
		out = append(out, d.dial)
	}
	return out
}

func pendingCount(n int) string {
	if n == 1 {
		return ui.Bridge("1 hut is Unhoused", "1 Pod is Pending")
	}
	return ui.Bridge(fmt.Sprintf("%d huts are Unhoused", n), fmt.Sprintf("%d Pods are Pending", n))
}

func liveKind(kind string) string {
	switch kind {
	case "StatefulSet":
		return ui.Bridge("stateful crew", kind)
	case "DaemonSet":
		return ui.Bridge("shrine", kind)
	}
	return ui.Bridge("crew", kind)
}

func objectName(ns, name string) string { return ns + "/" + name }

const liveStoryWhy = "switch NAMES to read why"

func schedulerSaid(reason string) string {
	switch {
	case reason == "":
		return ui.Bridge("no word from the Quartermaster yet", "no reason from the scheduler yet")
	case !ui.Naming():
		return liveStoryWhy
	}
	return ui.Verbatim(reason)
}

func LiveCounsel(f Frame) string {
	c := f.LiveCouncil
	if len(c.Pending) > 0 {
		p := c.Pending[0]
		switch {
		case p.Reason == "":
			return pendingCount(len(c.Pending)) + "."
		case !ui.Naming():
			return pendingCount(len(c.Pending)) + ": " + liveStoryWhy + "."
		}
		return ui.Verbatim(pendingCount(len(c.Pending)) + ". " + objectName(p.Namespace, p.Name) + ": " + p.Reason)
	}
	if down := c.Nodes - c.NodesReady; down > 0 && !liveUnread(f, "nodes") {
		return ui.Bridge(plural(down, "Keep is", "Keeps are")+" dark: their Reeve does not say they are ready.", plural(down, "node is", "nodes are")+" not Ready.")
	}
	if len(c.Short) > 0 {
		w := c.Short[0]
		return ui.Bridge(fmt.Sprintf("The %s %s has %d of %d huts Fit for Duty.", liveKind(w.Kind), objectName(w.Namespace, w.Name), w.Ready, w.Desired),
			fmt.Sprintf("%s %s has %d of %d replicas ready.", w.Kind, objectName(w.Namespace, w.Name), w.Ready, w.Desired))
	}
	if len(c.Unserved) > 0 {
		s := c.Unserved[0]
		return ui.Bridge("The depot "+objectName(s.Namespace, s.Name)+" has no hut Fit for Duty on its Guild Roll.", "Service "+objectName(s.Namespace, s.Name)+" has no ready endpoint.")
	}
	for _, s := range keepStructures(f.Site) {
		if f.Metrics.Nodes[s.Object.Name].DiskPressure {
			return nodeLabel(f.Site, s.Object.Name) + " has DiskPressure: its Reeve evicts huts to clear its ground."
		}
	}
	if len(c.Unread) > 0 {
		return ui.Bridge("Your servants could not read: ", "Kubernites could not read: ") + strings.Join(c.Unread, ", ") + "."
	}
	return ui.Bridge("Every Keep stands, every hut has a Keep, every crew its huts and every depot a hut Fit for Duty.",
		"Every node is Ready, every Pod has a node, every workload its replicas and every Service a ready endpoint.")
}

func liveQuartermasterPage(f Frame) CouncilPage {
	page := QuartermasterPage(f)
	c := f.LiveCouncil
	pending := Section{Title: ui.Bridge("UNHOUSED", "PENDING"), Cols: []int{0, 220}, Empty: ui.Bridge("No hut waits for a Keep.", "No Pod waits for a node.")}
	if liveUnread(f, "pods") {
		pending.Empty = ui.Bridge("Your servants could not read the huts.", "Kubernites could not read the Pods.")
	}
	for _, p := range c.Pending {
		pending.Rows = append(pending.Rows, []Cell{cell(objectName(p.Namespace, p.Name), ToneBad), cell(schedulerSaid(p.Reason), ToneDim)})
	}
	pending.Rows = capRows(pending.Rows, livePendingRows)
	page.Sections[1] = pending
	if !f.Metrics.Augury {
		page.Sections[0] = withoutColumn(page.Sections[0], 2)
	}
	page.Line = quartermasterLine(f, 0)
	if len(c.Pending) > 0 {
		page.Line = pendingCount(len(c.Pending)) + ui.Bridge(": the Quartermaster's reason stands beside each.", ": the scheduler's reason stands beside each.")
	}
	return page
}

func withoutColumn(s Section, col int) Section {
	out := s
	out.Cols = slices.Delete(slices.Clone(s.Cols), col, col+1)
	out.Head = slices.Delete(slices.Clone(s.Head), col, col+1)
	out.Rows = make([][]Cell, len(s.Rows))
	for i, r := range s.Rows {
		if len(r) > col {
			r = slices.Delete(slices.Clone(r), col, col+1)
		}
		out.Rows[i] = r
	}
	return out
}

func liveStewardsPage(f Frame) CouncilPage {
	c := f.LiveCouncil
	s := Section{Title: ui.Bridge("SHORT OF HUTS", "BELOW DESIRED REPLICAS"), Cols: []int{0, 300, 440}, Head: []string{ui.Bridge("CREW", "WORKLOAD"), "KIND", "READY"},
		Empty: ui.Bridge("Every crew, stateful crew and shrine has its huts Fit for Duty.", "Every Deployment, StatefulSet and DaemonSet has its desired replicas ready.")}
	if liveUnread(f, "deployments", "statefulsets", "daemonsets") {
		s.Empty = ui.Bridge("Your servants could not read every crew.", "Kubernites could not read every workload list.")
	}
	for _, w := range c.Short {
		tone := ToneWarn
		if w.Ready == 0 {
			tone = ToneBad
		}
		s.Rows = append(s.Rows, []Cell{cell(objectName(w.Namespace, w.Name), ToneText), cell(liveKind(w.Kind), ToneDim), cell(fmt.Sprintf("%d of %d ready", w.Ready, w.Desired), tone)})
	}
	s.Rows = capRows(s.Rows, liveShortRows)
	line := ui.Bridge(fmt.Sprintf("%d of %d crews, stateful crews and shrines have every hut they want Fit for Duty.", c.WorkloadsFull, c.Workloads),
		fmt.Sprintf("%d of %d workloads have their desired replicas ready.", c.WorkloadsFull, c.Workloads))
	return CouncilPage{Line: line, Sections: []Section{s}}
}

func liveWayfinderPage(f Frame) CouncilPage {
	c := f.LiveCouncil
	s := Section{Title: ui.Bridge("DEPOTS WITH NO HUT FIT FOR DUTY", "SERVICES WITHOUT A READY ENDPOINT"), Cols: []int{0, 300}, Head: []string{ui.Bridge("DEPOT", "SERVICE"), ui.Bridge("GUILD ROLL", "ENDPOINTS")},
		Empty: ui.Bridge("Every depot that looks for a badge has a hut Fit for Duty.", "Every Service with a selector has a ready endpoint.")}
	if liveUnread(f, "services", "endpointslices") {
		s.Empty = ui.Bridge("Your servants could not read every depot and Guild Roll.", "Kubernites could not read the Services or their EndpointSlices.")
	}
	for _, u := range c.Unserved {
		roll := ui.Bridge("its Guild Roll is empty", "no endpoint")
		if u.Endpoints > 0 {
			roll = ui.Bridge(fmt.Sprintf("%d on its Guild Roll, none Fit for Duty", u.Endpoints), fmt.Sprintf("%d, none ready", u.Endpoints))
		}
		s.Rows = append(s.Rows, []Cell{cell(objectName(u.Namespace, u.Name), ToneText), cell(roll, ToneBad)})
	}
	s.Rows = capRows(s.Rows, liveUnservedRows)
	line := ui.Bridge(fmt.Sprintf("%d of %d depots that look for a badge have a hut Fit for Duty.", c.ServicesServed, c.Services),
		fmt.Sprintf("%d of %d Services with a selector have a ready endpoint.", c.ServicesServed, c.Services))
	return CouncilPage{Line: line, Sections: []Section{s}}
}

func liveAugurPage(f Frame) CouncilPage {
	c := f.LiveCouncil
	top := Section{Title: ui.Bridge("BUSIEST HUTS", "BUSIEST PODS"), Cols: []int{0, 300, 420}, Head: []string{ui.Bridge("HUT", "POD"), ui.Bridge("FOOD", "CPU"), ui.Bridge("HOUSING", "MEMORY")},
		Empty: ui.Bridge("No omens: no Observatory answers in this realm.", "metrics-server does not answer in this cluster.")}
	if f.Metrics.Augury {
		top.Empty = ui.Bridge("No hut burns yet.", "metrics-server reports no Pod yet.")
	}
	for _, p := range c.Busiest {
		top.Rows = append(top.Rows, []Cell{cell(objectName(p.Namespace, p.Name), ToneText), cell(ui.Bridge(foodCount(p.CPUMilli), Hands(p.CPUMilli)), ToneText), cell(fmt.Sprintf("%dMi", int(p.MemoryBytes/(1<<20))), ToneText)})
	}
	cellars := Section{Title: ui.Bridge("THE CELLARS", "EPHEMERAL STORAGE"), Cols: []int{0, 220, 470}, Head: []string{ui.Bridge("KEEP", "NODE"), ui.Bridge("PLEDGED", "REQUESTED"), "STATE"},
		Empty: ui.Bridge("No Keep reports its ground.", "No node reports allocatable ephemeral storage.")}
	for _, s := range keepStructures(f.Site) {
		m, ok := f.Metrics.Nodes[s.Object.Name]
		if !ok || m.EphemeralAllocatableBytes <= 0 {
			continue
		}
		state, tone := ui.Bridge("no Cellar Ward", "no DiskPressure"), ToneGood
		if m.DiskPressure {
			state, tone = ui.Bridge("CELLAR WARD", "DISK PRESSURE"), ToneBad
		}
		cellars.Rows = append(cellars.Rows, []Cell{cell(nodeLabel(f.Site, s.Object.Name), ToneText), cell(ephemeralPledge(m), ToneText), cell(state, tone)})
	}
	cellars.Rows = capRows(cellars.Rows, liveCellarRows)
	line := BlindAugur
	switch {
	case !f.Metrics.Augury:
		line = ui.Bridge("The Augur sees nothing: no Observatory answers in this realm.", "metrics-server does not answer: Kubernites shows no usage.")
	case len(c.Busiest) > 0:
		p := c.Busiest[0]
		line = fmt.Sprintf("%s works hardest, eating %s.", objectName(p.Namespace, p.Name), ui.Bridge(foodOf(p.CPUMilli), Hands(p.CPUMilli)+" of CPU"))
	default:
		line = "The omens are calm."
	}
	for _, s := range keepStructures(f.Site) {
		if f.Metrics.Nodes[s.Object.Name].DiskPressure {
			line = nodeLabel(f.Site, s.Object.Name) + " has DiskPressure: its Reeve evicts huts to clear its ground."
			break
		}
	}
	return CouncilPage{Line: line, Sections: []Section{top, cellars}}
}

func ephemeralPledge(m view.NodeMetrics) string {
	story := fmt.Sprintf("%s of %s", view.Grouped(m.EphemeralRequestedBytes/Mib), view.Barrels(float64(m.EphemeralAllocatableBytes/Mib)))
	return ui.Bridge(story, fmt.Sprintf("%s of %s", MemQuantity(m.EphemeralRequestedBytes), MemQuantity(m.EphemeralAllocatableBytes)))
}

func liveCouncilPage(tab string, f Frame) CouncilPage {
	switch tab {
	case TabStewards:
		return liveStewardsPage(f)
	case TabWayfinder:
		return liveWayfinderPage(f)
	case TabAugur:
		return liveAugurPage(f)
	}
	return liveQuartermasterPage(f)
}
