package limitranger

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func list(pairs ...string) corev1.ResourceList {
	out := corev1.ResourceList{}
	for i := 0; i+1 < len(pairs); i += 2 {
		out[corev1.ResourceName(pairs[i])] = resource.MustParse(pairs[i+1])
	}
	return out
}

func docsRange() *corev1.LimitRange {
	return &corev1.LimitRange{ObjectMeta: metav1.ObjectMeta{Name: "cpu-limit-range", Namespace: "default"},
		Spec: corev1.LimitRangeSpec{Limits: []corev1.LimitRangeItem{{Type: corev1.LimitTypeContainer,
			Default: list("cpu", "1"), DefaultRequest: list("cpu", "500m")}}}}
}

func podWith(r corev1.ResourceRequirements) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "default-cpu-demo", Namespace: "default"},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "default-cpu-demo-ctr", Image: "nginx", Resources: r}}}}
}

func admitted(p *corev1.Pod, ranges ...*corev1.LimitRange) *corev1.Pod {
	out := p.DeepCopy()
	RequestsFromLimits(&out.Spec)
	Mutate(out, ranges)
	return out
}

func quantity(l corev1.ResourceList, name corev1.ResourceName) string {
	q, ok := l[name]
	if !ok {
		return "<unset>"
	}
	return q.String()
}

func TestDefaultsFollowTheLiveDocsCases(t *testing.T) {
	cases := []struct {
		name         string
		in           corev1.ResourceRequirements
		request, lim string
	}{
		{"nothing set: default request and default limit", corev1.ResourceRequirements{}, "500m", "1"},
		{"only a limit: the request matches the limit", corev1.ResourceRequirements{Limits: list("cpu", "1")}, "1", "1"},
		{"only a request: the default limit", corev1.ResourceRequirements{Requests: list("cpu", "0.75")}, "750m", "1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := admitted(podWith(c.in), docsRange())
			res := out.Spec.Containers[0].Resources
			if quantity(res.Requests, corev1.ResourceCPU) != c.request || quantity(res.Limits, corev1.ResourceCPU) != c.lim {
				t.Fatalf("requests %v limits %v", res.Requests, res.Limits)
			}
		})
	}
}

func TestMutateAnnotatesWhatItSetLikeTheUpstreamPlugin(t *testing.T) {
	lr := docsRange()
	lr.Spec.Limits[0].DefaultRequest = list("cpu", "200m", "memory", "64Mi")
	lr.Spec.Limits[0].Default = list("cpu", "500m", "memory", "1Gi")
	p := podWith(corev1.ResourceRequirements{Requests: list("cpu", "100m")})
	p.Spec.InitContainers = []corev1.Container{{Name: "raven"}}
	out := admitted(p, lr)
	want := "LimitRanger plugin set: memory request for container default-cpu-demo-ctr; cpu, memory limit for container default-cpu-demo-ctr; cpu, memory request for init container raven; cpu, memory limit for init container raven"
	if got := out.Annotations[Annotation]; got != want {
		t.Fatalf("annotation = %q", got)
	}
	if !DefaultedRequest(out, corev1.ResourceMemory) || DefaultedRequest(out, "gpu") {
		t.Fatal("memory request was defaulted for the first container")
	}
	if p.Annotations != nil || len(p.Spec.Containers[0].Resources.Limits) != 0 {
		t.Fatal("mutate changed its input")
	}
	full := admitted(podWith(corev1.ResourceRequirements{Requests: list("cpu", "1", "memory", "1Gi"), Limits: list("cpu", "1", "memory", "1Gi")}), lr)
	if _, ok := full.Annotations[Annotation]; ok {
		t.Fatal("annotated a pod it did not change")
	}
	if DefaultedRequest(full, corev1.ResourceCPU) {
		t.Fatal("an explicit request reads as defaulted")
	}
}

func TestDefaultRequirementsMergesContainerItemsOnly(t *testing.T) {
	lr := docsRange()
	lr.Spec.Limits = append(lr.Spec.Limits, corev1.LimitRangeItem{Type: corev1.LimitTypePod, Max: list("cpu", "2")},
		corev1.LimitRangeItem{Type: corev1.LimitTypeContainer, DefaultRequest: list("memory", "64Mi")})
	d := DefaultRequirements(lr)
	if quantity(d.Requests, corev1.ResourceCPU) != "500m" || quantity(d.Requests, corev1.ResourceMemory) != "64Mi" || quantity(d.Limits, corev1.ResourceCPU) != "1" {
		t.Fatalf("defaults = %+v", d)
	}
}

func TestDefaultedSpecLeavesTheTemplateAlone(t *testing.T) {
	spec := podWith(corev1.ResourceRequirements{}).Spec
	out := DefaultedSpec(spec, []*corev1.LimitRange{docsRange()})
	if quantity(out.Containers[0].Resources.Requests, corev1.ResourceCPU) != "500m" || len(spec.Containers[0].Resources.Requests) != 0 {
		t.Fatalf("spec %v template %v", out.Containers[0].Resources, spec.Containers[0].Resources)
	}
	if bare := DefaultedSpec(spec, nil); len(bare.Containers[0].Resources.Requests) != 0 {
		t.Fatalf("no range, yet %v", bare.Containers[0].Resources)
	}
}

func constrained(item corev1.LimitRangeItem) []*corev1.LimitRange {
	return []*corev1.LimitRange{{ObjectMeta: metav1.ObjectMeta{Name: "c", Namespace: "default"}, Spec: corev1.LimitRangeSpec{Limits: []corev1.LimitRangeItem{item}}}}
}

func TestValidateUsesTheUpstreamMessages(t *testing.T) {
	cases := []struct {
		name string
		item corev1.LimitRangeItem
		res  corev1.ResourceRequirements
		want string
	}{
		{"min without a request", corev1.LimitRangeItem{Type: corev1.LimitTypeContainer, Min: list("cpu", "100m")}, corev1.ResourceRequirements{},
			"minimum cpu usage per Container is 100m.  No request is specified"},
		{"min above the request", corev1.LimitRangeItem{Type: corev1.LimitTypeContainer, Min: list("cpu", "100m")}, corev1.ResourceRequirements{Requests: list("cpu", "50m")},
			"minimum cpu usage per Container is 100m, but request is 50m"},
		{"min above the limit", corev1.LimitRangeItem{Type: corev1.LimitTypeContainer, Min: list("cpu", "100m")},
			corev1.ResourceRequirements{Requests: list("cpu", "200m"), Limits: list("cpu", "50m")}, "minimum cpu usage per Container is 100m, but limit is 50m"},
		{"max without a limit", corev1.LimitRangeItem{Type: corev1.LimitTypeContainer, Max: list("memory", "1Gi")}, corev1.ResourceRequirements{},
			"maximum memory usage per Container is 1Gi.  No limit is specified"},
		{"max below the limit", corev1.LimitRangeItem{Type: corev1.LimitTypeContainer, Max: list("memory", "1Gi")}, corev1.ResourceRequirements{Limits: list("memory", "2Gi")},
			"maximum memory usage per Container is 1Gi, but limit is 2Gi"},
		{"max below the request", corev1.LimitRangeItem{Type: corev1.LimitTypeContainer, Max: list("memory", "1Gi")},
			corev1.ResourceRequirements{Requests: list("memory", "2Gi"), Limits: list("memory", "1Gi")}, "maximum memory usage per Container is 1Gi, but request is 2Gi"},
		{"ratio without a request", corev1.LimitRangeItem{Type: corev1.LimitTypeContainer, MaxLimitRequestRatio: list("cpu", "2")},
			corev1.ResourceRequirements{Limits: list("cpu", "1")}, "cpu max limit to request ratio per Container is 2, but no request is specified or request is 0"},
		{"ratio without a limit", corev1.LimitRangeItem{Type: corev1.LimitTypeContainer, MaxLimitRequestRatio: list("cpu", "2")},
			corev1.ResourceRequirements{Requests: list("cpu", "1")}, "cpu max limit to request ratio per Container is 2, but no limit is specified or limit is 0"},
		{"ratio exceeded", corev1.LimitRangeItem{Type: corev1.LimitTypeContainer, MaxLimitRequestRatio: list("cpu", "2")},
			corev1.ResourceRequirements{Requests: list("cpu", "100m"), Limits: list("cpu", "300m")}, "cpu max limit to request ratio per Container is 2, but provided ratio is 3.000000"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Validate(podWith(c.res), constrained(c.item))
			if err == nil || err.Error() != c.want {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
		})
	}
}

func TestValidateAggregatesAndPassesWithinBounds(t *testing.T) {
	item := corev1.LimitRangeItem{Type: corev1.LimitTypeContainer, Min: list("cpu", "100m"), Max: list("cpu", "1")}
	err := Validate(podWith(corev1.ResourceRequirements{}), constrained(item))
	if err == nil || !strings.HasPrefix(err.Error(), "[minimum cpu usage per Container is 100m.  No request is specified, maximum cpu usage") {
		t.Fatalf("err = %v", err)
	}
	ok := podWith(corev1.ResourceRequirements{Requests: list("cpu", "200m"), Limits: list("cpu", "500m")})
	if err := Validate(ok, constrained(item)); err != nil {
		t.Fatal(err)
	}
	ratio := corev1.LimitRangeItem{Type: corev1.LimitTypeContainer, MaxLimitRequestRatio: list("cpu", "3")}
	if err := Validate(ok, constrained(ratio)); err != nil {
		t.Fatal(err)
	}
}

func TestPodItemsSumContainersAndSidecars(t *testing.T) {
	always := corev1.ContainerRestartPolicyAlways
	p := podWith(corev1.ResourceRequirements{Requests: list("cpu", "300m"), Limits: list("cpu", "300m")})
	p.Spec.InitContainers = []corev1.Container{
		{Name: "raven", RestartPolicy: &always, Resources: corev1.ResourceRequirements{Requests: list("cpu", "300m"), Limits: list("cpu", "300m")}},
		{Name: "squire", Resources: corev1.ResourceRequirements{Requests: list("cpu", "900m"), Limits: list("cpu", "900m")}},
	}
	req, lim := podTotals(p)
	if quantity(req, corev1.ResourceCPU) != "1200m" || quantity(lim, corev1.ResourceCPU) != "1200m" {
		t.Fatalf("totals %v %v", req, lim)
	}
	err := Validate(p, constrained(corev1.LimitRangeItem{Type: corev1.LimitTypePod, Max: list("cpu", "1")}))
	if err == nil || err.Error() != "maximum cpu usage per Pod is 1, but limit is 1200m" {
		t.Fatalf("err = %v", err)
	}
	p.Spec.Resources = &corev1.ResourceRequirements{Limits: list("cpu", "1")}
	if err := Validate(p, constrained(corev1.LimitRangeItem{Type: corev1.LimitTypePod, Max: list("cpu", "1")})); err == nil || !strings.Contains(err.Error(), "request is 1200m") {
		t.Fatalf("pod-level limit: %v", err)
	}
}

func TestTheFirstFailingRangeStopsValidation(t *testing.T) {
	first := constrained(corev1.LimitRangeItem{Type: corev1.LimitTypeContainer, Min: list("memory", "1Gi")})
	second := constrained(corev1.LimitRangeItem{Type: corev1.LimitTypeContainer, Min: list("cpu", "1")})
	err := Validate(podWith(corev1.ResourceRequirements{}), append(first, second...))
	if err == nil || err.Error() != "minimum memory usage per Container is 1Gi.  No request is specified" {
		t.Fatalf("err = %v", err)
	}
}

func TestHugeQuantitiesCompareByValue(t *testing.T) {
	item := corev1.LimitRangeItem{Type: corev1.LimitTypeContainer, Max: list("memory", "1Ei")}
	if err := Validate(podWith(corev1.ResourceRequirements{Limits: list("memory", "2Ei")}), constrained(item)); err == nil {
		t.Fatal("2Ei passed a 1Ei maximum")
	}
}
