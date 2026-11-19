package soulstone

import "testing"

func TestEveryPlayableRepositoryHasAOneLineTagline(t *testing.T) {
	for _, repo := range Repositories() {
		line := taglines[repo]
		if line == "" {
			t.Errorf("repository %s has no tagline: the registry's repository card would be blank", repo)
		}
		if len([]rune(line)) > TaglineChars {
			t.Errorf("tagline of %s is %d characters, the card fits %d: %q", repo, len([]rune(line)), TaglineChars, line)
		}
	}
	if Tagline("porter:1.1") != taglines["porter"] || Tagline("nothing:9") != "" {
		t.Error("Tagline does not follow the repository")
	}
}
