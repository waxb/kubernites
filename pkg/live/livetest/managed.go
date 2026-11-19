package livetest

import (
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/waxb/kubernites/pkg/live"
)

const (
	ManagedServer  = "https://203.0.113.10"
	ManagedVersion = "v1.37.0-gke.1100000"
	managedPool    = "pool-alder"
	managedZone    = "north-9a"
	managedType    = "e2-standard-2"
	nodePoolLabel  = "cloud.google.com/gke-nodepool"
	ingressClass   = "nginx"
)

type managed struct {
	s     live.Snapshot
	born  time.Time
	seq   int
	nodes []string
	ips   int
}

type workload struct {
	ns, name string
	replicas int
	claims   []string
	class    string
}

var managedNamespaces = []string{"default", "kube-system", "kube-public", "kube-node-lease", "gatehouse", "sealwrights", "keybearers", "market", "watchtower"}

var systemDeployments = []workload{
	{ns: "kube-system", name: "dns-herald", replicas: 2}, {ns: "kube-system", name: "tunnel-agent", replicas: 2},
	{ns: "kube-system", name: "dns-herald-autoscaler", replicas: 1}, {ns: "kube-system", name: "tunnel-agent-autoscaler", replicas: 1},
	{ns: "kube-system", name: "augur", replicas: 1}, {ns: "kube-system", name: "event-crier", replicas: 1},
	{ns: "kube-system", name: "default-backend", replicas: 1}, {ns: "kube-system", name: "seal-relay", replicas: 1},
	{ns: "kube-system", name: "trouble-watch", replicas: 1},
}

var appDeployments = []workload{
	{ns: "gatehouse", name: "gate-controller", replicas: 1},
	{ns: "sealwrights", name: "sealwright", replicas: 1}, {ns: "sealwrights", name: "sealwright-cainjector", replicas: 1}, {ns: "sealwrights", name: "sealwright-webhook", replicas: 1},
	{ns: "keybearers", name: "keybearer", replicas: 1}, {ns: "keybearers", name: "keybearer-webhook", replicas: 1}, {ns: "keybearers", name: "keybearer-cert-controller", replicas: 1},
	{ns: "watchtower", name: "lookout", replicas: 1}, {ns: "watchtower", name: "dashboards", replicas: 1}, {ns: "watchtower", name: "alert-bell", replicas: 1},
	{ns: "market", name: "storefront", replicas: 1}, {ns: "market", name: "checkout", replicas: 1}, {ns: "market", name: "catalog", replicas: 1},
	{ns: "market", name: "pricing", replicas: 1}, {ns: "market", name: "mailer", replicas: 1}, {ns: "market", name: "accounts", replicas: 1},
	{ns: "market", name: "reviews", replicas: 1}, {ns: "market", name: "admin-portal", replicas: 1},
}

var managedStatefulSets = []workload{
	{ns: "watchtower", name: "metrics-vault", replicas: 1, claims: []string{"data", "wal"}, class: "sc-balanced"},
	{ns: "watchtower", name: "log-vault", replicas: 1, claims: []string{"data", "index"}, class: "sc-balanced"},
	{ns: "market", name: "orders-db", replicas: 1, claims: []string{"data", "wal"}, class: "sc-fast"},
	{ns: "market", name: "search", replicas: 1, claims: []string{"data", "snapshots"}, class: "sc-standard"},
	{ns: "market", name: "queue", replicas: 1, claims: []string{"data", "journal"}, class: "sc-standard"},
	{ns: "market", name: "sessions", replicas: 1, claims: []string{"data"}, class: "sc-standard"},
	{ns: "market", name: "kv", replicas: 1, claims: []string{"data", "backup"}, class: "sc-fast"},
	{ns: "market", name: "dispatcher", replicas: 1},
	{ns: "market", name: "archive", replicas: 0, claims: []string{"data"}, class: "sc-legacy"},
}

var runningDaemonSets = []workload{
	{ns: "kube-system", name: "path-keeper"}, {ns: "kube-system", name: "log-courier"}, {ns: "kube-system", name: "metadata-warden"},
	{ns: "kube-system", name: "disk-node"}, {ns: "kube-system", name: "mask-agent"}, {ns: "kube-system", name: "mesh-agent"},
	{ns: "watchtower", name: "node-lookout"},
}

var idleSelectors = []map[string]string{
	{"kubernetes.io/os": "windows"}, {"kubernetes.io/arch": "arm64"}, {"cloud.example/accelerator": "gpu"}, {"cloud.example/tpu": "true"},
	{"cloud.example/sandbox": "true"}, {"cloud.example/confidential": "true"}, {"cloud.example/local-ssd": "true"}, {"cloud.example/spot-pool": "true"},
}

var managedServices = []struct {
	ns, name, app string
	kind          corev1.ServiceType
	headless      bool
}{
	{ns: "default", name: "kubernetes"},
	{ns: "kube-system", name: "dns-herald", app: "dns-herald"}, {ns: "kube-system", name: "augur", app: "augur"},
	{ns: "kube-system", name: "default-backend", app: "default-backend"}, {ns: "kube-system", name: "seal-relay", app: "seal-relay"},
	{ns: "kube-system", name: "trouble-watch", app: "trouble-watch"}, {ns: "kube-system", name: "event-crier", app: "event-crier"},
	{ns: "gatehouse", name: "gate-controller", app: "gate-controller", kind: corev1.ServiceTypeLoadBalancer},
	{ns: "gatehouse", name: "gate-controller-admission", app: "gate-controller"},
	{ns: "sealwrights", name: "sealwright", app: "sealwright"}, {ns: "sealwrights", name: "sealwright-webhook", app: "sealwright-webhook"},
	{ns: "keybearers", name: "keybearer-webhook", app: "keybearer-webhook"},
	{ns: "watchtower", name: "lookout", app: "lookout"}, {ns: "watchtower", name: "dashboards", app: "dashboards"},
	{ns: "watchtower", name: "alert-bell", app: "alert-bell"}, {ns: "watchtower", name: "metrics-vault", app: "metrics-vault", headless: true},
	{ns: "watchtower", name: "log-vault", app: "log-vault", headless: true}, {ns: "watchtower", name: "node-lookout", app: "node-lookout"},
	{ns: "market", name: "storefront", app: "storefront"}, {ns: "market", name: "checkout", app: "checkout"}, {ns: "market", name: "catalog", app: "catalog"},
	{ns: "market", name: "pricing", app: "pricing"}, {ns: "market", name: "mailer", app: "mailer"}, {ns: "market", name: "accounts", app: "accounts"},
	{ns: "market", name: "reviews", app: "reviews"}, {ns: "market", name: "admin-portal", app: "admin-portal", kind: corev1.ServiceTypeNodePort},
	{ns: "market", name: "orders-db", app: "orders-db", headless: true}, {ns: "market", name: "orders-db-read", app: "orders-db"},
	{ns: "market", name: "search", app: "search", headless: true}, {ns: "market", name: "queue", app: "queue", headless: true},
	{ns: "market", name: "sessions", app: "sessions", headless: true}, {ns: "market", name: "kv", app: "kv", headless: true},
	{ns: "market", name: "dispatcher", app: "dispatcher", headless: true}, {ns: "market", name: "archive", app: "archive", headless: true},
}

var managedIngresses = []struct{ ns, name, service string }{
	{"market", "storefront", "storefront"}, {"market", "checkout", "checkout"}, {"market", "catalog", "catalog"}, {"market", "accounts", "accounts"},
	{"market", "admin", "admin-portal"}, {"watchtower", "dashboards", "dashboards"}, {"watchtower", "alerts", "alert-bell"},
}

var managedCRDs = []live.CRD{
	{Group: "cert-manager.io", Kind: "Certificate", Plural: "certificates", Scope: "Namespaced", Version: "v1"},
	{Group: "cert-manager.io", Kind: "CertificateRequest", Plural: "certificaterequests", Scope: "Namespaced", Version: "v1"},
	{Group: "cert-manager.io", Kind: "Issuer", Plural: "issuers", Scope: "Namespaced", Version: "v1"},
	{Group: "cert-manager.io", Kind: "ClusterIssuer", Plural: "clusterissuers", Scope: "Cluster", Version: "v1"},
	{Group: "acme.cert-manager.io", Kind: "Order", Plural: "orders", Scope: "Namespaced", Version: "v1"},
	{Group: "acme.cert-manager.io", Kind: "Challenge", Plural: "challenges", Scope: "Namespaced", Version: "v1"},
	{Group: "external-secrets.io", Kind: "ExternalSecret", Plural: "externalsecrets", Scope: "Namespaced", Version: "v1"},
	{Group: "external-secrets.io", Kind: "SecretStore", Plural: "secretstores", Scope: "Namespaced", Version: "v1"},
	{Group: "external-secrets.io", Kind: "ClusterSecretStore", Plural: "clustersecretstores", Scope: "Cluster", Version: "v1"},
	{Group: "external-secrets.io", Kind: "PushSecret", Plural: "pushsecrets", Scope: "Namespaced", Version: "v1alpha1"},
	{Group: "cilium.io", Kind: "CiliumEndpoint", Plural: "ciliumendpoints", Scope: "Namespaced", Version: "v2"},
	{Group: "cilium.io", Kind: "CiliumIdentity", Plural: "ciliumidentities", Scope: "Cluster", Version: "v2"},
	{Group: "cilium.io", Kind: "CiliumNode", Plural: "ciliumnodes", Scope: "Cluster", Version: "v2"},
	{Group: "cilium.io", Kind: "CiliumNetworkPolicy", Plural: "ciliumnetworkpolicies", Scope: "Namespaced", Version: "v2"},
	{Group: "cloud.google.com", Kind: "BackendConfig", Plural: "backendconfigs", Scope: "Namespaced", Version: "v1"},
	{Group: "networking.gke.io", Kind: "ManagedCertificate", Plural: "managedcertificates", Scope: "Namespaced", Version: "v1"},
	{Group: "networking.gke.io", Kind: "FrontendConfig", Plural: "frontendconfigs", Scope: "Namespaced", Version: "v1beta1"},
	{Group: "monitoring.googleapis.com", Kind: "PodMonitoring", Plural: "podmonitorings", Scope: "Namespaced", Version: "v1"},
	{Group: "nodemanagement.gke.io", Kind: "UpdateInfo", Plural: "updateinfos", Scope: "Cluster", Version: "v1alpha1"},
}

var managedCustom = []struct {
	kind, ns string
	n        int
}{
	{"Certificate", "market", 3}, {"Certificate", "watchtower", 1}, {"CertificateRequest", "market", 3}, {"CertificateRequest", "watchtower", 1},
	{"Issuer", "market", 1}, {"ClusterIssuer", "", 2}, {"ExternalSecret", "market", 4}, {"ExternalSecret", "watchtower", 2},
	{"SecretStore", "market", 1}, {"ClusterSecretStore", "", 1}, {"CiliumIdentity", "", 24}, {"CiliumNode", "", 3},
	{"BackendConfig", "market", 1}, {"ManagedCertificate", "market", 1}, {"PodMonitoring", "watchtower", 2}, {"UpdateInfo", "", 1},
}

func Managed() live.Snapshot {
	m := &managed{born: time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)}
	m.s = live.Snapshot{TakenAt: time.Date(2026, 10, 27, 14, 5, 0, 0, time.UTC), Version: ManagedVersion,
		Context:  live.ContextInfo{Name: "staging-north", Cluster: "staging-north", Server: ManagedServer, User: "staging-north-user"},
		Identity: live.Identity{Username: "player@example.com", Groups: []string{"system:authenticated"}}}
	for _, ns := range managedNamespaces {
		m.s.Namespaces = append(m.s.Namespaces, corev1.Namespace{ObjectMeta: m.meta("", ns)})
	}
	for i := 0; i < 3; i++ {
		m.node(fmt.Sprintf("gke-staging-north-%s-7f3c9a1e-%c%c2k", managedPool, 'b'+i, 'q'+i))
	}
	for _, d := range systemDeployments {
		m.deployment(d)
	}
	for _, d := range appDeployments {
		m.deployment(d)
	}
	for _, d := range runningDaemonSets {
		m.daemonSet(d, nil)
	}
	for i := 0; i < 32; i++ {
		ns := "kube-system"
		if i%8 == 7 {
			ns = "watchtower"
		}
		m.daemonSet(workload{ns: ns, name: fmt.Sprintf("idle-shrine-%02d", i)}, idleSelectors[i%len(idleSelectors)])
	}
	for _, st := range managedStatefulSets {
		m.statefulSet(st)
	}
	m.storageClasses()
	for _, j := range []workload{{ns: "market", name: "migrate-schema-41"}, {ns: "watchtower", name: "compact-2026-10-26"}, {ns: "sealwrights", name: "startup-check"}} {
		m.job(j)
	}
	for _, svc := range managedServices {
		m.service(svc.ns, svc.name, svc.app, svc.kind, svc.headless)
	}
	for _, ing := range managedIngresses {
		m.ingress(ing.ns, ing.name, ing.service)
	}
	m.custom()
	return m.s
}

func (m *managed) uid() types.UID {
	m.seq++
	return types.UID(fmt.Sprintf("6b1f0c2e-0000-4a00-9000-%012d", m.seq))
}

func (m *managed) meta(ns, name string) metav1.ObjectMeta {
	m.seq++
	return metav1.ObjectMeta{Namespace: ns, Name: name, UID: types.UID(fmt.Sprintf("6b1f0c2e-0000-4a00-9000-%012d", m.seq)),
		CreationTimestamp: metav1.NewTime(m.born.Add(time.Duration(m.seq) * time.Minute)), ResourceVersion: fmt.Sprint(1000 + m.seq)}
}

func (m *managed) node(name string) {
	n := corev1.Node{ObjectMeta: m.meta("", name)}
	n.Labels = map[string]string{live.LabelZone: managedZone, live.LabelInstanceType: managedType, nodePoolLabel: managedPool,
		"kubernetes.io/os": "linux", "kubernetes.io/arch": "amd64", "kubernetes.io/hostname": name}
	n.Status = corev1.NodeStatus{
		Capacity:    corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2"), corev1.ResourceMemory: resource.MustParse("8148424Ki"), corev1.ResourcePods: resource.MustParse("110")},
		Allocatable: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1930m"), corev1.ResourceMemory: resource.MustParse("6085060Ki"), corev1.ResourcePods: resource.MustParse("110")},
		Conditions:  []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue, Reason: "KubeletReady"}},
		NodeInfo:    corev1.NodeSystemInfo{KubeletVersion: ManagedVersion, OSImage: "Container-Optimized OS", ContainerRuntimeVersion: "containerd://2.1.4", OperatingSystem: "linux", Architecture: "amd64"},
		Addresses:   []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: fmt.Sprintf("10.20.0.%d", len(m.nodes)+2)}},
	}
	m.nodes = append(m.nodes, name)
	m.s.Nodes = append(m.s.Nodes, n)
}

func ref(kind string, o metav1.ObjectMeta, api string) []metav1.OwnerReference {
	yes := true
	return []metav1.OwnerReference{{APIVersion: api, Kind: kind, Name: o.Name, UID: o.UID, Controller: &yes, BlockOwnerDeletion: &yes}}
}

func labels(app string) map[string]string {
	return map[string]string{"app": app}
}

func (m *managed) template(w workload, image string) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels(w.name)}, Spec: corev1.PodSpec{Containers: []corev1.Container{{
		Name: w.name, Image: image,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("50m"), corev1.ResourceMemory: resource.MustParse("64Mi")},
			Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")},
		},
	}}}}
}

func (m *managed) pod(w workload, owner []metav1.OwnerReference, name, node string, tpl corev1.PodTemplateSpec) *corev1.Pod {
	p := corev1.Pod{ObjectMeta: m.meta(w.ns, name), Spec: *tpl.Spec.DeepCopy()}
	p.Labels, p.OwnerReferences = labels(w.name), owner
	p.Spec.NodeName = node
	m.ips++
	ready := []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}, {Type: corev1.ContainersReady, Status: corev1.ConditionTrue}, {Type: corev1.PodScheduled, Status: corev1.ConditionTrue}}
	started := metav1.NewTime(m.born.Add(time.Hour))
	p.Status = corev1.PodStatus{Phase: corev1.PodRunning, Conditions: ready, PodIP: fmt.Sprintf("10.40.%d.%d", m.ips/200, m.ips%200+2), QOSClass: corev1.PodQOSBurstable,
		ContainerStatuses: []corev1.ContainerStatus{{Name: w.name, Image: tpl.Spec.Containers[0].Image, Ready: true, Started: new(bool),
			State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: started}}}}}
	*p.Status.ContainerStatuses[0].Started = true
	m.s.Pods = append(m.s.Pods, p)
	return &m.s.Pods[len(m.s.Pods)-1]
}

func (m *managed) deployment(w workload) {
	replicas := int32(w.replicas)
	d := appsv1.Deployment{ObjectMeta: m.meta(w.ns, w.name)}
	d.Labels = labels(w.name)
	d.Spec = appsv1.DeploymentSpec{Replicas: &replicas, Selector: &metav1.LabelSelector{MatchLabels: labels(w.name)}, Template: m.template(w, "registry.example/"+w.name+":1.4.2")}
	d.Status = appsv1.DeploymentStatus{Replicas: replicas, ReadyReplicas: replicas, AvailableReplicas: replicas, UpdatedReplicas: replicas}
	m.s.Deployments = append(m.s.Deployments, d)
	hash := fmt.Sprintf("%x", 0x5c4d3b2a+m.seq)[:8]
	rs := appsv1.ReplicaSet{ObjectMeta: m.meta(w.ns, w.name+"-"+hash)}
	rs.Labels, rs.OwnerReferences = labels(w.name), ref("Deployment", d.ObjectMeta, "apps/v1")
	rs.Spec = appsv1.ReplicaSetSpec{Replicas: &replicas, Selector: d.Spec.Selector, Template: d.Spec.Template}
	rs.Status = appsv1.ReplicaSetStatus{Replicas: replicas, ReadyReplicas: replicas, AvailableReplicas: replicas}
	m.s.ReplicaSets = append(m.s.ReplicaSets, rs)
	for i := 0; i < w.replicas; i++ {
		m.pod(w, ref("ReplicaSet", rs.ObjectMeta, "apps/v1"), fmt.Sprintf("%s-%s-%c%c%c%c%c", w.name, hash, 'a'+i, 'k'+i%7, 'p', 'x'+i%3, 'z'-i), m.nodes[(m.seq+i)%len(m.nodes)], rs.Spec.Template)
	}
}

func (m *managed) daemonSet(w workload, selector map[string]string) {
	ds := appsv1.DaemonSet{ObjectMeta: m.meta(w.ns, w.name)}
	ds.Labels = labels(w.name)
	ds.Spec = appsv1.DaemonSetSpec{Selector: &metav1.LabelSelector{MatchLabels: labels(w.name)}, Template: m.template(w, "registry.example/"+w.name+":2.0.1")}
	ds.Spec.Template.Spec.NodeSelector = selector
	if selector == nil {
		n := int32(len(m.nodes))
		ds.Status = appsv1.DaemonSetStatus{DesiredNumberScheduled: n, CurrentNumberScheduled: n, NumberReady: n, NumberAvailable: n, UpdatedNumberScheduled: n}
	}
	m.s.DaemonSets = append(m.s.DaemonSets, ds)
	if selector != nil {
		return
	}
	for i, node := range m.nodes {
		m.pod(w, ref("DaemonSet", ds.ObjectMeta, "apps/v1"), fmt.Sprintf("%s-%c%c%c%c%c", w.name, 'f'+i, 'q', 'w'+i, 'b', 'm'+i), node, ds.Spec.Template)
	}
}

func (m *managed) statefulSet(w workload) {
	replicas := int32(w.replicas)
	st := appsv1.StatefulSet{ObjectMeta: m.meta(w.ns, w.name)}
	st.Labels = labels(w.name)
	st.Spec = appsv1.StatefulSetSpec{Replicas: &replicas, ServiceName: w.name, Selector: &metav1.LabelSelector{MatchLabels: labels(w.name)}, Template: m.template(w, "registry.example/"+w.name+":7.1")}
	class := w.class
	for _, c := range w.claims {
		st.Spec.VolumeClaimTemplates = append(st.Spec.VolumeClaimTemplates, corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: c},
			Spec: corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, StorageClassName: &class,
				Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("10Gi")}}}})
		st.Spec.Template.Spec.Containers[0].VolumeMounts = append(st.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{Name: c, MountPath: "/var/" + c})
	}
	st.Status = appsv1.StatefulSetStatus{Replicas: replicas, ReadyReplicas: replicas, CurrentReplicas: replicas, AvailableReplicas: replicas}
	m.s.StatefulSets = append(m.s.StatefulSets, st)
	ordinals := max(w.replicas, 1)
	for i := 0; i < ordinals; i++ {
		name := fmt.Sprintf("%s-%d", w.name, i)
		var p *corev1.Pod
		if i < w.replicas {
			p = m.pod(w, ref("StatefulSet", st.ObjectMeta, "apps/v1"), name, m.nodes[(m.seq+i)%len(m.nodes)], st.Spec.Template)
		}
		for _, c := range w.claims {
			claim := c + "-" + name
			m.volume(w.ns, claim, class)
			if p != nil {
				p.Spec.Volumes = append(p.Spec.Volumes, corev1.Volume{Name: c, VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claim}}})
			}
		}
	}
}

func (m *managed) volume(ns, claim, class string) {
	pvc := corev1.PersistentVolumeClaim{ObjectMeta: m.meta(ns, claim)}
	pv := corev1.PersistentVolume{ObjectMeta: m.meta("", fmt.Sprintf("pvc-%s", pvc.UID))}
	size := resource.MustParse("10Gi")
	pvc.Spec = corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, StorageClassName: &class, VolumeName: pv.Name,
		Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: size}}}
	pvc.Status = corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound, Capacity: corev1.ResourceList{corev1.ResourceStorage: size}, AccessModes: pvc.Spec.AccessModes}
	pv.Spec = corev1.PersistentVolumeSpec{Capacity: corev1.ResourceList{corev1.ResourceStorage: size}, AccessModes: pvc.Spec.AccessModes, StorageClassName: class,
		PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
		ClaimRef:                      &corev1.ObjectReference{Kind: "PersistentVolumeClaim", Namespace: ns, Name: claim, UID: pvc.UID},
		PersistentVolumeSource:        corev1.PersistentVolumeSource{CSI: &corev1.CSIPersistentVolumeSource{Driver: "disk.csi.example", VolumeHandle: "disks/" + pv.Name}}}
	pv.Status = corev1.PersistentVolumeStatus{Phase: corev1.VolumeBound}
	m.s.PVCs = append(m.s.PVCs, pvc)
	m.s.PVs = append(m.s.PVs, pv)
}

func (m *managed) storageClasses() {
	wait, now := storagev1.VolumeBindingWaitForFirstConsumer, storagev1.VolumeBindingImmediate
	del := corev1.PersistentVolumeReclaimDelete
	yes := true
	for i, name := range []string{"sc-standard", "sc-fast", "sc-balanced", "sc-legacy"} {
		sc := storagev1.StorageClass{ObjectMeta: m.meta("", name), Provisioner: "disk.csi.example", ReclaimPolicy: &del, VolumeBindingMode: &wait, AllowVolumeExpansion: &yes}
		if i == 3 {
			sc.VolumeBindingMode = &now
		}
		if i == 0 {
			sc.Annotations = map[string]string{"storageclass.kubernetes.io/is-default-class": "true"}
		}
		m.s.StorageClasses = append(m.s.StorageClasses, sc)
	}
}

func (m *managed) job(w workload) {
	one := int32(1)
	j := batchv1.Job{ObjectMeta: m.meta(w.ns, w.name)}
	j.Spec = batchv1.JobSpec{Completions: &one, Parallelism: &one, Template: m.template(w, "registry.example/"+w.name+":1")}
	j.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
	done := metav1.NewTime(m.born.Add(48 * time.Hour))
	j.Status = batchv1.JobStatus{Succeeded: 1, CompletionTime: &done, Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}}}
	m.s.Jobs = append(m.s.Jobs, j)
}

func (m *managed) service(ns, name, app string, kind corev1.ServiceType, headless bool) {
	svc := corev1.Service{ObjectMeta: m.meta(ns, name)}
	if kind == "" {
		kind = corev1.ServiceTypeClusterIP
	}
	port := corev1.ServicePort{Name: "http", Port: 80, TargetPort: intstr.FromInt32(8080), Protocol: corev1.ProtocolTCP}
	if kind != corev1.ServiceTypeClusterIP {
		port.NodePort = 30080 + int32(m.seq%100)
	}
	svc.Spec = corev1.ServiceSpec{Type: kind, Ports: []corev1.ServicePort{port}, ClusterIP: fmt.Sprintf("10.60.%d.%d", m.seq/200, m.seq%200+10)}
	if headless {
		svc.Spec.ClusterIP = corev1.ClusterIPNone
	}
	if app != "" {
		svc.Spec.Selector = labels(app)
	}
	if kind == corev1.ServiceTypeLoadBalancer {
		svc.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{{IP: "198.51.100.24"}}
	}
	if name == "kubernetes" {
		port = corev1.ServicePort{Name: "https", Port: 443, TargetPort: intstr.FromInt32(443), Protocol: corev1.ProtocolTCP}
		svc.Spec.Ports = []corev1.ServicePort{port}
	}
	m.s.Services = append(m.s.Services, svc)
	if app == "" {
		return
	}
	slice := discoveryv1.EndpointSlice{ObjectMeta: m.meta(ns, fmt.Sprintf("%s-%c%c%c%c%c", name, 'a'+m.seq%20, 'k', 'q'+m.seq%5, 'x', 'z'-m.seq%9)), AddressType: discoveryv1.AddressTypeIPv4}
	slice.Labels = map[string]string{discoveryv1.LabelServiceName: name, discoveryv1.LabelManagedBy: "endpointslice-controller.k8s.io"}
	slice.OwnerReferences = ref("Service", svc.ObjectMeta, "v1")
	pname, proto, pnum := "http", corev1.ProtocolTCP, int32(8080)
	slice.Ports = []discoveryv1.EndpointPort{{Name: &pname, Protocol: &proto, Port: &pnum}}
	yes := true
	for _, p := range m.s.Pods {
		if p.Namespace != ns || p.Labels["app"] != app {
			continue
		}
		node := p.Spec.NodeName
		slice.Endpoints = append(slice.Endpoints, discoveryv1.Endpoint{Addresses: []string{p.Status.PodIP}, NodeName: &node,
			Conditions: discoveryv1.EndpointConditions{Ready: &yes, Serving: &yes},
			TargetRef:  &corev1.ObjectReference{Kind: "Pod", Namespace: ns, Name: p.Name, UID: p.UID}})
	}
	m.s.EndpointSlices = append(m.s.EndpointSlices, slice)
}

func (m *managed) ingress(ns, name, service string) {
	class := ingressClass
	prefix := networkingv1.PathTypePrefix
	ing := networkingv1.Ingress{ObjectMeta: m.meta(ns, name)}
	host := name + "." + ns + ".staging.example.org"
	ing.Spec = networkingv1.IngressSpec{IngressClassName: &class,
		TLS: []networkingv1.IngressTLS{{Hosts: []string{host}, SecretName: name + "-tls"}},
		Rules: []networkingv1.IngressRule{{Host: host, IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{
			Path: "/", PathType: &prefix, Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: service, Port: networkingv1.ServiceBackendPort{Number: 80}}},
		}}}}}}}
	ing.Status.LoadBalancer.Ingress = []networkingv1.IngressLoadBalancerIngress{{IP: "198.51.100.24"}}
	m.s.Ingresses = append(m.s.Ingresses, ing)
}

func (m *managed) custom() {
	for _, c := range managedCRDs {
		c.Name, c.UID = c.Plural+"."+c.Group, m.uid()
		m.s.CRDs = append(m.s.CRDs, c)
	}
	group := map[string]string{}
	for _, c := range managedCRDs {
		group[c.Kind] = c.Group
	}
	for _, cr := range managedCustom {
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

func toLower(s string) string {
	out := []byte(s)
	for i, c := range out {
		if c >= 'A' && c <= 'Z' {
			out[i] = c + 'a' - 'A'
		}
	}
	return string(out)
}

const stagingSeq = 9000

var stagingNamespaces = []string{"gmp-system", "gmp-public", "gke-managed-system", "gke-managed-cim", "gke-managed-filestorecsi", "gke-managed-volumepopulator"}

func Staging() live.Snapshot {
	m := &managed{s: Managed(), seq: stagingSeq, born: time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)}
	m.s.Context = live.ContextInfo{Name: "gke-staging", Cluster: "gke-staging", Server: ManagedServer, User: "gke-staging-user"}
	for _, ns := range stagingNamespaces {
		m.s.Namespaces = append(m.s.Namespaces, corev1.Namespace{ObjectMeta: m.meta("", ns)})
	}
	return m.s
}
