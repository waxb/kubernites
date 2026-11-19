package app

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/view"
)

const readyPrefix = "ready:"

type stepCue struct {
	id     string
	waited bool
	chimed string
}

func (s *Session) stepCues() []sound.SFX {
	o, ok := currentStepOf(s.State)
	if !ok {
		return nil
	}
	ready := strings.HasPrefix(o.Progress, readyPrefix)
	if o.ID != s.step.id {
		s.step = stepCue{id: o.ID, waited: !ready, chimed: s.step.chimed}
		return nil
	}
	if !ready {
		s.step.waited = true
		return nil
	}
	if s.step.waited && s.step.chimed != o.ID {
		s.step.chimed = o.ID
		return []sound.SFX{sound.SFXReady}
	}
	return nil
}

func currentStepOf(gs view.GameState) (view.ObjectiveView, bool) {
	for _, o := range gs.Objectives {
		if !o.Done {
			return o, true
		}
	}
	return view.ObjectiveView{}, false
}
