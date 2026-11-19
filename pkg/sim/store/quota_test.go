package store

import (
	"errors"
	"testing"
)

type recordedWrite struct {
	op            EventType
	kind          string
	before, after bool
}

type fakeMeter struct {
	refuse error
	writes []recordedWrite
	costs  []int
}

func (m *fakeMeter) Admit(kind string, obj Object) error {
	m.costs = append(m.costs, SizeOf(obj))
	return m.refuse
}

func (m *fakeMeter) Wrote(op EventType, kind string, before, after Object) {
	m.writes = append(m.writes, recordedWrite{op: op, kind: kind, before: before != nil, after: after != nil})
}

func TestMeterSeesEveryWriteWithBeforeAndAfter(t *testing.T) {
	s := cluster()
	m := &fakeMeter{}
	s.SetMeter(m)
	created, err := s.Create(newPod("a"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Update(created); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(KeyOf(created)); err != nil {
		t.Fatal(err)
	}
	want := []recordedWrite{{Added, KindPod, false, true}, {Modified, KindPod, true, true}, {Deleted, KindPod, true, false}}
	if len(m.writes) != len(want) {
		t.Fatalf("writes = %+v", m.writes)
	}
	for i := range want {
		if m.writes[i] != want[i] {
			t.Fatalf("write %d = %+v, want %+v", i, m.writes[i], want[i])
		}
	}
	if len(m.costs) != 2 || m.costs[0] <= 0 {
		t.Fatalf("admit costs = %v", m.costs)
	}
}

func TestMeterRefusesCreatesAndUpdatesButNeverDeletes(t *testing.T) {
	s := cluster()
	created, err := s.Create(newPod("a"))
	if err != nil {
		t.Fatal(err)
	}
	m := &fakeMeter{refuse: ErrNoSpace}
	s.SetMeter(m)
	rv := s.ResourceVersion()
	if _, err := s.Create(newPod("b")); !errors.Is(err, ErrNoSpace) {
		t.Fatalf("create err = %v", err)
	}
	if _, err := s.Update(created); !errors.Is(err, ErrNoSpace) {
		t.Fatalf("update err = %v", err)
	}
	if s.ResourceVersion() != rv {
		t.Fatalf("a refused write moved the revision: %d -> %d", rv, s.ResourceVersion())
	}
	if err := s.Delete(KeyOf(created)); err != nil {
		t.Fatalf("delete refused: %v", err)
	}
	if len(m.writes) != 1 || m.writes[0].op != Deleted {
		t.Fatalf("writes = %+v", m.writes)
	}
}

func TestMeterDoesNotHideExistenceErrors(t *testing.T) {
	s := cluster()
	if _, err := s.Create(newPod("a")); err != nil {
		t.Fatal(err)
	}
	s.SetMeter(&fakeMeter{refuse: ErrNoSpace})
	if _, err := s.Create(newPod("a")); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate create err = %v", err)
	}
	if _, err := s.Update(newPod("zz")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing update err = %v", err)
	}
}

func TestNoSpaceCarriesTheEtcdMessage(t *testing.T) {
	if ErrNoSpace.Error() != "etcdserver: mvcc: database space exceeded" {
		t.Fatalf("message = %q", ErrNoSpace.Error())
	}
}

func TestSizeOfUsesProtobufSize(t *testing.T) {
	small := SizeOf(newPod("a"))
	big := newPod("a")
	big.Labels = map[string]string{"app": "lantern", "tier": "a-rather-long-label-value-for-weight"}
	if small <= 0 || SizeOf(big) <= small {
		t.Fatalf("sizes small=%d big=%d", small, SizeOf(big))
	}
	if SizeOf(nil) != 0 {
		t.Fatal("nil has no size")
	}
}
