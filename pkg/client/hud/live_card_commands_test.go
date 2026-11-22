package hud_test

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
)

type liveKind struct {
	namespaced bool
	objects    map[string]bool
}

func (k liveKind) holds(ns, name string) bool {
	if !k.namespaced {
		ns = ""
	}
	return k.objects[ns+"/"+name]
}

func liveKinds(s live.Snapshot) map[string]*liveKind {
	out := map[string]*liveKind{}
	add := func(names []string, namespaced bool, ns, name string) {
		var k *liveKind
		for _, n := range names {
			if out[n] != nil {
				k = out[n]
			}
		}
		if k == nil {
			k = &liveKind{namespaced: namespaced, objects: map[string]bool{}}
		}
		for _, n := range names {
			out[n] = k
		}
		if name != "" {
			if !namespaced {
				ns = ""
			}
			k.objects[ns+"/"+name] = true
		}
	}
	builtin := func(names string, namespaced bool) []string {
		add(strings.Fields(names), namespaced, "", "")
		return strings.Fields(names)
	}
	pods, svcs := builtin("pod pods po", true), builtin("service services svc", true)
	nodes, nss := builtin("node nodes no", false), builtin("namespace namespaces ns", false)
	pvcs, pvs := builtin("persistentvolumeclaim persistentvolumeclaims pvc", true), builtin("persistentvolume persistentvolumes pv", false)
	deps, rss := builtin("deployment deployments deploy", true), builtin("replicaset replicasets rs", true)
	dss, stss := builtin("daemonset daemonsets ds", true), builtin("statefulset statefulsets sts", true)
	builtin("endpointslice endpointslices", true)
	builtin("event events ev", true)
	ings, gws := builtin("ingress ingresses ing", true), builtin("gateway gateways gtw", true)
	hpas, jobs := builtin("horizontalpodautoscaler horizontalpodautoscalers hpa", true), builtin("job jobs", true)
	cjs, scs := builtin("cronjob cronjobs cj", true), builtin("storageclass storageclasses sc", false)
	pdbs, pcs := builtin("poddisruptionbudget poddisruptionbudgets pdb", true), builtin("priorityclass priorityclasses pc", false)
	for _, o := range s.Ingresses {
		add(ings, true, o.Namespace, o.Name)
	}
	for _, o := range s.Gateways {
		add(gws, true, o.Namespace, o.Name)
	}
	for _, o := range s.HPAs {
		add(hpas, true, o.Namespace, o.Name)
	}
	for _, o := range s.Jobs {
		add(jobs, true, o.Namespace, o.Name)
	}
	for _, o := range s.CronJobs {
		add(cjs, true, o.Namespace, o.Name)
	}
	for _, o := range s.StorageClasses {
		add(scs, false, "", o.Name)
	}
	for _, o := range s.PDBs {
		add(pdbs, true, o.Namespace, o.Name)
	}
	for _, o := range s.PriorityClasses {
		add(pcs, false, "", o.Name)
	}
	for _, p := range s.Pods {
		add(pods, true, p.Namespace, p.Name)
	}
	for _, o := range s.Services {
		add(svcs, true, o.Namespace, o.Name)
	}
	for _, o := range s.Nodes {
		add(nodes, false, "", o.Name)
	}
	for _, o := range s.Namespaces {
		add(nss, false, "", o.Name)
	}
	for _, o := range s.PVCs {
		add(pvcs, true, o.Namespace, o.Name)
	}
	for _, o := range s.PVs {
		add(pvs, false, "", o.Name)
	}
	for _, o := range s.Deployments {
		add(deps, true, o.Namespace, o.Name)
	}
	for _, o := range s.ReplicaSets {
		add(rss, true, o.Namespace, o.Name)
	}
	for _, o := range s.DaemonSets {
		add(dss, true, o.Namespace, o.Name)
	}
	for _, o := range s.StatefulSets {
		add(stss, true, o.Namespace, o.Name)
	}
	for _, c := range s.CRDs {
		names := []string{c.Plural, strings.ToLower(c.Kind), c.Plural + "." + c.Group}
		add(names, c.Scope == "Namespaced", "", "")
		for _, r := range s.Custom {
			if r.Group == c.Group && r.Kind == c.Kind {
				add(names, c.Scope == "Namespaced", r.Namespace, r.Name)
			}
		}
	}
	return out
}

func checkCommand(kinds map[string]*liveKind, cmd string) string {
	f := strings.Fields(cmd)
	if len(f) == 0 {
		return "empty"
	}
	switch f[0] {
	case "version", "cluster-info", "api-resources", "top":
		return ""
	case "describe", "get", "explain":
	default:
		return ""
	}
	if len(f) < 2 {
		return "no type"
	}
	k := kinds[f[1]]
	if k == nil {
		return "kubectl knows no type " + f[1] + " on this cluster"
	}
	if f[0] == "explain" {
		return ""
	}
	name, ns, named := "", "", false
	for i := 2; i < len(f); i++ {
		switch {
		case f[i] == "-n" && i+1 < len(f):
			ns = f[i+1]
			i++
		case (f[i] == "-o" || f[i] == "-l") && i+1 < len(f):
			i++
		case strings.HasPrefix(f[i], "-"):
		case !named:
			name, named = f[i], true
		}
	}
	if !named {
		return ""
	}
	if k.namespaced && ns == "" {
		ns = "default"
	}
	if !k.holds(ns, name) {
		return "no " + f[1] + " " + name + " in namespace " + ns
	}
	return ""
}

func TestEveryLiveCardAndTabCommandFindsWhatTheCardShows(t *testing.T) {
	snap := livetest.Managed()
	w, err := live.Build(snap)
	if err != nil {
		t.Fatal(err)
	}
	kinds := liveKinds(snap)
	checked := 0
	for _, st := range w.Site.Structures {
		sel := hud.Selection{Kind: "structure", Structure: st}
		origin := st.Kind + " (" + st.Object.Kind + " " + st.Object.Namespace + "/" + st.Object.Name + ")"
		var cmds []string
		for _, tab := range []string{hud.DetailDescribe, hud.DetailYAML, hud.DetailExplain} {
			for _, c := range hud.DetailCommands(sel, tab) {
				cmds = append(cmds, c)
			}
		}
		for _, a := range hud.StructureActions(st, hud.Frame{}) {
			cmds = append(cmds, a.Command)
		}
		for _, c := range cmds {
			if strings.TrimSpace(c) == "" {
				continue
			}
			checked++
			if why := checkCommand(kinds, c); why != "" {
				t.Errorf("%s: kubectl %s: %s", origin, c, why)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no command checked")
	}
}
