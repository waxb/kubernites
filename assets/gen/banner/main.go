package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	captureW   = 960
	captureH   = 540
	cropTop    = 24
	bannerW    = captureW
	bannerH    = 320
	bandTop    = 244
	bandH      = 76
	nameScale  = 6
	inkOffset  = 3
	maxScale   = 4
	bannerName = "KUBERNITES"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "banner:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("banner", flag.ContinueOnError)
	in := fs.String("in", "", "a 960x540 capture of the live realm")
	out := fs.String("out", "banner.png", "output PNG path")
	scale := fs.Int("scale", 2, "pixel scale of the output, 1 to 4")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *scale < 1 || *scale > maxScale {
		return errors.New("scale must be between 1 and 4")
	}
	src, err := readPNG(*in)
	if err != nil {
		return err
	}
	if b := src.Bounds(); b.Dx() != captureW || b.Dy() != captureH {
		return fmt.Errorf("%s is %dx%d, not a %dx%d capture", *in, b.Dx(), b.Dy(), captureW, captureH)
	}
	g := &composer{src: crop(src)}
	ebiten.SetWindowSize(bannerW, bannerH)
	ebiten.SetWindowTitle("banner")
	if err := ebiten.RunGameWithOptions(g, &ebiten.RunGameOptions{InitUnfocused: true}); err != nil {
		return err
	}
	if g.out == nil {
		return errors.New("the banner was not drawn")
	}
	return writePNG(*out, upscale(g.out, *scale))
}

func readPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func crop(src image.Image) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, bannerW, bannerH))
	draw.Draw(dst, dst.Bounds(), src, image.Pt(0, cropTop), draw.Src)
	return dst
}

func upscale(src *image.RGBA, scale int) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx()*scale, b.Dy()*scale))
	for y := 0; y < b.Dy()*scale; y++ {
		for x := 0; x < b.Dx()*scale; x++ {
			dst.SetRGBA(x, y, src.RGBAAt(x/scale, y/scale))
		}
	}
	return dst
}

type composer struct {
	src *image.RGBA
	out *image.RGBA
}

func (c *composer) Update() error {
	if c.out != nil {
		return ebiten.Termination
	}
	return nil
}

func (c *composer) Draw(screen *ebiten.Image) {
	if c.out != nil {
		return
	}
	canvas := ebiten.NewImage(bannerW, bannerH)
	canvas.DrawImage(ebiten.NewImageFromImage(c.src), nil)
	band := ui.Rect{X: 0, Y: bandTop, W: bannerW, H: bandH}
	ui.Fill(canvas, band, color.RGBA{12, 16, 24, 200})
	ui.Fill(canvas, ui.Rect{X: 0, Y: band.Y, W: bannerW, H: 1}, ui.Palette.Gold)
	ui.Fill(canvas, ui.Rect{X: 0, Y: band.Y + band.H - 1, W: bannerW, H: 1}, ui.Palette.Gold)
	name := ui.Rect{X: 0, Y: band.Y + (band.H-ui.CapHeight*nameScale)/2 - nameScale, W: bannerW, H: ui.CapHeight * nameScale}
	ui.Font{}.DrawCentered(canvas, bannerName, name.Offset(inkOffset, inkOffset), nameScale, ui.Palette.Ink)
	ui.Font{}.DrawCentered(canvas, bannerName, name, nameScale, ui.Palette.GoldBright)
	out := image.NewRGBA(image.Rect(0, 0, bannerW, bannerH))
	canvas.ReadPixels(out.Pix)
	c.out = out
	screen.DrawImage(canvas, nil)
}

func (c *composer) Layout(int, int) (int, int) {
	return bannerW, bannerH
}
