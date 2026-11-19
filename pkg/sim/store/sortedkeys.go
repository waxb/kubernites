package store

import "sort"

type entry struct {
	key Key
	obj Object
}

func keyLess(a, b Key) bool {
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	if a.Namespace != b.Namespace {
		return a.Namespace < b.Namespace
	}
	return a.Name < b.Name
}

func (s *Store) entriesOf(kind string) []entry {
	if es, cached := s.sorted[kind]; cached {
		return es
	}
	es := []entry{}
	for k, obj := range s.objects {
		if k.Kind == kind {
			es = append(es, entry{key: k, obj: obj})
		}
	}
	sort.Slice(es, func(i, j int) bool { return keyLess(es[i].key, es[j].key) })
	s.sorted[kind] = es
	return es
}

func (s *Store) keysOf(kind string) []Key {
	es := s.entriesOf(kind)
	keys := make([]Key, len(es))
	for i, e := range es {
		keys[i] = e.key
	}
	return keys
}

func searchEntry(es []entry, k Key) int {
	return sort.Search(len(es), func(i int) bool { return !keyLess(es[i].key, k) })
}

func (s *Store) KindVersion(kind string) uint64 {
	return s.kindRV[kind] + s.restores<<40
}

func (s *Store) putEntry(k Key, obj Object) {
	s.kindRV[k.Kind] = s.rv
	es, cached := s.sorted[k.Kind]
	if !cached {
		return
	}
	i := searchEntry(es, k)
	if i < len(es) && es[i].key == k {
		es[i].obj = obj
		return
	}
	es = append(es, entry{})
	copy(es[i+1:], es[i:])
	es[i] = entry{key: k, obj: obj}
	s.sorted[k.Kind] = es
}

func (s *Store) dropEntry(k Key) {
	s.kindRV[k.Kind] = s.rv
	es, cached := s.sorted[k.Kind]
	if !cached {
		return
	}
	i := searchEntry(es, k)
	if i == len(es) || es[i].key != k {
		delete(s.sorted, k.Kind)
		return
	}
	s.sorted[k.Kind] = append(es[:i], es[i+1:]...)
}

func inNamespace(es []entry, namespace string) []entry {
	if namespace == "" {
		return es
	}
	lo := sort.Search(len(es), func(i int) bool { return es[i].key.Namespace >= namespace })
	hi := sort.Search(len(es), func(i int) bool { return es[i].key.Namespace > namespace })
	return es[lo:hi]
}
