package soulstone

import (
	"strings"
	"testing"
)

func TestProfilesNameTheirClassAndKinInStoryWords(t *testing.T) {
	for _, p := range append(Playable(), Familiars()...) {
		if strings.Contains(p.Name, "class of latest") {
			t.Errorf("%s is named %q: a tag, not a class", p.Image, p.Name)
		}
		if strings.Contains(p.Description, "fluent-bit") {
			t.Errorf("%s says %q: a product name in story mode", p.Image, p.Description)
		}
	}
}
