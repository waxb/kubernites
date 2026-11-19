package hud

import (
	"maps"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	OverlayBridge  = "bridge"
	bridgePrefix   = "bridge"
	BridgeSee      = bridgePrefix + ":see"
	BridgeAlmanac  = bridgePrefix + ":almanac"
	BridgeTry      = bridgePrefix + ":try"
	bridgeBook     = bridgePrefix + ":book"
	BridgeMended   = bridgePrefix + ":mended"
	bridgeLabel    = "SEE IT IN KUBERNETES"
	tryLabel       = "TRY IT IN THE STONE"
	AlmanacLabel   = "ALMANAC"
	FromAlmanac    = "the Almanac"
	OfferSee       = "see"
	OfferAlmanac   = "almanac"
	OfferTry       = "try"
	bridgeSeeTip   = "The step you just did, as Kubernetes: the command, the YAML it changed, the rule and the docs."
	bridgeBookTip  = "Its Kubernetes side waits in the Almanac, with your own command."
	tryTip         = "You saw this in Kubernetes: speak it on the Stone this time. The command is written for you; press Enter."
	bridgeMendTip  = "The trouble you just mended, as Kubernetes: the command, the YAML, the rule and the docs."
	podObjectStart = "pod/"
	depotPage      = "depot"
)

type kubeLayer struct {
	bridge bridgeState
	Recap  recapState  `json:"-"`
	Ending endingState `json:"-"`
}

type bridgeState struct {
	view  view.BridgeView
	from  string
	tried map[string]string
}

type bridgeOffer struct {
	Kind   string `json:"-"`
	PageID string `json:"-"`
	Title  string `json:"-"`
	line   CommandLine
	pod    string
}

func bridgeLabelText(label string) string { return ui.Settle(label) }

func bridged(f Frame, page string) bool {
	return f.State.Kube.Bridged[page].Presses > 0
}

func saidLines(f Frame) []view.SaidLine {
	out := make([]view.SaidLine, 0, len(f.State.Kube.Said))
	for _, s := range f.State.Kube.Said {
		out = append(out, view.SaidLine{Command: s.Command, YAML: s.YAML, Typed: s.Typed, At: s.At})
	}
	return out
}

func ConceptPage(id string) (string, bool) {
	concept := view.Content.ObjectiveConcept(id)
	if concept == "" {
		return "", false
	}
	p, ok := view.Content.ForSubject(view.SubjectConcept, concept)
	if !ok || !view.Content.HasBridge(p.ID) {
		return "", false
	}
	return p.ID, true
}

func firstOfConcept(f Frame, at int) bool {
	concept := view.Content.ObjectiveConcept(f.State.Objectives[at].ID)
	for _, o := range f.State.Objectives[:at] {
		if o.Done && view.Content.ObjectiveConcept(o.ID) == concept {
			return false
		}
	}
	return true
}

func (h *HUD) stepOffer(f Frame) (bridgeOffer, bool) {
	d, on := f.ShownObjective()
	at := slices.IndexFunc(f.State.Objectives, func(o view.ObjectiveView) bool { return o.ID == d.ID })
	if !on || !d.Done || at < 0 || !firstOfConcept(f, at) {
		return bridgeOffer{}, false
	}
	page, ok := ConceptPage(d.ID)
	if !ok {
		return bridgeOffer{}, false
	}
	if !bridged(f, page) {
		return bridgeOffer{Kind: OfferSee, PageID: page, Title: d.Text}, true
	}
	if bookMet(f, page) {
		return bridgeOffer{Kind: OfferAlmanac, PageID: page, Title: d.Text}, true
	}
	return bridgeOffer{}, false
}

func triedInStone(f Frame, page string) bool {
	since := f.State.Kube.Bridged[page].First
	for _, s := range f.State.Kube.Said {
		if s.Typed && s.At >= since && view.Content.BridgeMatches(page, s.Command) {
			return true
		}
	}
	return false
}

func (h *HUD) tryOffer(f Frame, l CommandLine, where string) (bridgeOffer, bool) {
	if l.Command == "" || !f.Revealed("stone") {
		return bridgeOffer{}, false
	}
	if len(f.State.Kube.Bridged) == 0 {
		return bridgeOffer{}, false
	}
	words := view.CommandWords(kubectlVerb + BareCommandOf(l.Command))
	for _, page := range slices.Sorted(maps.Keys(f.State.Kube.Bridged)) {
		if !bridged(f, page) || !view.Content.BridgeMatchesWords(page, words) || triedInStone(f, page) {
			continue
		}
		if spent, ok := h.Kube.bridge.tried[page]; ok && spent != where {
			continue
		}
		return bridgeOffer{Kind: OfferTry, PageID: page, line: l}, true
	}
	return bridgeOffer{}, false
}

func (h *HUD) ObjectiveOffer(f Frame) (bridgeOffer, bool) {
	see, seen := h.heldStepOffer(f)
	if seen && see.Kind == OfferSee {
		return see, true
	}
	if o, ok := f.current(); ok && f.SpeakCommand() != "" {
		if l, lined := f.ObjectiveLine(); lined {
			if try, ok := h.tryOffer(f, l, "step:"+o.ID); ok {
				return try, true
			}
		}
	}
	return see, seen
}

func (h *HUD) spendTry(o bridgeOffer, where string) {
	if o.Kind != OfferTry {
		return
	}
	if h.Kube.bridge.tried == nil {
		h.Kube.bridge.tried = map[string]string{}
	}
	if _, ok := h.Kube.bridge.tried[o.PageID]; !ok {
		h.Kube.bridge.tried[o.PageID] = where
	}
}

func bridgeAsBuilt(page string, f Frame) (view.Bridge, bool) {
	if b, ok := view.Content.BridgeFor(page); ok && page == "door-hut" && f.State.Tutorial {
		b.YAML = view.ChapterDoorYAML
		return b, true
	}
	if page == depotPage && f.State.Roll.Bare {
		addresses := make([]string, 0, len(f.State.Roll.Entries))
		for _, e := range f.State.Roll.Entries {
			addresses = append(addresses, e.Address)
		}
		return view.HandDepotBridge(addresses), true
	}
	return view.Content.BridgeFor(page)
}

func (h *HUD) OpenBridge(page, pod, from string, f Frame) []Intent {
	b, ok := bridgeAsBuilt(page, f)
	if !ok {
		return nil
	}
	b = b.Filled(pod, "")
	own, mine := view.OwnCommand(b, saidLines(f))
	bv := b.View("", "")
	if mine {
		bv = b.View(own.Command, own.YAML)
	}
	h.Kube.bridge.view, h.Kube.bridge.from = bv, from
	h.OverlayID = OverlayBridge
	return []Intent{ActFrom(view.Action{Action: view.ActBridge, ID: page}, from), {Kind: IntentSound, Command: BookSound}}
}

func (h *HUD) BridgeView() view.BridgeView { return h.Kube.bridge.view }

func (h *HUD) pressBridge(rest string, f Frame) []Intent {
	switch rest {
	case "see":
		if o, ok := h.ObjectiveOffer(f); ok && o.Kind == OfferSee {
			return h.OpenBridge(o.PageID, "", FromObjectiveCard, f)
		}
	case "almanac":
		if o, ok := h.ObjectiveOffer(f); ok && o.Kind == OfferAlmanac {
			return h.OpenBook(o.PageID, f)
		}
	case "try":
		if o, ok := h.ObjectiveOffer(f); ok && o.Kind == OfferTry {
			current, _ := f.current()
			h.spendTry(o, "step:"+current.ID)
			return h.UseLine(o.line, FromObjectiveCard)
		}
	case "book":
		return h.OpenBook(h.Kube.bridge.view.Page, f)
	case "mended":
		if o, ok := h.CalamityOffer(f); ok && o.Kind == OfferSee {
			return h.OpenBridge(o.PageID, o.pod, FromCalamityCard, f)
		}
	case "calamity-try":
		if o, ok := h.CalamityOffer(f); ok && o.Kind == OfferTry {
			v, _ := h.ShownCalamity(f)
			h.spendTry(o, "calamity:"+v.ID)
			return h.UseLine(o.line, FromCalamityCard)
		}
	}
	return nil
}

func calamityBridge(f Frame, v view.CalamityView) (view.Bridge, bool) {
	ids := splitIDs(v.Kind)
	if inc, ok := incidentOf(f, v.Incident); ok {
		ids = append(splitIDs(inc.Kind), ids...)
	}
	for _, id := range ids {
		for _, kind := range calamityKinds {
			if p, ok := view.Content.ForSubject(kind, id); ok {
				return view.Content.BridgeFor(p.ID)
			}
		}
	}
	return view.Bridge{}, false
}

func calamityPod(v view.CalamityView) string {
	if pod, ok := strings.CutPrefix(v.Object, podObjectStart); ok {
		return pod
	}
	return ""
}

func (h *HUD) CalamityOffer(f Frame) (bridgeOffer, bool) {
	v, ok := h.ShownCalamity(f)
	if !ok || h.live != nil {
		return bridgeOffer{}, false
	}
	if v.Mended != "" {
		b, ok := calamityBridge(f, v)
		if !ok || bridged(f, b.Page) {
			return bridgeOffer{}, false
		}
		return bridgeOffer{Kind: OfferSee, PageID: b.Page, pod: calamityPod(v)}, true
	}
	for _, l := range h.calamityLines(f) {
		if o, ok := h.tryOffer(f, l, "calamity:"+v.ID); ok {
			return o, true
		}
	}
	return bridgeOffer{}, false
}

func init() {
	prefixHandlers[bridgePrefix] = (*HUD).pressBridge
}
