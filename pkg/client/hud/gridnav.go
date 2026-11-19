package hud

import (
	"strings"
)

var (
	boardOnly   = []string{"slider", "buykeep", "step:", "pick:", "close"}
	objectHeads = []string{ActionPrefix + ":", "keep:retire", "sel:details", "sel:logs", LandButton, linePrefix + ":"}
)

func objectTarget(target string) bool {
	for _, p := range objectHeads {
		if strings.HasPrefix(target, p) {
			return true
		}
	}
	return false
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func orderMode(rest string) string {
	head, tail, _ := strings.Cut(rest, ":")
	switch {
	case head == "roll" || head == "history":
		return CrewModeRoll
	case head == "image":
		return CrewModeImage
	case head == "res":
		return CrewModeRes
	case head == "horn" && tail != "":
		return CrewModeHorn
	case head == CrewModeProbes && tail != "":
		return CrewModeProbes
	}
	return CrewModeChain
}

func modeOpener(mode string) string {
	if mode == CrewModeProbes {
		return CrewPrefix + ":" + CrewModeProbes
	}
	return CrewModeOpen + mode
}

func crewSteps(target string, st NavState) ([]string, bool) {
	rest, ok := strings.CutPrefix(target, CrewPrefix+":")
	if _, place := CrewPlace(target); !ok || place || rest == "board" {
		return nil, false
	}
	if hasAnyPrefix(rest, boardOnly) {
		return boardSteps(target, st), true
	}
	out, mode := []string{}, st.Mode
	if !st.Crew {
		out, mode = append(out, ChainButton), CrewModeChain
	}
	if want := orderMode(rest); want != mode && !(rest == "makeroom" && mode != CrewModeChain) {
		if mode != CrewModeChain {
			out = append(out, crewBack)
		}
		if want != CrewModeChain {
			out = append(out, modeOpener(want))
		}
	}
	return append(out, target), true
}

func boardSteps(target string, st NavState) []string {
	switch {
	case !st.Chain:
		return []string{ChainButton, target}
	case st.Crew && st.Mode != CrewModeChain:
		return []string{crewBack, target}
	}
	return []string{target}
}

func portSteps(target string, st NavState) ([]string, bool) {
	_, id, ok := PortTarget(target)
	if !ok || st.Harbor && st.Haven == id {
		return nil, false
	}
	return []string{"haven:" + id, target}, true
}
