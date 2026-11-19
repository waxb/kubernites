package hud

import (
	"fmt"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func raisesForemen(f Frame) bool {
	return f.State.RaiseKeeper == view.KeeperReplicaSet
}

func foremenOnly(f Frame) bool {
	return raisesForemen(f) && !f.State.Unlocks[view.DeploymentsPage]
}

func crewKeeper(f Frame, crew string) string {
	for _, c := range f.State.Crews {
		if c.Name == crew {
			return c.Keeper
		}
	}
	return ""
}

func raiseTitle(f Frame) string {
	if raisesForemen(f) {
		return ui.Bridge("RAISE A SHIFT", "CREATE A REPLICASET")
	}
	return ui.Bridge("RAISE A CREW", "CREATE A DEPLOYMENT")
}

func foremanAction(r RaiseRow, image string, workers int) view.Action {
	return view.Action{Action: "raiseMuster", Name: r.Crew, Soulstone: image, Replicas: workers}
}

func foremanChooserLine(r RaiseRow, workers int) string {
	return ui.Bridge(fmt.Sprintf("A shift keeps exactly %s standing: it takes in the bare ones wearing app=%s, takes down any beyond %d, and raises the rest.", plural(workers, "hut", "huts"), r.Crew, workers),
		fmt.Sprintf("A ReplicaSet keeps exactly %s running: it adopts the Pods labelled app=%s, deletes any beyond %d, and creates the rest from its template.", plural(workers, "Pod", "Pods"), r.Crew, workers))
}

func foremanTip(r RaiseRow, workers int) string {
	return fmt.Sprintf(ui.Bridge("Writes a shift (a ReplicaSet) that keeps %d %s: no kubectl word makes one, so it is written as a manifest.",
		"Applies a ReplicaSet manifest keeping %d %s: kubectl has no imperative create for ReplicaSets."), workers, r.Crew)
}

const foremanWorkers = view.HandPorterTarget

func foremanWord(rows []RaiseRow, crew, foreman string) string {
	if len(rows) > 0 && rows[0].Foreman {
		return foreman
	}
	return crew
}

func crewKept(c model.CrewChain) bool {
	return !c.Bare || c.Name != ""
}

func chainKeeper(c model.CrewChain) string {
	if c.Bare {
		return view.KeeperReplicaSet
	}
	return ""
}

const deployCell = 2

func deployable(c model.CrewChain, f Frame) bool {
	return c.Bare && c.Name != "" && f.State.Unlocks[view.DeploymentsPage]
}

func deployButton(c model.CrewChain, f Frame) Button {
	b := order("deploy", ui.Bridge("RAISE CREW", "DEPLOY"), ui.Bridge("Raise a crew: it takes this shift in as it stands, and nothing restarts. A crew runs one shift per class and swaps them hut by hut.",
		"Create a Deployment whose selector matches this ReplicaSet: it adopts the ReplicaSet as it stands, and no Pod restarts. A Deployment rolls out each new image."),
		view.DeployCommand(c.Name, c.Image, c.Desired), f)
	b.Icon = "icon/roll"
	return b
}

func deployForeman(c model.CrewChain) []Intent {
	a := view.Action{Action: view.ActDeployCrew, Name: c.Name}
	return []Intent{ActFrom(a, FromCrewCard)}
}

func crewButtonFor(f Frame) Button {
	b := crewButton
	if raisesForemen(f) {
		b.Label = "SHIFT"
	}
	if len(f.State.Crews) > 0 && foremenOnly(f) {
		b.Tip = ui.Bridge("Opens your shifts: scale one or open its board.", "Opens the ReplicaSets you run: scale one or open its board.")
		return b
	}
	if len(f.State.Crews) > 0 {
		b.Tip = ui.Bridge("Opens THE CREWS: every crew you keep, to scale one or open its board. Raise another from there.",
			"Opens DEPLOYMENTS: every Deployment you run, to scale one or open its board. Create another from there.")
		return b
	}
	if !raisesForemen(f) {
		return b
	}
	b.Tip = ui.Bridge("Opens RAISE A SHIFT: pick a trade and the count it keeps. It takes in the bare ones wearing its badge.",
		"Opens CREATE A REPLICASET: pick a trade and its replicas. It adopts the bare Pods its selector matches.")
	return b
}
