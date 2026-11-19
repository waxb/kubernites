package hud

import (
	"fmt"
	"reflect"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

const liveCrewName = "porters"

type liveCrew struct {
	name      string
	Replicas  int    `json:"-"`
	Image     string `json:"-"`
	resources any
	Text      string `json:"-"`
}

func at(v any, keys ...string) any {
	for _, k := range keys {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func labelsOf(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func indented(v any, pad string) []string {
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	for i, l := range lines {
		lines[i] = pad + l
	}
	return lines
}

func labelBlock(head string, v any, pad string) []string {
	if len(labelsOf(v)) == 0 {
		return nil
	}
	return append([]string{pad + head + ":"}, indented(v, pad+"  ")...)
}

var containerNoise = map[string]bool{
	"name": true, "image": true, "terminationMessagePath": true, "terminationMessagePolicy": true, "imagePullPolicy": true,
}

var podExtras = []string{"nodeSelector", "tolerations", "affinity", "topologySpreadConstraints", "priorityClassName", "runtimeClassName", "volumes"}

func present(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case map[string]any:
		return len(x) > 0
	case []any:
		return len(x) > 0
	case string:
		return x != ""
	}
	return true
}

func containerLines(c any) []string {
	m, _ := c.(map[string]any)
	out := []string{"      - name: " + str(m["name"]), "        image: " + str(m["image"])}
	rest := map[string]any{}
	for k, v := range m {
		if !containerNoise[k] && present(v) {
			rest[k] = v
		}
	}
	if len(rest) > 0 {
		out = append(out, indented(rest, "        ")...)
	}
	return out
}

func containerBlock(head string, obj map[string]any) []string {
	list, _ := at(obj, "spec", "template", "spec", head).([]any)
	if len(list) == 0 {
		return nil
	}
	out := []string{"      " + head + ":"}
	for _, c := range list {
		out = append(out, containerLines(c)...)
	}
	return out
}

func podExtraLines(obj map[string]any) []string {
	out := []string{}
	for _, key := range podExtras {
		if v := at(obj, "spec", "template", "spec", key); present(v) {
			out = append(out, indented(map[string]any{key: v}, "      ")...)
		}
	}
	return out
}

func crewManifestLines(obj map[string]any) []string {
	out := []string{"apiVersion: apps/v1", "kind: Deployment", "metadata:", "  name: " + str(at(obj, "metadata", "name"))}
	out = append(out, labelBlock("labels", at(obj, "metadata", "labels"), "  ")...)
	out = append(out, "spec:", fmt.Sprintf("  replicas: %d", replicasOf(obj)), "  selector:")
	out = append(out, labelBlock("matchLabels", at(obj, "spec", "selector", "matchLabels"), "    ")...)
	out = append(out, "  template:", "    metadata:")
	out = append(out, labelBlock("labels", at(obj, "spec", "template", "metadata", "labels"), "      ")...)
	out = append(out, "    spec:")
	out = append(out, podExtraLines(obj)...)
	out = append(out, containerBlock("initContainers", obj)...)
	return append(out, containerBlock("containers", obj)...)
}

func replicasOf(obj map[string]any) int {
	switch n := at(obj, "spec", "replicas").(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

func firstContainer(obj map[string]any) map[string]any {
	list, _ := at(obj, "spec", "template", "spec", "containers").([]any)
	if len(list) == 0 {
		return nil
	}
	m, _ := list[0].(map[string]any)
	return m
}

func CrewFromYAML(raw string) (liveCrew, bool) {
	var obj map[string]any
	if err := yaml.Unmarshal([]byte(raw), &obj); err != nil || str(obj["kind"]) != "Deployment" {
		return liveCrew{}, false
	}
	c := firstContainer(obj)
	if c == nil || str(at(obj, "metadata", "name")) == "" {
		return liveCrew{}, false
	}
	return liveCrew{name: str(at(obj, "metadata", "name")), Replicas: replicasOf(obj), Image: str(c["image"]), resources: pledgeOf(c["resources"]),
		Text: strings.Join(crewManifestLines(obj), "\n")}, true
}

func (h *HUD) liveCrewOf(name string) (liveCrew, bool) {
	res := h.kubectl("get deployment " + name + " -o yaml")
	if !res.OK {
		return liveCrew{}, false
	}
	return CrewFromYAML(res.Output)
}

func (h *HUD) liveCrewText() (string, bool) {
	c, ok := h.liveCrewOf(liveCrewName)
	if !ok {
		return "", false
	}
	if ui.StoryNaming() {
		return lexicon.StoryManifest(c.Text), true
	}
	return c.Text, true
}

func (h *HUD) ScriptText() string { return h.Script.Editor.Text() }

func (h *HUD) Pinned() bool { return h.Pin.On() }

func pledgeOf(v any) any {
	if len(labelsOf(v)) == 0 {
		return nil
	}
	return v
}

func classOf(image string) string {
	if _, tag, ok := strings.Cut(image, ":"); ok {
		return tag
	}
	return image
}

func changeParts(live, next liveCrew) []string {
	parts := []string{}
	if next.Replicas < live.Replicas {
		parts = append(parts, fmt.Sprintf("%s %d to %d", live.name, live.Replicas, next.Replicas))
	}
	if next.Image != live.Image {
		parts = append(parts, fmt.Sprintf("class %s to %s", classOf(live.Image), classOf(next.Image)))
	}
	if live.resources != nil && !reflect.DeepEqual(live.resources, next.resources) {
		parts = append(parts, "food pledge changed")
	}
	return parts
}

func (h *HUD) applyWarning(text string) string {
	if ui.StoryNaming() {
		text = lexicon.ManifestFromStory(text)
	}
	next, ok := CrewFromYAML(text)
	if !ok {
		return ""
	}
	live, ok := h.liveCrewOf(next.name)
	if !ok {
		return ""
	}
	if next.Replicas >= live.Replicas && next.Image == live.Image {
		return ""
	}
	return strings.Join(changeParts(live, next), ", ") + ": apply?"
}
