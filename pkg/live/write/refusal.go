package write

import (
	"errors"
	"fmt"
)

var ErrRefused = errors.New("refused by the write gate before it left Kubernites")

type Why int

const (
	NeedsConfirm Why = iota + 1
	NeedsClass
	NotAtLevel
	Stale
)

func (w Why) String() string {
	switch w {
	case NeedsConfirm:
		return "needs a preview and CONFIRM"
	case NeedsClass:
		return "needs a class warning"
	case NotAtLevel:
		return "not allowed at this level"
	case Stale:
		return "the realm is stale"
	}
	return "refused"
}

type Refusal struct {
	Method string
	Path   string
	Why    Why
	Class  Class
	Level  Level
	Detail string
}

func (r *Refusal) Error() string {
	reason := r.Why.String()
	switch r.Why {
	case NeedsConfirm:
		reason = "at WRITES every change needs its dry-run preview and CONFIRM first"
	case NeedsClass:
		reason = "the " + ClassTitle(r.Class) + " warning has not been passed for this context"
	case NotAtLevel:
		reason = "not allowed at " + r.Level.String()
		if r.Detail != "" {
			reason += ": " + r.Detail
		}
	case Stale:
		reason = "the realm is stale, so nothing is sent until the watch recovers"
		if r.Detail != "" {
			reason += " (" + r.Detail + ")"
		}
	}
	return fmt.Sprintf("%v: %s %s (%s)", ErrRefused, r.Method, r.Path, reason)
}

func (r *Refusal) Unwrap() error {
	return ErrRefused
}

func ClassTitle(c Class) string {
	switch c {
	case Secrets:
		return "Secrets"
	case Workloads:
		return "reaching into workloads"
	case Destroy:
		return "destroying more than one thing"
	case Unpreviewed:
		return "changing without a preview"
	case Tokens:
		return "minting a token"
	}
	return string(c)
}
