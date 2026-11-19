package model

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func slice(name, service string, ready ...*bool) *discoveryv1.EndpointSlice {
	s := &discoveryv1.EndpointSlice{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Labels: map[string]string{discoveryv1.LabelServiceName: service}}}
	for i, r := range ready {
		e := discoveryv1.Endpoint{Addresses: []string{fmt.Sprintf("10.0.0.%d", i+1)}, Conditions: discoveryv1.EndpointConditions{Ready: r}}
		if i > 0 {
			e.TargetRef = &corev1.ObjectReference{Kind: "Pod", Name: fmt.Sprintf("%s-%c", service, 'o'+rune(i))}
		}
		s.Endpoints = append(s.Endpoints, e)
	}
	return s
}

func TestGuildRollsCountReadyEndpointsPerService(t *testing.T) {
	yes, no := true, false
	c := NewCache()
	c.Reset([]store.Object{
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "parcels", Namespace: "default"}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "kubernetes", Namespace: "default"}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "kube-dns", Namespace: "kube-system"}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "archive", Namespace: "default"}},
		slice("porter-a", "parcels", &yes, &no, nil),
		slice("porter-b", "parcels", &yes),
		slice("dns", "kube-dns", &yes),
	})
	got := c.GuildRolls()
	if len(got) != 2 || got[0].Service != "archive" || got[1].Service != "parcels" {
		t.Fatalf("rolls = %+v", got)
	}
	if got[1].Endpoints != 3 || got[1].Ready != 2 {
		t.Errorf("parcels roll = %+v, want 2 of 3 ready: nil counts as ready, and one address on two slices is one endpoint, as kube-proxy merges them", got[1])
	}
	if got[0].Endpoints != 0 {
		t.Errorf("archive roll = %+v", got[0])
	}
	if m := got[1].Members; len(m) != 3 || m[1].Ready || !m[2].Ready || m[0].Name != "10.0.0.1" || m[2].Name != "parcels-q" {
		t.Errorf("members = %+v", m)
	}
}

func TestNodeViewsReadCordonAndWards(t *testing.T) {
	c := NewCache()
	c.Reset([]store.Object{&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "stone-keep-2"}, Spec: corev1.NodeSpec{Unschedulable: true,
		Taints: []corev1.Taint{{Key: "ward", Value: "true", Effect: corev1.TaintEffectNoSchedule}}}}})
	v := c.NodeViews()["stone-keep-2"]
	if !v.Unschedulable || len(v.Taints) != 1 || v.Taints[0] != "ward" {
		t.Errorf("node view = %+v", v)
	}
}
