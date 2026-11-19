package hud

import (
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	keyRollAdd    = "roll-add"
	keyRollStrike = "roll-strike"
	keyLookFor    = "look-for"
	keyTearUp     = "tear-up"
	rollAddTip    = "Put this hut on the depot's roll: the depot hands its boxes in turn to every hut on its roll, and to no other. You keep this roll by hand (an EndpointSlice you write)."
	strikeTip     = "Strike a fallen hut off the roll: the depot keeps sending boxes to an address nobody answers until you do, and they spoil there."
	lookForTip    = "Let the depot look for the badge app=porters (a selector): the Stewards then list every ready porter on its roll and strike each fallen one by themselves."
	lookForLocked = "Learn Badges in the Grimoire first: a depot that looks for a badge is its power."
	tearUpTip     = "Tear up your hand roll (delete the EndpointSlice you wrote): the Stewards keep only the roll they write, and a hut that falls would stay on yours."
	tearUpEarly   = "The Stewards keep no roll for this depot yet: tear yours up and no hut would get a box. LOOK FOR a badge first."
	idleLine      = "IDLE: on no depot's roll, so no box comes to it"
)

func kubectlless(command string) string {
	return strings.TrimPrefix(command, "kubectl ")
}

func rollIdle(roll view.RollView, name string) (view.RollEntry, bool) {
	for _, e := range roll.Idle {
		if e.Hut == name {
			return e, true
		}
	}
	return view.RollEntry{}, false
}

func withRollActions(acts []CardAction, p model.PodView, f Frame) []CardAction {
	roll := f.State.Roll
	e, idle := rollIdle(roll, p.Name)
	if !roll.Hand || !idle || p.Namespace != defaultNS {
		return acts
	}
	add := CardAction{Key: keyRollAdd, Label: "ADD TO ROLL", Command: kubectlless(view.EnrollCommand(e.Address, p.Name)), Tip: rollAddTip,
		Act: view.Action{Action: view.ActEnroll, ID: p.Name}}
	return append([]CardAction{add}, acts...)
}

func firstGone(roll view.RollView) (view.RollEntry, int, bool) {
	for i, e := range roll.Entries {
		if !e.Live {
			return e, i, true
		}
	}
	return view.RollEntry{}, 0, false
}

func withRollDepot(acts []CardAction, service string, f Frame) []CardAction {
	roll := f.State.Roll
	if service != parcelsService || !roll.Hand && !roll.Bare {
		return acts
	}
	out := []CardAction{}
	if gone, i, ok := firstGone(roll); ok {
		out = append(out, CardAction{Key: keyRollStrike, Label: "STRIKE", Command: kubectlless(view.StrikeCommand(i, gone.Address)), Tip: gone.Hut + ": " + strikeTip,
			Act: view.Action{Action: view.ActStrike, ID: gone.Address}})
	}
	if !roll.LooksFor {
		look := CardAction{Key: keyLookFor, Label: "LOOK FOR", Command: kubectlless(view.LookForCommand()), Tip: lookForTip,
			Act: view.Action{Action: view.ActLookFor}}
		if !f.State.Unlocks[view.BadgesPage] {
			look.Disabled, look.Tip = true, lookForLocked
		}
		out = append(out, look)
	}
	if roll.Hand {
		tear := CardAction{Key: keyTearUp, Label: "TEAR UP", Command: kubectlless(view.TearUpCommand()), Tip: tearUpTip,
			Act: view.Action{Action: view.ActTearUpRoll}}
		if !roll.LooksFor {
			tear.Disabled, tear.Tip = true, tearUpEarly
		}
		out = append(out, tear)
	}
	return append(out, acts...)
}

const parcelsService = "parcels"

func rollLines(p model.PodView, f Frame) []Line {
	roll := f.State.Roll
	if _, idle := rollIdle(roll, p.Name); !idle || !roll.Hand {
		return nil
	}
	return []Line{{Text: idleLine, Clr: ui.Palette.RedText}}
}

func goneMembers(f Frame) []string {
	out := []string{}
	for _, e := range f.State.Roll.Entries {
		if !e.Live {
			out = append(out, e.Hut)
		}
	}
	return out
}

func isGone(f Frame, name string) bool {
	return slices.Contains(goneMembers(f), name)
}
