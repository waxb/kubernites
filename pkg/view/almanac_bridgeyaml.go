package view

import (
	"fmt"
	"strings"
)

type parsedCommand struct {
	words []string
	flags map[string]string
}

func parseCommand(command string) parsedCommand {
	out := parsedCommand{flags: map[string]string{}}
	fields := strings.Fields(command)
	for i := 0; i < len(fields); i++ {
		tok := fields[i]
		if i == 0 && tok == "kubectl" {
			continue
		}
		if !strings.HasPrefix(tok, "-") {
			out.words = append(out.words, tok)
			continue
		}
		name, value, valued := strings.Cut(tok, "=")
		if !valued && name == "-c" && i+1 < len(fields) {
			value, i = fields[i+1], i+1
		}
		out.flags[strings.TrimLeft(name, "-")] = value
	}
	return out
}

var objectKinds = map[string]string{"deployment": "Deployment", "replicaset": "ReplicaSet", "pod": "Pod", "node": "Node", "service": "Service"}

func (p parsedCommand) target(from int) (string, string, bool) {
	if len(p.words) <= from {
		return "", "", false
	}
	kind, name, split := strings.Cut(p.words[from], "/")
	if !split {
		if len(p.words) <= from+1 {
			return "", "", false
		}
		name = p.words[from+1]
	}
	if alias, ok := kindAliases[kind]; ok {
		kind = alias
	}
	title, ok := objectKinds[kind]
	return title, name, ok && name != ""
}

func YAMLOf(command string) (string, bool) {
	p := parseCommand(command)
	if len(p.words) == 0 {
		return "", false
	}
	switch p.words[0] {
	case "scale":
		return p.scaleYAML()
	case "set":
		return p.setYAML()
	case "autoscale":
		return p.autoscaleYAML()
	case "create":
		return p.createYAML()
	case "run":
		return p.runYAML()
	case "expose":
		return p.exposeYAML()
	case "drain", "cordon":
		if len(p.words) < 2 {
			return "", false
		}
		return header("Node", p.words[1]) + "spec:\n  unschedulable: true", true
	}
	return "", false
}

func header(kind, name string) string {
	return fmt.Sprintf("kind: %s\nmetadata:\n  name: %s\n", kind, name)
}

func (p parsedCommand) scaleYAML() (string, bool) {
	kind, name, ok := p.target(1)
	replicas, set := p.flags["replicas"]
	if !ok || !set {
		return "", false
	}
	return header(kind, name) + "spec:\n  replicas: " + replicas, true
}

func containerOf(crew string, flags map[string]string) string {
	if c := flags["c"]; c != "" && c != "*" {
		return c
	}
	if c := flags["containers"]; c != "" && c != "*" {
		return c
	}
	return strings.TrimSuffix(crew, "s")
}

func (p parsedCommand) setYAML() (string, bool) {
	if len(p.words) < 2 {
		return "", false
	}
	kind, name, ok := p.target(2)
	if !ok {
		return "", false
	}
	template := header(kind, name) + "spec:\n  template:\n    spec:\n      containers:\n"
	switch p.words[1] {
	case "image":
		var b strings.Builder
		b.WriteString(template)
		first := 4
		if strings.Contains(p.words[2], "/") {
			first = 3
		}
		for _, pair := range p.words[min(first, len(p.words)):] {
			c, img, ok := strings.Cut(pair, "=")
			if !ok {
				continue
			}
			fmt.Fprintf(&b, "      - name: %s\n        image: %s\n", c, img)
		}
		return strings.TrimRight(b.String(), "\n"), strings.Contains(b.String(), "image: ")
	case "resources":
		body := resourceLines(p.flags)
		if body == "" {
			return "", false
		}
		return template + "      - name: " + containerOf(name, p.flags) + "\n        resources:\n" + body, true
	}
	return "", false
}

func resourceLines(flags map[string]string) string {
	var b strings.Builder
	for _, part := range []string{"requests", "limits"} {
		value := flags[part]
		if value == "" {
			continue
		}
		fmt.Fprintf(&b, "          %s:\n", part)
		for _, pair := range strings.Split(value, ",") {
			key, amount, ok := strings.Cut(pair, "=")
			if ok {
				fmt.Fprintf(&b, "            %s: %s\n", key, amount)
			}
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (p parsedCommand) autoscaleYAML() (string, bool) {
	kind, name, ok := p.target(1)
	if !ok {
		return "", false
	}
	var b strings.Builder
	fmt.Fprintf(&b, "apiVersion: autoscaling/v2\nkind: HorizontalPodAutoscaler\nmetadata:\n  name: %s\nspec:\n  scaleTargetRef:\n    apiVersion: apps/v1\n    kind: %s\n    name: %s\n", name, kind, name)
	if v := p.flags["min"]; v != "" {
		fmt.Fprintf(&b, "  minReplicas: %s\n", v)
	}
	if v := p.flags["max"]; v != "" {
		fmt.Fprintf(&b, "  maxReplicas: %s\n", v)
	}
	if cpu := p.flags["cpu"]; cpu != "" {
		b.WriteString("  metrics:\n  - type: Resource\n    resource:\n      name: cpu\n      target:\n")
		if pct, ok := strings.CutSuffix(cpu, "%"); ok {
			fmt.Fprintf(&b, "        type: Utilization\n        averageUtilization: %s\n", pct)
		} else {
			fmt.Fprintf(&b, "        type: AverageValue\n        averageValue: %s\n", cpu)
		}
	}
	return strings.TrimRight(b.String(), "\n"), true
}

func (p parsedCommand) createYAML() (string, bool) {
	if len(p.words) < 3 || p.words[1] != "deployment" || p.flags["image"] == "" {
		return "", false
	}
	name, image := p.words[2], p.flags["image"]
	replicas := p.flags["replicas"]
	if replicas == "" {
		replicas = "1"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: %s\nspec:\n  replicas: %s\n  selector:\n    matchLabels:\n      app: %s\n  template:\n    metadata:\n      labels:\n        app: %s\n    spec:\n      containers:\n      - name: %s\n        image: %s", name, replicas, name, name, imageName(image), image)
	if port := p.flags["port"]; port != "" {
		fmt.Fprintf(&b, "\n        ports:\n        - containerPort: %s", port)
	}
	return b.String(), true
}

func imageName(image string) string {
	name, _, _ := strings.Cut(image, ":")
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return name
}

func (p parsedCommand) runYAML() (string, bool) {
	if len(p.words) < 2 || p.flags["image"] == "" {
		return "", false
	}
	var b strings.Builder
	fmt.Fprintf(&b, "apiVersion: v1\nkind: Pod\nmetadata:\n  name: %s\n", p.words[1])
	if labels := p.flags["labels"]; labels != "" {
		b.WriteString("  labels:\n")
		for _, pair := range strings.Split(labels, ",") {
			if k, v, ok := strings.Cut(pair, "="); ok {
				fmt.Fprintf(&b, "    %s: %s\n", k, v)
			}
		}
	}
	fmt.Fprintf(&b, "spec:\n  containers:\n  - name: %s\n    image: %s", p.words[1], p.flags["image"])
	return b.String(), true
}

func (p parsedCommand) exposeYAML() (string, bool) {
	kind, name, ok := p.target(1)
	if !ok || kind == "Pod" || p.flags["name"] == "" {
		return "", false
	}
	var b strings.Builder
	fmt.Fprintf(&b, "apiVersion: v1\nkind: Service\nmetadata:\n  name: %s\nspec:\n  selector:\n    app: %s\n", p.flags["name"], name)
	if port := p.flags["port"]; port != "" {
		fmt.Fprintf(&b, "  ports:\n  - port: %s\n", port)
		if target := p.flags["target-port"]; target != "" {
			fmt.Fprintf(&b, "    targetPort: %s\n", target)
		}
	}
	return strings.TrimRight(b.String(), "\n"), true
}
