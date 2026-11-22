package write

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

const (
	DrainTries = 120
	DrainEvery = 5 * time.Second
	mirrorKey  = "kubernetes.io/config.mirror"
)

type Drain struct {
	Node    string
	Cordon  *Order
	Evict   []Order
	Skipped []string
	Orphans []string
	Classes []Class
	Command string
}

func PlanDrain(n *corev1.Node, pods []corev1.Pod) (Drain, error) {
	d := Drain{Node: n.Name, Command: DrainCommand(n.Name)}
	if !n.Spec.Unschedulable {
		o, err := Cordon(n, true)
		if err != nil {
			return Drain{}, err
		}
		d.Cordon = &o
	}
	var on []corev1.Pod
	for _, p := range pods {
		if p.Spec.NodeName == n.Name {
			on = append(on, p)
		}
	}
	slices.SortFunc(on, func(a, b corev1.Pod) int {
		return strings.Compare(a.Namespace+"/"+a.Name, b.Namespace+"/"+b.Name)
	})
	for i := range on {
		p := &on[i]
		who := p.Namespace + "/" + p.Name
		if _, ok := p.Annotations[mirrorKey]; ok {
			d.Skipped = append(d.Skipped, who+": a mirror pod, which only its node's kubelet can remove")
			continue
		}
		if owner := controllerOf(p); owner == "DaemonSet" {
			d.Skipped = append(d.Skipped, who+": a DaemonSet pod")
			continue
		} else if owner == "" {
			d.Orphans = append(d.Orphans, who)
		}
		o, err := Evict(p)
		if err != nil {
			return Drain{}, err
		}
		d.Evict = append(d.Evict, o)
	}
	if len(d.Orphans) > 0 {
		d.Classes = []Class{Destroy}
	}
	return d, nil
}

func controllerOf(p *corev1.Pod) string {
	for _, ref := range p.OwnerReferences {
		if ref.Controller != nil && *ref.Controller {
			return ref.Kind
		}
	}
	return ""
}

type DrainReport struct {
	Hut     string
	Cordon  string
	Code    int
	Done    bool
	Gone    bool
	Blocked string
	Failed  string
	Tries   int
}

func blockingText(a Answer) string {
	if len(a.Causes) > 0 {
		return strings.Join(a.Causes, "; ")
	}
	return a.Text()
}

func (g *Gate) IssueDrain(d Drain) {
	if d.Cordon != nil {
		g.Issue(d.Cordon.request())
	}
	for _, o := range d.Evict {
		g.issue(o.request(), DrainTries)
	}
}

func (g *Gate) RunDrain(ctx context.Context, d Drain, every time.Duration, report func(DrainReport)) error {
	defer func() {
		if d.Cordon != nil {
			g.Revoke(d.Cordon.Ticket())
		}
		for _, o := range d.Evict {
			g.Revoke(o.Ticket())
		}
	}()
	if d.Cordon != nil {
		ans := g.Send(ctx, *d.Cordon)
		r := DrainReport{Cordon: d.Node, Code: ans.Code, Tries: 1, Done: ans.OK()}
		if !ans.OK() {
			r.Failed = ans.Text()
		}
		report(r)
		if !ans.OK() {
			return fmt.Errorf("cordon %s: %s", d.Node, ans.Text())
		}
	}
	for _, o := range d.Evict {
		hut := o.Target.Namespace + "/" + o.Target.Name
		for try := 1; ; try++ {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			ans := g.Send(ctx, o)
			r := DrainReport{Hut: hut, Tries: try}
			switch {
			case ans.OK():
				r.Done = true
			case ans.Err == nil && ans.Code == http.StatusNotFound:
				r.Done, r.Gone = true, true
			case ans.Blocked():
				r.Blocked = blockingText(ans)
			default:
				r.Failed = ans.Text()
			}
			if r.Blocked != "" && try >= DrainTries {
				r.Failed = fmt.Sprintf("still blocked after %d tries: %s", try, r.Blocked)
				r.Blocked = ""
			}
			report(r)
			if r.Blocked == "" {
				g.Revoke(o.Ticket())
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(every):
			}
		}
	}
	return nil
}
