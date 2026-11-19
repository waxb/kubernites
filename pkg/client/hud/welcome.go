package hud

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	OverlayWelcome = "welcome"
	WelcomeInfo    = "info"
	WelcomeRoad    = "road"
	legendColumn   = 262
	legendNameW    = 96
)

var WelcomeBox = ui.Rect{X: Width/2 - 300, Y: TopH + 24, W: 600, H: 310}

var WelcomeStory = []string{
	"Boxes are requests. Ships land them on your quay, where they wait.",
	"Carry one and it pays gold. Leave one too long and its seal turns red:",
	"it spoils, a dropped request. The share delivered fresh is your reliability.",
	"",
	"For now YOU carry them: click the parcels on the quay.",
}

func welcomeSpend() [2]string {
	return [2]string{
		ui.Bridge("Spend the gold on buildings that raise crews of workers, until the crews", "Spend the money on buildings that are real Kubernetes parts, until crews"),
		ui.Bridge("carry every box for you.", "of Pods carry every box for you."),
	}
}

const (
	goalMark     = "GOAL:"
	WelcomeInset = 24
	welcomeGoal  = goalMark + " reach the Feudal Age: its Age card lists all it asks."
)

func WelcomeGoal(f Frame) string {
	for _, a := range f.State.Road.Ages {
		if a.ID == view.AgeFeudal && a.Asks != "" {
			return goalMark + " reach the Feudal Age. It asks " + a.Asks + "."
		}
	}
	return welcomeGoal
}

var WelcomeLegend = [][3]string{
	{"Box", "a request", "what a user asks for"}, {"Harbor", "where requests arrive", "where ships dock"},
	{"Keep", "a Node (a machine)", "a machine for your huts"}, {"Codex", "etcd (the database)", "the realm's database"},
	{"Chancery", "kube-apiserver", "where every order is filed"}, {"Hut", "a Pod (your app)", "your app, running"},
	{"Shift", "a ReplicaSet", "keeps a count of huts"}, {"Speaking Stone", "kubectl", "speak orders to the realm"},
	{"Crew", "a Deployment", "huts of one trade"}, {"Forum", "Pods with no Node yet", "where huts wait for a Keep with room"},
}

type experience struct {
	id      string
	key     ebiten.Key
	button  string
	label   string
	caption string
	tip     string
}

var experiences = []experience{
	{id: view.GuidanceNovice, key: ebiten.Key1, button: "welcome:begin", label: "1  NEW TO STRATEGY",
		caption: "The full guide: a step at a time, a card for each new idea.",
		tip:     "Start with the full guide [1 or Enter]"},
	{id: view.GuidanceStrategist, key: ebiten.Key2, button: "welcome:strategy", label: "2  I KNOW STRATEGY GAMES",
		caption: "No step holds you back. Advice comes when a situation needs it.",
		tip:     "Start without step holds: advice only when a situation needs it [2]"},
	{id: view.GuidanceKubernetes, key: ebiten.Key3, button: "welcome:kube", label: "3  I KNOW KUBERNETES",
		caption: "A light guide, NAMES on, the Kubernetes side first.",
		tip:     "Start with a light guide and NAMES on: every object under its real name [3]"},
}

var experienceNames = map[string]string{
	view.GuidanceNovice: "NEW", view.GuidanceStrategist: "STRATEGY", view.GuidanceKubernetes: "KUBERNETES",
}

func nextExperience(id string) string {
	for i, e := range experiences {
		if e.id == id {
			return experiences[(i+1)%len(experiences)].id
		}
	}
	return view.GuidanceNovice
}

func experienceOf(button string) (experience, bool) {
	for _, e := range experiences {
		if e.button == button {
			return e, true
		}
	}
	return experience{}, false
}

func (f Frame) Experience() string {
	if f.State.Guidance == "" {
		return view.GuidanceNovice
	}
	return f.State.Guidance
}

func (f Frame) KubernetesFirst() bool {
	return f.Experience() == view.GuidanceKubernetes
}

func (f Frame) gameSide() int {
	if f.KubernetesFirst() {
		return 1
	}
	return 0
}

func WelcomeInfoRect() ui.Rect {
	r := WelcomeBox
	return ui.Rect{X: r.X + WelcomeInset, Y: r.Y + 128, W: r.W - 2*WelcomeInset, H: 18}
}

func welcomeStrip() ui.Rect {
	r := WelcomeBox
	return ui.Rect{X: r.X + 12, Y: r.Y + 156, W: r.W - 24, H: 140}
}

func welcomeChoiceRects() []ui.Rect {
	strip := welcomeStrip()
	const w, gap = 168, 6
	total := 3*w + 2*gap
	x := strip.X + (strip.W-total)/2
	out := make([]ui.Rect, 3)
	for i := range out {
		out[i] = ui.Rect{X: x + i*(w+gap), Y: strip.Y + 30, W: w, H: 22}
	}
	return out
}

func (h *HUD) welcomeButtons(_ Frame) []Button {
	r := WelcomeBox
	if h.welcomePage != "" {
		return []Button{{ID: "welcome:back", Rect: ui.Rect{X: r.X + r.W/2 - 60, Y: r.Y + r.H - 30, W: 120, H: 18}, Label: "BACK [ESC]",
			Tip: "Back to the choice of experience"}, closeButtonAs(r, "welcome:back", "Back [Esc]")}
	}
	bar := WelcomeInfoRect()
	half := (bar.W - 8) / 2
	out := []Button{
		{ID: "welcome:info", Rect: ui.Rect{X: bar.X, Y: bar.Y, W: half, H: bar.H}, Label: "REALM INFORMATION", Tip: "How the realm works, and what is what"},
		{ID: "welcome:road", Rect: ui.Rect{X: bar.X + half + 8, Y: bar.Y, W: half, H: bar.H}, Label: "ROAD OF AGES", Tip: "The Ages the demo holds and the ones the full release adds"},
	}
	for i, e := range experiences {
		out = append(out, Button{ID: e.button, Rect: welcomeChoiceRects()[i], Label: e.label, Tip: e.tip})
	}
	return append(out, closeButtonAs(r, "welcome:close", "Start with the full guide [Esc]"))
}

func (h *HUD) pressWelcome(rest string, f Frame) []Intent {
	click := Intent{Kind: IntentSound, Command: "click"}
	switch rest {
	case WelcomeInfo, WelcomeRoad:
		h.welcomePage = rest
		return []Intent{click}
	case "back":
		h.welcomePage = ""
		return []Intent{click}
	case "close":
		rest = "begin"
	}
	e, ok := experienceOf("welcome:" + rest)
	if !ok {
		return []Intent{click}
	}
	return h.beginAs(e.id, f, click)
}

func (h *HUD) beginAs(id string, f Frame, click Intent) []Intent {
	h.OverlayID, h.welcomePage = OverlayNone, ""
	out := []Intent{click}
	if id != view.GuidanceNovice {
		out = append(out, act(view.Action{Action: view.ActSetGuidance, ID: id}))
	}
	if id == view.GuidanceKubernetes && !f.TrueNames {
		out = append(out, Intent{Kind: IntentNames})
	}
	return out
}

func (h *HUD) welcomeKeys(in Input, f Frame) ([]Intent, bool) {
	if h.welcomePage != "" {
		if in.Pressed(ebiten.KeyEscape) || in.Pressed(ebiten.KeyBackspace) {
			return h.pressWelcome("back", f), true
		}
		return nil, len(in.Keys) > 0
	}
	for _, e := range experiences {
		if in.Pressed(e.key) {
			return h.beginAs(e.id, f, Intent{Kind: IntentSound, Command: "click"}), true
		}
	}
	if in.Pressed(ebiten.KeyEnter) || in.Pressed(ebiten.KeyEscape) || in.Pressed(ebiten.KeySpace) {
		return h.pressWelcome("begin", f), true
	}
	return nil, len(in.Keys) > 0
}

func (h *HUD) OpenWelcome(page string) {
	h.OverlayID, h.welcomePage = OverlayWelcome, page
}
