package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/waxb/kubernites/pkg/client/sound"
)

type options struct {
	out   string
	loops int
	rate  int
}

type clip struct {
	name string
	pcm  []byte
}

type wavHeader struct {
	Riff     [4]byte
	Size     uint32
	Wave     [4]byte
	Fmt      [4]byte
	FmtSize  uint32
	Format   uint16
	Channels uint16
	Rate     uint32
	ByteRate uint32
	Align    uint16
	Bits     uint16
	Data     [4]byte
	DataSize uint32
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "soundcheck:", err)
		os.Exit(1)
	}
}

func parseFlags(args []string) (options, error) {
	fs := flag.NewFlagSet("soundcheck", flag.ContinueOnError)
	out := fs.String("out", "", "directory for the .wav files (default: a new temp directory)")
	loops := fs.Int("loops", 2, "how many times each music loop repeats in its file")
	rate := fs.Int("rate", sound.SampleRate, "sample rate in Hz")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if *loops < 1 {
		return options{}, errors.New("-loops must be at least 1")
	}
	return options{out: *out, loops: *loops, rate: *rate}, nil
}

func run(args []string, w io.Writer) error {
	opts, err := parseFlags(args)
	if err != nil {
		return err
	}
	dir, err := outputDir(opts.out)
	if err != nil {
		return err
	}
	start := time.Now()
	clips, err := renderAll(opts)
	if err != nil {
		return err
	}
	elapsed := time.Since(start)
	for _, c := range clips {
		if err := writeWAV(filepath.Join(dir, c.name+".wav"), c.pcm, opts.rate); err != nil {
			return err
		}
		fmt.Fprintln(w, describe(c, opts.rate))
	}
	fmt.Fprintf(w, "rendered %d clips in %v\nwrote %s\n", len(clips), elapsed.Round(time.Millisecond), dir)
	return nil
}

func renderAll(opts options) ([]clip, error) {
	clips := make([]clip, 0, len(sound.AllMusic())+len(sound.AllSFX()))
	for _, m := range sound.AllMusic() {
		pcm, err := sound.RenderMusic(m, opts.rate)
		if err != nil {
			return nil, err
		}
		clips = append(clips, clip{name: "music_" + string(m), pcm: bytes.Repeat(pcm, opts.loops)})
	}
	for _, s := range sound.AllSFX() {
		pcm, err := sound.RenderSFX(s, opts.rate)
		if err != nil {
			return nil, err
		}
		clips = append(clips, clip{name: "sfx_" + string(s), pcm: pcm})
	}
	for _, v := range sound.VoiceSamples() {
		pcm, err := sound.RenderVoice(v.Voice, v.Line, opts.rate)
		if err != nil {
			return nil, err
		}
		clips = append(clips, clip{name: "voice_" + v.Name, pcm: pcm})
	}
	return clips, nil
}

func outputDir(dir string) (string, error) {
	if dir == "" {
		tmp, err := os.MkdirTemp("", "kubernites-soundcheck-")
		if err != nil {
			return "", fmt.Errorf("create temp directory: %w", err)
		}
		return tmp, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	return dir, nil
}

func encodeWAV(pcm []byte, rate int) ([]byte, error) {
	h := wavHeader{
		Riff: [4]byte{'R', 'I', 'F', 'F'}, Size: uint32(36 + len(pcm)), Wave: [4]byte{'W', 'A', 'V', 'E'},
		Fmt: [4]byte{'f', 'm', 't', ' '}, FmtSize: 16, Format: 1, Channels: 2, Rate: uint32(rate),
		ByteRate: uint32(rate * 4), Align: 4, Bits: 16, Data: [4]byte{'d', 'a', 't', 'a'}, DataSize: uint32(len(pcm)),
	}
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, h); err != nil {
		return nil, fmt.Errorf("encode wav header: %w", err)
	}
	buf.Write(pcm)
	return buf.Bytes(), nil
}

func writeWAV(path string, pcm []byte, rate int) error {
	data, err := encodeWAV(pcm, rate)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func describe(c clip, rate int) string {
	peak, sum := 0.0, 0.0
	n := len(c.pcm) / 2
	for i := 0; i < n; i++ {
		v := float64(int16(binary.LittleEndian.Uint16(c.pcm[2*i:]))) / 32768
		peak = math.Max(peak, math.Abs(v))
		sum += v * v
	}
	rms := math.Sqrt(sum / math.Max(float64(n), 1))
	seconds := float64(len(c.pcm)) / 4 / float64(rate)
	return fmt.Sprintf("%-22s %6.2fs  peak %6.1f dBFS  rms %6.1f dBFS", c.name, seconds, decibels(peak), decibels(rms))
}

func decibels(v float64) float64 {
	if v <= 0 {
		return math.Inf(-1)
	}
	return 20 * math.Log10(v)
}
