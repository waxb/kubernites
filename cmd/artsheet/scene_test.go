package main

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
)

func TestSceneSpritesExistForEveryAge(t *testing.T) {
	set := art.Build()
	for age := 0; age < art.AgeTiers; age++ {
		for _, s := range sceneSprites(set, age) {
			if _, ok := s.atlas.Regions[s.name]; !ok {
				t.Errorf("age %d: the scene names the missing sprite %s", age, s.name)
			}
		}
	}
	if img := renderScene(set, 2); img.Rect.Dx() == 0 {
		t.Fatal("the Castle scene rendered nothing")
	}
}
