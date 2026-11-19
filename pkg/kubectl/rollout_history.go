package kubectl

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"text/tabwriter"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/sim/rolloututil"
)

func RevisionFlag(inv Invocation, name string) (int64, error) {
	raw, ok := inv.Flags[name]
	if !ok {
		return 0, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error: invalid argument %q for \"--%s\" flag: %v", raw, name, err)
	}
	return v, nil
}

func RevisionsOf(c Cluster, d *appsv1.Deployment) map[int64]*appsv1.ReplicaSet {
	out := map[int64]*appsv1.ReplicaSet{}
	for _, rs := range OwnedSets(c, d) {
		v, err := strconv.ParseInt(rs.Annotations[rolloututil.RevisionAnnotation], 10, 64)
		if err == nil {
			out[v] = rs
		}
	}
	return out
}

func rolloutHistory(c Cluster, inv Invocation, d *appsv1.Deployment) (string, error) {
	revision, err := RevisionFlag(inv, "revision")
	if err != nil {
		return "", err
	}
	history := RevisionsOf(c, d)
	if len(history) == 0 {
		return "deployment.apps/" + d.Name + " \nNo rollout history found.", nil
	}
	if revision > 0 {
		rs, ok := history[revision]
		if !ok {
			return "", errors.New("error: unable to find the specified revision")
		}
		t := newTabbed()
		describePodTemplate(t, withChangeCause(rs))
		return fmt.Sprintf("deployment.apps/%s with revision #%d\n%s", d.Name, revision, t.String()), nil
	}
	return "deployment.apps/" + d.Name + " \n" + historyTable(history), nil
}

func withChangeCause(rs *appsv1.ReplicaSet) *corev1.PodTemplateSpec {
	tmpl := rs.Spec.Template.DeepCopy()
	if cause := rs.Annotations[ChangeCauseAnnotation]; cause != "" {
		annotations := map[string]string{ChangeCauseAnnotation: cause}
		for k, v := range tmpl.Annotations {
			annotations[k] = v
		}
		tmpl.Annotations = annotations
	}
	return tmpl
}

func historyTable(history map[int64]*appsv1.ReplicaSet) string {
	revisions := make([]int64, 0, len(history))
	for r := range history {
		revisions = append(revisions, r)
	}
	sort.Slice(revisions, func(i, j int) bool { return revisions[i] < revisions[j] })
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 8, 2, ' ', 0)
	fmt.Fprintf(w, "REVISION\tCHANGE-CAUSE\n")
	for _, r := range revisions {
		cause := history[r].Annotations[ChangeCauseAnnotation]
		if cause == "" {
			cause = "<none>"
		}
		fmt.Fprintf(w, "%d\t%s\n", r, cause)
	}
	_ = w.Flush()
	return buf.String()
}
