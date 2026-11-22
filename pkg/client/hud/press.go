package hud

import (
	"strconv"
	"strings"

	"github.com/waxb/kubernites/pkg/view"
)

var pageButtons = map[string]string{
	"cmd:build": PageBuild, "cmd:summon": PageSummon, CmdMore: PageMore,
}

var overlayButtons = map[string]string{
	"cmd:grimoire": OverlayGrimoire, "cmd:soulvault": OverlaySoulvault,
	"cmd:chronicles": OverlayChronicles, "cmd:age": OverlayAge, "top:menu": OverlayMenu, "well:tech": OverlayGrimoire,
	"well:yaml": OverlayScriptorium, "cmd:voyage": OverlayVoyage, "cmd:council": OverlayCouncil,
	LedgerButton: OverlayLedger, "menu:resume": OverlayMenu,
}

var simpleIntents = map[string]Intent{
	"top:pause":       {Kind: IntentPause},
	PinnedResume:      {Kind: IntentPause},
	"menu:save":       {Kind: IntentSave},
	"menu:mute":       {Kind: IntentMute},
	"menu:title":      {Kind: IntentTitle},
	"menu:reduce":     {Kind: IntentReduce},
	"menu:edge":       {Kind: IntentEdge},
	"menu:names":      {Kind: IntentNames},
	StoneMenuID:       {Kind: IntentStone},
	"menu:fullscreen": {Kind: IntentFullscreen},
	"menu:coins":      {Kind: IntentCoinFlights},
	"menu:pause":      {Kind: IntentPausePolicy},
	"title:new":       {Kind: IntentNew},
	"title:load":      {Kind: IntentLoad},
	"title:quit":      {Kind: IntentQuit},
	"title:live":      {Kind: IntentLive},
	TitleLearn:        {Kind: IntentTutorial, Command: TutorialOpen},
	"cmd:door":        act(view.Action{Action: "doorHearth"}),
	BerthButton:       act(view.Action{Action: view.ActOpenBerth}),
	CloseBerthButton:  act(view.Action{Action: view.ActCloseBerth}),
	"age:advance":     act(view.Action{Action: "ageUp"}),
}

func act(a view.Action) Intent { return Intent{Kind: IntentAction, Action: a} }

func weave(id string) view.Action { return view.Action{Action: "weave", ID: id} }

func (h *HUD) Press(id string, f Frame) []Intent {
	h.spotPressed(id)
	h.dropUsedNews(id, f)
	click := Intent{Kind: IntentSound, Command: "click"}
	if page, ok := pageButtons[id]; ok {
		h.PageID = page
		return []Intent{click}
	}
	if overlay, ok := overlayButtons[id]; ok {
		h.toggleOverlay(overlay)
		return append(h.readAgeIntents(f), click)
	}
	if id == MenuPanels {
		return h.resetPanels()
	}
	if id == menuLiveChoices {
		h.OverlayID = OverlayLiveChoices
		return []Intent{click, {Kind: IntentLiveChoices, Command: "open"}}
	}
	if id == DawnMore {
		h.Dawn.More = !h.Dawn.More
		return []Intent{click}
	}
	if id == "menu:experience" {
		return h.beginAs(nextExperience(f.Experience()), f, click)
	}
	if id == "menu:calm" {
		h.OverlayID = OverlayNone
		mode := "on"
		if f.State.CalmRealm {
			mode = "off"
		}
		return []Intent{click, act(view.Action{Action: view.ActSetCalm, ID: mode})}
	}
	if id == TrialsButton {
		h.OpenTrials(h.Trial.Focus)
		return []Intent{click}
	}
	if id == TutorialButton {
		h.OpenChapters(h.Yard.tmap.Record)
		return []Intent{click}
	}
	if id == HelpButton {
		return h.pressHelp()
	}
	if cmd, ok := strings.CutPrefix(id, volumePrefix); ok {
		return volumeIntents(cmd, click)
	}
	if in, ok := simpleIntents[id]; ok {
		if strings.HasPrefix(id, "menu:") {
			h.OverlayID = OverlayNone
		}
		return []Intent{in}
	}
	prefix, rest, _ := strings.Cut(id, ":")
	if prefix == IntentNotice {
		return h.PressNotice(rest, f)
	}
	if handler, ok := prefixHandlers[prefix]; ok {
		return handler(h, rest, f)
	}
	return []Intent{click}
}

var prefixHandlers = map[string]func(*HUD, string, Frame) []Intent{
	"cmd":             (*HUD).pressCommand,
	"harbor":          PressPort,
	"freight":         (*HUD).pressFreight,
	"tab":             (*HUD).pressTab,
	"well":            (*HUD).pressWell,
	"speed":           pressSpeed,
	"build":           pressBuild,
	"summon":          pressSummon,
	"sel":             (*HUD).pressSelection,
	"det":             (*HUD).pressDetails,
	"grim":            (*HUD).pressGrimoire,
	"kmap":            (*HUD).PressKnowledge,
	"learn":           PressLearn,
	"vault":           pressVault,
	acadPrefix:        (*HUD).pressAcademy,
	"chron":           pressChronicle,
	"overlay":         (*HUD).pressClose,
	"obj":             (*HUD).pressObjective,
	"scr":             (*HUD).pressScript,
	"stone":           (*HUD).pressStone,
	"reforge":         (*HUD).pressReforge,
	"loom":            pressLoom,
	"info":            func(*HUD, string, Frame) []Intent { return nil },
	liveQuietPrefix:   (*HUD).pressLiveTrouble,
	liveForwardPrefix: (*HUD).pressLiveForward,
	"card":            (*HUD).pressCard,
	"siege":           (*HUD).pressSiege,
	"welcome":         (*HUD).pressWelcome,
	"road":            (*HUD).pressRoad,
	"alert":           (*HUD).pressAlert,
	"voyage":          (*HUD).pressVoyage,
	"keep":            (*HUD).pressKeep,
	"blip":            (*HUD).pressBlip,
	"age":             (*HUD).pressAge,
	"council":         (*HUD).pressCouncil,
	"intro":           (*HUD).PressIntro,
	questPrefix:       (*HUD).PressQuest,
	"tiding":          (*HUD).PressTiding,
	"act":             (*HUD).PressAction,
	"lens":            (*HUD).pressLens,
	"lesson":          (*HUD).pressLesson,
	"pick":            (*HUD).pressPick,
	trialPrefix:       (*HUD).PressTrial,
	bookPrefix:        (*HUD).pressBook,
	tipPrefix:         (*HUD).pressTip,
}

func pressLoom(_ *HUD, id string, _ Frame) []Intent {
	return []Intent{act(weave(id))}
}

func (h *HUD) toggleOverlay(o string) {
	if h.OverlayID == o {
		h.OverlayID = OverlayNone
		return
	}
	h.OverlayID = o
}

func (h *HUD) pressCommand(rest string, f Frame) []Intent {
	switch rest {
	case "back":
		h.PageID = pageParent(h.PageID)
	case "siege":
		return h.OpenSiege(f)
	case "crew":
		return h.OpenRaise(f)
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) pressTab(rest string, _ Frame) []Intent {
	h.TabID = rest
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) pressWell(rest string, _ Frame) []Intent {
	if rest == "stone" {
		h.OpenStone("")
	}
	return nil
}

func (h *HUD) pressClose(rest string, _ Frame) []Intent {
	if rest == "close" {
		h.OverlayID = OverlayNone
	}
	return nil
}

func pressSpeed(_ *HUD, rest string, f Frame) []Intent {
	v, err := strconv.ParseFloat(rest, 64)
	if err != nil {
		return nil
	}
	set := act(view.Action{Action: "speed", Speed: v})
	if f.Paused {
		return []Intent{set, {Kind: IntentPause}}
	}
	return []Intent{set}
}

func pressBuild(h *HUD, kind string, _ Frame) []Intent {
	return []Intent{{Kind: IntentPlace, Target: kind}}
}

func pressSummon(_ *HUD, image string, _ Frame) []Intent {
	return []Intent{ActFrom(view.Action{Action: "summon", Soulstone: image}, FromCommandCard)}
}

func PressLearn(h *HUD, id string, _ Frame) []Intent {
	h.Grimoire, h.Kmap.Focus, h.Kmap.Held = id, id, true
	h.Kmap.heldAt, h.Kmap.heldSet = [2]int{h.MouseX, h.MouseY}, true
	return []Intent{act(view.Action{Action: "unlock", ID: id})}
}

func pressVault(_ *HUD, image string, _ Frame) []Intent {
	if image == "owned" {
		return nil
	}
	return []Intent{act(view.Action{Action: "buySoulstone", Image: image})}
}

func pressChronicle(_ *HUD, rest string, _ Frame) []Intent {
	id, answer, _ := strings.Cut(rest, ":")
	return []Intent{act(view.Action{Action: "chronicle", ID: id, Answer: answer})}
}

func (h *HUD) pressGrimoire(id string, _ Frame) []Intent {
	h.Grimoire = id
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) readClosedAge(f Frame) []Intent {
	m := f.State.AgeMoment
	unread := m != nil && !m.Read && m.Age == f.State.Age
	switch {
	case !unread:
		h.ageOpen = false
	case h.OverlayID == OverlayAge:
		h.ageOpen = true
	case h.ageOpen:
		h.ageOpen = false
		return []Intent{act(view.Action{Action: view.ActReadAge})}
	}
	return nil
}

func (h *HUD) readAgeIntents(f Frame) []Intent {
	m := f.State.AgeMoment
	if h.OverlayID != OverlayAge || m == nil || m.Read {
		return nil
	}
	return []Intent{act(view.Action{Action: view.ActReadAge})}
}
