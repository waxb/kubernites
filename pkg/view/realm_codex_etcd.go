package view

import (
	"fmt"
)

const StatusCommand = "etcdctl endpoint status --write-out=table"

const AlarmsCommand = "etcdctl alarm list"

const DefragCommand = "etcdctl defrag"

const DisarmCommand = "etcdctl alarm disarm"

const compactCommand = "etcdctl compact %d"

func CompactCommand(rev int64) string {
	return fmt.Sprintf(compactCommand, rev)
}
