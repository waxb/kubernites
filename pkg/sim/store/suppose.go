package store

func (s *Store) Suppose(gone []Key, put []Object, fn func()) {
	saved := map[Key]Object{}
	present := map[Key]bool{}
	keep := func(k Key) {
		if _, seen := present[k]; seen {
			return
		}
		obj, ok := s.objects[k]
		present[k] = ok
		if ok {
			saved[k] = obj
		}
	}
	for _, k := range gone {
		keep(k)
		delete(s.objects, k)
		delete(s.sorted, k.Kind)
	}
	for _, obj := range put {
		k := KeyOf(obj)
		keep(k)
		s.objects[k] = obj
		delete(s.sorted, k.Kind)
	}
	s.restores++
	s.supposing++
	defer func() {
		s.supposing--
		for k, was := range present {
			if was {
				s.objects[k] = saved[k]
			} else {
				delete(s.objects, k)
			}
			delete(s.sorted, k.Kind)
		}
		s.restores++
	}()
	fn()
}
