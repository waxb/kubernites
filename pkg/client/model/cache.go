package model

import (
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
)

type Cache struct {
	objects map[store.Key]store.Object
	version uint64
}

func NewCache() *Cache {
	return &Cache{objects: map[store.Key]store.Object{}}
}

func (c *Cache) Reset(objs []store.Object) {
	next := make(map[store.Key]store.Object, len(objs))
	for _, o := range objs {
		next[store.KeyOf(o)] = o
	}
	c.objects = next
	c.version++
}

func (c *Cache) Apply(e store.Event) {
	key := store.KeyOf(e.Object)
	if e.Type == store.Deleted {
		delete(c.objects, key)
	} else {
		c.objects[key] = e.Object
	}
	c.version++
}

func (c *Cache) Version() uint64 { return c.version }

func (c *Cache) Get(kind, namespace, name string) (store.Object, bool) {
	o, ok := c.objects[store.Key{Kind: kind, Namespace: namespace, Name: name}]
	return o, ok
}

func (c *Cache) keys(kind string) []store.Key {
	out := []store.Key{}
	for k := range c.objects {
		if k.Kind == kind {
			out = append(out, k)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func list[T store.Object](c *Cache, kind string) []T {
	out := []T{}
	for _, k := range c.keys(kind) {
		if typed, ok := c.objects[k].(T); ok {
			out = append(out, typed)
		}
	}
	return out
}

func (c *Cache) Pods() []*corev1.Pod     { return list[*corev1.Pod](c, store.KindPod) }
func (c *Cache) Nodes() []*corev1.Node   { return list[*corev1.Node](c, store.KindNode) }
func (c *Cache) Events() []*corev1.Event { return list[*corev1.Event](c, store.KindEvent) }
func (c *Cache) ReplicaSets() []*appsv1.ReplicaSet {
	return list[*appsv1.ReplicaSet](c, store.KindReplicaSet)
}
func (c *Cache) Services() []*corev1.Service { return list[*corev1.Service](c, store.KindService) }
func (c *Cache) Slices() []*discoveryv1.EndpointSlice {
	return list[*discoveryv1.EndpointSlice](c, store.KindEndpointSlice)
}

func (c *Cache) Len() int { return len(c.objects) }

func (c *Cache) Names(kind string) []string {
	keys := c.keys(kind)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k.Name)
	}
	return out
}
