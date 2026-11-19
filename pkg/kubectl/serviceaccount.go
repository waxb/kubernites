package kubectl

import (
	"bytes"
	"fmt"
	"text/tabwriter"

	corev1 "k8s.io/api/core/v1"
)

func describeServiceAccount(c Cluster, sa *corev1.ServiceAccount) string {
	buf := &bytes.Buffer{}
	w := tabwriter.NewWriter(buf, 0, 8, 2, ' ', 0)
	fmt.Fprintf(w, "Name:\t%s\n", sa.Name)
	fmt.Fprintf(w, "Namespace:\t%s\n", sa.Namespace)
	for _, f := range fieldLines("Labels", labelLines(sa.Labels)) {
		fmt.Fprintf(w, "%s\t%s\n", f.label(), f.value)
	}
	for _, f := range fieldLines("Annotations", annotationLines(sa.Annotations)) {
		fmt.Fprintf(w, "%s\t%s\n", f.label(), f.value)
	}
	pulls := []string{}
	for _, s := range sa.ImagePullSecrets {
		pulls = append(pulls, s.Name)
	}
	if len(pulls) == 0 {
		pulls = []string{"<none>"}
	}
	for _, f := range fieldLines("Image pull secrets", pulls) {
		fmt.Fprintf(w, "%s\t%s\n", f.label(), f.value)
	}
	events := eventsFor(c, sa.UID)
	if len(events) == 0 {
		fmt.Fprintf(w, "Events:\t<none>\n")
	}
	_ = w.Flush()
	if len(events) > 0 {
		return buf.String() + eventsSection(c, sa.UID)
	}
	return buf.String()
}
