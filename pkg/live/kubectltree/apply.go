package kubectltree

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

type Conflict struct {
	Manager     string
	Subresource string
	Fields      []string
}

type Applied struct {
	Result
	Conflicts []Conflict
	Stale     bool
}

func ApplyArgs(file string, force bool) []string {
	args := []string{"apply", "--server-side", "--field-manager=" + FieldManager, "-f", file}
	if force {
		args = append(args, "--force-conflicts")
	}
	return args
}

func (t *Tree) Apply(ctx context.Context, text string, force bool) Applied {
	return t.apply(ctx, text, force, hooks{})
}

func (t *Tree) apply(ctx context.Context, text string, force bool, h hooks) Applied {
	fail := func(err error) Applied {
		res := Result{Code: 1, Output: "error: " + err.Error() + "\n", Command: ApplyArgs("-", force)}
		res.Stderr = res.Output
		return Applied{Result: res}
	}
	body, err := Sendable(text)
	if err != nil {
		return fail(err)
	}
	in, feed, err := os.Pipe()
	if err != nil {
		return fail(err)
	}
	go func() {
		feed.Write(body)
		feed.Close()
	}()
	h.stdin = in
	res := t.runAs(ctx, ApplyArgs("-", force), h)
	in.Close()
	out := Applied{Result: res}
	if res.Code != 0 {
		out.Conflicts = Conflicts(res.Stderr)
		out.Stale = len(out.Conflicts) == 0 && strings.Contains(res.Stderr, "the object has been modified")
	}
	return out
}

func Sendable(text string) ([]byte, error) {
	var obj map[string]any
	if err := yaml.Unmarshal([]byte(text), &obj); err != nil {
		return nil, fmt.Errorf("the Scriptorium cannot read this: %w", err)
	}
	if len(obj) == 0 {
		return nil, fmt.Errorf("the Scriptorium holds no object")
	}
	u := &unstructured.Unstructured{Object: obj}
	if u.GetAPIVersion() == "" || u.GetKind() == "" || u.GetName() == "" {
		return nil, fmt.Errorf("the Scriptorium needs the object's apiVersion, kind and name")
	}
	u.SetManagedFields(nil)
	u.SetGeneration(0)
	u.SetUID("")
	u.SetSelfLink("")
	unstructured.RemoveNestedField(u.Object, "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(u.Object, "status")
	return yaml.Marshal(u.Object)
}

var (
	conflictHead = regexp.MustCompile(`conflicts? with "([^"]+)"(?: with subresource "([^"]+)")?(?: using \S+(?: at \S+)?)?:(.*)$`)
	conflictLine = regexp.MustCompile(`^\s*-\s+(\S.*)$`)
)

func Conflicts(stderr string) []Conflict {
	var out []Conflict
	var cur *Conflict
	for _, line := range strings.Split(stderr, "\n") {
		if m := conflictHead.FindStringSubmatch(line); m != nil {
			out = append(out, Conflict{Manager: m[1], Subresource: m[2]})
			cur = &out[len(out)-1]
			if f := strings.TrimSpace(m[3]); f != "" {
				cur.Fields = append(cur.Fields, f)
			}
			continue
		}
		if cur == nil {
			continue
		}
		if m := conflictLine.FindStringSubmatch(line); m != nil {
			cur.Fields = append(cur.Fields, strings.TrimSpace(m[1]))
			continue
		}
		cur = nil
	}
	return out
}
