package store

import "errors"

var ErrNoSpace = errors.New("etcdserver: mvcc: database space exceeded")

type Meter interface {
	Admit(kind string, obj Object) error
	Wrote(op EventType, kind string, before, after Object)
}

type protoSized interface {
	Size() int
}

func SizeOf(obj Object) int {
	if obj == nil {
		return 0
	}
	if sized, ok := obj.(protoSized); ok {
		return sized.Size()
	}
	return 0
}

func (s *Store) SetMeter(m Meter) {
	s.meter = m
}

func (s *Store) admitWrite(kind string, obj Object) error {
	if s.meter == nil {
		return nil
	}
	return s.meter.Admit(kind, obj)
}

func (s *Store) meterWrite(op EventType, kind string, before, after Object) {
	if s.meter != nil {
		s.meter.Wrote(op, kind, before, after)
	}
}
