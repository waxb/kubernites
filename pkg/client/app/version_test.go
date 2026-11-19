package app

import "testing"

func TestTheVersionNamesTheBuild(t *testing.T) {
	was := Commit
	defer func() { Commit = was }()
	Commit = ""
	if got := VersionLabel(); got != "v"+Version {
		t.Fatalf("a build without its commit reads %q", got)
	}
	Commit = "3e84d42"
	if got := VersionLabel(); got != "v"+Version+" (3e84d42)" {
		t.Fatalf("a build reads %q", got)
	}
}
