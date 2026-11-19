package store

func (s *Store) Mirror(obj Object) error {
	kind, err := KindOf(obj)
	if err != nil {
		return err
	}
	key := Key{Kind: kind, Namespace: obj.GetNamespace(), Name: obj.GetName()}
	stored := copyOf(obj)
	setGVK(stored, kind)
	_, existed := s.objects[key]
	s.rv++
	s.objects[key] = stored
	s.putEntry(key, stored)
	if existed {
		s.emit(Modified, stored)
	} else {
		s.emit(Added, stored)
	}
	return nil
}

func (s *Store) Unmirror(key Key) bool {
	current, ok := s.objects[key]
	if !ok {
		return false
	}
	delete(s.objects, key)
	s.rv++
	s.dropEntry(key)
	s.emit(Deleted, current)
	return true
}
