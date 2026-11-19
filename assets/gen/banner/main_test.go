package main

import (
	"image"
	"image/color"
	"path/filepath"
	"strings"
	"testing"
)

func TestCropTakesTheWorldBelowTheTopBar(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, captureW, captureH))
	mark := color.RGBA{1, 2, 3, 255}
	src.SetRGBA(7, cropTop, mark)
	got := crop(src)
	if got.Bounds() != image.Rect(0, 0, bannerW, bannerH) {
		t.Fatalf("crop is %v", got.Bounds())
	}
	if got.RGBAAt(7, 0) != mark {
		t.Fatalf("the crop does not start at row %d", cropTop)
	}
	if cropTop+bannerH > captureH {
		t.Fatalf("the crop runs past the capture")
	}
}

func TestUpscaleIsNearest(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	a, b := color.RGBA{10, 20, 30, 255}, color.RGBA{200, 100, 50, 255}
	src.SetRGBA(0, 0, a)
	src.SetRGBA(1, 0, b)
	got := upscale(src, 3)
	if got.Bounds() != image.Rect(0, 0, 6, 3) {
		t.Fatalf("upscale is %v", got.Bounds())
	}
	if got.RGBAAt(2, 2) != a || got.RGBAAt(3, 0) != b {
		t.Fatal("upscale is not nearest-neighbour")
	}
}

func TestOnlyAFullCaptureMakesABanner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "small.png")
	if err := writePNG(path, image.NewRGBA(image.Rect(0, 0, 10, 10))); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-in", path, "-out", filepath.Join(t.TempDir(), "b.png")}); err == nil || !strings.Contains(err.Error(), "not a 960x540 capture") {
		t.Fatalf("a small picture answered %v", err)
	}
	if err := run([]string{"-in", path, "-scale", "9"}); err == nil {
		t.Fatal("a scale of 9 was accepted")
	}
}
