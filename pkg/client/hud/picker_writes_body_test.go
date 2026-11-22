package hud

import (
	"strings"
	"testing"
)

func TestThePickersWritesBodySaysWhatIsReadInWholeWords(t *testing.T) {
	body := strings.Join(liveWritesBody(), " ")
	if strings.Contains(body, " as at ") {
		t.Errorf("the WRITES confirm body drops its verb: %q", body)
	}
	for _, line := range liveWritesBody() {
		if w := liveFont.Width(line, 1); w > LiveBox.W-40 {
			t.Errorf("%q is %d pixels, the box holds %d", line, w, LiveBox.W-40)
		}
	}
}
