package write

import (
	"bytes"
	"encoding/json"
	"strings"

	"sigs.k8s.io/yaml"
)

type Reveal func(namespace, name, key string) bool

func MaskSecrets(text string, reveal Reveal) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return text, false
	}
	asJSON := strings.HasPrefix(trimmed, "{")
	data := []byte(trimmed)
	if !asJSON {
		var err error
		if data, err = yaml.YAMLToJSON(data); err != nil {
			return text, false
		}
	}
	var doc any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		return text, false
	}
	if !maskTree(doc, reveal) {
		return text, false
	}
	if asJSON {
		out, err := json.MarshalIndent(doc, "", "    ")
		if err != nil {
			return Masked, true
		}
		return string(out) + "\n", true
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return Masked, true
	}
	return string(out), true
}

func maskTree(node any, reveal Reveal) bool {
	obj, ok := node.(map[string]any)
	if !ok {
		return false
	}
	masked := false
	if items, ok := obj["items"].([]any); ok {
		for _, it := range items {
			masked = maskTree(it, reveal) || masked
		}
	}
	if obj["kind"] != "Secret" {
		return masked
	}
	meta, _ := obj["metadata"].(map[string]any)
	ns, _ := meta["namespace"].(string)
	name, _ := meta["name"].(string)
	for _, field := range []string{"data", "stringData"} {
		values, ok := obj[field].(map[string]any)
		if !ok {
			continue
		}
		for key := range values {
			if reveal == nil || !reveal(ns, name, key) {
				values[key] = Masked
			}
		}
		masked = true
	}
	if ann, ok := meta["annotations"].(map[string]any); ok {
		if _, ok := ann["kubectl.kubernetes.io/last-applied-configuration"]; ok {
			ann["kubectl.kubernetes.io/last-applied-configuration"] = Masked
			masked = true
		}
	}
	return masked
}
