package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/waxb/kubernites/pkg/client/art"
)

const (
	minSize = 16
	maxSize = 4096
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "icon:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("icon", flag.ContinueOnError)
	out := fs.String("out", "icon.png", "output PNG path")
	size := fs.Int("size", 1024, "icon size in pixels")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *size < minSize || *size > maxSize {
		return errors.New("size must be between 16 and 4096")
	}
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	if err := png.Encode(f, render(*size)); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func render(size int) *image.NRGBA {
	src := art.AppIcon()
	if size < art.AppIconSize {
		return boxDown(src, size)
	}
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dst.SetNRGBA(x, y, src.NRGBAAt(x*art.AppIconSize/size, y*art.AppIconSize/size))
		}
	}
	return dst
}

func boxDown(src *image.NRGBA, size int) *image.NRGBA {
	block := art.AppIconSize / size
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dst.SetNRGBA(x, y, average(src, x*block, y*block, block))
		}
	}
	return dst
}

func average(src *image.NRGBA, x0, y0, block int) color.NRGBA {
	var r, g, b, a int
	for y := y0; y < y0+block; y++ {
		for x := x0; x < x0+block; x++ {
			c := src.NRGBAAt(x, y)
			r, g, b, a = r+int(c.R)*int(c.A), g+int(c.G)*int(c.A), b+int(c.B)*int(c.A), a+int(c.A)
		}
	}
	if a == 0 {
		return color.NRGBA{}
	}
	return color.NRGBA{R: uint8(r / a), G: uint8(g / a), B: uint8(b / a), A: uint8(a / (block * block))}
}
