package live

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func (s Snapshot) Objects() []store.Object {
	var out []store.Object
	for i := range s.Nodes {
		out = append(out, s.Nodes[i].DeepCopy())
	}
	for i := range s.Namespaces {
		out = append(out, s.Namespaces[i].DeepCopy())
	}
	for i := range s.Pods {
		out = append(out, s.Pods[i].DeepCopy())
	}
	for i := range s.Services {
		out = append(out, s.Services[i].DeepCopy())
	}
	for i := range s.EndpointSlices {
		out = append(out, s.EndpointSlices[i].DeepCopy())
	}
	for i := range s.Deployments {
		out = append(out, s.Deployments[i].DeepCopy())
	}
	for i := range s.ReplicaSets {
		out = append(out, s.ReplicaSets[i].DeepCopy())
	}
	for i := range s.DaemonSets {
		out = append(out, s.DaemonSets[i].DeepCopy())
	}
	for i := range s.StatefulSets {
		out = append(out, s.StatefulSets[i].DeepCopy())
	}
	for i := range s.HPAs {
		out = append(out, s.HPAs[i].DeepCopy())
	}
	for i := range s.PVCs {
		out = append(out, s.PVCs[i].DeepCopy())
	}
	for i := range s.PVs {
		out = append(out, s.PVs[i].DeepCopy())
	}
	for i := range s.StorageClasses {
		out = append(out, s.StorageClasses[i].DeepCopy())
	}
	for i := range s.LimitRanges {
		out = append(out, s.LimitRanges[i].DeepCopy())
	}
	for i := range s.PriorityClasses {
		out = append(out, s.PriorityClasses[i].DeepCopy())
	}
	for i := range s.Events {
		out = append(out, s.Events[i].DeepCopy())
	}
	return out
}

func NewMirror(s Snapshot) (*store.Store, error) {
	taken := metav1.NewTime(s.TakenAt)
	st := store.New(func() metav1.Time { return taken })
	if err := st.Restore(s.Objects(), 0, 0); err != nil {
		return nil, err
	}
	return st, nil
}

func (r *Realm) mirrorOne(typ string, obj metav1.Object) {
	o, ok := obj.(store.Object)
	if !ok {
		return
	}
	kind, err := store.KindOf(o)
	if err != nil {
		return
	}
	if typ == watchDeleted {
		key := store.Key{Kind: kind, Namespace: o.GetNamespace(), Name: o.GetName()}
		if cur, ok := r.mirror.Peek(key); ok && cur.GetUID() == o.GetUID() {
			r.mirror.Unmirror(key)
		}
		return
	}
	_ = r.mirror.Mirror(o)
}

func (r *Realm) mirrorSwap(before, after []metav1.Object) {
	kept := map[string]bool{}
	for _, o := range after {
		kept[string(o.GetUID())] = true
	}
	for _, o := range before {
		if !kept[string(o.GetUID())] {
			r.mirrorOne(watchDeleted, o)
		}
	}
	for _, o := range after {
		r.mirrorOne(watchModified, o)
	}
}
