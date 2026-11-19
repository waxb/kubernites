package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
)

type tipLeadText struct {
	story string
	k8s   string
	page  string
}

func (t tipLeadText) text() string {
	if t.k8s == "" {
		return t.story
	}
	return ui.Bridge(t.story, t.k8s)
}

var hutLead = tipLeadText{"A hut placed by hand stands alone: if it is razed, nothing raises it again.",
	"A bare Pod is one you create by hand: if it is deleted, no controller replaces it.", "hut"}

var depotLead = tipLeadText{story: "A depot gives one kind of box a fixed address and hands each box to a ready hut.",
	k8s: "A Service gives a set of Pods one stable name and sends each request to a ready Pod."}

var tipLeads = map[string]tipLeadText{
	"cmd:door": {story: "A porter at the door is a free hut your Keep runs by itself.",
		k8s: "A static Pod is one the kubelet runs from a file on its own Node."},
	"cmd:summon": hutLead,
	"freight:tier": {story: "The harbor tier is how deep your ports are: deeper ports take more ships and richer boxes.",
		k8s: "The harbor tier sets how much traffic arrives: higher tiers bring more and bigger requests."},
}

var tipLeadPrefixes = []struct {
	prefix string
	lead   tipLeadText
}{
	{"summon:", hutLead},
	{"stall:", depotLead},
}

func tipLead(id string) tipLeadText {
	if t, ok := tipLeads[id]; ok {
		return t
	}
	for _, p := range tipLeadPrefixes {
		if strings.HasPrefix(id, p.prefix) {
			return p.lead
		}
	}
	return tipLeadText{}
}

func WithLeads(buttons []Button, f Frame) []Button {
	out := make([]Button, len(buttons))
	for i, b := range buttons {
		out[i] = b
		t := tipLead(b.ID)
		lead := t.text()
		switch {
		case lead == "" || b.Tip == "" || strings.Contains(b.Tip, lead):
		case t.page != "" && bookMet(f, t.page):
		case t.page != "":
			out[i].Tip = beforeClickNote(b.Tip, lead)
		default:
			out[i].Tip = lead + "\n" + b.Tip
		}
	}
	return out
}

func beforeClickNote(tip, line string) string {
	head, last, ok := cutLastLine(tip)
	if ok && strings.HasPrefix(last, "Click:") {
		return head + "\n" + line + "\n" + last
	}
	return tip + "\n" + line
}

func cutLastLine(text string) (string, string, bool) {
	at := strings.LastIndex(text, "\n")
	if at < 0 {
		return "", text, false
	}
	return text[:at], text[at+1:], true
}
