package hud

func (h *HUD) stepWindows(in Input, f Frame) ([]Intent, bool) {
	if h.Win.Grab.Id != "" {
		return h.dragWindow(in), true
	}
	if !in.Click {
		return nil, false
	}
	id, r, ok := h.windowAt(f, in.MX, in.MY)
	if !ok {
		return nil, false
	}
	h.Win = h.Win.raised(id)
	if !titleBar(id, r).Contains(in.MX, in.MY) || h.Hovering && isControl(h.Hover) {
		return nil, false
	}
	h.consumed = true
	switch {
	case in.DoubleClick:
		return h.snapHome(id), true
	case in.Held:
		h.Win = h.Win.grabbed(panelGrab{Id: id, fromX: in.MX, fromY: in.MY, start: r})
	}
	return nil, true
}

func (h *HUD) dragWindow(in Input) []Intent {
	g := h.Win.Grab
	h.consumed, h.Hover, h.Hovering = true, Button{}, false
	if !in.Held {
		h.Win = h.Win.grabbed(panelGrab{})
		if g.moving {
			return []Intent{{Kind: IntentPanels}}
		}
		return nil
	}
	dx, dy := in.MX-g.fromX, in.MY-g.fromY
	if !g.moving && max(iabs(dx), iabs(dy)) < dragSlop {
		return nil
	}
	g.moving = true
	h.Win = h.Win.pinned(g.Id, clampPanel(g.start.Offset(dx, dy))).grabbed(g)
	return nil
}

func (h *HUD) snapHome(id string) []Intent {
	if _, ok := h.Win.pins[id]; !ok {
		return nil
	}
	h.Win = h.Win.homed(id)
	return []Intent{{Kind: IntentPanels}, {Kind: IntentSound, Command: "click"}}
}
