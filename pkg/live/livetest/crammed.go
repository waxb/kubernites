package livetest

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/live"
)

var crammedNamespaces = []string{"default", "kube-system", "kube-public", "kube-node-lease", "gke-managed-system", "gke-managed-volumepopulator", "gke-managed-networking-dra-driver",
	"ledgerhall", "lookouts", "strongroom", "steward", "sealwrights", "gatehouse", "harbourside", "keybearers"}

var crammedDeployments = []workload{
	{ns: "kube-system", name: "dns-herald", replicas: 2}, {ns: "kube-system", name: "tunnel-agent", replicas: 2}, {ns: "kube-system", name: "augur", replicas: 1},
	{ns: "kube-system", name: "event-crier", replicas: 1}, {ns: "kube-system", name: "default-backend", replicas: 1}, {ns: "kube-system", name: "trouble-watch", replicas: 1},
	{ns: "ledgerhall", name: "ledger-proxy", replicas: 1},
	{ns: "lookouts", name: "lookout", replicas: 2}, {ns: "lookouts", name: "dashboards", replicas: 1}, {ns: "lookouts", name: "alert-bell", replicas: 1},
	{ns: "sealwrights", name: "sealwright", replicas: 1}, {ns: "sealwrights", name: "sealwright-cainjector", replicas: 1}, {ns: "sealwrights", name: "sealwright-webhook", replicas: 1},
	{ns: "gatehouse", name: "gate-controller", replicas: 1},
	{ns: "harbourside", name: "storefront", replicas: 2}, {ns: "harbourside", name: "checkout", replicas: 1}, {ns: "harbourside", name: "catalog", replicas: 1},
	{ns: "harbourside", name: "mailer", replicas: 1}, {ns: "harbourside", name: "accounts", replicas: 1}, {ns: "harbourside", name: "pricing", replicas: 1},
	{ns: "keybearers", name: "keybearer", replicas: 1}, {ns: "keybearers", name: "keybearer-webhook", replicas: 1}, {ns: "keybearers", name: "keybearer-cert-controller", replicas: 1},
}

var crammedDaemonSets = []workload{
	{ns: "kube-system", name: "path-keeper"}, {ns: "kube-system", name: "log-courier"}, {ns: "kube-system", name: "metadata-warden"},
	{ns: "kube-system", name: "disk-node"}, {ns: "kube-system", name: "mask-agent"}, {ns: "kube-system", name: "mesh-agent"}, {ns: "kube-system", name: "node-lookout"},
}

var crammedStatefulSets = []workload{
	{ns: "ledgerhall", name: "orders-db", replicas: 1, claims: []string{"data"}, class: "sc-fast"},
	{ns: "ledgerhall", name: "kv", replicas: 1, claims: []string{"data"}, class: "sc-fast"},
	{ns: "ledgerhall", name: "search", replicas: 1, claims: []string{"data"}, class: "sc-standard"},
	{ns: "ledgerhall", name: "queue", replicas: 1, claims: []string{"data"}, class: "sc-standard"},
	{ns: "strongroom", name: "strongbox", replicas: 1, claims: []string{"data", "audit"}, class: "sc-balanced"},
	{ns: "steward", name: "steward", replicas: 1, claims: []string{"data"}, class: "sc-standard"},
	{ns: "harbourside", name: "sessions", replicas: 1, claims: []string{"data", "wal"}, class: "sc-fast"},
	{ns: "harbourside", name: "cache", replicas: 1, claims: []string{"data", "snapshots"}, class: "sc-standard"},
	{ns: "harbourside", name: "inbox", replicas: 1, claims: []string{"data", "journal"}, class: "sc-standard"},
	{ns: "harbourside", name: "reports", replicas: 1, claims: []string{"data"}, class: "sc-balanced"},
}

var crammedServices = []struct {
	ns, name, app string
	kind          corev1.ServiceType
	headless      bool
}{
	{ns: "default", name: "kubernetes"},
	{ns: "kube-system", name: "dns-herald", app: "dns-herald"}, {ns: "kube-system", name: "augur", app: "augur"},
	{ns: "kube-system", name: "default-backend", app: "default-backend", kind: corev1.ServiceTypeNodePort}, {ns: "kube-system", name: "trouble-watch", app: "trouble-watch"},
	{ns: "ledgerhall", name: "orders-db", app: "orders-db", headless: true}, {ns: "ledgerhall", name: "kv", app: "kv", headless: true},
	{ns: "ledgerhall", name: "search", app: "search", headless: true}, {ns: "ledgerhall", name: "queue", app: "queue", headless: true},
	{ns: "ledgerhall", name: "orders-db-read", app: "orders-db"}, {ns: "ledgerhall", name: "kv-read", app: "kv"}, {ns: "ledgerhall", name: "search-api", app: "search"},
	{ns: "ledgerhall", name: "queue-api", app: "queue"}, {ns: "ledgerhall", name: "ledger-proxy", app: "ledger-proxy"},
	{ns: "strongroom", name: "strongbox", app: "strongbox", headless: true}, {ns: "strongroom", name: "strongbox-api", app: "strongbox"},
	{ns: "steward", name: "steward", app: "steward"},
	{ns: "sealwrights", name: "sealwright", app: "sealwright"}, {ns: "sealwrights", name: "sealwright-webhook", app: "sealwright-webhook"},
	{ns: "gatehouse", name: "gate-controller", app: "gate-controller", kind: corev1.ServiceTypeLoadBalancer}, {ns: "gatehouse", name: "gate-controller-admission", app: "gate-controller"},
	{ns: "harbourside", name: "storefront", app: "storefront"}, {ns: "harbourside", name: "checkout", app: "checkout"}, {ns: "harbourside", name: "catalog", app: "catalog"},
	{ns: "harbourside", name: "mailer", app: "mailer"}, {ns: "harbourside", name: "accounts", app: "accounts"}, {ns: "harbourside", name: "pricing", app: "pricing"},
	{ns: "harbourside", name: "sessions", app: "sessions"}, {ns: "harbourside", name: "cache", app: "cache"}, {ns: "harbourside", name: "inbox", app: "inbox"},
	{ns: "harbourside", name: "reports", app: "reports"}, {ns: "harbourside", name: "storefront-admin", app: "storefront"},
	{ns: "keybearers", name: "keybearer-webhook", app: "keybearer-webhook"},
}

var crammedCustom = []struct {
	kind, ns string
	n        int
}{
	{"Certificate", "ledgerhall", 2}, {"CertificateRequest", "ledgerhall", 2}, {"Issuer", "ledgerhall", 1}, {"ExternalSecret", "ledgerhall", 3},
	{"Certificate", "strongroom", 1}, {"CertificateRequest", "strongroom", 1}, {"Issuer", "strongroom", 1},
	{"Certificate", "steward", 1}, {"CertificateRequest", "steward", 1}, {"Issuer", "steward", 1},
	{"ExternalSecret", "lookouts", 2}, {"PodMonitoring", "kube-system", 2},
	{"Certificate", "harbourside", 4}, {"CertificateRequest", "harbourside", 4}, {"Issuer", "harbourside", 1}, {"ExternalSecret", "harbourside", 5},
	{"ClusterIssuer", "", 2}, {"ClusterSecretStore", "", 1}, {"CiliumIdentity", "", 30}, {"CiliumNode", "", 3}, {"UpdateInfo", "", 1},
}

func Crammed() live.Snapshot {
	m := &managed{born: time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)}
	m.s = live.Snapshot{TakenAt: time.Date(2026, 10, 30, 15, 30, 0, 0, time.UTC), Version: ManagedVersion,
		Context:  live.ContextInfo{Name: "gke-crammed", Cluster: "gke-crammed", Server: ManagedServer, User: "gke-crammed-user"},
		Identity: live.Identity{Username: "player@example.com", Groups: []string{"system:authenticated"}}}
	for _, ns := range crammedNamespaces {
		m.s.Namespaces = append(m.s.Namespaces, corev1.Namespace{ObjectMeta: m.meta("", ns)})
	}
	for i := 0; i < 3; i++ {
		m.node(fmt.Sprintf("gke-crammed-%s-4e2d8b7c-%c%c3n", managedPool, 'c'+i, 'r'+i))
	}
	for _, d := range crammedDeployments {
		m.deployment(d)
	}
	for _, d := range crammedDaemonSets {
		m.daemonSet(d, nil)
	}
	for _, st := range crammedStatefulSets {
		m.statefulSet(st)
	}
	m.storageClasses()
	for _, svc := range crammedServices {
		m.service(svc.ns, svc.name, svc.app, svc.kind, svc.headless)
	}
	m.customFrom(crammedCustom)
	return m.s
}

func (m *managed) customFrom(list []struct {
	kind, ns string
	n        int
}) {
	for _, c := range managedCRDs {
		c.Name, c.UID = c.Plural+"."+c.Group, m.uid()
		m.s.CRDs = append(m.s.CRDs, c)
	}
	group := map[string]string{}
	for _, c := range managedCRDs {
		group[c.Kind] = c.Group
	}
	for _, cr := range list {
		for i := 0; i < cr.n; i++ {
			m.s.Custom = append(m.s.Custom, live.CustomResource{Group: group[cr.kind], Kind: cr.kind, Namespace: cr.ns, Name: fmt.Sprintf("%s-%d", toLower(cr.kind), i+1), UID: m.uid()})
		}
	}
	for _, p := range m.s.Pods {
		if p.Namespace != "kube-system" {
			m.s.Custom = append(m.s.Custom, live.CustomResource{Group: "cilium.io", Kind: "CiliumEndpoint", Namespace: p.Namespace, Name: p.Name, UID: m.uid()})
		}
	}
}
