package hud

import (
	"fmt"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

const FromHarborBoard = "the harbor board"

var NeckNames = map[string][2]string{
	view.FlowCrew:     {"HUTS", "REPLICAS"},
	view.FlowCPU:      {"FOOD", "CPU"},
	view.FlowMemory:   {"HOUSING", "MEMORY"},
	view.FlowService:  {"DEPOT", "SERVICE"},
	view.FlowQuay:     {"QUAY", "QUEUE"},
	view.FlowCrash:    {"CRASHES", "CRASHLOOP"},
	view.FlowPending:  {"ROOM", "PENDING"},
	view.FlowStarting: {"HUTS STARTING", "PODS STARTING"},
	view.FlowBurst:    {"A BURST OF SHIPS", "A TRAFFIC SPIKE"},
}

var neckLines = map[string][2]string{
	view.FlowCrew:     {"Too few huts: their workers cannot carry every box that arrives.", "Too few replicas serve the requests that arrive."},
	view.FlowCPU:      {"No food is left on your Keeps for another hut.", "No CPU is left to request on your Nodes."},
	view.FlowMemory:   {"No housing is left on your Keeps for another hut.", "No memory is left to request on your Nodes."},
	view.FlowService:  {"A kind of box has no depot to send it on.", "A kind of request has no Service."},
	view.FlowQuay:     {"The quay is full: ships turn boxes away.", "The queue is full: requests are dropped."},
	view.FlowCrash:    {"Workers stop and are relit before they carry.", "Pods crash-loop before they serve."},
	view.FlowPending:  {"New huts wait Unhoused: no Keep has room for their pledge.", "New Pods wait Pending for a Node."},
	view.FlowStarting: {"New huts are starting: their workers carry once the hut is ready.", "New Pods are starting: they serve once Ready."},
	view.FlowBurst:    {"A burst of ships landed more than the crews carry at once: it passes.", "A traffic spike exceeded what the replicas serve: it passes."},
}

func EconomyOf(st view.GameState) view.EconomyView {
	e := st.Economy
	if e.Hand > 0 {
		e.Income = max(e.Income-e.Hand, 0)
		e.Profit -= e.Hand
	}
	return e
}

func NeckName(neck string) string {
	n, ok := NeckNames[neck]
	if !ok {
		return ""
	}
	return ui.Bridge(n[0], n[1])
}

func NeckLine(neck string) string {
	n, ok := neckLines[neck]
	if !ok {
		return ""
	}
	return ui.Bridge(n[0], n[1])
}

func ShortFixLabel(o view.CommandOffer) string {
	a := o.Action
	switch a.Action {
	case "openDepot":
		if d, ok := soulstone.DepotFor(a.Kind); ok {
			return ui.Bridge("OPEN "+strings.ToUpper(d.Service), "EXPOSE "+strings.ToUpper(d.Crew))
		}
	case "raiseCrew":
		name := strings.ToUpper(a.Name)
		return ui.Bridge("RAISE "+name, "CREATE "+name)
	case "buyKeep":
		return ui.Bridge("BUY A KEEP", "ADD A NODE")
	case view.ActScaleCrew:
		return fmt.Sprintf("SCALE TO %d", a.Replicas)
	case "buySoulstone":
		return ui.Bridge("ENDOW LEANER", "BUY LEANER IMAGE")
	case view.ActCrewResources:
		if a.Crew.Limits != "" {
			return ui.Bridge("LOOSEN BINDING", "RAISE LIMIT")
		}
		return ui.Bridge("PLEDGE LESS", "LOWER REQUEST")
	case view.ActMoveWorker:
		return ui.Bridge("RAZE A HUT", "DELETE A POD")
	case view.ActCrewImage:
		return ui.Bridge("ROLL OUT", "SET IMAGE")
	case view.ActCloseBerth:
		return "CLOSE A BERTH"
	}
	return FixLabel(o)
}

func FixLabel(o view.CommandOffer) string {
	label := strings.TrimSpace(o.Label)
	if _, rest, ok := strings.Cut(label, ": "); ok && rest != "" {
		label = rest
	}
	label, _, _ = strings.Cut(label, " (")
	return strings.ToUpper(label)
}
