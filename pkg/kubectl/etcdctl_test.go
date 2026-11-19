package kubectl

import (
	"errors"
	"strings"
	"testing"
)

type fakeEtcd struct {
	status    EtcdStatus
	defragged int
	disarmed  int
	compacted int64
}

func (f *fakeEtcd) EtcdStatus() EtcdStatus { return f.status }

func (f *fakeEtcd) Defragment() float64 {
	f.defragged++
	return 1.25
}

func (f *fakeEtcd) Compact(rev int64) error {
	switch {
	case rev > f.status.Revision:
		return errors.New("etcdserver: mvcc: required revision is a future revision")
	case rev <= f.status.CompactRevision:
		return errors.New("etcdserver: mvcc: required revision has been compacted")
	}
	f.compacted = rev
	return nil
}

func (f *fakeEtcd) AlarmDisarm() []string {
	f.disarmed++
	out := f.status.Alarms
	f.status.Alarms = nil
	return out
}

func healthyEtcd() *fakeEtcd {
	return &fakeEtcd{status: EtcdStatus{
		Endpoint: "127.0.0.1:2379", MemberID: 0x8e9e05c52164694d, Version: "3.6.0", DBSize: 3_200_000, DBSizeInUse: 1_100_000,
		Quota: 8 * 1024 * 1024, RaftTerm: 2, RaftIndex: 4211, Revision: 4200, CompactRevision: 3900,
	}}
}

func TestEtcdctlEndpointStatusTable(t *testing.T) {
	e := healthyEtcd()
	res := Etcdctl(e, "etcdctl endpoint status --write-out=table")
	if !res.OK {
		t.Fatalf("not ok: %q", res.Output)
	}
	lines := strings.Split(strings.TrimRight(res.Output, "\n"), "\n")
	if len(lines) != 5 || !strings.HasPrefix(lines[0], "+----------------+") || lines[0] != lines[2] || lines[0] != lines[4] {
		t.Fatalf("table shape:\n%s", res.Output)
	}
	header := "|    ENDPOINT    |        ID        | VERSION | STORAGE VERSION | DB SIZE | IN USE | PERCENTAGE NOT IN USE | QUOTA  | IS LEADER | IS LEARNER | RAFT TERM | RAFT INDEX | RAFT APPLIED INDEX | ERRORS | DOWNGRADE TARGET VERSION | DOWNGRADE ENABLED |"
	if lines[1] != header {
		t.Fatalf("header\n got %s\nwant %s", lines[1], header)
	}
	row := "| 127.0.0.1:2379 | 8e9e05c52164694d |   3.6.0 |           3.6.0 |  3.2 MB | 1.1 MB |                   66% | 8.4 MB |      true |      false |         2 |       4211 |               4211 |        |                          |             false |"
	if lines[3] != row {
		t.Fatalf("row\n got %s\nwant %s", lines[3], row)
	}
}

func TestEtcdctlEndpointStatusShowsTheAlarmInErrors(t *testing.T) {
	e := healthyEtcd()
	e.status.Alarms = []string{"memberID:10276657743932975437 alarm:NOSPACE"}
	res := Etcdctl(e, "etcdctl -w table endpoint status")
	lines := strings.Split(strings.TrimRight(res.Output, "\n"), "\n")
	if len(lines) != 6 || !strings.Contains(lines[3], "| memberID:10276657743932975437 |") || !strings.Contains(lines[4], "|         alarm:NOSPACE         |") && !strings.Contains(lines[4], "alarm:NOSPACE |") {
		t.Fatalf("alarm row:\n%s", res.Output)
	}
}

func TestEtcdctlEndpointStatusSimple(t *testing.T) {
	res := Etcdctl(healthyEtcd(), "etcdctl endpoint status")
	want := "127.0.0.1:2379, 8e9e05c52164694d, 3.6.0, 3.6.0, 3.2 MB, 1.1 MB, 66%, 8.4 MB, true, false, 2, 4211, 4211, , , false\n"
	if res.Output != want {
		t.Fatalf("got  %q\nwant %q", res.Output, want)
	}
}

func TestEtcdctlAlarmListAndDisarm(t *testing.T) {
	e := healthyEtcd()
	if res := Etcdctl(e, "etcdctl alarm list"); !res.OK || res.Output != "" {
		t.Fatalf("quiet list = %+v", res)
	}
	e.status.Alarms = []string{"memberID:10276657743932975437 alarm:NOSPACE"}
	if res := Etcdctl(e, "etcdctl alarm list"); res.Output != "memberID:10276657743932975437 alarm:NOSPACE\n" {
		t.Fatalf("list = %q", res.Output)
	}
	if res := Etcdctl(e, "etcdctl alarm disarm"); res.Output != "memberID:10276657743932975437 alarm:NOSPACE\n" || e.disarmed != 1 {
		t.Fatalf("disarm = %q", res.Output)
	}
	if res := Etcdctl(e, "etcdctl alarm disarm extra"); res.OK || res.Output != "Error: alarm disarm command accepts no arguments\n" {
		t.Fatalf("bad disarm = %+v", res)
	}
}

func TestEtcdctlDefragReportsTheMember(t *testing.T) {
	e := healthyEtcd()
	res := Etcdctl(e, "etcdctl defrag")
	if res.Output != "Finished defragmenting etcd member[127.0.0.1:2379]. took 1.25s\n" || e.defragged != 1 || res.Verb != "defrag" {
		t.Fatalf("defrag = %+v", res)
	}
}

func TestEtcdctlCompact(t *testing.T) {
	e := healthyEtcd()
	if res := Etcdctl(e, "etcdctl compact 4100"); res.Output != "compacted revision 4100\n" || e.compacted != 4100 {
		t.Fatalf("compact = %+v", res)
	}
	if res := Etcdctl(e, "etcdctl compact 9999"); res.OK || res.Output != "Error: etcdserver: mvcc: required revision is a future revision\n" {
		t.Fatalf("future = %+v", res)
	}
	if res := Etcdctl(e, "etcdctl compact"); res.OK || res.Output != "Error: compaction command needs 1 argument\n" {
		t.Fatalf("missing = %+v", res)
	}
	if res := Etcdctl(e, "etcdctl compact abc"); res.OK || !strings.HasPrefix(res.Output, "Error: bad number for revision") {
		t.Fatalf("bad = %+v", res)
	}
}

func TestEtcdctlVersionAndUnknownWords(t *testing.T) {
	if res := Etcdctl(healthyEtcd(), "etcdctl version"); res.Output != "etcdctl version: 3.7.0\nAPI version: 3.7\n" {
		t.Fatalf("version = %q", res.Output)
	}
	for _, cmd := range []string{"etcdctl put a b", "etcdctl", "etcdctl endpoint", "etcdctl alarm"} {
		if res := Etcdctl(healthyEtcd(), cmd); res.OK || !strings.Contains(res.Output, "etcdctl endpoint status") {
			t.Fatalf("%s = %+v", cmd, res)
		}
	}
	if res := Etcdctl(healthyEtcd(), "etcdctl endpoint status -w json"); res.OK {
		t.Fatalf("json = %+v", res)
	}
}

func TestHumanBytesMatchesGoHumanize(t *testing.T) {
	cases := map[uint64]string{0: "0 B", 9: "9 B", 25_000: "25 kB", 8388608: "8.4 MB", 2147483648: "2.1 GB", 18_000_000: "18 MB", 999_999: "1000 kB"}
	for in, want := range cases {
		if got := humanBytes(in); got != want {
			t.Fatalf("humanBytes(%d) = %q, want %q", in, got, want)
		}
	}
}
