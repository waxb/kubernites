package write

import (
	"slices"
	"strings"
)

type Level int

const (
	ReadOnly Level = iota
	Writes
	Seamless
)

func (l Level) String() string {
	switch l {
	case Writes:
		return "WRITES"
	case Seamless:
		return "DANGEROUSLY SEAMLESS"
	}
	return "READ ONLY"
}

func ParseLevel(s string) (Level, bool) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "READ ONLY", "READ-ONLY", "READONLY":
		return ReadOnly, true
	case "WRITES":
		return Writes, true
	case "DANGEROUSLY SEAMLESS", "SEAMLESS":
		return Seamless, true
	}
	return ReadOnly, false
}

type Class string

const (
	Secrets     Class = "secrets"
	Workloads   Class = "workloads"
	Destroy     Class = "destroy"
	Unpreviewed Class = "unpreviewed"
	Tokens      Class = "tokens"
)

var AllClasses = []Class{Secrets, Workloads, Destroy, Unpreviewed, Tokens}

func KnownClass(c Class) bool {
	return slices.Contains(AllClasses, c)
}

type Context struct {
	Name      string
	Server    string
	User      string
	Identity  string `json:",omitempty"`
	ClusterID string `json:",omitempty"`
}

type Choice struct {
	Context Context
	Level   Level
	Classes []Class
}

func UserAgent(l Level) string {
	return "kubernites-engineer-mode (" + strings.ToLower(l.String()) + ")"
}

const FieldManager = "kubernites"
