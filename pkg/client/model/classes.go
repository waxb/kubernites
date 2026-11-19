package model

import (
	"image/color"

	"github.com/waxb/kubernites/pkg/client/ui"
)

var classColors = map[string]color.RGBA{
	"balanced": ui.Palette.Green, "compute": ui.Palette.GoldBright, "memory": ui.Palette.Blue, "rich": ui.Palette.Purple,
}

func ClassColor(class string) color.RGBA {
	if c, ok := classColors[class]; ok {
		return c
	}
	return ui.Palette.Text
}
