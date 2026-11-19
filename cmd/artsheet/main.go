package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/waxb/kubernites/pkg/client/art"
)

func main() {
	out := flag.String("out", "artsheet", "output directory")
	focus := flag.String("focus", "", "comma separated name prefixes rendered to focus.png at 4x")
	strip := flag.String("strip", "", "comma separated name prefixes, one row each, rendered to strip.png at 4x")
	age := flag.Int("age", 0, "age tier (0 dark to 3 empire) of the buildings and huts in scene.png")
	flag.Parse()
	if *strip != "" {
		if err := runStrip(*out, *strip); err != nil {
			fmt.Fprintln(os.Stderr, "artsheet:", err)
			os.Exit(1)
		}
		return
	}
	if err := run(*out, *focus, *age); err != nil {
		fmt.Fprintln(os.Stderr, "artsheet:", err)
		os.Exit(1)
	}
}

func run(dir, focus string, age int) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}
	set := art.Build()
	atlases := map[string]art.Atlas{
		"terrain": set.Terrain, "buildings": set.Buildings, "walkers": set.Walkers, "effects": set.Effects,
		"icons": set.Icons, "portraits": set.Portraits, "ui": set.UI,
	}
	for name, a := range atlases {
		if err := writePNG(filepath.Join(dir, name+".png"), a.Image); err != nil {
			return err
		}
		if err := writePNG(filepath.Join(dir, "contact-"+name+".png"), contact(a, 3)); err != nil {
			return err
		}
	}
	if focus != "" {
		if err := writePNG(filepath.Join(dir, "focus.png"), contact(focused(atlases, focus), 4)); err != nil {
			return err
		}
	}
	for _, k := range []int{4, 32} {
		if err := writePNG(filepath.Join(dir, fmt.Sprintf("appicon-%dx.png", k)), scale(art.AppIcon(), k)); err != nil {
			return err
		}
	}
	if err := writePNG(filepath.Join(dir, "variants.png"), variantSheet(set.Walkers, 4)); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(dir, "keeps.png"), keepSheet()); err != nil {
		return err
	}
	return writePNG(filepath.Join(dir, "scene.png"), scale(renderScene(set, age), 2))
}

func runStrip(dir, prefixes string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}
	set := art.Build()
	atlases := map[string]art.Atlas{
		"terrain": set.Terrain, "buildings": set.Buildings, "walkers": set.Walkers, "effects": set.Effects,
		"icons": set.Icons, "portraits": set.Portraits, "ui": set.UI,
	}
	return writePNG(filepath.Join(dir, "strip.png"), stripSheet(atlases, prefixes, 4))
}

func focused(atlases map[string]art.Atlas, focus string) art.Atlas {
	keys := make([]string, 0, len(atlases))
	for k := range atlases {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	w, h := 1, 1
	for _, k := range keys {
		b := atlases[k].Image.Bounds()
		w, h = max(w, b.Dx()), h+b.Dy()
	}
	out := art.Atlas{Image: image.NewNRGBA(image.Rect(0, 0, w, h)), Regions: map[string]art.Region{}}
	y := 0
	for _, k := range keys {
		a := atlases[k]
		draw.Draw(out.Image, a.Image.Bounds().Add(image.Pt(0, y)), a.Image, image.Point{}, draw.Src)
		for n, r := range a.Regions {
			for _, p := range strings.Split(focus, ",") {
				if strings.HasPrefix(n, p) {
					r.Y += y
					out.Regions[n] = r
					break
				}
			}
		}
		y += a.Image.Bounds().Dy()
	}
	return out
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("failed to encode %s: %w", path, err)
	}
	return nil
}

func contact(a art.Atlas, k int) *image.NRGBA {
	names := art.Names(a)
	cellW, cellH := 0, 0
	for _, n := range names {
		r := a.Regions[n]
		cellW, cellH = max(cellW, r.W*k), max(cellH, r.H*k)
	}
	cellW, cellH = max(cellW, 200)+8, cellH+24
	cols := max(1, min(len(names), 2000/cellW))
	rows := (len(names) + cols - 1) / cols
	img := image.NewNRGBA(image.Rect(0, 0, cols*cellW, rows*cellH))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{58, 74, 64, 255}), image.Point{}, draw.Src)
	for i, n := range names {
		r := a.Regions[n]
		x, y := (i%cols)*cellW+4, (i/cols)*cellH+4
		sub := a.Image.SubImage(image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H))
		blitScaled(img, sub, x, y, k)
		label(img, n, x, y+r.H*k+14)
	}
	return img
}

func blitScaled(dst *image.NRGBA, src image.Image, x, y, k int) {
	b := src.Bounds()
	for sy := b.Min.Y; sy < b.Max.Y; sy++ {
		for sx := b.Min.X; sx < b.Max.X; sx++ {
			c := color.NRGBAModel.Convert(src.At(sx, sy)).(color.NRGBA)
			if c.A == 0 {
				continue
			}
			for dy := 0; dy < k; dy++ {
				for dx := 0; dx < k; dx++ {
					px, py := x+(sx-b.Min.X)*k+dx, y+(sy-b.Min.Y)*k+dy
					dst.Set(px, py, blend(dst.NRGBAAt(px, py), c))
				}
			}
		}
	}
}

func blend(dst, src color.NRGBA) color.NRGBA {
	a := float64(src.A) / 255
	mixc := func(d, s uint8) uint8 { return uint8(float64(d)*(1-a) + float64(s)*a) }
	return color.NRGBA{mixc(dst.R, src.R), mixc(dst.G, src.G), mixc(dst.B, src.B), 255}
}

func label(img *image.NRGBA, text string, x, y int) {
	d := &font.Drawer{Dst: img, Src: image.NewUniform(color.NRGBA{240, 236, 214, 255}), Face: basicfont.Face7x13, Dot: fixed.P(x, y)}
	d.DrawString(text)
}

func scale(src *image.NRGBA, k int) *image.NRGBA {
	b := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()*k, b.Dy()*k))
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			out.SetNRGBA(x, y, src.NRGBAAt(x/k, y/k))
		}
	}
	return out
}
