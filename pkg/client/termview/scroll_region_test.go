package termview

import (
	"runtime/debug"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestAScrollRegionPastTheScreenNeverCrashes(t *testing.T) {
	for _, c := range []struct {
		before, after [2]int
		out           string
	}{
		{[2]int{154, 10}, [2]int{154, 10}, "\x1b[1;11r" + strings.Repeat("\n", 20)},
		{[2]int{154, 27}, [2]int{154, 10}, "\x1b[1;27r"},
	} {
		func() {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("%q at %v then %v panicked: %v\n%s", c.out, c.before, c.after, p, debug.Stack())
				}
			}()
			v := New(c.before[0], c.before[1], func([]byte) {})
			defer v.Close()
			v.Write([]byte(c.out))
			v.Resize(c.after[0], c.after[1])
			v.Write([]byte(strings.Repeat("line\r\n", 30)))
		}()
	}
}

func TestEveryMarginAndPositionIsClampedToItsOwnAxis(t *testing.T) {
	axis := map[byte]string{'r': "rr", 's': "cc", 'H': "rc", 'f': "rc", 'd': "r", 'e': "r", 'A': "r", 'B': "r", 'E': "r", 'F': "r", 'G': "c", '`': "c", 'C': "c", 'D': "c", 'a': "c"}
	for _, size := range [][2]int{{20, 5}, {5, 20}, {154, 10}} {
		cols, rows := size[0], size[1]
		for _, seq := range []string{
			"1;11r", "3;300r", "300;300H", "300;300f", "300d", "300G", "300`", "300B", "300E", "300C", "300a", "300e", "300A", "300D", "300F",
			"2147483646;2147483646r", "99999999999999999999H",
		} {
			forms := introduced(seq)
			forms = append(forms, "\x1b[?69h\x1b[1;300s", "\x1b[?69h\x1b\x1b[2;30s\x1b[?6h", "\x1b[?69h\x9b1;300s")
			for _, form := range forms {
				for _, parts := range splits(form) {
					for _, d := range through(parts, cols, rows).csi {
						final := ansi.Cmd(d.cmd).Final()
						ax, ok := axis[final]
						if !ok || ansi.Cmd(d.cmd).Prefix() != 0 || ansi.Cmd(d.cmd).Intermediate() != 0 {
							continue
						}
						for i, v := range d.vals {
							limit := cols
							if a := ax[min(i, len(ax)-1)]; a == 'r' {
								limit = rows
							}
							if v > limit {
								t.Fatalf("%q split %q at %dx%d: the emulator got %q with parameter %d = %d, past its axis (%d)", form, parts, cols, rows, final, i, v, limit)
							}
						}
					}
				}
				v := New(cols, rows, func([]byte) {})
				v.Write([]byte(form + strings.Repeat("text\tmore\r\n\x1bD\x1bM\x1b[L\x1b[M", 40)))
				v.Resize(rows, cols)
				v.Write([]byte(strings.Repeat("line\r\n", 30)))
				if err := v.Broken(); err != nil {
					t.Errorf("%q at %dx%d reached the emulator unclamped: %v", form, cols, rows, err)
				}
				v.Close()
			}
		}
	}
}

func TestAnEmulatorPanicEndsTheSessionAndNeverTheProcess(t *testing.T) {
	v := New(20, 5, func([]byte) {})
	defer v.Close()
	v.mu.Lock()
	v.guard(func() { panic("index out of range") })
	v.mu.Unlock()
	if _, err := v.Write([]byte("more\r\n")); err == nil || !strings.Contains(err.Error(), ErrCannotDraw.Error()) {
		t.Fatalf("a write after the emulator broke answered %v", err)
	}
	v.Resize(30, 8)
	v.Key(Key{Rune: 'a'})
	v.Type("b")
	_ = v.Text()
}
