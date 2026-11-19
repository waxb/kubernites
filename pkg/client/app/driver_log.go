package app

import (
	"fmt"
	"strings"
)

func (d *Driver) logHover(tip Tooltip) {
	var b strings.Builder
	at, _ := d.clock()
	fmt.Fprintf(&b, "- `%s` hover `%s`", at, tip.ID)
	for _, part := range []struct{ name, text string }{{"tooltip", tip.Tip}, {"purpose", tip.Purpose}, {"how", tip.How}, {"why", tip.Why}} {
		if part.text != "" {
			fmt.Fprintf(&b, "\n  - %s: %s", part.name, strings.ReplaceAll(part.text, "\n", " / "))
		}
	}
	d.logLine(b.String())
}

func (d *Driver) logRegions(line string, regions []LookRegion) {
	var b strings.Builder
	b.WriteString(line)
	for _, r := range regions {
		fmt.Fprintf(&b, "\n  - %s: %s", r.Label, strings.Join(r.Lines, " / "))
	}
	d.logLine(b.String())
}
