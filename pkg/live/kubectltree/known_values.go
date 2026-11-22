package kubectltree

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"slices"
	"strings"
)

const (
	knownAnchor  = 64
	knownMinimum = 4
)

type knownEdge struct {
	b  byte
	to int32
}

type knownMatcher struct {
	root  [256]int32
	edges [][]knownEdge
	depth []int32
	fail  []int32
	dict  []int32
	out   [][]string
}

func knownForms(known []string) []string {
	var forms []string
	for _, v := range known {
		if len(v) < knownMinimum {
			continue
		}
		forms = append(forms, v, base64.StdEncoding.EncodeToString([]byte(v)))
		if q, err := json.Marshal(v); err == nil {
			forms = append(forms, string(q[1:len(q)-1]))
		}
	}
	slices.Sort(forms)
	return slices.Compact(forms)
}

func (m *knownMatcher) child(node int32, b byte) int32 {
	if node == 0 {
		return m.root[b]
	}
	for _, e := range m.edges[node] {
		if e.b == b {
			return e.to
		}
	}
	return 0
}

func (m *knownMatcher) add() int32 {
	m.edges = append(m.edges, nil)
	m.depth = append(m.depth, 0)
	m.fail = append(m.fail, 0)
	m.dict = append(m.dict, 0)
	m.out = append(m.out, nil)
	return int32(len(m.edges) - 1)
}

func newKnownMatcher(forms []string) *knownMatcher {
	m := &knownMatcher{}
	m.add()
	for _, f := range forms {
		node := int32(0)
		for i := 0; i < len(f) && i < knownAnchor; i++ {
			next := m.child(node, f[i])
			if next == 0 {
				next = m.add()
				m.depth[next] = m.depth[node] + 1
				if node == 0 {
					m.root[f[i]] = next
				} else {
					m.edges[node] = append(m.edges[node], knownEdge{b: f[i], to: next})
				}
			}
			node = next
		}
		m.out[node] = append(m.out[node], f)
	}
	for i := range m.out {
		slices.SortFunc(m.out[i], func(a, b string) int { return len(b) - len(a) })
	}
	var queue []int32
	for b := range 256 {
		if c := m.root[b]; c != 0 {
			queue = append(queue, c)
		}
	}
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		for _, e := range m.edges[u] {
			f := m.fail[u]
			for f != 0 && m.child(f, e.b) == 0 {
				f = m.fail[f]
			}
			if t := m.child(f, e.b); t != e.to {
				m.fail[e.to] = t
			}
			if t := m.fail[e.to]; len(m.out[t]) > 0 {
				m.dict[e.to] = t
			} else {
				m.dict[e.to] = m.dict[t]
			}
			queue = append(queue, e.to)
		}
	}
	return m
}

func matchedAt(text string, at int, form string, work *int) bool {
	if len(text)-at < len(form) {
		return false
	}
	for i := 0; i < len(form); i++ {
		*work++
		if text[at+i] != form[i] {
			return false
		}
	}
	return true
}

func maskKnownValues(ctx context.Context, text string, known []string, work *int) (string, bool) {
	forms := knownForms(known)
	if len(forms) == 0 || text == "" {
		return text, true
	}
	m := newKnownMatcher(forms)
	reach := make([]int32, len(text))
	next := *work + maskCheckEvery
	state := int32(0)
	for i := 0; i < len(text); i++ {
		if *work >= next {
			next = *work + maskCheckEvery
			if ctx.Err() != nil {
				return "", false
			}
		}
		c := text[i]
		*work++
		for state != 0 && m.child(state, c) == 0 {
			state = m.fail[state]
			*work++
		}
		state = m.child(state, c)
		node := state
		if len(m.out[node]) == 0 {
			node = m.dict[node]
		}
		for ; node != 0; node = m.dict[node] {
			start := i + 1 - int(m.depth[node])
			for _, f := range m.out[node] {
				if matchedAt(text, start, f, work) {
					if end := int32(start + len(f)); end > reach[start] {
						reach[start] = end
					}
					break
				}
			}
		}
	}
	var out strings.Builder
	for i := 0; i < len(text); {
		if int(reach[i]) <= i {
			out.WriteByte(text[i])
			i++
			continue
		}
		end := int(reach[i])
		for j := i; j < end; j++ {
			end = max(end, int(reach[j]))
		}
		out.WriteString(Masked)
		*work += end - i
		i = end
	}
	return out.String(), true
}
