package kubectl

import (
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const maxAnnotationLen = 140

var terminalEscaper = strings.NewReplacer("\x1b", "^[", "\r", "\\r")

func annotationLines(annotations map[string]string) []string {
	out := []string{}
	for _, key := range sortedKeys(annotations) {
		if key == corev1.LastAppliedConfigAnnotation {
			continue
		}
		for _, line := range annotationLine(key, strings.TrimSuffix(annotations[key], "\n")) {
			out = append(out, terminalEscaper.Replace(line))
		}
	}
	if len(out) == 0 {
		return []string{"<none>"}
	}
	return out
}

func annotationLine(key, value string) []string {
	if len(value)+len(key)+2 <= maxAnnotationLen && !strings.Contains(value, "\n") {
		return []string{key + ": " + value}
	}
	out := []string{key + ":"}
	for _, line := range strings.Split(value, "\n") {
		out = append(out, "  "+shorten(line, maxAnnotationLen-2))
	}
	return out
}

func labelLines(labels map[string]string) []string {
	keys := sortedKeys(labels)
	if len(keys) == 0 {
		return []string{"<none>"}
	}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+labels[key])
	}
	return out
}

func WriteAnnotations(w io.Writer, title string, annotations map[string]string) {
	fmt.Fprintf(w, "%s:\t%s\n", title, strings.Join(annotationLines(annotations), "\n\t"))
}

func shorten(s string, limit int) string {
	if len(s) > limit {
		return s[:limit] + "..."
	}
	return s
}

func fieldLines(key string, values []string) []field {
	out := make([]field, 0, len(values))
	for i, value := range values {
		label := key
		if i > 0 {
			label = ""
		}
		out = append(out, field{label, value})
	}
	return out
}

func (f field) label() string {
	if f.key == "" {
		return ""
	}
	return f.key + ":"
}

func (b *block) row(indent, width int, label, value string) {
	b.lines = append(b.lines, fmt.Sprintf("%s%-*s%s", strings.Repeat(" ", indent), width, label, value))
}

func (b *block) kvLines(indent, width int, key string, values []string) {
	for _, f := range fieldLines(key, values) {
		b.row(indent, width, f.label(), f.value)
	}
}
