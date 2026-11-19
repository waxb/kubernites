package write

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

func BodyShown(path, contentType string, body []byte) string {
	text := bytes.TrimSpace(body)
	if len(text) == 0 {
		return ""
	}
	doc, ok := jsonDoc(text)
	if !ok {
		kind := strings.TrimSpace(strings.Split(contentType, ";")[0])
		if kind == "" {
			kind = "an unnamed type"
		}
		return fmt.Sprintf("%d bytes of %s, not shown (only JSON is shown)", len(body), kind)
	}
	if UnderSecrets(path) || holdsSecret(doc) {
		var names []string
		fieldNames(doc, "", &names)
		return "fields " + strings.Join(names, ", ") + " (a Secret's values are never shown)"
	}
	return Mask(string(text))
}

func UnderSecrets(path string) bool {
	return parse(path).resource == "secrets"
}

func jsonDoc(text []byte) (any, bool) {
	if !json.Valid(text) {
		return nil, false
	}
	var doc any
	dec := json.NewDecoder(bytes.NewReader(text))
	dec.UseNumber()
	if dec.Decode(&doc) != nil {
		return nil, false
	}
	return doc, true
}

func holdsSecret(doc any) bool {
	switch x := doc.(type) {
	case map[string]any:
		if x["kind"] == "Secret" {
			return true
		}
		for _, v := range x {
			if holdsSecret(v) {
				return true
			}
		}
	case []any:
		return slices.ContainsFunc(x, holdsSecret)
	}
	return false
}

func fieldNames(doc any, at string, out *[]string) {
	switch x := doc.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			name := k
			if at != "" {
				name = at + "." + k
			}
			fieldNames(x[k], name, out)
		}
		if len(x) == 0 && at != "" {
			*out = append(*out, at)
		}
	case []any:
		for i, v := range x {
			fieldNames(v, at+"["+strconv.Itoa(i)+"]", out)
		}
		if len(x) == 0 && at != "" {
			*out = append(*out, at)
		}
	default:
		if at != "" {
			*out = append(*out, at)
		}
	}
}

func maskedAnswer(path string, data []byte) []byte {
	if !UnderSecrets(path) || len(bytes.TrimSpace(data)) == 0 {
		return data
	}
	doc, ok := jsonDoc(bytes.TrimSpace(data))
	if !ok {
		return nil
	}
	maskTree(doc, nil)
	out, err := json.Marshal(doc)
	if err != nil {
		return nil
	}
	return out
}
