package kubectl

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
)

func renderTable(headers []string, rows [][]string) string {
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 6, 4, 3, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	_ = w.Flush()
	return b.String()
}

func age(c Cluster, t metav1.Time) string {
	if t.IsZero() {
		return "<unknown>"
	}
	d := c.Clock().Since(t)
	if d < 0 {
		d = 0
	}
	return duration.HumanDuration(d.Truncate(time.Second))
}

func labelString(labels map[string]string) string {
	if len(labels) == 0 {
		return "<none>"
	}
	keys := sortedKeys(labels)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+labels[k])
	}
	return strings.Join(parts, ",")
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

func formatMilli(v float64) string {
	return fmt.Sprintf("%dm", int64(v+0.5))
}

func formatMi(bytes float64) string {
	return fmt.Sprintf("%dMi", int64(bytes/(1024*1024)+0.5))
}
