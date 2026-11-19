package app

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	titleRegion   = "title"
	mapRegion     = "titlemap"
	liveRegion    = "titlelive"
	shadowReach   = 4
	panelTitle    = "title"
	panelChapters = "chapters"
	panelLive     = "live"
	titleTime     = "00:00"
	titleWaits    = "no realm runs on the title page: choose one of its modes"
	titleQuits    = "QUIT closes the game here; the harness stays on the title page"
	titleWaitCap  = 10.0
)

var errTitleOpen = errors.New("the title page is open: no realm runs yet, so there is no Speaking Stone")

func (d *Driver) atTitle() bool { return d.G.Scene == SceneTitle }

func (d *Driver) titleLayer() []hud.Button {
	switch {
	case d.G.Chapters.Active():
		return d.G.Chapters.TitleButtons()
	case d.G.Picker.Active():
		return d.G.Picker.Buttons()
	}
	return d.G.Title.Buttons()
}

func (d *Driver) titlePanel() string {
	switch {
	case d.G.Chapters.Active():
		return panelChapters
	case d.G.Picker.Active():
		return panelLive
	}
	return panelTitle
}

func (d *Driver) titleButton(id string) (hud.Button, error) {
	for _, b := range d.titleLayer() {
		if b.ID == id && b.Rect.W > 0 {
			if b.Disabled {
				return b, fmt.Errorf("%s is shown but cannot be pressed now", id)
			}
			return b, nil
		}
	}
	return hud.Button{}, fmt.Errorf("no button %q on the %s page: look again", id, d.titlePanel())
}

func (d *Driver) titleCover() (ui.Rect, string, bool) {
	switch {
	case d.G.Chapters.Active():
		return hud.KnowledgeBox, mapRegion, true
	case d.G.Picker.Active():
		return hud.LiveBox, liveRegion, true
	}
	return ui.Rect{}, titleRegion, false
}

func (d *Driver) captureTitle() []ui.TextRun {
	var page, layer []ui.TextRun
	cover, region, covered := d.titleCover()
	d.onThread(func() {
		d.Dst.Fill(background)
		stop := ui.RecordText(func(r ui.TextRun) { page = append(page, r) })
		ui.InRegion(titleRegion, func() { d.G.drawTitlePage(d.Dst) })
		stop()
		stop = ui.RecordText(func(r ui.TextRun) { layer = append(layer, r) })
		ui.InRegion(region, func() { d.G.drawTitleLayers(d.Dst) })
		stop()
	})
	out := make([]ui.TextRun, 0, len(page)+len(layer))
	for i, r := range page {
		if i > 0 && shadowOf(page[i-1], r) {
			continue
		}
		if covered && cover.Contains(r.Rect.X+r.Rect.W/2, r.Rect.Y+r.Rect.H/2) {
			continue
		}
		out = append(out, r)
	}
	return append(out, layer...)
}

func shadowOf(shadow, r ui.TextRun) bool {
	dx, dy := shadow.Rect.X-r.Rect.X, shadow.Rect.Y-r.Rect.Y
	return shadow.Text == r.Text && dx >= 0 && dx <= shadowReach && dy >= 0 && dy <= shadowReach
}

func (d *Driver) titleButtons(runs []ui.TextRun) []LookButton {
	out := []LookButton{}
	seen := map[string]bool{}
	for _, b := range d.titleLayer() {
		if b.Rect.W <= 0 || b.Rect.H <= 0 || seen[b.ID] {
			continue
		}
		seen[b.ID] = true
		region := regionOfRect(runs, b.Rect)
		label := ui.Plain(b.Label)
		text := buttonText(runs, b.Rect, label)
		if label == "" {
			label, text = bareLabel(b.ID, region, text), ""
		}
		out = append(out, LookButton{ID: b.ID, Label: label, Text: text, Region: region, Enabled: !b.Disabled})
	}
	return out
}

func (d *Driver) lookTitle(shot bool) Look {
	runs := d.captureTitle()
	st := PlayText(runs)
	d.diff(st, exactLine)
	look := Look{Time: titleTime, Regions: lookRegions(st), Buttons: d.titleButtons(runs), World: []LookObject{}, Panel: d.titlePanel()}
	if shot {
		look.Shot = d.shot(titleTime)
	}
	line := fmt.Sprintf("- `%s` look at the %s page", titleTime, look.Panel)
	if look.Shot != "" {
		line += ", shot " + look.Shot
	}
	d.logRegions(line, look.Regions)
	return look
}

func (d *Driver) clickTitle(id string) (Changes, error) {
	b, err := d.titleButton(id)
	if err != nil {
		d.logLine(fmt.Sprintf("- `%s` click `%s`: %v", titleTime, id, err))
		return Changes{}, err
	}
	x, y := b.Rect.X+b.Rect.W/2, b.Rect.Y+b.Rect.H/2
	d.Step(hud.Input{MX: x, MY: y}, 0)
	d.Step(hud.Input{MX: x, MY: y, Click: true}, 0)
	d.Step(hud.Input{MX: x, MY: y}, 0)
	ch := d.changes()
	if d.quit {
		ch.Note, d.quit = titleQuits, false
	}
	d.Settle()
	d.logAction("click", id, ch)
	return ch, nil
}

func (d *Driver) hoverTitle(id string) (Tooltip, error) {
	b, err := d.titleButton(id)
	if err != nil {
		return Tooltip{}, err
	}
	d.Step(hud.Input{MX: idleX, MY: idleY}, 0)
	d.onThread(func() {
		for range hud.TitleTipTicks {
			d.Step(hud.Input{MX: b.Rect.X + b.Rect.W/2, MY: b.Rect.Y + b.Rect.H/2}, 0)
		}
	})
	st := PlayText(d.Capture())
	tip := Tooltip{ID: id, Label: ui.Plain(b.Label), Tip: cmp.Or(strings.Join(st.Region(tooltipRegion), "\n"), NoTip), Page: hud.NoPage}
	d.logHover(tip)
	return tip, nil
}

func (d *Driver) waitTitle(seconds float64) Changes {
	ticks := int(math.Round(min(max(seconds, 0), titleWaitCap) / TickSeconds))
	d.onThread(func() {
		for range ticks {
			d.Step(hud.Input{MX: idleX, MY: idleY}, TickSeconds)
		}
	})
	ch := d.changesBy(maskedLine)
	ch.Note = titleWaits
	d.logAction("wait", fmt.Sprint(seconds), ch)
	return ch
}

func (d *Driver) keyTitle(name string, k ebiten.Key, mods chordMods) Changes {
	d.Step(hud.Input{MX: idleX, MY: idleY, Keys: []ebiten.Key{k}, Chars: keyChar(k, mods), Shift: mods.shift, Ctrl: mods.ctrl, Alt: mods.alt}, 0)
	d.Step(hud.Input{MX: idleX, MY: idleY}, 0)
	ch := d.changes()
	if d.quit {
		ch.Note, d.quit = titleQuits, false
	}
	d.Settle()
	d.logAction("key", name, ch)
	return ch
}
