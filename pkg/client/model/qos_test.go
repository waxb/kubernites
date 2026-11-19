package model

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func withResources(p *corev1.Pod, req, lim corev1.ResourceList) *corev1.Pod {
	next := p.DeepCopy()
	next.Spec.Containers[0].Resources = corev1.ResourceRequirements{Requests: req, Limits: lim}
	return next
}

func cpuMem(cpu, mem string) corev1.ResourceList {
	return corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpu), corev1.ResourceMemory: resource.MustParse(mem)}
}

func TestQoSComesFromStatusThenFromTheSpec(t *testing.T) {
	base := pod("p", "n1", "porter:1.0")
	cases := []struct {
		name string
		pod  *corev1.Pod
		want string
	}{
		{"no pledge", base, QoSBestEffort},
		{"pledge only", withResources(base, cpuMem("100m", "64Mi"), nil), QoSBurstable},
		{"pledge equals binding", withResources(base, cpuMem("100m", "64Mi"), cpuMem("100m", "64Mi")), QoSGuaranteed},
	}
	for _, c := range cases {
		if got := QoSOf(c.pod); got != c.want {
			t.Errorf("%s: QoS = %q, want %q", c.name, got, c.want)
		}
		if got := ViewOf(c.pod).QoS; got != c.want {
			t.Errorf("%s: view QoS = %q, want %q", c.name, got, c.want)
		}
	}
	reported := base.DeepCopy()
	reported.Status.QOSClass = corev1.PodQOSGuaranteed
	if got := QoSOf(reported); got != QoSGuaranteed {
		t.Errorf("status.qosClass ignored: %q", got)
	}
}

func TestQoSLooksTitlesAndRanks(t *testing.T) {
	cases := []struct {
		qos, look, title string
		rank             int
	}{
		{QoSBestEffort, "vagrant", "Vagrant shack (BestEffort)", 0},
		{QoSBurstable, "freeman", "Freeman cottage (Burstable)", 1},
		{QoSGuaranteed, "landed", "Landed manor (Guaranteed)", 2},
		{"", "", "", -1},
	}
	for _, c := range cases {
		if got := QoSLook(c.qos); got != c.look {
			t.Errorf("look(%q) = %q", c.qos, got)
		}
		if got := QoSTitle(c.qos); got != c.title {
			t.Errorf("title(%q) = %q", c.qos, got)
		}
		if got := QoSRank(c.qos); got != c.rank {
			t.Errorf("rank(%q) = %d", c.qos, got)
		}
	}
}

func TestQoSSentencesAreHonestAboutEviction(t *testing.T) {
	for _, qos := range []string{QoSBestEffort, QoSBurstable, QoSGuaranteed} {
		s := QoSSentence(qos)
		if s == "" || !strings.Contains(strings.ToLower(s), "evicted") || strings.Count(s, ".") != 1 || !strings.HasSuffix(s, ".") {
			t.Errorf("%s: sentence %q is not one plain sentence about eviction", qos, s)
		}
		if strings.Contains(strings.ToLower(s), "displac") {
			t.Errorf("%s: node-pressure eviction is not displacement (preemption): %q", qos, s)
		}
	}
	if !strings.Contains(QoSSentence(QoSGuaranteed), "not immune") {
		t.Error("the Landed manor sentence must say it is not immune")
	}
	if !strings.Contains(QoSSentence(QoSBurstable), "more than it pledged") {
		t.Error("the Freeman sentence must name usage beyond the Pledge")
	}
	if QoSSentence("") != "" {
		t.Error("unknown class has a sentence")
	}
}

func TestRanksFromPriorityClassNames(t *testing.T) {
	cases := map[string]string{"realm-high": RankHigh, "realm-normal": RankNormal, "realm-low": RankLow, "": "", "system-node-critical": ""}
	for class, want := range cases {
		if got := RankOf(class); got != want {
			t.Errorf("RankOf(%q) = %q, want %q", class, got, want)
		}
	}
	for _, r := range Ranks {
		if RankOf(RankClass(r)) != r {
			t.Errorf("rank %q does not round-trip", r)
		}
	}
	p := pod("p", "n1", "porter:1.0")
	priority := int32(1000)
	p.Spec.PriorityClassName, p.Spec.Priority = "realm-high", &priority
	v := ViewOf(p)
	if v.Rank != RankHigh || v.Priority != 1000 {
		t.Errorf("view rank = %q priority %d", v.Rank, v.Priority)
	}
}
