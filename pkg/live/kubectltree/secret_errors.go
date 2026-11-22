package kubectltree

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/waxb/kubernites/pkg/live/write"
)

const maxMaskedErrors = 1 << 20

func makesSecretCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil && c.Parent() != nil; c = c.Parent() {
		if c.Name() == "secret" && c.Parent().Name() == "create" {
			return true
		}
	}
	return false
}

func typedSecretValues(cmd *cobra.Command) []string {
	var out []string
	for _, name := range []string{"from-literal", "docker-password"} {
		f := cmd.Flags().Lookup(name)
		if f == nil || !f.Changed {
			continue
		}
		for _, v := range flagValues(f) {
			if name == "from-literal" {
				_, v, _ = strings.Cut(v, "=")
			}
			out = append(out, v)
		}
	}
	return out
}

func leafStrings(v any) []string {
	var out []string
	switch x := v.(type) {
	case string:
		out = append(out, x)
		if b, err := base64.StdEncoding.DecodeString(x); err == nil && len(b) > 0 {
			out = append(out, string(b))
		}
	case map[string]any:
		for _, e := range x {
			out = append(out, leafStrings(e)...)
		}
	case []any:
		for _, e := range x {
			out = append(out, leafStrings(e)...)
		}
	}
	return out
}

func maskErrors(text string, known []string) string {
	out, _, _ := maskErrorsIn(context.Background(), text, known)
	return out
}

const maskCheckEvery = 64 << 10

var errMaskStopped = "error: stopped: the Stone held errors about a Secret and STOP ended their masking, so they are not shown\n"

func maskErrorsIn(ctx context.Context, text string, known []string) (string, int, bool) {
	if text == "" {
		return text, 0, true
	}
	if len(text) > maxMaskedErrors {
		return fmt.Sprintf("error: the Stone held %d bytes of errors about a Secret and could not mask them, so they are not shown\n", len(text)), 0, true
	}
	work := len(text)
	if masked, ok := write.MaskSecrets(text, nil); ok {
		text = masked
	}
	next := maskCheckEvery
	stopped := func() bool {
		if work < next {
			return false
		}
		next = work + maskCheckEvery
		return ctx.Err() != nil
	}
	var out strings.Builder
	for i := 0; i < len(text); {
		if stopped() {
			return errMaskStopped, work, false
		}
		c := text[i]
		if c != '{' && c != '[' {
			out.WriteByte(c)
			i++
			work++
			continue
		}
		end, ok, spans := scanJSON(text, i)
		work += end - i
		if ok {
			spans = [][2]int{{i, end}}
		}
		pos := i
		for _, sp := range spans {
			out.WriteString(text[pos:sp[0]])
			out.WriteString(maskSpan(text[sp[0]:sp[1]]))
			work += sp[1] - sp[0]
			pos = sp[1]
		}
		out.WriteString(text[pos:end])
		i = end
	}
	text = maskKeyedObjects(out.String())
	work += len(text)
	if ctx.Err() != nil {
		return errMaskStopped, work, false
	}
	text, ok := maskKnownValues(ctx, text, known, &work)
	if !ok {
		return errMaskStopped, work, false
	}
	return text, work, true
}

func maskSpan(span string) string {
	dec := json.NewDecoder(strings.NewReader(span))
	dec.UseNumber()
	var doc any
	if dec.Decode(&doc) != nil || !maskFragment(doc, "") {
		return span
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return Masked
	}
	return string(b)
}

func jsonSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func scanString(text string, i int) (int, bool) {
	for j := i + 1; j < len(text); j++ {
		switch c := text[j]; {
		case c == '"':
			return j + 1, true
		case c == '\\':
			j++
			if j >= len(text) {
				return j, false
			}
			switch text[j] {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
			case 'u':
				if j+4 >= len(text) {
					return len(text), false
				}
				for k := j + 1; k <= j+4; k++ {
					if !strings.ContainsRune("0123456789abcdefABCDEF", rune(text[k])) {
						return k, false
					}
				}
				j += 4
			default:
				return j, false
			}
		case c < 0x20:
			return j, false
		}
	}
	return len(text), false
}

func scanScalar(text string, i int) (int, bool) {
	for _, lit := range []string{"true", "false", "null"} {
		if strings.HasPrefix(text[i:], lit) {
			return i + len(lit), true
		}
	}
	j := i
	if j < len(text) && text[j] == '-' {
		j++
	}
	digits := func() bool {
		k := j
		for j < len(text) && text[j] >= '0' && text[j] <= '9' {
			j++
		}
		return j > k
	}
	if j < len(text) && text[j] == '0' {
		j++
	} else if !digits() {
		return j, false
	}
	if j < len(text) && text[j] == '.' {
		j++
		if !digits() {
			return j, false
		}
	}
	if j < len(text) && (text[j] == 'e' || text[j] == 'E') {
		j++
		if j < len(text) && (text[j] == '+' || text[j] == '-') {
			j++
		}
		if !digits() {
			return j, false
		}
	}
	return j, true
}

const (
	wantFirst = iota
	wantKey
	wantColon
	wantValue
	wantNext
)

func scanJSON(text string, start int) (int, bool, [][2]int) {
	type frame struct {
		open  int
		kind  byte
		state int
	}
	var stack []frame
	var spans [][2]int
	stack = append(stack, frame{open: start, kind: text[start], state: wantFirst})
	i := start + 1
	for {
		for i < len(text) && jsonSpace(text[i]) {
			i++
		}
		if i >= len(text) {
			return len(text), false, spans
		}
		top := &stack[len(stack)-1]
		c := text[i]
		closing := byte('}')
		if top.kind == '[' {
			closing = ']'
		}
		switch {
		case c == closing && (top.state == wantFirst || top.state == wantNext):
			open := top.open
			stack = stack[:len(stack)-1]
			i++
			for len(spans) > 0 && spans[len(spans)-1][0] >= open {
				spans = spans[:len(spans)-1]
			}
			spans = append(spans, [2]int{open, i})
			if len(stack) == 0 {
				return i, true, spans
			}
			continue
		case top.state == wantNext:
			if c != ',' {
				return i, false, spans
			}
			i++
			top.state = wantValue
			if top.kind == '{' {
				top.state = wantKey
			}
			continue
		case top.kind == '{' && (top.state == wantFirst || top.state == wantKey):
			if c != '"' {
				return i, false, spans
			}
			end, ok := scanString(text, i)
			if !ok {
				return end, false, spans
			}
			i, top.state = end, wantColon
			continue
		case top.state == wantColon:
			if c != ':' {
				return i, false, spans
			}
			i++
			top.state = wantValue
			continue
		}
		top.state = wantNext
		switch {
		case c == '{' || c == '[':
			stack = append(stack, frame{open: i, kind: c, state: wantFirst})
			i++
		case c == '"':
			end, ok := scanString(text, i)
			if !ok {
				return end, false, spans
			}
			i = end
		default:
			end, ok := scanScalar(text, i)
			if !ok {
				return max(end, i+1), false, spans
			}
			i = end
		}
	}
}

var keyedObject = regexp.MustCompile(`"(?:data|stringData)"\s*:\s*\{|"` + regexp.QuoteMeta(lastConfig) + `"\s*:\s*"`)

func maskKeyedObjects(text string) string {
	var out strings.Builder
	pos := 0
	for _, m := range keyedObject.FindAllStringIndex(text, -1) {
		if m[0] < pos {
			continue
		}
		out.WriteString(text[pos:m[1]])
		i := m[1]
		if text[i-1] == '"' {
			end, _ := scanString(text, i-1)
			out.WriteString(Masked + `"`)
			pos = max(end, i)
			continue
		}
		depth := 1
		value := false
		for i < len(text) && depth > 0 {
			switch c := text[i]; c {
			case '{', '[':
				depth++
				out.WriteByte(c)
				i++
			case '}', ']':
				depth--
				out.WriteByte(c)
				i++
			case ':':
				value = true
				out.WriteByte(c)
				i++
			case ',':
				value = false
				out.WriteByte(c)
				i++
			case '"':
				end, _ := scanString(text, i)
				if value && depth == 1 {
					out.WriteString(`"` + Masked + `"`)
				} else {
					out.WriteString(text[i:end])
				}
				i = end
			default:
				out.WriteByte(c)
				i++
			}
		}
		pos = i
	}
	out.WriteString(text[pos:])
	return out.String()
}

func maskFragment(v any, parent string) bool {
	changed := false
	switch x := v.(type) {
	case map[string]any:
		if p, ok := x["path"].(string); ok && (strings.HasPrefix(p, "/data") || strings.HasPrefix(p, "/stringData")) {
			if _, has := x["value"]; has {
				x["value"] = Masked
				changed = true
			}
		}
		for k, e := range x {
			switch {
			case (k == "data" || k == "stringData") && parent != "metadata":
				if m, ok := e.(map[string]any); ok {
					for key := range m {
						m[key] = Masked
					}
					changed = true
					continue
				}
			case k == lastConfig:
				x[k] = maskLastApplied(e)
				changed = true
				continue
			}
			if maskFragment(e, k) {
				changed = true
			}
		}
	case []any:
		for _, e := range x {
			if maskFragment(e, parent) {
				changed = true
			}
		}
	}
	return changed
}
