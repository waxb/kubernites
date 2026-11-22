package app

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestTheVersionNamesTheBuild(t *testing.T) {
	was := Commit
	defer func() { Commit = was }()
	Commit = ""
	if got := VersionLabel(); got != "v"+Version && !strings.HasPrefix(got, "v"+Version+" (") {
		t.Fatalf("a build without a set commit reads %q", got)
	}
	Commit = "3e84d42"
	if got := VersionLabel(); got != "v"+Version+" (3e84d42)" {
		t.Fatalf("a build reads %q", got)
	}
}

func TestCapturesShowOnlyTheMajorAndMinorVersionSoAPatchBumpChangesNoPicture(t *testing.T) {
	was, wasCommit := captureLabel, Commit
	defer func() { captureLabel, Commit = was, wasCommit }()
	captureLabel, Commit = true, "3e84d42"
	parts := strings.SplitN(Version, ".", 3)
	if got, want := VersionLabel(), "v"+parts[0]+"."+parts[1]; got != want {
		t.Fatalf("a capture reads %q, want %q (no patch level, no commit)", got, want)
	}
}

func TestTheChangelogsNewestEntryIsThisVersion(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^## ([0-9][0-9.]*)( .*)?$`).FindStringSubmatch(string(data))
	if m == nil || m[1] != Version {
		t.Fatalf("CHANGELOG.md's newest entry is %v, but the version is %s: add an entry for it", m, Version)
	}
}
