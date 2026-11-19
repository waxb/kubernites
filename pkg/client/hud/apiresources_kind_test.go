package hud

import (
	"strings"
	"testing"
)

func TestStoryAPIResourcesKeepsAKindColumn(t *testing.T) {
	lines := []string{
		"NAME                       SHORTNAMES   APIVERSION                           NAMESPACED   KIND",
		"pods                       po           v1                                   true         Pod",
		"replicasets                rs           apps/v1                              true         ReplicaSet",
	}
	out := storyAPIResources(lines)
	if !strings.Contains(out[0], "KIND") {
		t.Fatalf("header = %q, want a KIND column", out[0])
	}
	if !strings.Contains(out[2], "shift") || strings.Contains(out[2], "ReplicaSet") {
		t.Errorf("the shifts row = %q, want its story kind", out[2])
	}
}
