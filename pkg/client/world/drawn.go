package world

import "github.com/waxb/kubernites/pkg/client/ui"

const (
	DrawnLabel   = "label"
	DrawnCallout = "callout"
	DrawnSpeech  = "speech"
	DrawnTag     = "tag"
)

type DrawnBox struct {
	Kind string
	Key  string
	Text string
	Rect ui.Rect
}

func (r *Renderer) Drawn() []DrawnBox { return append([]DrawnBox(nil), r.boxes...) }

func (r *Renderer) note(kind, key, text string, box ui.Rect) {
	r.boxes = append(r.boxes, DrawnBox{Kind: kind, Key: key, Text: text, Rect: box})
}
