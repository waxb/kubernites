package hud

import (
	"fmt"
	"github.com/waxb/kubernites/pkg/client/ui"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	CpuStep      = 50
	CpuMost      = 4000
	memStep      = 32
	memMost      = 4096
	useSuffix    = "use"
	HornPitch    = 70
	hornPitchMin = 10
	hornPitchMax = 100
	hornPitchStp = 10
	hornCeiling  = 50
	hornSpan     = 3
)

type HornPick struct {
	Min, Max, CPU int
}

type resField struct {
	key   string
	label string
	rune  string
	step  int64
	most  int64
	unit  string
	get   func(model.Resources) int64
	set   func(model.Resources, int64) model.Resources
	story string
	k8s   string
}

var (
	resFields = []resField{
		{"cpu-req", "Food Pledge", "resources.requests.cpu", CpuStep, CpuMost, "m",
			func(r model.Resources) int64 { return r.CPURequest }, func(r model.Resources, v int64) model.Resources { r.CPURequest = v; return r },
			"Each Keep reserves this much food for every worker: a bigger pledge fits fewer huts, but no worker goes hungry.",
			"requests.cpu: the scheduler reserves this per Pod. Higher means fewer Pods per Node, each sure of its share."},
		{"mem-req", "Housing Pledge", "resources.requests.memory", memStep, memMost, "Mi",
			func(r model.Resources) int64 { return r.MemRequest }, func(r model.Resources, v int64) model.Resources { r.MemRequest = v; return r },
			"Each Keep reserves this many rooms for every worker, and fits one only where that much housing is free.",
			"requests.memory: reserved per Pod; the scheduler places a Pod only on a Node with this much free."},
		{"cpu-lim", "Food Binding", "resources.limits.cpu", CpuStep, CpuMost, "m",
			func(r model.Resources) int64 { return r.CPULimit }, func(r model.Resources, v int64) model.Resources { r.CPULimit = v; return r },
			"A worker may eat at most this much food: past it, it slows down but never dies.",
			"limits.cpu: the container is throttled above this. It is never killed for CPU."},
		{"mem-lim", "Housing Binding", "resources.limits.memory", memStep, memMost, "Mi",
			func(r model.Resources) int64 { return r.MemLimit }, func(r model.Resources, v int64) model.Resources { r.MemLimit = v; return r },
			"A worker taking more rooms than this is Voidtaken and rebuilt, so its neighbours keep theirs.",
			"limits.memory: a container above this is OOMKilled and restarted, so its neighbours keep their memory."},
	}
)

func (rf resField) icon() string {
	if rf.unit == "m" {
		return ui.IconFood
	}
	return ui.IconHousing
}

func knownFields(f Frame) []resField {
	out := []resField{}
	for _, rf := range resFields {
		if RuneKnown(f, rf.rune) {
			out = append(out, rf)
		}
	}
	return out
}

func snapResources(r model.Resources, _ Frame) model.Resources { return r }

func StepValue(step, most, v int64, up bool) int64 {
	if up {
		return min((v/step+1)*step, most)
	}
	if v%step != 0 {
		return v / step * step
	}
	return max(v-step, 0)
}

func roundedUp(v, step, most int64) int64 {
	return min((v+step-1)/step*step, most)
}

func crewPeakOf(f Frame, c model.CrewChain) view.UsePeakView {
	cv, ok := crewByName(f, c.Name)
	if !ok {
		return view.UsePeakView{}
	}
	return cv.Peak
}

func (rf resField) measured(p view.UsePeakView) int64 {
	if !p.Known() {
		return 0
	}
	switch rf.key {
	case "cpu-req":
		return p.CPU
	case "mem-req":
		return roundedUp(p.Memory/mebi, rf.step, rf.most)
	case "cpu-lim":
		return p.Raise
	}
	return 0
}

func FieldNamed(key string) (resField, bool) {
	for _, rf := range resFields {
		if rf.key == key {
			return rf, true
		}
	}
	return resField{}, false
}

func (h *HUD) pressResources(c model.CrewChain, arg string, f Frame) []Intent {
	click := Intent{Kind: IntentSound, Command: "click"}
	if arg == "apply" {
		a, ok := ResourcesAction(c, h.Crew.Res, f)
		if !ok {
			return nil
		}
		h.Crew.Mode = CrewModeChain
		return []Intent{ActFrom(a, FromCrewCard)}
	}
	key, dir, _ := strings.Cut(arg, ":")
	rf, ok := FieldNamed(key)
	if !ok || !RuneKnown(f, rf.rune) {
		return nil
	}
	if dir == useSuffix {
		n := rf.measured(crewPeakOf(f, c))
		if n <= 0 {
			return nil
		}
		h.Crew.Res = rf.set(h.Crew.Res, n)
		return []Intent{click}
	}
	h.Crew.Res = rf.set(h.Crew.Res, StepValue(rf.step, rf.most, rf.get(EffectiveResources(h.Crew.Res, f)), dir == "up"))
	return []Intent{click}
}

func EffectiveResources(r model.Resources, f Frame) model.Resources {
	lr := f.State.LimitRange
	if r.CPURequest == 0 {
		r.CPURequest = r.CPULimit
	}
	if r.MemRequest == 0 {
		r.MemRequest = r.MemLimit
	}
	if lr == nil {
		return r
	}
	if r.CPURequest == 0 {
		r.CPURequest = lr.CPU
	}
	if r.MemRequest == 0 {
		r.MemRequest = lr.Memory / mebi
	}
	if r.CPULimit == 0 {
		r.CPULimit = lr.CPULimit
	}
	if r.MemLimit == 0 {
		r.MemLimit = lr.MemoryLimit / mebi
	}
	return r
}

func resValue(rf resField, pick model.Resources, f Frame) (string, bool) {
	own, n := rf.get(pick), rf.get(EffectiveResources(pick, f))
	if n <= 0 {
		return "none", false
	}
	return ui.Bridge(rf.icon()+strconv.FormatInt(n, 10), quantity(n, rf.unit)), own == 0
}

func quantity(v int64, unit string) string {
	return strconv.FormatInt(v, 10) + unit
}

func ResourcesAction(c model.CrewChain, pick model.Resources, f Frame) (view.Action, bool) {
	var req, lim []string
	for _, rf := range knownFields(f) {
		if rf.get(pick) == rf.get(c.Resources) {
			continue
		}
		pair := strings.SplitN(rf.rune, ".", 3)
		entry := pair[2] + "=" + quantity(rf.get(pick), rf.unit)
		if pair[1] == "requests" {
			req = append(req, entry)
		} else {
			lim = append(lim, entry)
		}
	}
	if len(req)+len(lim) == 0 || c.Bare || PledgeOverBinding(pick) {
		return view.Action{}, false
	}
	return view.Action{Action: view.ActCrewResources, Name: c.Name, Crew: view.CrewOrder{Container: c.Container,
		Requests: strings.Join(req, ","), Limits: strings.Join(lim, ",")}}, true
}

func PledgeOverBinding(r model.Resources) bool {
	return r.CPULimit > 0 && r.CPURequest > r.CPULimit || r.MemLimit > 0 && r.MemRequest > r.MemLimit
}

func PickQoS(r model.Resources) string {
	req, lim := corev1.ResourceList{}, corev1.ResourceList{}
	put := func(rl corev1.ResourceList, name corev1.ResourceName, v int64, unit string) {
		if v > 0 {
			rl[name] = resource.MustParse(quantity(v, unit))
		}
	}
	put(req, corev1.ResourceCPU, r.CPURequest, "m")
	put(req, corev1.ResourceMemory, r.MemRequest, "Mi")
	put(lim, corev1.ResourceCPU, r.CPULimit, "m")
	put(lim, corev1.ResourceMemory, r.MemLimit, "Mi")
	spec := corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Resources: corev1.ResourceRequirements{Requests: req, Limits: lim}}}}
	return model.TemplateQoS(spec)
}

func FirstHorn(c model.CrewChain) HornPick {
	if c.Horn != nil {
		return HornPick{Min: c.Horn.Min, Max: c.Horn.Max, CPU: max(c.Horn.Goal, hornPitchMin)}
	}
	low := max(c.Desired, 1)
	return HornPick{Min: low, Max: min(low+hornSpan, hornCeiling), CPU: HornPitch}
}

func roomyHorn(p HornPick, c model.CrewChain, k CrewFacts) HornPick {
	if c.Horn != nil || !k.Bounded() {
		return p
	}
	next := p
	next.Max = max(next.Min, min(p.Max, c.Desired+max(k.Fed, 0)))
	return next
}

func (h *HUD) pressHorn(c model.CrewChain, arg string) []Intent {
	switch arg {
	case "sound":
		p := h.Crew.Horn
		h.Crew.Mode = CrewModeChain
		return []Intent{ActFrom(view.Action{Action: view.ActAutoscaleCrew, Name: c.Name, Crew: view.CrewOrder{Min: p.Min, Max: p.Max, CPU: p.CPU}}, FromCrewCard)}
	case "silence":
		h.Crew.Mode = CrewModeChain
		return []Intent{ActFrom(view.Action{Action: view.ActSilenceHorn, Name: c.Name}, FromCrewCard)}
	case "retune":
		p := h.Crew.Horn
		if c.Horn == nil || p == FirstHorn(c) {
			return nil
		}
		h.Crew.Mode = CrewModeChain
		return []Intent{ActFrom(view.Action{Action: view.ActRetuneHorn, Name: c.Name, Crew: view.CrewOrder{Min: p.Min, Max: p.Max, CPU: p.CPU}}, FromCrewCard)}
	}
	field, dir, _ := strings.Cut(arg, ":")
	h.Crew.Horn = StepHorn(h.Crew.Horn, field, dir == "up")
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func StepHorn(p HornPick, field string, up bool) HornPick {
	d := 1
	if !up {
		d = -1
	}
	next := p
	switch field {
	case "min":
		next.Min = min(max(p.Min+d, 1), p.Max)
	case "max":
		next.Max = min(max(p.Max+d, p.Min), hornCeiling)
	case "cpu":
		next.CPU = min(max(p.CPU+d*hornPitchStp, hornPitchMin), hornPitchMax)
	}
	return next
}

func NewerClasses(f Frame, image string) []string {
	cur, ok := soulstone.Lookup(image)
	if !ok {
		return nil
	}
	out := []string{}
	for _, s := range f.State.Soulstones {
		p, found := soulstone.Lookup(s.Image)
		if s.Owned && !s.Cursed && found && p.Repository == cur.Repository && soulstone.TagNewer(p.Tag, cur.Tag) {
			out = append(out, p.Image)
		}
	}
	return out
}

func HornLine(hn *model.Horn) string {
	if !hn.Measured {
		return fmt.Sprintf(ui.Bridge("Floor %d, Ceiling %d, Pitch %d%%: now %d; it has not read the food yet, as its new huts are not ready", "minReplicas %d, maxReplicas %d, target %d%%: now %d; it has not read the CPU yet, as its new Pods are not ready"), hn.Min, hn.Max, hn.Goal, hn.Current)
	}
	return fmt.Sprintf(ui.Bridge("Floor %d, Ceiling %d, Pitch %d%%: now %d, using %d%%", "minReplicas %d, maxReplicas %d, target %d%%: now %d, using %d%%"), hn.Min, hn.Max, hn.Goal, hn.Current, hn.Utilization)
}
