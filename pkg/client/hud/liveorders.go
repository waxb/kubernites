package hud

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	LiveOrderPrefix  = "liveorder"
	IntentLiveOrder  = "live:order"
	IntentLiveScroll = "live:scroll"
	OverlayLiveOrder = "liveorder"
	liveOrderMore    = "more"
	ScrollGo         = "go"
	ScrollCancel     = "cancel"
	ScrollStop       = "stop"
	FieldSep         = "\x1f"
)

type LiveOrder struct {
	Key      string
	Label    string
	Tip      string
	Icon     string
	Cell     int
	Page     int
	Disabled bool
	Why      string
	Chord    string
	Replaces []string
}

type OrderInput struct {
	Label   string
	Value   string
	Hint    string
	Choices []string
	Tick    bool
	Number  bool
}

type OrderChange struct {
	Label  string
	Before string
	After  string
}

type OrderForm struct {
	ID       int
	Key      string
	Title    string
	Object   string
	Level    string
	Lines    []string
	Inputs   []OrderInput
	Locked   bool
	Verdict  string
	Changes  []OrderChange
	Note     string
	Bad      bool
	Warning  bool
	Go       string
	GoTip    string
	GoOff    bool
	GoKey    ebiten.Key
	Stop     bool
	Command  string
	Progress []string
}

type liveWrites struct {
	orders  []LiveOrder
	page    int
	form    *OrderForm
	focus   int
	choices []LiveChoice
}

var ChordKeys = map[string]ebiten.Key{
	"B": ebiten.KeyB, "D": ebiten.KeyD, "F": ebiten.KeyF, "J": ebiten.KeyJ, "K": ebiten.KeyK, "L": ebiten.KeyL,
	"N": ebiten.KeyN, "O": ebiten.KeyO, "T": ebiten.KeyT, "V": ebiten.KeyV, "X": ebiten.KeyX, "Y": ebiten.KeyY,
}

func ChordName(letter string) string {
	if _, ok := ChordKeys[letter]; !ok {
		return ""
	}
	return "CTRL-" + letter
}

func (h *HUD) SetLiveOrders(list []LiveOrder) {
	h.lw.orders = list
}

func (h *HUD) LiveOrders() []LiveOrder {
	return h.lw.orders
}

func (h *HUD) liveOrderButton(o LiveOrder) Button {
	tip := o.Tip
	if name := ChordName(o.Chord); name != "" {
		tip += " [" + name + "]"
	}
	return Button{ID: LiveOrderPrefix + ":" + o.Key, Label: o.Label, Icon: o.Icon, Tip: tip, Disabled: o.Disabled, Why: o.Why, Menu: o.Key == liveOrderMore}
}

func (h *HUD) withLiveOrders(slots []gridSlot) []gridSlot {
	if h.live == nil || len(h.lw.orders) == 0 {
		return slots
	}
	taken, replaced := map[int]bool{}, map[string]bool{}
	var mine []gridSlot
	for _, o := range h.lw.orders {
		if o.Page != h.lw.page || o.Cell < 0 {
			continue
		}
		taken[o.Cell] = true
		for _, id := range o.Replaces {
			replaced[id] = true
		}
		mine = append(mine, gridSlot{cell: o.Cell, button: h.liveOrderButton(o)})
	}
	out := []gridSlot{}
	for _, s := range slots {
		if taken[s.cell] || replaced[s.button.ID] || (h.lw.page > 0 && s.cell != BackCell) {
			continue
		}
		out = append(out, s)
	}
	if h.lw.page > 0 {
		out = []gridSlot{{cell: BackCell, button: Button{ID: LiveOrderPrefix + ":" + liveOrderMore, Label: "BACK", Icon: "icon/back", Tip: "Back to the first orders [Esc]"}}}
	}
	return append(out, mine...)
}

func (h *HUD) pressLiveOrder(rest string, f Frame) []Intent {
	click := Intent{Kind: IntentSound, Command: "click"}
	if rest == liveOrderMore {
		h.lw.page = 1 - h.lw.page
		return []Intent{click}
	}
	return []Intent{click, {Kind: IntentLiveOrder, Target: rest}}
}

func IsLiveOrder(id string) bool {
	return strings.HasPrefix(id, LiveOrderPrefix+":")
}

func (h *HUD) liveChord(in Input, f Frame) ([]Intent, bool) {
	if h.live == nil || !in.Control {
		return nil, false
	}
	pressed := false
	for _, k := range in.Keys {
		if k == ebiten.KeyControl || k == ebiten.KeyControlLeft || k == ebiten.KeyControlRight {
			continue
		}
		pressed = true
		for _, o := range h.lw.orders {
			if key, ok := ChordKeys[o.Chord]; !ok || key != k {
				continue
			}
			if o.Disabled {
				h.Toast(o.Tip, true)
				return []Intent{{Kind: IntentSound, Command: "error"}}, true
			}
			return []Intent{{Kind: IntentSound, Command: "click"}, {Kind: IntentLiveOrder, Target: o.Key}}, true
		}
	}
	return nil, pressed
}

var OrderBox = ui.Rect{X: Overlay.X + (Overlay.W-520)/2, Y: Overlay.Y + 18, W: 520, H: 320}

func (h *HUD) OpenOrder(form OrderForm) {
	if h.lw.form == nil || h.lw.form.ID != form.ID {
		h.lw.focus = 0
	} else {
		for i := range form.Inputs {
			if i < len(h.lw.form.Inputs) && !form.Locked {
				form.Inputs[i].Value = h.lw.form.Inputs[i].Value
			}
		}
	}
	h.lw.form = &form
	h.OverlayID = OverlayLiveOrder
}

func (h *HUD) CloseOrder() {
	h.lw.form = nil
	if h.OverlayID == OverlayLiveOrder {
		h.OverlayID = OverlayNone
	}
}

func (h *HUD) OrderOpen() (OrderForm, bool) {
	if h.lw.form == nil || h.OverlayID != OverlayLiveOrder {
		return OrderForm{}, false
	}
	return *h.lw.form, true
}

func (h *HUD) OrderValues() string {
	if h.lw.form == nil {
		return ""
	}
	vals := make([]string, len(h.lw.form.Inputs))
	for i, in := range h.lw.form.Inputs {
		vals[i] = in.Value
	}
	return strings.Join(vals, FieldSep)
}

func (h *HUD) scrollIntent(cmd string) []Intent {
	return []Intent{{Kind: IntentSound, Command: "click"}, {Kind: IntentLiveScroll, Command: cmd, Target: h.OrderValues()}}
}

const (
	orderRowY   = 86
	orderRowH   = 20
	orderFieldX = 150
	orderFieldW = 220
)

func (h *HUD) orderInputRect(i int) ui.Rect {
	return ui.Rect{X: OrderBox.X + orderFieldX, Y: OrderBox.Y + orderRowY + i*orderRowH, W: orderFieldW, H: orderRowH - 4}
}

func (h *HUD) orderButtons() []Button {
	form := h.lw.form
	if form == nil {
		return []Button{closeButtonAs(OrderBox, LiveOrderPrefix+":scroll:close", "Close, nothing sent [Esc]")}
	}
	r := OrderBox
	y := r.Y + r.H - 28
	out := []Button{closeButtonAs(r, LiveOrderPrefix+":scroll:close", "Close, nothing sent [Esc]")}
	cancel := Button{ID: LiveOrderPrefix + ":scroll:" + ScrollCancel, Rect: ui.Rect{X: r.X + 16, Y: y, W: 110, H: 18}, Label: "CANCEL", Tip: "Close, nothing sent [Esc]"}
	if form.Stop || len(form.Progress) > 0 && form.Go == "" {
		cancel.Label, cancel.Tip = "CLOSE", "Close this scroll; a running drain goes on until STOP [Esc]"
	}
	out = append(out, cancel)
	if form.Stop {
		out = append(out, Button{ID: LiveOrderPrefix + ":scroll:" + ScrollStop, Rect: ui.Rect{X: r.X + 138, Y: y, W: 110, H: 18}, Label: "STOP", Tip: "End the drain: nothing more is evicted"})
	}
	if form.Go != "" {
		out = append(out, Button{ID: LiveOrderPrefix + ":scroll:" + ScrollGo, Rect: ui.Rect{X: r.X + r.W - 16 - 200, Y: y, W: 200, H: 18}, Label: form.Go, Tip: form.GoTip, Disabled: form.GoOff})
	}
	if !form.Locked {
		for i, in := range form.Inputs {
			out = append(out, Button{ID: LiveOrderPrefix + ":field:" + string(rune('0'+i)), Rect: h.orderInputRect(i), Tip: in.Hint})
		}
	}
	return out
}

func (h *HUD) pressOrderScroll(rest string) []Intent {
	form := h.lw.form
	if form == nil {
		h.CloseOrder()
		return []Intent{{Kind: IntentSound, Command: "click"}}
	}
	switch {
	case rest == "scroll:close" || rest == "scroll:"+ScrollCancel:
		return h.scrollIntent(ScrollCancel)
	case rest == "scroll:"+ScrollStop:
		return h.scrollIntent(ScrollStop)
	case rest == "scroll:"+ScrollGo:
		if form.GoOff {
			return []Intent{{Kind: IntentSound, Command: "error"}}
		}
		return h.scrollIntent(ScrollGo)
	case strings.HasPrefix(rest, "field:"):
		i := int(rest[len("field:")] - '0')
		if i < 0 || i >= len(form.Inputs) || form.Locked {
			return nil
		}
		h.lw.focus = i
		in := &form.Inputs[i]
		switch {
		case in.Tick:
			in.Value = tickValue(in.Value != "on")
		case len(in.Choices) > 0:
			in.Value = nextChoice(in.Choices, in.Value)
		}
		return []Intent{{Kind: IntentSound, Command: "click"}}
	}
	return nil
}

func tickValue(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

func nextChoice(choices []string, cur string) string {
	for i, c := range choices {
		if c == cur {
			return choices[(i+1)%len(choices)]
		}
	}
	return choices[0]
}

func (h *HUD) orderTyping() bool {
	form := h.lw.form
	if form == nil || form.Locked || h.lw.focus >= len(form.Inputs) {
		return false
	}
	in := form.Inputs[h.lw.focus]
	return !in.Tick && len(in.Choices) == 0
}

func (h *HUD) orderKeys(in Input) ([]Intent, bool) {
	form := h.lw.form
	if form == nil {
		return nil, false
	}
	switch {
	case in.Pressed(ebiten.KeyEscape):
		return h.scrollIntent(ScrollCancel), true
	case in.Pressed(ebiten.KeyTab) && !form.Locked && len(form.Inputs) > 0:
		h.lw.focus = (h.lw.focus + 1) % len(form.Inputs)
		return nil, true
	case in.Pressed(ebiten.KeyEnter) || in.Pressed(ebiten.KeyNumpadEnter):
		if form.GoKey == ebiten.KeyEnter && !form.GoOff && form.Go != "" {
			return h.scrollIntent(ScrollGo), true
		}
		return nil, true
	}
	if h.orderTyping() {
		f := &form.Inputs[h.lw.focus]
		if in.Pressed(ebiten.KeyBackspace) && f.Value != "" {
			f.Value = f.Value[:len(f.Value)-1]
		}
		for _, r := range in.Chars {
			if r < 32 || r > 126 || len(f.Value) >= 120 || f.Number && (r < '0' || r > '9') {
				continue
			}
			f.Value += string(r)
		}
		return nil, len(in.Keys) > 0 || len(in.Chars) > 0
	}
	if form.GoKey == ebiten.KeyY && in.Pressed(ebiten.KeyY) && !in.Control && !form.GoOff && form.Go != "" {
		return h.scrollIntent(ScrollGo), true
	}
	return nil, len(in.Keys) > 0 || len(in.Chars) > 0
}

func (h *HUD) drawOrderScroll(dst *ebiten.Image, f Frame) {
	form := h.lw.form
	if form == nil {
		return
	}
	ui.Fill(dst, Overlay, ui.Tint(ui.Palette.Shadow, reforgeDim))
	r := OrderBox
	ui.DarkPanel(dst, r)
	titleClr := ui.Palette.GoldBright
	if form.Warning {
		titleClr = ui.Palette.RedText
	}
	h.Font.DrawCentered(dst, form.Title, ui.Rect{X: r.X, Y: r.Y + 8, W: r.W, H: 16}, 2, titleClr)
	x, y := r.X+16, r.Y+32
	liveFont.Draw(dst, form.Level, r.X+r.W-16-liveFont.Width(form.Level, 1), y, 1, LevelColour(form.Level))
	h.Font.Draw(dst, ui.Clip(form.Object, 70), x, y, 1, ui.Palette.Text)
	y += ui.LineHeight + 2
	if form.Verdict != "" {
		clr := ui.Palette.TextDim
		if strings.HasPrefix(form.Verdict, "!") {
			clr = ui.Palette.RedText
		}
		for _, l := range h.Font.WrapLines(form.Verdict, r.W-32, 1, 2) {
			h.Font.Draw(dst, l, x, y, 1, clr)
			y += ui.LineHeight
		}
	}
	y = max(y, r.Y+orderRowY-6)
	for i, in := range form.Inputs {
		ry := r.Y + orderRowY + i*orderRowH
		h.Font.Draw(dst, in.Label, x, ry+4, 1, ui.Palette.Gold)
		box := h.orderInputRect(i)
		fill := ui.Palette.ConsoleEdge
		if i == h.lw.focus && !form.Locked {
			fill = ui.Palette.BlueDark
		}
		ui.Fill(dst, box, fill)
		ui.Stroke(dst, box, ui.Palette.Gold)
		value := in.Value
		switch {
		case in.Tick && in.Value == "on":
			value = "[x]"
		case in.Tick:
			value = "[ ]"
		case len(in.Choices) > 0:
			value = "< " + in.Value + " >"
		case i == h.lw.focus && !form.Locked && int(h.Now*2)%2 == 0:
			value += "_"
		}
		liveFont.Draw(dst, ui.Clip(value, 36), box.X+4, box.Y+4, 1, ui.Palette.Text)
		if in.Hint != "" && len(in.Choices) == 0 && !in.Tick {
			liveFont.Draw(dst, ui.Clip(in.Hint, 22), box.X+box.W+8, box.Y+4, 1, ui.Palette.TextDim)
		}
		y = ry + orderRowH
	}
	for _, l := range form.Lines {
		for _, w := range h.Font.WrapLines(l, r.W-32, 1, 5) {
			h.Font.Draw(dst, w, x, y, 1, ui.Palette.Cream)
			y += ui.LineHeight
		}
	}
	if len(form.Changes) > 0 {
		y += 2
		h.Font.Draw(dst, ui.Bridge("WHAT THE CHANCERY ANSWERS", "THE SERVER'S DRY RUN"), x, y, 1, ui.Palette.Gold)
		y += ui.LineHeight
		for _, c := range form.Changes {
			before, after := c.Before, c.After
			if before == "" {
				before = "(none)"
			}
			if after == "" {
				after = "(none)"
			}
			liveFont.Draw(dst, ui.Clip(c.Label+": "+before+" -> "+after, 84), x+8, y, 1, ui.Palette.Text)
			y += ui.LineHeight
		}
	}
	for _, p := range form.Progress {
		for _, w := range liveFont.WrapLines(p, r.W-32, 1, 3) {
			liveFont.Draw(dst, w, x, y, 1, ui.Palette.TextDim)
			y += ui.LineHeight
		}
	}
	if form.Command != "" {
		liveFont.Draw(dst, ui.Clip("> "+form.Command, 84), x, r.Y+r.H-58, 1, ui.Palette.Blue)
	}
	if form.Note != "" {
		clr := ui.Palette.Green
		if form.Bad {
			clr = ui.Palette.RedText
		}
		lines := h.Font.WrapLines(form.Note, r.W-32, 1, 2)
		for i, l := range lines {
			h.Font.Draw(dst, l, x, r.Y+r.H-46+i*ui.LineHeight-(len(lines)-1)*ui.LineHeight, 1, clr)
		}
	}
	for _, b := range h.orderButtons() {
		switch {
		case strings.HasSuffix(b.ID, ":scroll:"+ScrollGo) && !b.Disabled:
			h.drawPrimary(dst, b)
		case strings.HasPrefix(b.ID, LiveOrderPrefix+":field:"):
		default:
			h.drawButton(dst, b)
		}
	}
}

const liveDigitTip = "Plain keys only look around: an order that changes the cluster takes a click or its Control chord."

func init() {
	prefixHandlers[LiveOrderPrefix] = func(h *HUD, rest string, f Frame) []Intent {
		if strings.HasPrefix(rest, "scroll:") || strings.HasPrefix(rest, "field:") {
			return h.pressOrderScroll(rest)
		}
		return h.pressLiveOrder(rest, f)
	}
	Overlays[OverlayLiveOrder] = OverlaySpec{func(h *HUD, _ Frame) []Button { return h.orderButtons() }, (*HUD).drawOrderScroll}
	ModalOverlays = append(ModalOverlays, OverlayLiveOrder)
}

func (h *HUD) orderChord(id string) string {
	key := strings.TrimPrefix(id, LiveOrderPrefix+":")
	for _, o := range h.lw.orders {
		if o.Key == key {
			if name := ChordName(o.Chord); name != "" {
				return "^" + strings.TrimPrefix(name, "CTRL-")
			}
		}
	}
	return ""
}

var orderLabels = map[string][2]string{
	"rollback": {"UNDO", "UNDO"}, "resources": {"RESOURCES", "RESOURCES"}, "image": {"CLASS", "IMAGE"},
	"delete": {"DELETE", "DELETE"}, "drain": {"DRAIN", "DRAIN"}, "uncordon": {"UNCORDON", "UNCORDON"}, "retune": {"WAR HORN", "HPA RANGE"},
}

func OrderLabel(full string) string {
	key, _, _ := strings.Cut(full, "|")
	if l, ok := orderLabels[key]; ok {
		return ui.Bridge(l[0], l[1])
	}
	return strings.ToUpper(key)
}

func (h *HUD) liveRemedy(l CommandLine) CommandLine {
	if h.live == nil {
		return l
	}
	hints := h.live.Hints
	switch {
	case l.Order != "":
		l.Unwired = false
		if !hints {
			l.Shown = OrderLabel(l.Order)
		}
	case l.Unwired:
		if !hints {
			l.Shown = "EDIT"
		}
	case l.Command != "" && !hints:
		l.Shown = ui.Bridge("ASK THE STONE", "RUN IT")
	}
	return l
}

func (h *HUD) withOrderState(buttons []Button) []Button {
	for i, b := range buttons {
		if strings.HasSuffix(b.ID, ":"+UnwiredOp) && h.live != nil && h.live.Level != "" && h.live.Level != LevelReadOnly {
			buttons[i].Tip = ui.Bridge("This change is an edit of the crew by hand, which is the Scriptorium's work, not one of the card's orders.", "This change is a kubectl edit, which is the Scriptorium's work, not one of the card's orders.")
			continue
		}
		if !IsLiveOrder(b.ID) {
			continue
		}
		key := strings.TrimPrefix(b.ID, LiveOrderPrefix+":")
		found := false
		for _, o := range h.lw.orders {
			if o.Key == key {
				found = true
				buttons[i].Disabled, buttons[i].Why = o.Disabled, o.Why
				buttons[i].Tip = b.Tip + "\n" + o.Tip
			}
		}
		if !found {
			buttons[i].Disabled = true
		}
	}
	return buttons
}

func (h *HUD) drawOrderChip(dst *ebiten.Image, b Button) bool {
	if !IsLiveOrder(b.ID) || h.live == nil || !h.live.Hints {
		return false
	}
	if b.Disabled {
		h.drawUnwiredChip(dst, b.Rect, b.Label)
		return true
	}
	h.drawCommandChip(dst, b.Rect, b.Label, h.Hovering && h.Hover.ID == b.ID)
	return true
}
