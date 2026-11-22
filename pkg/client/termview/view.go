package termview

import (
	"errors"
	"fmt"
	"image/color"
	"strings"
	"sync"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"
)

type View struct {
	mu     sync.Mutex
	writes sync.Mutex
	in     *filter
	emu    *vt.Emulator
	send   func([]byte)
	closed bool
	broken error
	done   chan struct{}
}

var ErrCannotDraw = errors.New("the Stone's terminal could not draw what the container sent, so the session ended")

func (v *View) guard(fn func()) {
	defer func() {
		if p := recover(); p != nil && v.broken == nil {
			v.broken = fmt.Errorf("%w (%v)", ErrCannotDraw, p)
		}
	}()
	if v.closed || v.broken != nil {
		return
	}
	fn()
}

func (v *View) Broken() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.broken
}

type Cell struct {
	Text      string
	Fg        color.RGBA
	Bg        color.RGBA
	HasFg     bool
	HasBg     bool
	Bold      bool
	Faint     bool
	Reverse   bool
	Underline bool
}

type Key struct {
	Name  string
	Rune  rune
	Ctrl  bool
	Alt   bool
	Shift bool
}

func New(cols, rows int, send func([]byte)) *View {
	v := &View{in: newFilter(), emu: vt.NewEmulator(max(cols, 1), max(rows, 1)), send: send, done: make(chan struct{})}
	v.emu.SetScrollbackSize(1)
	go v.pump()
	return v
}

func (v *View) pump() {
	defer close(v.done)
	buf := make([]byte, 4096)
	for {
		n, err := v.emu.Read(buf)
		if n > 0 && v.send != nil {
			v.send(append([]byte(nil), buf[:n]...))
		}
		if err != nil {
			return
		}
	}
}

func (v *View) Write(p []byte) (int, error) {
	v.writes.Lock()
	defer v.writes.Unlock()
	if err := v.Broken(); err != nil {
		return 0, err
	}
	cols, rows := v.Size()
	v.in.feed(p, cols, rows, func(piece []byte, _ int) {
		v.mu.Lock()
		defer v.mu.Unlock()
		v.guard(func() { v.emu.Write(piece) })
	})
	if err := v.Broken(); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (v *View) Resize(cols, rows int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.closed || cols == v.emu.Width() && rows == v.emu.Height() {
		return
	}
	v.guard(func() { v.emu.Resize(max(cols, 1), max(rows, 1)) })
}

func (v *View) Size() (int, int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.emu.Width(), v.emu.Height()
}

func (v *View) Cursor() (int, int, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	p := v.emu.CursorPosition()
	return p.X, p.Y, !v.emu.CursorHidden()
}

func (v *View) AltScreen() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.emu.IsAltScreen()
}

func (v *View) Row(y int) []Cell {
	v.mu.Lock()
	defer v.mu.Unlock()
	w := v.emu.Width()
	out := make([]Cell, w)
	for x := 0; x < w; x++ {
		c := v.cellAt(x, y)
		if c == nil {
			out[x].Text = " "
			continue
		}
		cell := Cell{Text: c.Content}
		if cell.Text == "" {
			cell.Text = " "
		}
		if c.Style.Fg != nil {
			cell.Fg, cell.HasFg = rgba(c.Style.Fg), true
		}
		if c.Style.Bg != nil {
			cell.Bg, cell.HasBg = rgba(c.Style.Bg), true
		}
		a := c.Style.Attrs
		cell.Bold, cell.Faint, cell.Reverse = a&uv.AttrBold != 0, a&uv.AttrFaint != 0, a&uv.AttrReverse != 0
		cell.Underline = c.Style.Underline != 0
		out[x] = cell
	}
	return out
}

func (v *View) Text() []string {
	_, rows := v.Size()
	out := make([]string, rows)
	for y := range rows {
		var line strings.Builder
		for _, c := range v.Row(y) {
			line.WriteString(c.Text)
		}
		out[y] = line.String()
	}
	return out
}

func (v *View) cellAt(x, y int) (c *uv.Cell) {
	defer func() {
		if recover() != nil {
			c = nil
		}
	}()
	return v.emu.CellAt(x, y)
}

func rgba(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

var named = map[string]rune{
	"enter": uv.KeyEnter, "tab": uv.KeyTab, "backspace": uv.KeyBackspace, "escape": uv.KeyEscape, "space": uv.KeySpace,
	"up": uv.KeyUp, "down": uv.KeyDown, "left": uv.KeyLeft, "right": uv.KeyRight,
	"home": uv.KeyHome, "end": uv.KeyEnd, "pgup": uv.KeyPgUp, "pgdown": uv.KeyPgDown, "insert": uv.KeyInsert, "delete": uv.KeyDelete,
	"f1": uv.KeyF1, "f2": uv.KeyF2, "f3": uv.KeyF3, "f4": uv.KeyF4, "f5": uv.KeyF5, "f6": uv.KeyF6,
	"f7": uv.KeyF7, "f8": uv.KeyF8, "f9": uv.KeyF9, "f10": uv.KeyF10, "f11": uv.KeyF11, "f12": uv.KeyF12,
}

func (v *View) Key(k Key) {
	code := k.Rune
	if k.Name != "" {
		c, ok := named[k.Name]
		if !ok {
			return
		}
		code = c
	}
	if code == 0 {
		return
	}
	var mod uv.KeyMod
	if k.Ctrl {
		mod |= uv.ModCtrl
	}
	if k.Alt {
		mod |= uv.ModAlt
	}
	if k.Shift {
		mod |= uv.ModShift
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.closed {
		return
	}
	if k.Name == "tab" && k.Shift {
		v.guard(func() { v.emu.SendText("\x1b[Z") })
		return
	}
	v.guard(func() { v.emu.SendKey(uv.KeyPressEvent{Code: code, Mod: mod}) })
}

func (v *View) Type(text string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.closed || text == "" {
		return
	}
	v.guard(func() { v.emu.SendText(text) })
}

func (v *View) Close() {
	v.mu.Lock()
	if v.closed {
		v.mu.Unlock()
		return
	}
	v.closed = true
	v.emu.Close()
	v.mu.Unlock()
	<-v.done
}
