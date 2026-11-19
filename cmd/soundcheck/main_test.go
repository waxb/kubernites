package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/sound"
)

func TestRunWritesEveryClipAsWAV(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := run([]string{"-out", dir, "-loops", "2", "-rate", "8000"}, &out); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.wav"))
	if err != nil {
		t.Fatal(err)
	}
	if want := len(sound.AllMusic()) + len(sound.AllSFX()) + len(sound.VoiceSamples()); len(files) != want {
		t.Fatalf("wrote %d files, want %d", len(files), want)
	}
	data, err := os.ReadFile(filepath.Join(dir, "music_dark.wav"))
	if err != nil {
		t.Fatal(err)
	}
	checkHeader(t, data, 8000)
	loop, err := sound.RenderMusic(sound.MusicDark, 8000)
	if err != nil {
		t.Fatal(err)
	}
	if len(data)-44 != 2*len(loop) {
		t.Errorf("two loops expected: %d data bytes for a %d byte loop", len(data)-44, len(loop))
	}
	if !strings.Contains(out.String(), dir) || !strings.Contains(out.String(), "sfx_calamity") {
		t.Errorf("summary should list clips and the directory:\n%s", out.String())
	}
}

func checkHeader(t *testing.T, data []byte, rate int) {
	t.Helper()
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" || string(data[12:16]) != "fmt " || string(data[36:40]) != "data" {
		t.Fatal("not a RIFF/WAVE file")
	}
	le := binary.LittleEndian
	if le.Uint32(data[4:8]) != uint32(len(data)-8) || le.Uint32(data[40:44]) != uint32(len(data)-44) {
		t.Error("chunk sizes do not match the file")
	}
	if le.Uint16(data[20:22]) != 1 || le.Uint16(data[22:24]) != 2 || le.Uint32(data[24:28]) != uint32(rate) || le.Uint16(data[34:36]) != 16 {
		t.Error("want 16-bit stereo PCM at the requested rate")
	}
}

func TestRunDefaultsToATempDirectory(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	var out bytes.Buffer
	if err := run([]string{"-rate", "8000", "-loops", "1"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "kubernites-soundcheck-") {
		t.Errorf("the temp directory is printed:\n%s", out.String())
	}
}

func TestRunRejectsBadFlags(t *testing.T) {
	for _, args := range [][]string{{"-loops", "0"}, {"-rate", "10", "-out", t.TempDir()}, {"-bogus"}} {
		if err := run(args, &bytes.Buffer{}); err == nil {
			t.Errorf("run(%v) should fail", args)
		}
	}
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-out", filepath.Join(blocker, "sub")}, &bytes.Buffer{}); err == nil {
		t.Error("an unusable output directory is reported")
	}
}

func TestDescribe(t *testing.T) {
	pcm := make([]byte, 8)
	binary.LittleEndian.PutUint16(pcm, uint16(16384))
	line := describe(clip{name: "sfx_test", pcm: pcm}, 2)
	if !strings.Contains(line, "sfx_test") || !strings.Contains(line, "-6.0 dBFS") {
		t.Errorf("describe = %q", line)
	}
	if !math.IsInf(decibels(0), -1) {
		t.Error("silence is -Inf dB")
	}
}
