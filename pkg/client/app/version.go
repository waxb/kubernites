package app

import (
	"runtime/debug"
	"strings"
)

const Version = "0.15.1"

var Commit = ""

var captureLabel bool

func captureVersion() string {
	parts := strings.SplitN(Version, ".", 3)
	if len(parts) < 2 {
		return Version
	}
	return parts[0] + "." + parts[1]
}

func VersionLabel() string {
	if captureLabel {
		return "v" + captureVersion()
	}
	c := Commit
	if c == "" {
		c = buildCommit()
	}
	if c == "" {
		return "v" + Version
	}
	return "v" + Version + " (" + c + ")"
}

func buildCommit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	rev, modified := "", false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if rev != "" && modified {
		rev += ", modified"
	}
	return rev
}
