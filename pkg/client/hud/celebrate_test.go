package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheFirstDeliveryRaisesABannerAndTheIncomeJump(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	h.Celebrate(f, "The first ledgers delivered", 4.6)
	if h.Juice.Banner != "THE FIRST LEDGERS DELIVERED" {
		t.Fatalf("banner = %q", h.Juice.Banner)
	}
	text, shown := h.GainFloat()
	if !shown || text != "+4.6/S" {
		t.Fatalf("gain float = %q %v", text, shown)
	}
}

func TestTheCelebrationReadsServedInKubernetesNames(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.TrueNames = true
	h.Celebrate(f, "The first ledgers delivered", 4.6)
	if h.Juice.Banner != "THE FIRST DATABASE REQUESTS SERVED" {
		t.Fatalf("banner = %q", h.Juice.Banner)
	}
	if text, _ := h.GainFloat(); text != "+$4.6/S" {
		t.Fatalf("gain float = %q", text)
	}
}

func TestTheIncomeJumpStaysAsLongAsTheBanner(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	h.Celebrate(f, "The first crates delivered", 12)
	h.events += GainLife + .3
	h.stepGain(f)
	if _, shown := h.GainFloat(); !shown {
		t.Fatal("the gain jump faded before its banner")
	}
	h.events += bannerIn + bannerHold + bannerOut
	h.stepGain(f)
	if _, shown := h.GainFloat(); shown {
		t.Fatal("the gain jump outlived its banner")
	}
}

func TestAFinishedRolloutRaisesABannerWithItsGain(t *testing.T) {
	defer ui.SetNamer(nil)
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Age = view.AgeFeudal
	f.State.Soulstones = append(f.State.Soulstones, view.SoulstoneEntry{Image: "porter:2.0", Owned: true, Visible: true})
	done := model.RolloutDone{Crew: "porters", Image: "porter:2.0", From: "porter:1.0", Huts: 6, Gain: 2.4}
	h.CelebrateRollout(f, done)
	title, cause := h.BannerLines()
	if title != "PORTERS: CLASS 2.0" || cause != "6 huts rebuilt one by one: +2.4 gold a second more than before" {
		t.Fatalf("banner = %q / %q", title, cause)
	}
	f.State.Age = view.AgeDark
	if _, dark := RolloutBannerIn(f, done); dark != "6 huts rebuilt a few at a time while the rest carried: the crew's roll-out. +2.4 gold a second so far." {
		t.Fatalf("the in-progress class 2.0 banner, which credits the roll-out, reads %q", dark)
	}
	if text, shown := h.GainFloat(); !shown || text != "+2.4/S" {
		t.Fatalf("gain float = %q %v", text, shown)
	}
	ui.SetNamer(names.Kubernetes(64))
	title, cause = RolloutBanner(done)
	if title != "PORTERS: PORTER:2.0 ROLLED OUT" || cause != "6 Pods updated one by one: +$2.4/s since the rollout began" {
		t.Fatalf("names banner = %q / %q", title, cause)
	}
	done.Gain = 0
	if _, cause := RolloutBanner(done); cause != "6 Pods updated one by one" {
		t.Fatalf("no gain = %q", cause)
	}
}

func TestARumouredOrFlawedClassIsNotCelebrated(t *testing.T) {
	for _, image := range []string{"porter:2.1", "porter:1.1"} {
		h := New(nil, Backend{})
		f := fullFrame(t)
		h.CelebrateRollout(f, model.RolloutDone{Crew: "porters", Image: image, From: "porter:1.0", Huts: 6, Gain: 2.4})
		if title, _ := h.BannerLines(); title != "" {
			t.Fatalf("%s raised the banner %q", image, title)
		}
		if _, shown := h.GainFloat(); shown {
			t.Fatalf("%s floated a gain", image)
		}
		toasts := h.Toasts()
		if len(toasts) != 1 || !toasts[0].Bad || !strings.Contains(toasts[0].Text, strings.TrimPrefix(image, "porter:")) {
			t.Fatalf("%s left the toasts %+v", image, toasts)
		}
	}
}
