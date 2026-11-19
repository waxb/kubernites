package store

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

type Object interface {
	metav1.Object
	runtime.Object
}

type Key struct {
	Kind      string
	Namespace string
	Name      string
}

func (k Key) String() string {
	if k.Namespace == "" {
		return k.Kind + "/" + k.Name
	}
	return k.Kind + "/" + k.Namespace + "/" + k.Name
}

type EventType string

const (
	Added    EventType = "ADDED"
	Modified EventType = "MODIFIED"
	Deleted  EventType = "DELETED"
)

type Event struct {
	Type   EventType
	Object Object
}

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrConflict      = errors.New("conflict: the object has been modified")
)

type Snapshot struct {
	Objects         []Object
	ResourceVersion uint64
	UIDSeq          uint64
}

type Store struct {
	objects   map[Key]Object
	rv        uint64
	uidSeq    uint64
	now       func() metav1.Time
	listeners []func(Event)
	sorted    map[string][]entry
	meter     Meter
	restores  uint64
	supposing int
	kindRV    map[string]uint64
}

func New(now func() metav1.Time) *Store {
	return &Store{objects: map[Key]Object{}, now: now, sorted: map[string][]entry{}, kindRV: map[string]uint64{}}
}

func KeyOf(obj Object) Key {
	kind, _ := KindOf(obj)
	return Key{Kind: kind, Namespace: obj.GetNamespace(), Name: obj.GetName()}
}

func (s *Store) Watch(fn func(Event)) {
	s.listeners = append(s.listeners, fn)
}

func (s *Store) Restores() uint64 {
	return s.restores
}

func (s *Store) Supposing() bool {
	return s.supposing > 0
}

func (s *Store) ResourceVersion() uint64 {
	return s.rv
}

func (s *Store) Create(obj Object) (Object, error) {
	kind, err := KindOf(obj)
	if err != nil {
		return nil, err
	}
	key := Key{Kind: kind, Namespace: obj.GetNamespace(), Name: obj.GetName()}
	if key.Name == "" {
		return nil, fmt.Errorf("%s: name is required", kind)
	}
	stored := copyOf(obj)
	if err := s.admit(stored, nil); err != nil {
		return nil, err
	}
	if _, exists := s.objects[key]; exists {
		return nil, fmt.Errorf("%s %q: %w", kind, key.Name, ErrAlreadyExists)
	}
	if err := s.admitWrite(kind, stored); err != nil {
		return nil, err
	}
	s.uidSeq++
	s.rv++
	stored.SetUID(types.UID(uidFor(s.uidSeq)))
	stored.SetResourceVersion(strconv.FormatUint(s.rv, 10))
	stored.SetCreationTimestamp(s.now())
	setGeneration(stored, kind, 1)
	setGVK(stored, kind)
	s.objects[key] = stored
	s.putEntry(key, stored)
	s.meterWrite(Added, kind, nil, stored)
	s.emit(Added, stored)
	return copyOf(stored), nil
}

func (s *Store) Update(obj Object) (Object, error) {
	return s.update(obj, true)
}

func (s *Store) UpdateStatus(obj Object) (Object, error) {
	return s.update(obj, false)
}

func (s *Store) update(obj Object, bumpGeneration bool) (Object, error) {
	kind, err := KindOf(obj)
	if err != nil {
		return nil, err
	}
	key := Key{Kind: kind, Namespace: obj.GetNamespace(), Name: obj.GetName()}
	current, exists := s.objects[key]
	if !exists {
		return nil, fmt.Errorf("%s %q: %w", kind, key.Name, ErrNotFound)
	}
	if rv := obj.GetResourceVersion(); rv != "" && rv != current.GetResourceVersion() {
		return nil, fmt.Errorf("%s %q: %w", kind, key.Name, ErrConflict)
	}
	stored := copyOf(obj)
	if err := keepDeletion(stored, current); err != nil {
		return nil, err
	}
	if shouldDeleteDuringUpdate(stored, current) {
		s.remove(key, current)
		return copyOf(current), nil
	}
	if err := s.admit(stored, current); err != nil {
		return nil, err
	}
	if err := s.admitWrite(kind, stored); err != nil {
		return nil, err
	}
	s.rv++
	stored.SetUID(current.GetUID())
	stored.SetCreationTimestamp(current.GetCreationTimestamp())
	stored.SetResourceVersion(strconv.FormatUint(s.rv, 10))
	generation := current.GetGeneration()
	if bumpGeneration && specChanged(current, stored) {
		generation++
	}
	setGeneration(stored, kind, generation)
	setGVK(stored, kind)
	s.objects[key] = stored
	s.putEntry(key, stored)
	s.meterWrite(Modified, kind, current, stored)
	s.emit(Modified, stored)
	return copyOf(stored), nil
}

func (s *Store) Delete(key Key) error {
	current, exists := s.objects[key]
	if !exists {
		return fmt.Errorf("%s %q: %w", key.Kind, key.Name, ErrNotFound)
	}
	if len(current.GetFinalizers()) > 0 {
		if !alreadyDeleting(current) {
			s.markAsDeleting(key, current)
		}
		return nil
	}
	s.remove(key, current)
	return nil
}

func (s *Store) remove(key Key, current Object) {
	delete(s.objects, key)
	s.rv++
	s.dropEntry(key)
	final := copyOf(current)
	final.SetResourceVersion(strconv.FormatUint(s.rv, 10))
	s.meterWrite(Deleted, key.Kind, current, nil)
	s.emit(Deleted, final)
}

func (s *Store) Peek(key Key) (Object, bool) {
	obj, ok := s.objects[key]
	return obj, ok
}

func (s *Store) Get(key Key) (Object, bool) {
	obj, ok := s.objects[key]
	if !ok {
		return nil, false
	}
	return copyOf(obj), true
}

func (s *Store) collect(kind, namespace string, pick func(Object) Object) []Object {
	es := inNamespace(s.entriesOf(kind), namespace)
	out := make([]Object, 0, len(es))
	for _, e := range es {
		out = append(out, pick(e.obj))
	}
	return out
}

func (s *Store) List(kind, namespace string) []Object {
	return s.collect(kind, namespace, copyOf)
}

func (s *Store) View(kind, namespace string) []Object {
	return s.collect(kind, namespace, func(obj Object) Object { return obj })
}

func (s *Store) Snapshot() Snapshot {
	keys := make([]Key, 0, len(s.objects))
	for k := range s.objects {
		keys = append(keys, k)
	}
	sortKeys(keys)
	objs := make([]Object, 0, len(keys))
	for _, k := range keys {
		objs = append(objs, copyOf(s.objects[k]))
	}
	return Snapshot{Objects: objs, ResourceVersion: s.rv, UIDSeq: s.uidSeq}
}

func (s *Store) Restore(objs []Object, rv, uidSeq uint64) error {
	restored := make(map[Key]Object, len(objs))
	for _, obj := range objs {
		kind, err := KindOf(obj)
		if err != nil {
			return err
		}
		stored := copyOf(obj)
		setGeneration(stored, kind, stored.GetGeneration())
		setGVK(stored, kind)
		restored[Key{Kind: kind, Namespace: obj.GetNamespace(), Name: obj.GetName()}] = stored
	}
	s.objects = restored
	s.sorted = map[string][]entry{}
	s.kindRV = map[string]uint64{}
	s.restores++
	s.rv = rv
	s.uidSeq = max(uidSeq, maxUIDSeq(objs))
	return nil
}

func maxUIDSeq(objs []Object) uint64 {
	var highest uint64
	for _, obj := range objs {
		uid := string(obj.GetUID())
		tail := uid[strings.LastIndex(uid, "-")+1:]
		if seq, err := strconv.ParseUint(tail, 16, 64); err == nil && seq > highest {
			highest = seq
		}
	}
	return highest
}

func Get[T Object](s *Store, kind, namespace, name string) (T, bool) {
	var zero T
	obj, ok := s.Get(Key{Kind: kind, Namespace: namespace, Name: name})
	if !ok {
		return zero, false
	}
	typed, ok := obj.(T)
	return typed, ok
}

func Peek[T Object](s *Store, kind, namespace, name string) (T, bool) {
	var zero T
	obj, ok := s.Peek(Key{Kind: kind, Namespace: namespace, Name: name})
	if !ok {
		return zero, false
	}
	typed, ok := obj.(T)
	return typed, ok
}

func List[T Object](s *Store, kind, namespace string) []T {
	return typed[T](s.List(kind, namespace))
}

func View[T Object](s *Store, kind, namespace string) []T {
	return typed[T](s.View(kind, namespace))
}

func typed[T Object](objs []Object) []T {
	out := make([]T, 0, len(objs))
	for _, obj := range objs {
		if typed, ok := obj.(T); ok {
			out = append(out, typed)
		}
	}
	return out
}

func (s *Store) emit(t EventType, obj Object) {
	for _, fn := range s.listeners {
		fn(Event{Type: t, Object: copyOf(obj)})
	}
}

func copyOf(obj Object) Object {
	return obj.DeepCopyObject().(Object)
}

func setGVK(obj Object, kind string) {
	if gvk, ok := gvks[kind]; ok {
		obj.GetObjectKind().SetGroupVersionKind(gvk)
	}
}

func uidFor(seq uint64) string {
	return fmt.Sprintf("%08x-0000-4000-8000-%012x", seq>>16, seq)
}

func sortKeys(keys []Key) {
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Kind != keys[j].Kind {
			return keys[i].Kind < keys[j].Kind
		}
		if keys[i].Namespace != keys[j].Namespace {
			return keys[i].Namespace < keys[j].Namespace
		}
		return keys[i].Name < keys[j].Name
	})
}
