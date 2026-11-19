package kubectl

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	EtcdVersion    = "3.7.0"
	etcdAPIVersion = "3.7"
	etcdctlWords   = "error: the Speaking Stone knows these etcdctl words: etcdctl endpoint status [--write-out=table], etcdctl alarm list, etcdctl alarm disarm, etcdctl defrag, etcdctl compact <revision>, etcdctl version"
)

type EtcdStatus struct {
	Endpoint        string
	MemberID        uint64
	Version         string
	DBSize          int64
	DBSizeInUse     int64
	Quota           int64
	RaftTerm        uint64
	RaftIndex       uint64
	Revision        int64
	CompactRevision int64
	Alarms          []string
}

type Etcd interface {
	EtcdStatus() EtcdStatus
	Defragment() float64
	Compact(rev int64) error
	AlarmDisarm() []string
}

type etcdInvocation struct {
	words    []string
	writeOut string
}

func Etcdctl(e Etcd, command string) Result {
	tokens, err := tokenize(command)
	if err != nil {
		return fail("etcdctl", "", err)
	}
	if len(tokens) > 0 && tokens[0] == "etcdctl" {
		tokens = tokens[1:]
	}
	inv, err := parseEtcd(tokens)
	if err != nil {
		return fail("etcdctl", "", err)
	}
	out, verb, err := runEtcd(e, inv)
	if err != nil {
		return fail(verb, "", err)
	}
	return Result{Output: ensureNewline(out), OK: true, Verb: verb, Resource: "etcd"}
}

func parseEtcd(tokens []string) (etcdInvocation, error) {
	inv := etcdInvocation{writeOut: "simple"}
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		name, value, hasValue := strings.Cut(strings.TrimLeft(tok, "-"), "=")
		switch {
		case !strings.HasPrefix(tok, "-"):
			inv.words = append(inv.words, tok)
		case name == "w" || name == "write-out":
			if !hasValue {
				if i+1 >= len(tokens) {
					return inv, fmt.Errorf("Error: flag needs an argument: %s", tok)
				}
				i++
				value = tokens[i]
			}
			inv.writeOut = value
		case name == "cluster" || name == "endpoints":
		default:
			return inv, fmt.Errorf("Error: unknown flag: %s", tok)
		}
	}
	return inv, nil
}

func runEtcd(e Etcd, inv etcdInvocation) (string, string, error) {
	words := inv.words
	switch {
	case len(words) == 2 && words[0] == "endpoint" && words[1] == "status":
		return endpointStatus(e.EtcdStatus(), inv.writeOut)
	case len(words) >= 2 && words[0] == "alarm" && words[1] == "list":
		return alarmLines(e.EtcdStatus().Alarms), "alarm", noArgs(words, "alarm list")
	case len(words) >= 2 && words[0] == "alarm" && words[1] == "disarm":
		if err := noArgs(words, "alarm disarm"); err != nil {
			return "", "alarm", err
		}
		return alarmLines(e.AlarmDisarm()), "alarm", nil
	case len(words) == 1 && words[0] == "defrag":
		return defragLine(e), "defrag", nil
	case len(words) >= 1 && words[0] == "compact":
		return compact(e, words[1:])
	case len(words) == 1 && words[0] == "version":
		return fmt.Sprintf("etcdctl version: %s\nAPI version: %s", EtcdVersion, etcdAPIVersion), "version", nil
	}
	return "", "etcdctl", errors.New(etcdctlWords)
}

func noArgs(words []string, command string) error {
	if len(words) > 2 {
		return fmt.Errorf("Error: %s command accepts no arguments", command)
	}
	return nil
}

func alarmLines(alarms []string) string {
	if len(alarms) == 0 {
		return ""
	}
	return strings.Join(alarms, "\n")
}

func defragLine(e Etcd) string {
	took := time.Duration(e.Defragment() * float64(time.Second)).Round(time.Millisecond)
	return fmt.Sprintf("Finished defragmenting etcd member[%s]. took %s", e.EtcdStatus().Endpoint, took)
}

func compact(e Etcd, args []string) (string, string, error) {
	if len(args) != 1 {
		return "", "compact", errors.New("Error: compaction command needs 1 argument")
	}
	rev, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return "", "compact", fmt.Errorf("Error: bad number for revision: %v", err)
	}
	if err := e.Compact(rev); err != nil {
		return "", "compact", fmt.Errorf("Error: %v", err)
	}
	return fmt.Sprintf("compacted revision %d", rev), "compact", nil
}

func endpointStatus(s EtcdStatus, writeOut string) (string, string, error) {
	header := []string{"endpoint", "ID", "version", "storage version", "db size", "in use", "percentage not in use", "quota", "is leader",
		"is learner", "raft term", "raft index", "raft applied index", "errors", "downgrade target version", "downgrade enabled"}
	row := []string{s.Endpoint, fmt.Sprintf("%x", s.MemberID), s.Version, s.Version, humanBytes(uint64(s.DBSize)), humanBytes(uint64(s.DBSizeInUse)),
		fmt.Sprintf("%d%%", notInUse(s)), humanBytes(uint64(s.Quota)), "true", "false", fmt.Sprint(s.RaftTerm), fmt.Sprint(s.RaftIndex),
		fmt.Sprint(s.RaftIndex), strings.Join(s.Alarms, ", "), "", "false"}
	switch writeOut {
	case "simple":
		return strings.Join(row, ", "), "endpoint", nil
	case "table":
		return renderEtcdTable(header, [][]string{row}), "endpoint", nil
	}
	return "", "endpoint", fmt.Errorf("Error: the Speaking Stone reads etcdctl output only as simple or table, not %q", writeOut)
}

func notInUse(s EtcdStatus) int64 {
	if s.DBSize <= 0 {
		return 0
	}
	return 100 - s.DBSizeInUse*100/s.DBSize
}

func humanBytes(s uint64) string {
	if s < 10 {
		return fmt.Sprintf("%d B", s)
	}
	sizes := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}
	e := math.Floor(math.Log(float64(s)) / math.Log(1000))
	val := math.Floor(float64(s)/math.Pow(1000, e)*10+0.5) / 10
	format := "%.0f %s"
	if val < 10 {
		format = "%.1f %s"
	}
	return fmt.Sprintf(format, val, sizes[int(e)])
}
