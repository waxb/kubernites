package kubectl

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/waxb/kubernites/pkg/sim/rolloututil"
	"github.com/waxb/kubernites/pkg/sim/store"
)

type Revisioned struct {
	Resource ResourceType
	Obj      store.Object
	selector *metav1.LabelSelector
	Template *corev1.PodTemplateSpec
}

type RevisionedVerb func(Cluster, Invocation, Revisioned) (string, error)

var revisionedVerbs = map[string]RevisionedVerb{
	"status": revisionedStatus, "history": revisionedHistory,
}

func rolloutRevisioned(c Cluster, inv Invocation, r ResourceType, names []string) (string, string, error) {
	verb, ok := revisionedVerbs[inv.Args[0]]
	if !ok {
		return "", r.Singular, unsupportedRollout(inv.Args[0], r, names[0])
	}
	out := []string{}
	for _, name := range names {
		v, err := loadRevisioned(c, r, inv.Namespace, name)
		if err != nil {
			return strings.Join(out, "\n"), r.Singular, err
		}
		line, err := verb(c, inv, v)
		if err != nil {
			return strings.Join(out, "\n"), r.Singular, err
		}
		out = append(out, strings.TrimRight(line, "\n"))
	}
	return strings.Join(out, "\n"), r.Singular, nil
}

func loadRevisioned(c Cluster, r ResourceType, namespace, name string) (Revisioned, error) {
	obj, ok := c.Store().Get(store.Key{Kind: r.Kind, Namespace: namespace, Name: name})
	if !ok {
		return Revisioned{}, NotFound(r, name)
	}
	switch o := obj.(type) {
	case *appsv1.DaemonSet:
		return Revisioned{Resource: r, Obj: o, selector: o.Spec.Selector, Template: &o.Spec.Template}, nil
	case *appsv1.StatefulSet:
		return Revisioned{Resource: r, Obj: o, selector: o.Spec.Selector, Template: &o.Spec.Template}, nil
	}
	return Revisioned{}, NotFound(r, name)
}

func (v Revisioned) Ref() string {
	return v.Resource.Qualified() + "/" + v.Obj.GetName()
}

func (v Revisioned) History(c Cluster) []*appsv1.ControllerRevision {
	selector, err := metav1.LabelSelectorAsSelector(v.selector)
	if err != nil {
		return nil
	}
	out := []*appsv1.ControllerRevision{}
	for _, h := range store.List[*appsv1.ControllerRevision](c.Store(), store.KindControllerRevision, v.Obj.GetNamespace()) {
		if selector.Matches(labels.Set(h.Labels)) && metav1.IsControlledBy(h, v.Obj) {
			out = append(out, h)
		}
	}
	rolloututil.SortRevisions(out)
	return out
}

func revisionedStatus(_ Cluster, _ Invocation, v Revisioned) (string, error) {
	if ds, ok := v.Obj.(*appsv1.DaemonSet); ok {
		return daemonSetRolloutStatus(ds)
	}
	return statefulSetRolloutStatus(v.Obj.(*appsv1.StatefulSet))
}

func revisionedHistory(c Cluster, inv Invocation, v Revisioned) (string, error) {
	revision, err := RevisionFlag(inv, "revision")
	if err != nil {
		return "", err
	}
	byNumber := map[int64]*appsv1.ControllerRevision{}
	for _, h := range v.History(c) {
		byNumber[h.Revision] = h
	}
	if len(byNumber) == 0 {
		return v.Ref() + " \nNo rollout history found.", nil
	}
	if revision <= 0 {
		return v.Ref() + " \n" + revisionTable(byNumber), nil
	}
	h, ok := byNumber[revision]
	if !ok {
		return "", errors.New("error: unable to find the specified revision")
	}
	tmpl, err := rolloututil.ApplyTemplate(h)
	if err != nil {
		return "", fmt.Errorf("error: unable to parse history %s", h.Name)
	}
	t := newTabbed()
	describePodTemplate(t, &tmpl)
	return fmt.Sprintf("%s with revision #%d\n%s", v.Ref(), revision, t.String()), nil
}

func revisionTable(byNumber map[int64]*appsv1.ControllerRevision) string {
	numbers := make([]int64, 0, len(byNumber))
	for n := range byNumber {
		numbers = append(numbers, n)
	}
	sort.Slice(numbers, func(i, j int) bool { return numbers[i] < numbers[j] })
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 8, 2, ' ', 0)
	fmt.Fprintf(w, "REVISION\tCHANGE-CAUSE\n")
	for _, n := range numbers {
		cause := byNumber[n].Annotations[ChangeCauseAnnotation]
		if cause == "" {
			cause = "<none>"
		}
		fmt.Fprintf(w, "%d\t%s\n", n, cause)
	}
	_ = w.Flush()
	return buf.String()
}
