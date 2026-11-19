package model

import "testing"

func TestCodexGaugeMaths(t *testing.T) {
	quota := float64(2 << 30)
	cases := []struct {
		name  string
		c     Codex
		fill  float64
		level int
		large bool
	}{
		{"unknown", Codex{}, 0, -1, false},
		{"lean", Codex{SizeBytes: quota * .1, InUseBytes: quota * .1, QuotaBytes: quota}, .1, -1, false},
		{"stirring", Codex{SizeBytes: quota * .4, QuotaBytes: quota}, .4, 0, false},
		{"swelling", Codex{SizeBytes: quota * .55, InUseBytes: quota * .2, QuotaBytes: quota}, .55, 1, true},
		{"heavy", Codex{SizeBytes: quota * .7, QuotaBytes: quota}, .7, 2, true},
		{"bloated", Codex{SizeBytes: quota * .9, QuotaBytes: quota}, .9, 3, true},
		{"past the quota", Codex{SizeBytes: quota * 1.2, QuotaBytes: quota}, 1, 3, true},
		{"alarm while lean", Codex{SizeBytes: quota * .2, QuotaBytes: quota, Alarm: true}, .2, CodexLevels - 1, true},
	}
	for _, c := range cases {
		if got := c.c.Fill(); got < c.fill-1e-9 || got > c.fill+1e-9 {
			t.Errorf("%s: fill = %v, want %v", c.name, got, c.fill)
		}
		if got := c.c.Level(); got != c.level {
			t.Errorf("%s: level = %d, want %d", c.name, got, c.level)
		}
		if got := c.c.Large(); got != c.large {
			t.Errorf("%s: large = %v", c.name, got)
		}
	}
	if (Codex{}).Known() || !(Codex{QuotaBytes: 1}).Known() {
		t.Error("a Codex is known once it reports a quota")
	}
	if got := (Codex{SizeBytes: 900, InUseBytes: 300}).Reclaimable(); got != 600 {
		t.Errorf("reclaimable = %v", got)
	}
	if got := (Codex{SizeBytes: 100, InUseBytes: 300}).Reclaimable(); got != 0 {
		t.Errorf("negative reclaim = %v", got)
	}
}

func TestMiddenFillAndHeapLevels(t *testing.T) {
	gi := float64(1 << 30)
	cases := []struct {
		m     Midden
		fill  float64
		level int
	}{
		{Midden{}, 0, -1},
		{Midden{UsedBytes: 1 * gi, CapacityBytes: 60 * gi}, 1.0 / 60, -1},
		{Midden{UsedBytes: 12 * gi, CapacityBytes: 60 * gi}, .2, 0},
		{Midden{UsedBytes: 27 * gi, CapacityBytes: 60 * gi}, .45, 1},
		{Midden{UsedBytes: 42 * gi, CapacityBytes: 60 * gi}, .7, 2},
		{Midden{UsedBytes: 57 * gi, CapacityBytes: 60 * gi}, .95, 3},
		{Midden{UsedBytes: 6 * gi, CapacityBytes: 60 * gi, Pressure: true}, .1, MiddenLevels - 1},
		{Midden{UsedBytes: 90 * gi, CapacityBytes: 60 * gi}, 1, 3},
	}
	for i, c := range cases {
		if got := c.m.Fill(); got < c.fill-1e-9 || got > c.fill+1e-9 {
			t.Errorf("case %d: fill = %v, want %v", i, got, c.fill)
		}
		if got := c.m.Level(); got != c.level {
			t.Errorf("case %d: level = %d, want %d", i, got, c.level)
		}
	}
}

func TestGaugeLevels(t *testing.T) {
	cases := []struct {
		fill  float64
		alarm bool
		want  int
	}{{0, false, GaugeCalm}, {.59, false, GaugeCalm}, {.6, false, GaugeWarn}, {.84, false, GaugeWarn}, {.85, false, GaugeDanger}, {.1, true, GaugeDanger}}
	for _, c := range cases {
		if got := GaugeLevel(c.fill, c.alarm); got != c.want {
			t.Errorf("GaugeLevel(%v, %v) = %d, want %d", c.fill, c.alarm, got, c.want)
		}
	}
}

func TestDecreeStateAndCountdown(t *testing.T) {
	d := Decree{ID: "d1", Kind: DecreeTribute, State: DecreeOffered, Deadline: 400}
	if !d.Open() || !d.Tribute() {
		t.Error("an offered probe is open")
	}
	if got := d.Left(155); got != 245 {
		t.Errorf("left = %v", got)
	}
	if got := d.Left(900); got != 0 {
		t.Errorf("left after the deadline = %v", got)
	}
	for _, state := range []string{DecreeMet, DecreeMissed} {
		if (Decree{State: state}).Open() {
			t.Errorf("%s brief still open", state)
		}
	}
	if (Decree{Kind: "lean-codex"}).Tribute() {
		t.Error("not every brief is a probe")
	}
	cases := map[float64]string{0: "0:00", -5: "0:00", 9.2: "0:10", 65: "1:05", 600: "10:00", 3725: "1:02:05"}
	for secs, want := range cases {
		if got := Countdown(secs); got != want {
			t.Errorf("Countdown(%v) = %q, want %q", secs, got, want)
		}
	}
}

func TestOpenDecreesSortByDeadline(t *testing.T) {
	c := Caesar{Decrees: []Decree{
		{ID: "late", State: DecreeOffered, Deadline: 900},
		{ID: "done", State: DecreeMet, Deadline: 100},
		{ID: "soon", State: DecreeActive, Deadline: 300},
	}}
	got := c.OpenDecrees()
	if len(got) != 2 || got[0].ID != "soon" || got[1].ID != "late" {
		t.Fatalf("open briefs = %+v", got)
	}
	if c.Decrees[0].ID != "late" {
		t.Error("sorting mutated the snapshot")
	}
}

func TestByteText(t *testing.T) {
	cases := map[float64]string{0: "0 KiB", 300 << 10: "300 KiB", 3 << 20: "3.0 MiB", 512 << 20: "512 MiB", 1 << 30: "1.0 GiB", 2.5 * (1 << 30): "2.5 GiB", 60 << 30: "60 GiB"}
	for v, want := range cases {
		if got := Bytes(v); got != want {
			t.Errorf("Bytes(%v) = %q, want %q", v, got, want)
		}
	}
}
