package kubectl

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/yaml"

	"github.com/waxb/kubernites/pkg/sim/store"
)

type getOptions struct {
	output       string
	showLabels   bool
	labelColumns []string
	allNS        bool
	prefix       bool
}

func get(c Cluster, inv Invocation) (string, string, error) {
	typeTok, names, err := SplitTypeName(inv.Args)
	if err != nil {
		return "", "", err
	}
	if typeTok == "all" {
		return getAll(c, inv), "all", nil
	}
	if strings.Contains(typeTok, ",") {
		return getMany(c, inv, strings.Split(typeTok, ","), names)
	}
	if gk, ok := gatewayKindOf(typeTok); ok {
		out, err := gatewayTable(c, gk, inv, names, false)
		return out, gk.plural, err
	}
	r, err := ResolveType(typeTok)
	if err != nil {
		return "", typeTok, err
	}
	objs, err := SelectObjects(c, r, inv, names)
	if err != nil {
		return "", r.Plural, err
	}
	opts := getOptions{output: inv.Flags["output"], showLabels: inv.Bools["show-labels"], labelColumns: labelColumns(inv), allNS: inv.allNS && r.Namespaced}
	if len(objs) == 0 {
		return noResources(r, inv), r.Plural, nil
	}
	if r.Kind == "Event" {
		sortEvents(objs)
	}
	out, err := render(c, r, objs, opts, len(names) == 1)
	return out, r.Plural, err
}

func getMany(c Cluster, inv Invocation, tokens, names []string) (string, string, error) {
	if len(names) > 0 {
		return "", "all", errors.New("error: you may only specify a single resource type")
	}
	output := inv.Flags["output"]
	sections, all := []string{}, []store.Object{}
	for _, token := range tokens {
		if gk, ok := gatewayKindOf(strings.TrimSpace(token)); ok && output != "yaml" && output != "json" {
			out, err := gatewayTable(c, gk, inv, nil, true)
			if err != nil {
				return "", gk.plural, err
			}
			if !strings.HasPrefix(out, "No resources found") {
				sections = append(sections, strings.TrimRight(out, "\n"))
			}
			continue
		}
		r, err := ResolveType(strings.TrimSpace(token))
		if err != nil {
			return "", "all", err
		}
		objs, err := SelectObjects(c, r, inv, nil)
		if err != nil {
			return "", r.Plural, err
		}
		if len(objs) == 0 {
			continue
		}
		if output == "yaml" || output == "json" {
			all = append(all, objs...)
			continue
		}
		opts := getOptions{output: output, showLabels: inv.Bools["show-labels"], labelColumns: labelColumns(inv), allNS: inv.allNS && r.Namespaced, prefix: true}
		out, err := render(c, r, objs, opts, false)
		if err != nil {
			return "", r.Plural, err
		}
		sections = append(sections, strings.TrimRight(out, "\n"))
	}
	if len(all) > 0 {
		out, err := Marshal(all, output, false)
		return out, "all", err
	}
	if len(sections) == 0 {
		return fmt.Sprintf("No resources found in %s namespace.", inv.Namespace), "all", nil
	}
	return strings.Join(sections, "\n\n"), "all", nil
}

func SelectObjects(c Cluster, r ResourceType, inv Invocation, names []string) ([]store.Object, error) {
	ns := NamespaceFor(r, inv)
	if inv.allNS {
		ns = ""
	}
	if len(names) > 0 {
		out := []store.Object{}
		for _, n := range names {
			obj, ok := c.Store().Get(store.Key{Kind: r.Kind, Namespace: ns, Name: n})
			if !ok {
				return nil, NotFound(r, n)
			}
			out = append(out, obj)
		}
		return out, nil
	}
	selector := labels.Everything()
	if raw := inv.Flags["selector"]; raw != "" {
		parsed, err := labels.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("error: unable to parse requirement: %v", err)
		}
		selector = parsed
	}
	byField, err := fieldSelector(r.Kind, inv.Flags["field-selector"])
	if err != nil {
		return nil, err
	}
	out := []store.Object{}
	for _, obj := range c.Store().List(r.Kind, ns) {
		if selector.Matches(labels.Set(obj.GetLabels())) && byField.Matches(objectFields(obj)) {
			out = append(out, obj)
		}
	}
	return out, nil
}

func noResources(r ResourceType, inv Invocation) string {
	if !r.Namespaced || inv.allNS {
		return "No resources found"
	}
	return fmt.Sprintf("No resources found in %s namespace.", inv.Namespace)
}

func render(c Cluster, r ResourceType, objs []store.Object, opts getOptions, single bool) (string, error) {
	switch opts.output {
	case "yaml", "json":
		return Marshal(objs, opts.output, single)
	case "name":
		lines := make([]string, 0, len(objs))
		for _, o := range objs {
			lines = append(lines, r.Qualified()+"/"+o.GetName())
		}
		return strings.Join(lines, "\n"), nil
	case "", "wide":
		return table(c, r, objs, opts), nil
	}
	return "", fmt.Errorf("error: unable to match a printer suitable for the output format %q, allowed formats are: json,name,wide,yaml", opts.output)
}

func Marshal(objs []store.Object, format string, single bool) (string, error) {
	var payload any = objs[0]
	if !single || len(objs) > 1 {
		payload = map[string]any{"apiVersion": "v1", "kind": "List", "items": objs, "metadata": map[string]string{"resourceVersion": ""}}
	}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return "", err
	}
	if format == "json" {
		j, err := yaml.YAMLToJSON(data)
		if err != nil {
			return "", err
		}
		return string(j), nil
	}
	return string(data), nil
}

func table(c Cluster, r ResourceType, objs []store.Object, opts getOptions) string {
	p := printerFor(r.Kind)
	headers := p.headers(opts.output == "wide")
	if opts.allNS {
		headers = append([]string{"NAMESPACE"}, headers...)
	}
	for _, key := range opts.labelColumns {
		headers = append(headers, labelColumnHeader(key))
	}
	if opts.showLabels {
		headers = append(headers, "LABELS")
	}
	rows := make([][]string, 0, len(objs))
	for _, o := range objs {
		row := p.row(c, o, opts.output == "wide")
		if opts.prefix {
			row[0] = r.Qualified() + "/" + row[0]
		}
		if opts.allNS {
			row = append([]string{o.GetNamespace()}, row...)
		}
		for _, key := range opts.labelColumns {
			row = append(row, o.GetLabels()[key])
		}
		if opts.showLabels {
			row = append(row, labelString(o.GetLabels()))
		}
		rows = append(rows, row)
	}
	return renderTable(headers, rows)
}

func labelColumns(inv Invocation) []string {
	out := []string{}
	for _, key := range strings.Split(inv.Flags["label-columns"], ",") {
		if key = strings.TrimSpace(key); key != "" {
			out = append(out, key)
		}
	}
	return out
}

func labelColumnHeader(key string) string {
	parts := strings.Split(key, "/")
	return strings.ToUpper(parts[len(parts)-1])
}

func getAll(c Cluster, inv Invocation) string {
	sections := []string{}
	for _, name := range []string{"pods", "services", "deployments", "replicasets", "horizontalpodautoscalers"} {
		r, _ := ResolveType(name)
		objs, _ := SelectObjects(c, r, inv, nil)
		if len(objs) == 0 {
			continue
		}
		sections = append(sections, table(c, r, objs, getOptions{prefix: true, allNS: inv.allNS}))
	}
	if len(sections) == 0 {
		return fmt.Sprintf("No resources found in %s namespace.", inv.Namespace)
	}
	return strings.Join(sections, "\n")
}

func sortEvents(objs []store.Object) {
	sort.SliceStable(objs, func(i, j int) bool {
		a, b := eventLastSeen(objs[i].(*corev1.Event)), eventLastSeen(objs[j].(*corev1.Event))
		return a.Before(&b)
	})
}
