package lexicon

import (
	"maps"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const (
	StoryCommand      = "speakstctl"
	CodexCommand      = "codexctl"
	ChaosEventMessage = "The chaos monkey deleted this Pod"
)

var storyPreferred = map[string]string{
	"Pod": "hut", "Pods": "huts", "pod": "hut", "pods": "huts",
	ChaosEventMessage: "The Simian Imp toppled this hut", "chaos monkey": "Simian Imp",
	"static Pod": "hut at the door", "static Pods": "huts at the door", "mirror Pod": "mirror hut",
	"Node": "Keep", "Nodes": "Keeps", "node": "Keep", "nodes": "Keeps",
	"control-plane Node": "Lone Tower", "control-plane Nodes": "Lone Towers",
	"Service": "depot", "Services": "depots",
	"ClusterIP Service": "Inner Hall", "NodePort": "Postern Gate",
	"Deployment": "crew", "Deployments": "crews", "deployment": "crew", "deployments": "crews",
	"ReplicaSet": "shift", "ReplicaSets": "shifts", "replicaset": "shift", "replicasets": "shifts",
	"DaemonSet": "Shrine", "DaemonSets": "Shrines", "StatefulSet": "Knightly Order", "StatefulSets": "Knightly Orders",
	"CronJob": "Ritual", "CronJobs": "Rituals", "Job": "Quest", "Jobs": "Quests",
	"ConfigMap": "Scroll", "ConfigMaps": "Scrolls", "Secret": "Sealed Scroll", "Secrets": "Sealed Scrolls",
	"ServiceAccount": "Writ", "ServiceAccounts": "Writs", "serviceaccount": "writ", "serviceaccounts": "writs",
	"service account": "writ", "service accounts": "writs", "Service Account": "Writ",
	"Namespace": "Fief", "Namespaces": "Fiefs", "namespace": "fief", "namespaces": "fiefs", "replica set": "shift", "replica sets": "shifts", "EndpointSlice": "Guild Roll", "EndpointSlices": "Guild Rolls",
	"HorizontalPodAutoscaler": "War Horn", "HorizontalPodAutoscalers": "War Horns", "HPA": "War Horn", "HPAs": "War Horns",
	"etcd": "the Codex", "etcd member": "Codex-keeper", "etcd members": "Codex-keepers",
	"kube-apiserver": "the Chancery", "kube-scheduler": "the Quartermaster",
	"kube-controller-manager": "the Stewards' Hall", "kube-proxy": "the Wayfinder", "CoreDNS": "the Herald",
	"metrics-server": "the Augur", "kubelet": "Reeve", "kubelets": "Reeves",
	"control plane": "Citadel", "control-plane": "Citadel", "cluster": "realm", "clusters": "realms",
	"CPU": "food", "CPUs": "food", "cpu": "food", "memory": "housing", "millicores": "rations", "millicore": "ration",
	"CPU request": "Food Pledge", "CPU requests": "Food Pledges", "memory request": "Housing Pledge", "memory requests": "Housing Pledges",
	"CPU limit": "Food Binding", "CPU limits": "Food Bindings", "memory limit": "Housing Binding", "memory limits": "Housing Bindings",
	"CPU throttling": "hunger", "throttling": "hunger", "throttled": "hungry",
	"Pending": "Unhoused", "CrashLoopBackOff": "Guttering", "OOMKilled": "Voidtaken", "Evicted": "Banished",
	"ImagePullBackOff": "the graduate never arrived", "ContainerCreating": "Kindling", "Terminating": "Last Rites",
	"ErrImagePull": "the graduate never arrived", "ErrImagePull / ImagePullBackOff": "the graduate never arrived",
	"Running": "Burning", "Completed": "Quest Fulfilled",
	"taint": "Ward", "taints": "Wards", "toleration": "Charm", "tolerations": "Charms",
	"container": "worker", "containers": "workers", "init container": "Squire", "init containers": "Squires",
	"sidecar": "Familiar", "sidecars": "Familiars", "image": "trade", "images": "trades", "image tag": "class", "image tags": "classes",
	"replicas": "count", "label": "badge", "labels": "badges", "selector": "badge it looks for", "selectors": "badges it looks for",
	"label selector": "badge it looks for", "scheduler": "Quartermaster", "controller": "Steward", "controllers": "Stewards",
	"endpoint": "roll entry", "endpoints": "roll entries",
	"Gateway": "harbor", "Gateways": "harbors", "GatewayClass": "harbor guild", "Ingress": "the Old Gate",
	"HTTPRoute": "road", "HTTPRoutes": "roads", "TCPRoute": "road", "TCPRoutes": "roads", "TLSRoute": "sealed road", "TLSRoutes": "sealed roads",
	"SLO": "Nines", "YAML": "script", "manifest": "script", "manifests": "scripts",
	"PodDisruptionBudget": "Garrison Charter", "PodDisruptionBudgets": "Garrison Charters",
	"PriorityClass": "Rank", "PriorityClasses": "Ranks", "preemption": "Displacement",
	"BestEffort": "Vagrant", "Burstable": "Freeman", "Guaranteed": "Landed", "QoS": "standing",
	"readiness probe": "Shutters", "readiness probes": "Shutters", "liveness probe": "Bell", "liveness probes": "Bells",
	"probe": "watch", "probes": "watches", "readiness": "Shutters", "liveness": "Bell",
	"rollout": "roll-out", "rollouts": "roll-outs", "autoscaler": "War Horn", "autoscalers": "War Horns",
	"PersistentVolume": "Vault", "PersistentVolumes": "Vaults", "PersistentVolumeClaim": "Deed", "PersistentVolumeClaims": "Deeds",
	"PVC": "Deed", "PVCs": "Deeds", "StorageClass": "vault kind", "StorageClasses": "vault kinds",
	"LimitRange": "default pledges", "ResourceQuota": "allowance", "Kubernetes": "the Kingdom's law", "k8s": "the Kingdom's law",
	"apiserver": "Chancery", "Event": "Tiding", "Events": "Tidings", "NotReady": "not yet ready",
	"memory land": "roomy land", "compute land": "fertile land", "memory-optimized": "roomy", "compute-optimized": "fertile",
	"TLS passthrough": "sealed passage", "TLS termination": "seal-breaking", "Node-pressure eviction": "the Crowding",
	"node-pressure eviction": "the Crowding", "HTTP listener": "cog quay", "TCP listener": "caravel quay", "HTTPS listener": "galleon quay",
	"TLS listener": "carrack quay", "listener": "quay", "listeners": "quays",
	"requests.cpu": "the Food Pledge", "requests.memory": "the Housing Pledge", "limits.cpu": "the Food Binding",
	"limits.memory": "the Housing Binding", "requests.ephemeral-storage": "the Cellar Pledge", "limits.ephemeral-storage": "the Cellar Binding",
	"ephemeral-storage": "Cellar", "memory.available": "free housing", "imagefs.available": "free ground", "nodefs": "the Keep's ground",
	"imagefs": "the Keep's common store", "node.kubernetes.io/instance-type": "the Keep-size badge", "node.kubernetes.io/unreachable": "the Storm Ward",
	"node.kubernetes.io/not-ready": "the Unready Ward", "node.kubernetes.io/memory-pressure": "the Crowding Ward",
	"node.kubernetes.io/pid-pressure": "the Throng Ward", "node.kubernetes.io/unschedulable": "the Cordon Ward",
	"node.kubernetes.io/network-unavailable": "the Fog Ward",
	"node.kubernetes.io/disk-pressure":       "the Cellar Ward", "topology.kubernetes.io/zone": "the isle badge", "etcd-defrag": "the Palimpsest Rite",
	"the Kubernetes docs": "the old scrolls", "Kubernetes docs": "the old scrolls", "pod-template-hash": "hut pattern",
	"Node-Selector": "Keep choice", "Node-Selectors": "Keep choices", "DaemonSet-managed": "Shrine-kept", "taint-based eviction": "Ward eviction",
	"Metrics API": "Omens", "workload": "crew", "workloads": "crews", "Kubernetes terms": "the Kingdom's law",
	"autoscale": "horn", "autoscaled": "horned", "Deployment.apps": "crew", "deployment.apps": "crew", "deployments.apps": "crews",
	"replicaset.apps": "shift", "replicasets.apps": "shifts", "horizontalpodautoscaler.autoscaling": "horn",
	"horizontalpodautoscalers.autoscaling": "horns", "priorityclass.scheduling.k8s.io": "rank", "priorityclasses.scheduling.k8s.io": "ranks",
	"controllerrevision.apps": "record", "controllerrevisions.apps": "records", "service/": "depot/", "configmap/": "scroll/", "namespace/": "fief/",
	"daemonset.apps": "shrine", "daemonsets.apps": "shrines", "daemonset": "shrine", "daemonsets": "shrines",
	"statefulset.apps": "order", "statefulsets.apps": "orders", "statefulset": "order", "statefulsets": "orders",
	"persistentvolume": "vault", "persistentvolumes": "vaults", "persistentvolumeclaim": "deed", "persistentvolumeclaims": "deeds",
	"storageclass.storage.k8s.io": "vault kind", "storageclass": "vault kind", "storageclasses": "vault kinds",
	"endpointslice.discovery.k8s.io": "roll", "endpointslices.discovery.k8s.io": "rolls",
	"old replicas": "old huts", "new replicas": "new huts", "updated replicas": "updated huts",
	"milliCPU": "rations", "cluster-wide": "realm-wide", "RollingUpdate": "rolling", "spec.containers[*].image": "the trade",
	"Levy": "hut", "Levies": "huts",
	"Pilgrim": "box", "Pilgrims": "boxes", "pilgrim": "box", "pilgrims": "boxes",
}

var StoryVerbs = map[string]bool{"get": true, "describe": true, "scale": true, "set": true, "top": true, "apply": true, "delete": true,
	"run": true, "create": true, "expose": true, "label": true, "autoscale": true, "rollout": true, "cordon": true, "drain": true,
	"uncordon": true, "taint": true, "logs": true, "explain": true, "edit": true, "annotate": true, "image": true, "resources": true,
	"status": true, "history": true, "undo": true, "restart": true, "pause": true, "resume": true, "version": true, "cluster-info": true,
	"api-resources": true, "realm-info": true, "alarm": true, "list": true, "disarm": true, "defrag": true, "compact": true, "dump": true}

var storyCommandNouns = map[string]string{
	"pods": "huts", "pod": "hut", "po": "huts", "nodes": "keeps", "node": "keep", "no": "keeps",
	"deployments": "crews", "deployment": "crew", "deploy": "crew", "services": "depots", "service": "depot", "svc": "depots",
	"replicasets": "shifts", "replicaset": "shift", "rs": "shifts", "hpa": "horns", "horizontalpodautoscalers": "horns",
	"daemonsets": "shrines", "daemonset": "shrine", "ds": "shrines", "statefulsets": "orders", "statefulset": "order", "sts": "orders",
	"events": "tidings", "configmaps": "scrolls", "configmap": "scroll", "namespaces": "fiefs", "namespace": "fief",
	"serviceaccounts": "writs", "serviceaccount": "writ", "sa": "writs",
	"endpoints": "rolls", "endpointslices": "rolls", "endpointslice": "roll", "pvc": "deeds", "persistentvolumeclaims": "deeds", "pv": "vaults",
	"persistentvolumes": "vaults", "jobs": "quests", "job": "quest", "cronjobs": "rituals", "cronjob": "ritual",
	"clusterip": "inner", "limitranges": "defaults", "limitrange": "defaults", "limits": "defaults",
}

var storyListNouns = map[string]string{
	"storageclasses": "vault kinds", "priorityclasses": "ranks", "ingresses": "signposts", "gateways": "harbor gates",
	"customresourcedefinitions": "foreign halls",
}

func StoryList(name string) string {
	if story, ok := storyListNouns[name]; ok {
		return story
	}
	if story, ok := storyCommandNouns[name]; ok {
		return story
	}
	return name
}

var storyCommandFlags = map[string]string{
	"--replicas": "--count", "--image": "--trade", "--requests": "--pledge", "--limits": "--binding",
	"--cpu-percent": "--food-percent", "--cpu": "--food", "--memory": "--housing", "--labels": "--badges",
	"--show-labels": "--show-badges", "--selector": "--looks-for", "--pod-selector": "--hut-looks-for", "--min": "--floor", "--max": "--ceiling", "--ignore-daemonsets": "--ignore-shrines",
	"--port": "--door", "--target-port": "--target-door", "--previous": "--last-life", "--clusterip": "--inner-door",
}

var (
	storyWords   = buildStoryWords()
	storyPattern = compileTrue(storyWords)
	commandStart = regexp.MustCompile(`\b(?:kubectl|etcdctl)\b`)
	milliCount   = regexp.MustCompile(`\b(\d[\d,]*)m(?: of CPU| CPU)?\b`)
	mebiCount    = regexp.MustCompile(`\b(\d[\d,]*) ?(Mi|Gi|Ki)B?(?: of memory| memory)?\b`)
	doubleThe    = regexp.MustCompile(`\b([Tt]he) the\b`)
	articleThe   = regexp.MustCompile(`\b([Aa]n?) the\b`)
)

func buildStoryWords() map[string]string {
	out := map[string]string{}
	add := func(from, to string) {
		if _, ok := out[from]; !ok {
			out[from] = to
		}
	}
	keys := make([]string, 0, len(storyPreferred))
	for k := range storyPreferred {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		to := storyPreferred[k]
		add(k, to)
		if r := rune(k[0]); unicode.IsLower(r) {
			add(capitalized(k), capitalized(to))
		}
		if up := strings.ToUpper(k); up != k {
			add(up, strings.ToUpper(to))
		}
	}
	return out
}

func StoryText(text string) string {
	return keepingNames(text, storyText)
}

func storyText(text string) string {
	switch {
	case text == "", strings.HasPrefix(strings.TrimLeft(text, " \n\t"), "{"), IsStoryManifest(text):
		return text
	case manifestDocument(text):
		return StoryManifest(text)
	}
	text = storyReplies.Replace(text)
	text = storyEndpoint.ReplaceAllString(text, "")
	text = storyDefaultFief.ReplaceAllString(text, "$1")
	text = storyWards(text)
	text = storyTaints(text)
	text = storyTypeList(text)
	text = storySystemDepots(text)
	text = storyLabelKeys.Replace(text)
	text = stripGlosses(text)
	text = usageUnits.ReplaceAllStringFunc(text, storyUsage)
	text = storyIdentifiersIn(text)
	var b strings.Builder
	last := 0
	for _, span := range commandSpans(text) {
		storyProse(&b, text[last:span[0]])
		b.WriteString(storyCommand(text[span[0]:span[1]]))
		last = span[1]
	}
	storyProse(&b, text[last:])
	out := doubleThe.ReplaceAllString(b.String(), "$1")
	return articleThe.ReplaceAllString(out, "the")
}

var usageUnits = regexp.MustCompile(`\b(?:CPU|MEMORY)\((?:cores|bytes|%)\)`)

func storyUsage(header string) string {
	name, unit, _ := strings.Cut(strings.TrimSuffix(header, ")"), "(")
	story := map[string]string{"CPU": "FOOD", "MEMORY": "HOUSING"}[name]
	if unit == "%" {
		return story + "(%)"
	}
	return story
}

var storyParenTerm = regexp.MustCompile(` \(([^()]+)\)`)

var storyLabelKeys = strings.NewReplacer("pod-template-hash=", "hut-pattern=", " container=", " worker=", " pod=", " hut=")

var storyReplies = strings.NewReplacer("the kubectl.kubernetes.io/last-applied-configuration annotation", "the copy the Scriptorium keeps",
	" from default namespace", "", " status is now: NodeNotSchedulable", " is now closed to new huts", " status is now: KeepClosed", " is now closed to new huts", "HTTP probe failed with statuscode:", "the watch was answered with")

func NumberSensitive(text string) bool {
	return storyEndpoint.MatchString(text)
}

var (
	storyEndpoint    = regexp.MustCompile(`\[\d{1,3}(?:\.\d{1,3}){3}:\d+\]`)
	storyDefaultFief = regexp.MustCompile(`(^|\s)default/`)
)

var KubernetesWords = regexp.MustCompile(`\b(?:(?i:kubectl)|kubeadm|[Pp]ods?|[Nn]odes?|Services?|Deployments?|ReplicaSets?|[Rr]eplica [Ss]ets?|DaemonSets?|StatefulSets?|CronJobs?|Jobs?|ConfigMaps?|Namespaces?|ServiceAccounts?|[Ss]erviceaccounts?|[Ss]ervice [Aa]ccounts?|[a-z]+ namespaces?|GI|EndpointSlices?|Ingress|Gateways?|HTTPRoutes?|TCPRoutes?|TLSRoutes?|GatewayClass|HPAs?|HorizontalPodAutoscalers?|etcd|kube-apiserver|kube-scheduler|kube-controller-manager|kubelets?|kube-proxy|CoreDNS|metrics-server|CPUs?|cpu|memory|MiB|GiB|millicores?|Pending|CrashLoopBackOff|OOMKilled|ImagePullBackOff|Evicted|ContainerCreating|Terminating|taints?|tolerations?|containers?|YAML|manifests?|clusters?|control[- ]plane|SLO|PodDisruptionBudgets?|PriorityClass(?:es)?|preemption|requests\.cpu|requests\.memory|limits\.cpu|limits\.memory|HTTPS?|TCP|TLS|schedulers?|controllers?|labels?|selectors?|Kubernetes|kubernetes\.io|k8s|apiserver|endpoints?|sidecars?|BestEffort|Burstable|Guaranteed|QoS|LimitRange|ResourceQuota|PersistentVolumes?|PersistentVolumeClaims?|StorageClass(?:es)?|PVCs?|images?|replicas|readiness|liveness|probes?|rollouts?|autoscal\w*|[a-z]+\.(?:apps|autoscaling|batch|policy|[a-z]+\.k8s\.io)|Node[A-Z]\w*|etc/kubernetes|\d+(?:Mi|Gi|Ki|m)\b)\b`)

var glossExtras = regexp.MustCompile(`\b(?:FailedScheduling|back-off|API server|rolling update|horizontal scaling|cordon|evict\w*|termination|passthrough|instance type|hostPort|manifest|SLO|HPA|zones?|[Nn]etwork partition|throttling|template|timed out|request|requests|label|spec\.[\w.]+)\b`)

func kubernetesGloss(inner string) bool {
	return KubernetesWords.MatchString(inner) || glossExtras.MatchString(inner)
}

func storyGlossSpans(text string) []storyGloss {
	out := storyGlosses(text)
	taken := map[int]bool{}
	for _, g := range out {
		taken[g.start] = true
	}
	for _, m := range storyParenTerm.FindAllStringSubmatchIndex(text, -1) {
		head, _, _ := strings.Cut(text[m[2]:m[3]], ": ")
		_, known := storyPreferred[head]
		if (known || kubernetesGloss(text[m[2]:m[3]])) && !taken[m[0]] {
			out = append(out, storyGloss{start: m[0], end: m[1], term: head})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].start < out[j].start })
	return out
}

func manifestDocument(text string) bool {
	head := strings.TrimLeft(text, " \n\t")
	return strings.HasPrefix(head, "apiVersion:") || strings.HasPrefix(head, "{") || strings.HasPrefix(head, "---\napiVersion:")
}

func stripGlosses(text string) string {
	var b strings.Builder
	last := 0
	for _, g := range storyGlossSpans(text) {
		b.WriteString(text[last:g.start])
		last = g.end
	}
	b.WriteString(text[last:])
	return b.String()
}

func commandSpans(text string) [][2]int {
	out := [][2]int{}
	for _, m := range commandStart.FindAllStringIndex(text, -1) {
		if len(out) > 0 && m[0] < out[len(out)-1][1] {
			continue
		}
		end := m[1]
		for end < len(text) && text[end] == ' ' {
			next := end + 1
			for next < len(text) && text[next] != ' ' && text[next] != '\n' {
				next++
			}
			tok := text[end+1 : next]
			if tok == "" || unicode.IsUpper(rune(tok[0])) {
				break
			}
			core, tail := splitTrailing(tok)
			if tail != "" {
				end = end + 1 + len(core)
				break
			}
			end = next
		}
		out = append(out, [2]int{m[0], end})
	}
	return out
}

func storyCommand(cmd string) string {
	tokens := strings.Split(cmd, " ")
	tokens[0] = storyToken(tokens[0])
	out := []string{tokens[0]}
	for i := 1; i < len(tokens); i++ {
		tok := tokens[i]
		if tokens[0] == CodexCommand && i == 1 && tok == "endpoint" {
			continue
		}
		out = append(out, storyCommandWord(tok))
	}
	return strings.Join(out, " ")
}

var storyVerbNames = map[string]string{"cluster-info": "realm-info", "image": "trade", "selector": "looks-for"}

var jsonKey = regexp.MustCompile(`"([A-Za-z][A-Za-z0-9]*)":`)

var (
	storyPatchKeys = withKeys(storyManifestKeys, map[string]string{"template": "pattern"})
	kubePatchKeys  = invert(storyPatchKeys, map[string]string{"badges": "labels"})
)

func withKeys(base, more map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(more))
	maps.Copy(out, base)
	maps.Copy(out, more)
	return out
}

func renameJSONKeys(tok string, names map[string]string) string {
	return jsonKey.ReplaceAllStringFunc(tok, func(m string) string {
		if to, ok := names[m[1:len(m)-2]]; ok {
			return `"` + to + `":`
		}
		return m
	})
}

func storyCommandWord(tok string) string {
	if strings.Contains(tok, `{"`) {
		return renameJSONKeys(tok, storyPatchKeys)
	}
	if core := strings.TrimRight(tok, "]).;:,"); core != tok && core != "" && !strings.HasPrefix(tok, "[") && !strings.HasPrefix(tok, "<") {
		return storyCommandWord(core) + tok[len(core):]
	}
	if story, ok := storyVerbNames[tok]; ok {
		return story
	}
	if StoryVerbs[tok] {
		return tok
	}
	if inner, ok := strings.CutPrefix(tok, "<"); ok && strings.Contains(inner, "|") {
		if core, closed := strings.CutSuffix(inner, ">"); closed {
			return "<" + storyCommandWord(core) + ">"
		}
	}
	if inner, ok := strings.CutPrefix(tok, "["); ok {
		if core, closed := strings.CutSuffix(inner, "]"); closed {
			return "[" + storyCommandWord(core) + "]"
		}
	}
	if strings.Contains(tok, "|") && !strings.HasPrefix(tok, "-") {
		parts := strings.Split(tok, "|")
		for i, part := range parts {
			parts[i] = storyCommandWord(part)
		}
		return strings.Join(parts, "|")
	}
	if story := storyToken(tok); story != tok {
		return story
	}
	if to, ok := storyWords[tok]; ok {
		return to
	}
	return tok
}

var proseFlag = regexp.MustCompile(`--[a-z][a-z-]*`)

func shouted(text string) bool {
	return strings.ContainsFunc(text, unicode.IsUpper) && !strings.ContainsFunc(text, unicode.IsLower)
}

func storyProse(b *strings.Builder, seg string) {
	seg = storyUnits(seg)
	seg = proseFlag.ReplaceAllStringFunc(seg, func(flag string) string {
		if story, ok := storyCommandFlags[flag]; ok {
			return story
		}
		return flag
	})
	prefix := b.String()
	full := prefix + seg
	last := 0
	for _, m := range storyPattern.FindAllStringIndex(seg, -1) {
		word := seg[m[0]:m[1]]
		at := len(prefix) + m[0]
		if inIdentifier(full, at, at+len(word)) || inPath(full, at, at+len(word)) {
			continue
		}
		to := storyWords[word]
		switch {
		case shouted(full):
			to = strings.ToUpper(to)
		case fullStop(full, at) && to != "" && unicode.IsLower(rune(to[0])):
			to = capitalized(to)
		}
		gap := seg[last:m[0]]
		before := b.String() + gap
		if fixed := fitArticle(before, to); len(fixed) >= b.Len() {
			gap = fixed[b.Len():]
		}
		b.WriteString(gap)
		b.WriteString(to)
		last = m[1]
	}
	b.WriteString(seg[last:])
}

func inPath(text string, start, end int) bool {
	return (start > 0 && text[start-1] == '.') || (end+1 < len(text) && text[end] == '.' && unicode.IsLetter(rune(text[end+1])))
}

func splitTrailing(tok string) (string, string) {
	end := len(tok)
	for end > 0 && strings.ContainsRune(".,;:!?)\"", rune(tok[end-1])) {
		end--
	}
	return tok[:end], tok[end:]
}

func storyToken(tok string) string {
	switch tok {
	case "kubectl":
		return StoryCommand
	case "etcdctl":
		return CodexCommand
	}
	if story, ok := storyPlaceholders[tok]; ok {
		return story
	}
	if left, right, ok := strings.Cut(tok, "="); ok && !strings.HasPrefix(tok, "-") {
		if l, known := storyPlaceholders[left]; known {
			if r, both := storyPlaceholders[right]; both {
				return l + "=" + r
			}
			return l + "=" + right
		}
	}
	if noun, ok := storyCommandNouns[tok]; ok {
		return noun
	}
	if kind, name, ok := strings.Cut(tok, "/"); ok {
		if noun, known := storyCommandNouns[kind]; known {
			return noun + "/" + name
		}
	}
	if strings.HasPrefix(tok, "-") {
		return storyFlag(tok)
	}
	return tok
}

func storyFlag(tok string) string {
	name, value, valued := strings.Cut(tok, "=")
	cpu := name == "--cpu"
	if story, ok := storyCommandFlags[name]; ok {
		name = story
	}
	if !valued {
		return name
	}
	if cpu {
		parts := strings.Split(value, "|")
		for i, part := range parts {
			if strings.HasSuffix(part, "m") {
				parts[i] = rationsOf(part)
			}
		}
		return name + "=" + strings.Join(parts, "|")
	}
	return name + "=" + storyValue(value)
}

var storyNamePrefixes = [][2]string{
	{"kube-controller-manager-", "stewards-"}, {"kube-apiserver-", "chancery-"}, {"kube-scheduler-", "quartermaster-"},
	{"kube-proxy-", "wayfinder-"}, {"metrics-server-", "augur-"}, {"coredns-", "herald-"}, {"etcd-", "codex-"},
}

var storyIdentifiers = map[string]string{
	"kube-system": "citadel", "node-role.kubernetes.io/control-plane": "citadel-keep", "kubernetes.io/hostname": "keep-name", "kubernetes.io/service-name": "depot-name",
	"kubernetes.io/os": "keep-ways", "kubernetes.io/arch": "keep-make", "node.kubernetes.io/instance-type": "keep-size",
	"topology.kubernetes.io/zone": "isle", "topology.kubernetes.io/region": "realm-region", "RegisteredNode": "RegisteredKeep",
	"node-controller": "keep-steward", "k8s-app": "citadel-app", "kube-node-lease": "candle-house", "kube-public": "citadel-notices",
	"NodeName": "KeepName", "NodeReady": "KeepReady", "NodeNotReady": "KeepNotReady", "NodeStatusUnknown": "KeepSilent", "NodeNotSchedulable": "KeepClosed", "NodeStatusNeverUpdated": "KeepNeverSpoke",
	"Deployment/metrics-server": "Deployment/augur", "Deployment/coredns": "Deployment/herald", "DaemonSet/kube-proxy": "DaemonSet/wayfinder",
	"kube-dns": "herald", "FailedScheduling": "NoKeepHasRoom", "kubernetes.io/config.mirror": "reeve-mirror",
	"kubernetes.io/config.hash": "reeve-scroll-hash", "kubernetes.io/config.seen": "reeve-scroll-seen", "kubernetes.io/config.source": "reeve-scroll-source",
	"NodeHasSufficientMemory": "KeepHasRoom", "NodeHasInsufficientMemory": "KeepCrowded", "NodeHasNoDiskPressure": "KeepGroundClear",
	"NodeHasDiskPressure": "KeepGroundFull", "NodeHasSufficientPID": "KeepUnthronged", "NodeHasInsufficientPID": "KeepThronged",
}

var storyIdentifierPattern = compileTrue(storyIdentifiers)

func storyName(tok string) string {
	for _, p := range storyNamePrefixes {
		if rest, ok := strings.CutPrefix(tok, p[0]); ok {
			return p[1] + rest
		}
	}
	return tok
}

func kubeName(tok string) string {
	for _, p := range storyNamePrefixes {
		if rest, ok := strings.CutPrefix(tok, p[1]); ok {
			return p[0] + rest
		}
	}
	if kube, ok := kubeIdentifiers[tok]; ok {
		return kube
	}
	return tok
}

var (
	kubeIdentifiers = invert(storyIdentifiers, map[string]string{chanceryDepot: "kubernetes"})
	systemName      = regexp.MustCompile(`\b(?:kube-controller-manager|kube-apiserver|kube-scheduler|kube-proxy|metrics-server|coredns|etcd)-[a-z0-9-]+`)
)

func StoryNames(text string) string {
	return keepingNames(text, func(text string) string { return storyIdentifiersIn(storyWards(text)) })
}

func storyIdentifiersIn(text string) string {
	text = systemName.ReplaceAllStringFunc(text, storyName)
	return storyIdentifierPattern.ReplaceAllStringFunc(text, func(id string) string { return storyIdentifiers[id] })
}

var storyPlaceholders = map[string]string{"<pod>": "<hut>", "<node>": "<keep>", "<container>": "<worker>", "<image>": "<trade>",
	"<service>": "<depot>", "<deployment>": "<crew>"}

func storyValue(value string) string {
	core := strings.TrimRight(value, "]).;:")
	return storyAmounts(core) + value[len(core):]
}

func storyAmounts(value string) string {
	parts := strings.Split(value, ",")
	for i, p := range parts {
		key, amount, ok := strings.Cut(p, "=")
		switch {
		case ok && key == "cpu":
			parts[i] = "food=" + rationsOf(amount)
		case ok && key == "memory":
			parts[i] = "housing=" + mebiOf(amount)
		case ok && key == "ephemeral-storage":
			parts[i] = "cellar=" + mebiOf(amount)
		}
	}
	return strings.Join(parts, ",")
}

func rationsOf(amount string) string {
	if milli, ok := strings.CutSuffix(amount, "m"); ok {
		return milli
	}
	if v, err := strconv.ParseFloat(amount, 64); err == nil {
		return strconv.Itoa(int(v * 1000))
	}
	return amount
}

func mebiOf(amount string) string {
	for suffix, scale := range map[string]float64{"Gi": 1024, "Mi": 1, "Ki": 1.0 / 1024} {
		if n, ok := strings.CutSuffix(amount, suffix); ok {
			if v, err := strconv.ParseFloat(n, 64); err == nil {
				return strconv.Itoa(int(v * scale))
			}
		}
	}
	return amount
}

func storyUnits(text string) string {
	text = milliCount.ReplaceAllStringFunc(text, func(m string) string {
		end := strings.IndexByte(m, 'm')
		n := strings.ReplaceAll(m[:end], ",", "")
		v, _ := strconv.Atoi(n)
		return storyCount(v, "ration", "rations")
	})
	var b strings.Builder
	last := 0
	for _, m := range mebiCount.FindAllStringSubmatchIndex(text, -1) {
		v, _ := strconv.ParseFloat(strings.ReplaceAll(text[m[2]:m[3]], ",", ""), 64)
		switch text[m[4]:m[5]] {
		case "Gi":
			v *= 1024
		case "Ki":
			v /= 1024
		}
		one, many := "room", "rooms"
		if cellarAmount(text, m[0], m[1]) {
			one, many = "barrel", "barrels"
		}
		b.WriteString(text[last:m[0]])
		if v > 0 && v < 1 {
			b.WriteString(strconv.FormatFloat(v, 'g', 2, 64) + " " + many)
		} else {
			b.WriteString(storyCount(int(v), one, many))
		}
		last = m[1]
	}
	b.WriteString(text[last:])
	return b.String()
}

func StoryBarrels(text string) string {
	return mebiCount.ReplaceAllStringFunc(text, func(m string) string {
		sub := mebiCount.FindStringSubmatch(m)
		v, _ := strconv.ParseFloat(strings.ReplaceAll(sub[1], ",", ""), 64)
		switch sub[2] {
		case "Gi":
			v *= 1024
		case "Ki":
			v /= 1024
		}
		return storyCount(int(v), "barrel", "barrels")
	})
}

var (
	cellarWords  = regexp.MustCompile(`(?i)ephemeral[- ]storage|imagefs|nodefs|\bcellar|\bbarrels?\b`)
	housingWords = regexp.MustCompile(`(?i)\bmemory\b|\bhousing\b|\brooms?\b`)
)

func cellarAmount(text string, start, end int) bool {
	lineStart := strings.LastIndexByte(text[:start], '\n') + 1
	lineEnd := len(text)
	if i := strings.IndexByte(text[end:], '\n'); i >= 0 {
		lineEnd = end + i
	}
	before, after := text[lineStart:start], text[end:lineEnd]
	cellar, housing := lastMatch(cellarWords, before), lastMatch(housingWords, before)
	if cellar >= 0 || housing >= 0 {
		return cellar > housing
	}
	return firstMatch(cellarWords, after) >= 0 && (firstMatch(housingWords, after) < 0 || firstMatch(cellarWords, after) < firstMatch(housingWords, after))
}

func lastMatch(re *regexp.Regexp, text string) int {
	all := re.FindAllStringIndex(text, -1)
	if len(all) == 0 {
		return -1
	}
	return all[len(all)-1][0]
}

func firstMatch(re *regexp.Regexp, text string) int {
	if m := re.FindStringIndex(text); m != nil {
		return m[0]
	}
	return -1
}

func storyCount(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return groupDigits(n) + " " + many
}

func groupDigits(n int) string {
	digits := strconv.Itoa(n)
	var b strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(d)
	}
	return b.String()
}

func CommandFromStory(line string) string {
	trimmed := strings.TrimSpace(line)
	tokens := strings.Fields(trimmed)
	if len(tokens) == 0 {
		return line
	}
	switch tokens[0] {
	case StoryCommand:
		tokens[0] = "kubectl"
	case CodexCommand:
		rest := strings.TrimPrefix(trimmed, CodexCommand)
		if strings.HasPrefix(rest, " status") {
			rest = " endpoint" + rest
		}
		return "etcdctl" + rest
	default:
		return line
	}
	for i := 1; i < len(tokens); i++ {
		tokens[i] = kubeToken(tokens[i])
	}
	return strings.Join(tokens, " ")
}

var (
	kubeNouns = invert(storyCommandNouns, map[string]string{"huts": "pods", "hut": "pod", "keeps": "nodes", "keep": "node",
		"crews": "deployments", "crew": "deployment", "depots": "services", "depot": "service", "shifts": "replicasets",
		"shift": "replicaset", "horns": "hpa", "horn": "hpa", "shrines": "daemonsets", "shrine": "daemonset",
		"orders": "statefulsets", "order": "statefulset", "rolls": "endpointslices", "deeds": "pvc", "deed": "pvc",
		"vaults": "pv", "vault": "pv", "quests": "jobs", "quest": "job", "rituals": "cronjobs", "ritual": "cronjob", "defaults": "limitranges"})
	kubeFlags = invert(storyCommandFlags, nil)
)

func invert(table, preferred map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range table {
		if _, taken := out[v]; !taken || len(k) > len(out[v]) {
			out[v] = k
		}
	}
	for k, v := range preferred {
		out[k] = v
	}
	return out
}

func kubeKindList(tok string) (string, bool) {
	if strings.HasPrefix(tok, "-") || !strings.Contains(tok, ",") || strings.ContainsAny(tok, "=/:{") {
		return "", false
	}
	parts := strings.Split(tok, ",")
	for i, p := range parts {
		noun, ok := kubeNouns[strings.ToLower(p)]
		if !ok {
			noun = p
		}
		parts[i] = noun
	}
	return strings.Join(parts, ","), true
}

func kubeToken(tok string) string {
	if strings.Contains(tok, `{"`) {
		return renameJSONKeys(tok, kubePatchKeys)
	}
	if list, ok := kubeKindList(tok); ok {
		return list
	}
	tok = kubeName(tok)
	for kube, story := range storyVerbNames {
		if tok == story {
			return kube
		}
	}
	for kube, story := range storyPlaceholders {
		if tok == story {
			return kube
		}
	}
	if noun, ok := kubeNouns[strings.ToLower(tok)]; ok {
		return noun
	}
	if noun, ok := storyMerged[strings.ToLower(tok)]; ok {
		return noun
	}
	if key, effect, ok := strings.Cut(tok, ":"); ok && !strings.HasPrefix(tok, "-") {
		if kube, known := kubeIdentifiers[key]; known {
			return kube + ":" + effect
		}
	}
	if kind, name, ok := strings.Cut(tok, "/"); ok {
		if noun, known := kubeNouns[kind]; known {
			return noun + "/" + name
		}
		if noun, merged := storyMerged[kind]; merged {
			return noun + "/" + name
		}
	}
	if key, value, ok := strings.Cut(tok, "="); ok && !strings.HasPrefix(tok, "-") {
		if kube, known := kubeIdentifiers[key]; known && strings.Contains(kube, "kubernetes.io/") {
			return kube + "=" + value
		}
	}
	if !strings.HasPrefix(tok, "-") {
		return tok
	}
	name, value, valued := strings.Cut(tok, "=")
	if kube, ok := kubeFlags[name]; ok {
		name = kube
	}
	if !valued {
		return name
	}
	parts := strings.Split(value, ",")
	for i, p := range parts {
		key, amount, ok := strings.Cut(p, "=")
		switch {
		case ok && key == "food":
			parts[i] = "cpu=" + amount + "m"
		case ok && key == "housing":
			parts[i] = "memory=" + amount + "Mi"
		case ok && key == "cellar":
			parts[i] = "ephemeral-storage=" + amount + "Mi"
		}
	}
	return name + "=" + strings.Join(parts, ",")
}

var (
	untoleratedTaint = regexp.MustCompile(`(\d+) node\(s\) had untolerated taint \{([^:{}]+):[^{}]*\}`)
	wardNames        = map[string]string{
		"node-role.kubernetes.io/control-plane": "the Royal Ward", "node.kubernetes.io/not-ready": "the Unready Ward",
		"node.kubernetes.io/unreachable": "the Storm Ward", "node.kubernetes.io/disk-pressure": "the Cellar Ward",
		"node.kubernetes.io/memory-pressure": "the Crowding Ward", "node.kubernetes.io/pid-pressure": "the Throng Ward",
		"node.kubernetes.io/unschedulable": "the Cordon Ward", "node.kubernetes.io/network-unavailable": "the Fog Ward",
	}
	wardRun     = regexp.MustCompile(`(behind the \w+ Ward), (\d+) Keeps? behind`)
	nodeReason  = regexp.MustCompile(`(\d+) node\(s\) ([^,.\n]+)`)
	shortReason = regexp.MustCompile(`(\d+) Insufficient (cpu|memory|ephemeral-storage)\b`)
	notHelpful  = regexp.MustCompile(`(\d+) Preemption is not helpful for scheduling`)
	nodeReasons = map[string][2]string{
		"were unschedulable":                                                {"was cordoned", "were cordoned"},
		"had untolerated taint(s)":                                          {"behind a Ward", "behind Wards"},
		"didn't match Pod's node affinity/selector":                         {"is not the hut's Keep choice", "are not the hut's Keep choice"},
		"didn't satisfy plugin(s) [NodeAffinity]":                           {"is not the hut's Keep choice", "are not the hut's Keep choice"},
		"didn't have free ports for the requested pod ports":                {"has its door taken", "have their doors taken"},
		"didn't match pod topology spread constraints":                      {"would tip the spread across the isles", "would tip the spread across the isles"},
		"didn't find available persistent volumes to bind":                  {"has no free Vault for its Deed", "have no free Vault for its Deed"},
		"didn't match PersistentVolume's node affinity":                     {"cannot reach the hut's Vault", "cannot reach the hut's Vault"},
		"unavailable due to one or more pvc(s) bound to non-existent pv(s)": {"waits on a Deed to a Vault that is gone", "wait on a Deed to a Vault that is gone"},
	}
	shortOf = map[string]string{"cpu": "food", "memory": "housing", "ephemeral-storage": "ground for cellars"}
)

func keeps(n string) string {
	if n == "1" {
		return n + " Keep"
	}
	return n + " Keeps"
}

func storyWards(text string) string {
	text = untoleratedTaint.ReplaceAllStringFunc(text, func(m string) string {
		parts := untoleratedTaint.FindStringSubmatch(m)
		return keeps(parts[1]) + " behind " + WardName(parts[2])
	})
	for run := wardRun.ReplaceAllString(text, "$1, $2 behind"); run != text; run = wardRun.ReplaceAllString(text, "$1, $2 behind") {
		text = run
	}
	text = nodeReason.ReplaceAllStringFunc(text, func(m string) string {
		parts := nodeReason.FindStringSubmatch(m)
		phrase, ok := nodeReasons[parts[2]]
		if !ok {
			return m
		}
		if parts[1] == "1" {
			return keeps(parts[1]) + " " + phrase[0]
		}
		return keeps(parts[1]) + " " + phrase[1]
	})
	text = shortReason.ReplaceAllStringFunc(text, func(m string) string {
		parts := shortReason.FindStringSubmatch(m)
		return keeps(parts[1]) + " short of " + shortOf[parts[2]]
	})
	return notHelpful.ReplaceAllStringFunc(text, func(m string) string {
		n := notHelpful.FindStringSubmatch(m)[1]
		if n == "1" {
			return keeps(n) + " holds no lower Rank to displace"
		}
		return keeps(n) + " hold no lower Rank to displace"
	})
}

func WardName(key string) string {
	if ward, ok := wardNames[key]; ok {
		return ward
	}
	return "a Ward"
}
