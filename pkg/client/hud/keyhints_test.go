package hud

import (
	"regexp"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

var keyHint = regexp.MustCompile(`\[([^\]]+)\]`)

func liveKeys() map[string]bool {
	out := map[string]bool{"Esc": true, "Space": true, "right click": true, "double click": true, "F1": true, "Tab": true, "Enter": true, "Shift+Enter": true,
		"C": true, "V": true, "O": true, "H": true, "B": true, "N": true, "L": true}
	for _, r := range gridKeyNames {
		out[string(r)] = true
	}
	return out
}

func staleCouncilKey(key string, b Button) bool {
	return key == "C" && !strings.Contains(gridKeyNames, "C") && !strings.Contains(b.Tip, "Council")
}

func hintedKeys(tip string) []string {
	out := []string{}
	for _, m := range keyHint.FindAllStringSubmatch(tip, -1) {
		for _, part := range strings.Split(strings.ReplaceAll(m[1], " or ", ", "), ", ") {
			out = append(out, strings.TrimSpace(part))
		}
	}
	return out
}

func hintFrames(t *testing.T) []Frame {
	t.Helper()
	base := modeFrame(t)
	base.Selection = Selection{}
	hut := base
	hut.Selection = Selection{Kind: "hearth", ID: "default/porters-7d9f-a", Pod: crewPod("porters-7d9f-a")}
	crew := base
	crew.Selection = crewSel()
	stall := depotBoardFrame()
	keep := selectStructure(base, view.Structure{ID: "keep-1", Kind: view.StructKeep, State: view.StateReady})
	return []Frame{base, hut, crew, stall, keep}
}

func TestEveryKeyNamedInATipHasALiveBinding(t *testing.T) {
	live := liveKeys()
	for i, f := range hintFrames(t) {
		for _, page := range []string{PageMain, PageBuild, PageSummon, PageMore} {
			h := New(nil, (&fakeRunner{}).backend())
			h.Update(Input{}, f)
			h.PageID = page
			for _, b := range h.Buttons(f) {
				for _, key := range hintedKeys(b.Tip) {
					if !live[key] || staleCouncilKey(key, b) {
						t.Errorf("frame %d page %s: %s promises [%s]: %q", i, page, b.ID, key, b.Tip)
					}
				}
			}
		}
		h := New(nil, (&fakeRunner{}).backend())
		h.Update(Input{}, f)
		h.Press(CrewPrefix+":mode:roll", f)
		for _, b := range h.Buttons(f) {
			for _, key := range hintedKeys(b.Tip) {
				if !live[key] || staleCouncilKey(key, b) {
					t.Errorf("frame %d rollout page: %s promises [%s]", i, b.ID, key)
				}
			}
		}
	}
}

func TestTheBackTipsNameTheBackCellsRealKey(t *testing.T) {
	key := gridKeyNames[BackCell : BackCell+1]
	for _, tip := range []string{homeTip, homeTipCrew, crewBackTip} {
		if !strings.Contains(tip, "["+"") || !strings.Contains(tip, key) && !strings.Contains(tip, "Esc") {
			t.Errorf("tip %q names neither the back cell's key %s nor Esc", tip, key)
		}
	}
	if strings.Contains(homeTip, "C or right") && key != "C" {
		t.Errorf("the home tip still promises C: %q", homeTip)
	}
}
