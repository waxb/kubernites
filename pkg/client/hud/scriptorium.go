package hud

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

const (
	TemplateReplicaSet = "REPLICASET"
	TemplateCrew       = "CREW"
	TemplateHearth     = "HEARTH"
	TemplateDepot      = "DEPOT"
	TemplateScroll     = "SCROLL"
	TemplateBlank      = "BLANK"

	resultFade = 6.0
	railW      = 160
	gutterW    = 18
	footerH    = 22
)

var templateOrder = []string{TemplateCrew, TemplateHearth, TemplateDepot, TemplateScroll, TemplateBlank}

func porterPodSpecK8s() string {
	return `    spec:
      containers:
      - name: porter
        image: porter:1.0`
}

var Templates = scriptoriumK8sTemplates()

func scriptoriumK8sTemplates() map[string]string {
	return map[string]string{
		TemplateReplicaSet: `apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: porters
spec:
  replicas: 3
  selector:
    matchLabels:
      app: porters
  template:
    metadata:
      labels:
        app: porters
` + porterPodSpecK8s(),
		TemplateCrew: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: porters
spec:
  replicas: 3
  selector:
    matchLabels:
      app: porters
  template:
    metadata:
      labels:
        app: porters
` + porterPodSpecK8s(),
		TemplateHearth: `apiVersion: v1
kind: Pod
metadata:
  name: porter-declared
  labels:
    app: porters
spec:
  containers:
  - name: porter
    image: porter:1.0`,
		TemplateDepot: `apiVersion: v1
kind: Service
metadata:
  name: parcels
spec:
  selector:
    app: porters
  ports:
  - port: 80
    targetPort: 8080`,
		TemplateScroll: `apiVersion: v1
kind: ConfigMap
metadata:
  name: realm-charter
data:
  motto: only what is declared survives`,
		TemplateBlank: "",
	}
}

var storyTips = map[string]string{
	TemplateCrew:   "A crew of porters that rolls out new classes",
	TemplateHearth: "A single hut",
	TemplateDepot:  "The parcels depot: it sends parcels to every porter wearing the porters badge",
	TemplateScroll: "A scroll of settings",
	TemplateBlank:  "Write a decree from nothing",
}

func templateTipsK8s() map[string]string {
	return map[string]string{
		TemplateCrew:   "A crew of porters that rolls out new classes (Deployment)",
		TemplateHearth: "A single hut (Pod)",
		TemplateDepot:  "The parcels depot: it sends parcels to every porter wearing app=porters (Service)",
		TemplateScroll: "A scroll of settings (ConfigMap)",
		TemplateBlank:  "Write a decree from nothing",
	}
}

func templateTip(name string) string {
	return ui.Bridge(storyTips[name], templateTipsK8s()[name])
}

func templateText(name string) string {
	if t := Templates[name]; ui.StoryNaming() && t != "" {
		return lexicon.StoryManifest(t)
	}
	return Templates[name]
}

var errorLine = regexp.MustCompile(`line (\d+)`)

type Script struct {
	Editor   Editor
	Template string
	Result   []string
	OK       bool
	ResultAt float64
	ErrLine  int
	Pending  bool
	Confirm  string
	Story    bool
	Live     func() (string, bool)
	Warn     string
	Approved bool
	Opened   bool
	Beside   bool
	Raw      bool
	beside   besideCache
}

func NewScript() Script {
	return Script{Editor: NewEditor(Templates[TemplateCrew]), Template: TemplateCrew}
}

func (s *Script) textOf(name string) string {
	if name == TemplateCrew && s.Live != nil {
		if t, ok := s.Live(); ok {
			return t
		}
	}
	return templateText(name)
}

func (s *Script) Load(name string) {
	s.Editor = NewEditor(s.textOf(name))
	s.Template, s.Confirm, s.ErrLine, s.Result, s.Story = name, "", 0, nil, ui.StoryNaming()
	s.Warn, s.Approved = "", false
}

func (h *HUD) syncScript() {
	s := &h.Script
	open := h.OverlayID == OverlayScriptorium
	if !open && h.liveEdit != nil {
		h.closeLiveEdit()
	}
	if open && !s.Opened && !s.Editor.Dirty {
		s.Load(s.Template)
	}
	s.Opened = open
}

func (s *Script) Sync(story bool) {
	if s.Story == story || s.Raw {
		return
	}
	if !s.Editor.Dirty {
		s.Load(s.Template)
		s.Story = story
		return
	}
	convert := lexicon.ManifestFromStory
	if story {
		convert = lexicon.StoryManifest
	}
	next := NewEditor(convert(s.Editor.Text()))
	next.Dirty = true
	s.Editor, s.Story = next, story
}

func (s *Script) Choose(name string) {
	if s.Editor.Dirty && name != s.Template {
		s.Confirm = name
		return
	}
	s.Load(name)
}

func ErrorLine(output string) int {
	m := errorLine.FindStringSubmatch(output)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

func (h *HUD) ApplyScript() []Intent {
	s := &h.Script
	s.Pending = false
	if h.backend.Apply == nil || s.Editor.Empty() {
		s.Result, s.OK, s.ResultAt = []string{"error: the decree is empty"}, false, h.Now
		return []Intent{{Kind: IntentSound, Command: "error"}}
	}
	if w := h.pendingWarning(); w != "" {
		s.Warn = w
		return []Intent{{Kind: IntentSound, Command: "click"}}
	}
	s.Approved = false
	out, ok := h.backend.Apply(s.Editor.Text())
	s.Result, s.OK, s.ResultAt, s.ErrLine = SplitOutput(out), ok, h.Now, 0
	sound := "rollout"
	if !ok {
		s.ErrLine, sound = ErrorLine(out), "error"
	}
	return []Intent{{Kind: IntentKubectl, Command: "apply"}, {Kind: IntentSound, Command: sound}}
}

var editorKeys = map[ebiten.Key]func(*Editor){
	ebiten.KeyBackspace:  (*Editor).Backspace,
	ebiten.KeyDelete:     (*Editor).Delete,
	ebiten.KeyTab:        (*Editor).Tab,
	ebiten.KeyArrowLeft:  func(e *Editor) { e.Move(0, -1) },
	ebiten.KeyArrowRight: func(e *Editor) { e.Move(0, 1) },
	ebiten.KeyArrowUp:    func(e *Editor) { e.Move(-1, 0) },
	ebiten.KeyArrowDown:  func(e *Editor) { e.Move(1, 0) },
	ebiten.KeyHome:       (*Editor).Home,
	ebiten.KeyEnd:        (*Editor).End,
	ebiten.KeyPageUp:     func(e *Editor) { e.Move(-scriptVisible(), 0) },
	ebiten.KeyPageDown:   func(e *Editor) { e.Move(scriptVisible(), 0) },
}

func (h *HUD) scriptKeys(in Input, _ Frame) []Intent {
	s := &h.Script
	s.Sync(ui.StoryNaming())
	if s.asking() {
		return h.confirmKeys(in)
	}
	for _, r := range in.Chars {
		s.Editor.Insert(r)
	}
	for _, k := range in.Keys {
		if edit, ok := editorKeys[k]; ok {
			edit(&s.Editor)
			continue
		}
		switch {
		case k == ebiten.KeyEscape:
			h.OverlayID = OverlayNone
		case (k == ebiten.KeyEnter || k == ebiten.KeyNumpadEnter) && in.Ctrl:
			s.Pending = true
		case k == ebiten.KeyEnter || k == ebiten.KeyNumpadEnter:
			s.Editor.Newline()
		}
	}
	if len(in.Chars) > 0 || len(in.Keys) > 0 {
		s.Editor.Follow(scriptVisible(), s.columns())
	}
	return nil
}

func (h *HUD) pendingWarning() string {
	if h.Script.Approved {
		return ""
	}
	return h.applyWarning(h.Script.Editor.Text())
}

func (s *Script) Answer(yes bool) {
	switch {
	case s.Warn != "" && yes:
		s.Warn, s.Approved, s.Pending = "", true, true
	case s.Warn != "":
		s.Warn = ""
	case yes:
		s.Load(s.Confirm)
	default:
		s.Confirm = ""
	}
}

func (s *Script) asking() bool { return s.Confirm != "" || s.Warn != "" }

func (h *HUD) confirmKeys(in Input) []Intent {
	s := &h.Script
	for _, r := range in.Chars {
		switch strings.ToLower(string(r)) {
		case "y":
			s.Answer(true)
		case "n":
			s.Answer(false)
		}
	}
	if in.Pressed(ebiten.KeyEscape) {
		s.Answer(false)
	}
	return nil
}

func (h *HUD) pressScript(rest string, _ Frame) []Intent {
	if live, ok := strings.CutPrefix(rest, "live:"); ok {
		return h.pressLiveEdit(live)
	}
	s := &h.Script
	s.Sync(ui.StoryNaming())
	switch {
	case rest == "apply":
		s.Pending = true
	case rest == "yes":
		s.Answer(true)
	case rest == "no":
		s.Answer(false)
	case rest == "beside":
		s.Beside = !s.Beside
	case strings.HasPrefix(rest, "tpl:"):
		s.Choose(strings.TrimPrefix(rest, "tpl:"))
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func scriptRail() ui.Rect {
	return ui.Rect{X: Overlay.X + 8, Y: Overlay.Y + 28, W: railW, H: Overlay.H - 28 - footerH - 14}
}

func ScriptEditor() ui.Rect {
	rail := scriptRail()
	return ui.Rect{X: rail.X + rail.W + 8, Y: rail.Y, W: Overlay.X + Overlay.W - 8 - (rail.X + rail.W + 8), H: rail.H}
}

func scriptFooter() ui.Rect {
	return ui.Rect{X: Overlay.X + 8, Y: scriptRail().Y + scriptRail().H + 6, W: Overlay.W - 16, H: footerH}
}

func scriptVisible() int {
	return ui.VisibleLines(ScriptEditor().H-6, 1)
}

func scriptColumns() int {
	return ui.MaxChars(ScriptEditor().W-gutterW-10, 1)
}

func (h *HUD) scriptButtons() []Button {
	if h.liveEdit != nil {
		return h.liveEditButtons()
	}
	rail := scriptRail()
	out := []Button{}
	for i, name := range templateOrder {
		out = append(out, Button{ID: "scr:tpl:" + name, Rect: ui.Rect{X: rail.X + 4, Y: rail.Y + 18 + i*22, W: rail.W - 8, H: 18},
			Label: templateLabel(name), Tip: templateTip(name), Selected: h.Script.Template == name})
	}
	foot := scriptFooter()
	if h.Script.asking() {
		yes, no := "Replace your draft [Y]", "Keep your draft [N]"
		if h.Script.Warn != "" {
			yes, no = "Apply it anyway [Y]", "Do not apply [N]"
		}
		out = append(out,
			Button{ID: "scr:yes", Rect: ui.Rect{X: foot.X + foot.W - 128, Y: foot.Y + 2, W: 60, H: 18}, Label: "YES", Tip: yes},
			Button{ID: "scr:no", Rect: ui.Rect{X: foot.X + foot.W - 64, Y: foot.Y + 2, W: 60, H: 18}, Label: "NO", Tip: no})
	} else {
		label := "APPLY ^ENTER"
		if h.Script.Pending {
			label = "..."
		}
		out = append(out, Button{ID: "scr:apply", Rect: ui.Rect{X: foot.X + foot.W - 104, Y: foot.Y + 2, W: 100, H: 18}, Label: label,
			Tip: "kubectl apply this decree [Ctrl+Enter]"})
	}
	return append(append(out, h.BesideButton()...), closeButton(Overlay))
}

func templateLabel(name string) string {
	switch name {
	case TemplateHearth:
		return "HUT"
	case TemplateScroll:
		return ui.Bridge(name, "CONFIGMAP")
	}
	return name
}
