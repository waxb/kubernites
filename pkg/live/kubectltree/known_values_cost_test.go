package kubectltree_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live/kubectltree"
)

func TestHeldErrorsWithManyKnownValuesAreMaskedInOnePassAndStop(t *testing.T) {
	line := "Error from server (Forbidden): error when creating \"dir/s.yaml\": the Stone at READ ONLY refuses this change\n"
	text := strings.Repeat(line, (512<<10)/len(line))
	known := make([]string, 0, 8000)
	for i := range 8000 {
		known = append(known, fmt.Sprintf("value-%06d", i))
	}
	started := time.Now()
	_, work, done := kubectltree.MaskErrorsIn(context.Background(), text, append([]string(nil), known...))
	t.Logf("held %d bytes, %d known values: %d bytes of work (%.0f per byte), %s", len(text), len(known), work, float64(work)/float64(len(text)), time.Since(started))
	if !done || work > 10*len(text) {
		t.Errorf("masking %d bytes of held errors with %d known values did %d bytes of work (%.0f per byte), more than one linear pass allows", len(text), len(known), work, float64(work)/float64(len(text)))
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancelled := make(chan time.Time, 1)
	timer := time.AfterFunc(20*time.Millisecond, func() {
		cancelled <- time.Now()
		cancel()
	})
	defer timer.Stop()
	out, _, done := kubectltree.MaskErrorsIn(ctx, text, append([]string(nil), known...))
	ended := time.Now()
	select {
	case at := <-cancelled:
		late := ended.Sub(at)
		t.Logf("STOP at +20ms; the masking ended %s later, done %v", late, done)
		if done || !strings.Contains(out, "STOP") {
			t.Errorf("STOP did not end the masking of held errors with many known values: it ran on %s after STOP and finished (done %v)", late, done)
		}
	default:
	}
}
