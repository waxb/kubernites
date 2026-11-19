package kubectl

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func eventLastSeen(e *corev1.Event) metav1.Time {
	switch {
	case e.Series != nil:
		return metav1.NewTime(e.Series.LastObservedTime.Time)
	case !e.LastTimestamp.IsZero():
		return e.LastTimestamp
	}
	return metav1.NewTime(e.EventTime.Time)
}

func eventFirstSeen(e *corev1.Event) metav1.Time {
	if !e.EventTime.IsZero() {
		return metav1.NewTime(e.EventTime.Time)
	}
	return e.FirstTimestamp
}

func events(c Cluster, inv Invocation) (string, string, error) {
	if len(inv.Args) > 0 {
		return "", "events", fmt.Errorf("error: unknown command %q for \"kubectl events\"", inv.Args[0])
	}
	ns := inv.Namespace
	if inv.allNS {
		ns = ""
	}
	types := map[string]bool{}
	for _, t := range strings.Split(inv.Flags["types"], ",") {
		if t = strings.TrimSpace(t); t != "" {
			types[strings.ToLower(t)] = true
		}
	}
	forKind, forName, err := eventsForFlag(inv.Flags["for"])
	if err != nil {
		return "", "events", err
	}
	var objs []store.Object
	for _, o := range c.Store().List(store.KindEvent, ns) {
		e := o.(*corev1.Event)
		if len(types) > 0 && !types[strings.ToLower(e.Type)] {
			continue
		}
		if forName != "" && (!strings.EqualFold(e.InvolvedObject.Kind, forKind) || e.InvolvedObject.Name != forName) {
			continue
		}
		objs = append(objs, o)
	}
	if len(objs) == 0 {
		if inv.allNS {
			return "No events found.", "events", nil
		}
		return fmt.Sprintf("No events found in %s namespace.", inv.Namespace), "events", nil
	}
	sortEvents(objs)
	if out := inv.Flags["output"]; out != "" {
		r, _ := ResolveType("events")
		text, err := render(c, r, objs, getOptions{output: out, allNS: inv.allNS}, false)
		return text, "events", err
	}
	return eventsTable(c, objs, inv.allNS), "events", nil
}

func eventsTable(c Cluster, objs []store.Object, allNS bool) string {
	headers := []string{"LAST SEEN", "TYPE", "REASON", "OBJECT", "MESSAGE"}
	if allNS {
		headers = append([]string{"NAMESPACE"}, headers...)
	}
	rows := make([][]string, 0, len(objs))
	for _, o := range objs {
		e := o.(*corev1.Event)
		row := []string{eventInterval(c, e), e.Type, e.Reason, e.InvolvedObject.Kind + "/" + e.InvolvedObject.Name, strings.TrimSpace(e.Message)}
		if allNS {
			row = append([]string{e.Namespace}, row...)
		}
		rows = append(rows, row)
	}
	return renderTable(headers, rows)
}

func eventInterval(c Cluster, e *corev1.Event) string {
	first := age(c, eventFirstSeen(e))
	switch {
	case e.Series != nil:
		return fmt.Sprintf("%s (x%d over %s)", age(c, eventLastSeen(e)), e.Series.Count, first)
	case e.Count > 1:
		return fmt.Sprintf("%s (x%d over %s)", age(c, e.LastTimestamp), e.Count, first)
	}
	return first
}

func eventsForFlag(raw string) (string, string, error) {
	if raw == "" {
		return "", "", nil
	}
	kind, name, ok := strings.Cut(raw, "/")
	if !ok || kind == "" || name == "" {
		return "", "", fmt.Errorf("error: --for must be in resource/name form")
	}
	r, err := ResolveType(kind)
	if err != nil {
		return "", "", err
	}
	return r.Kind, name, nil
}
