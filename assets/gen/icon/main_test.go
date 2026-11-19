package main

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
)

func TestRenderIsDeterministicAndOpaqueInTheMiddle(t *testing.T) {
	a, b := render(64), render(64)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("icon render is not deterministic")
	}
	if _, _, _, alpha := a.At(32, 32).RGBA(); alpha == 0 {
		t.Fatal("icon centre is transparent")
	}
	if _, _, _, alpha := a.At(0, 0).RGBA(); alpha != 0 {
		t.Fatal("icon corner is not transparent")
	}
}

func TestUpscaleIsNearestAndDownscaleKeepsShape(t *testing.T) {
	src := art.AppIcon()
	big := render(1024)
	if big.NRGBAAt(16*32+5, 16*32+5) != src.NRGBAAt(16, 16) {
		t.Fatal("upscale is not nearest-neighbour")
	}
	small := render(16)
	if small.Bounds().Dx() != 16 {
		t.Fatalf("small icon is %v", small.Bounds())
	}
	if small.NRGBAAt(8, 8).A == 0 || small.NRGBAAt(0, 0).A == 255 {
		t.Fatal("16px icon lost its shape")
	}
}

func TestRunWritesAPNG(t *testing.T) {
	out := filepath.Join(t.TempDir(), "icon.png")
	if err := run([]string{"-out", out, "-size", "32"}); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil || img.Bounds().Dx() != 32 {
		t.Fatalf("decoded %v err %v", img.Bounds(), err)
	}
}

func TestRunRefusals(t *testing.T) {
	cases := [][]string{
		{"-size", "8"},
		{"-size", "9000"},
		{"-nope"},
		{"-out", filepath.Join(t.TempDir(), "missing", "dir", "icon.png"), "-size", "32"},
	}
	for _, args := range cases {
		if err := run(args); err == nil {
			t.Errorf("%v accepted", args)
		}
	}
}
