package lexicon

import (
	_ "embed"
	"fmt"

	"sigs.k8s.io/yaml"
)

//go:embed lexicon.yaml
var raw []byte

type Entry struct {
	K8s    string `json:"k8s"`
	Name   string `json:"name"`
	Flavor string `json:"flavor"`
	Docs   string `json:"docs"`
}

type PlainWord struct {
	Term  string `json:"term"`
	Means string `json:"means"`
	First string `json:"first"`
	Quiet bool   `json:"quiet"`
}

type Lexicon struct {
	Entries []Entry     `json:"entries"`
	Plain   []PlainWord `json:"plain"`
}

func Load() (Lexicon, error) {
	return parse(raw)
}

func parse(data []byte) (Lexicon, error) {
	var lx Lexicon
	if err := yaml.UnmarshalStrict(data, &lx); err != nil {
		return Lexicon{}, fmt.Errorf("failed to parse embedded lexicon: %w", err)
	}
	for i, e := range lx.Entries {
		if e.K8s == "" || e.Name == "" {
			return Lexicon{}, fmt.Errorf("lexicon entry %d is missing k8s or name", i)
		}
	}
	for i, w := range lx.Plain {
		if w.Term == "" || w.Means == "" {
			return Lexicon{}, fmt.Errorf("plain word %d is missing its term or its meaning", i)
		}
	}
	return lx, nil
}
