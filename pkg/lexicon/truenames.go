package lexicon

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var (
	storyOnly  = map[string]bool{"Isle": true, "Drain": true, "Spread": true}
	plainWords = map[string]bool{
		"Scroll": true, "Scrolls": true, "Quest": true, "Quests": true, "Ritual": true, "Rituals": true, "Charm": true,
		"Charms": true, "Ward": true, "Wards": true, "Binding": true, "Bindings": true, "Banner": true, "Banners": true,
		"Storm": true, "Storms": true, "Shrine": true, "Shrines": true,
		"Vows": true, "Rank": true, "Ranks": true, "Hand": true, "Horn": true, "Horns": true,
	}
)

var trueTable = map[string]string{
	"hut at the door": "static Pod", "Hut at the door": "static Pod", "hut at the Lone Tower's door": "static Pod", "Hut at the Lone Tower's door": "static Pod", "LAMPLIGHTERS": "LANTERN PODS", "REFORGE KEEPS": "REBUILD KEEPS", "REFORGE LOSES": "REBUILD LOSES", "REFORGE": "REBUILD", "huts at the door": "static Pods", "Huts at the door": "static Pods", "mirror hut": "mirror Pod", "Mirror hut": "mirror Pod",
	"Keep's Door": "hostPort", "Lone Tower": "control-plane Node", "Keep size": "instance type",
	"Harbor gate": "entrance", "Harbor gates": "entrances", "harbor gate": "entrance", "harbor gates": "entrances",
	"Signpost": "Ingress", "Signposts": "Ingresses", "signpost": "Ingress", "signposts": "Ingresses",
	"Foreign hall": "custom resource kind", "Foreign halls": "custom resource kinds", "foreign hall": "custom resource kind", "foreign halls": "custom resource kinds",
	"Citadel across the sea": "managed control plane", "Heraldry view": "Namespace view", "HERALDRY": "NAMESPACES", "Crest": "namespace mark", "Crests": "namespace marks",
	"crest": "namespace mark", "crests": "namespace marks", "Pennant": "namespace flag", "Pennants": "namespace flags", "pennant": "namespace flag", "pennants": "namespace flags",
	"Stone Keep": "xlarge Node", "Stone Keeps": "xlarge Nodes", "Great Keep": "2xlarge Node",
	"Great Keeps": "2xlarge Nodes", "Citadel Keep": "4xlarge Node", "Citadel Keeps": "4xlarge Nodes",
	"Stewards' Hall": "kube-controller-manager", "Speaking Stone": "kubectl",
	"War Horn": "HorizontalPodAutoscaler", "War Horns": "HorizontalPodAutoscalers",
	"Horn's Pitch": "target utilization", "Horn's Memory": "stabilization window", "Horn's Ceiling": "maxReplicas",
	"Muster Floor and Ceiling": "minReplicas and maxReplicas",
	"Labor Pledge":             "CPU request", "Lore Pledge": "memory request", "Labor Binding": "CPU limit", "Lore Binding": "memory limit",
	"Food Pledge": "CPU request", "Housing Pledge": "memory request", "Food Binding": "CPU limit", "Housing Binding": "memory limit",
	"Guild Roll": "EndpointSlice", "Guild Rolls": "EndpointSlices", "roll": "EndpointSlice", "rolls": "EndpointSlices", "hand roll": "hand-kept EndpointSlice",
	"Royal Ward": "control-plane taint", "Fit for Duty": "Ready", "Reeve's Candle": "node lease", "Wayfinder's Post": "kube-proxy",
	"Sky Barons": "cloud provider", "Sky Envoy": "cloud-controller-manager", "Weary Hands": "CPU throttling",
	"guttering hut": "CrashLoopBackOff", "Guttering hut": "CrashLoopBackOff", "guttering huts": "CrashLoopBackOff Pods", "Guttering huts": "CrashLoopBackOff Pods", "Last Rites": "Terminating",
	"Quest Fulfilled": "Completed", "The Graduate Never Arrived": "ImagePullBackOff",
	"Diploma seal": "image digest", "Academy seal": "imagePullSecret", "Plot of Land": "machine",
	"lantern hut": "lantern Pod", "Lantern hut": "lantern Pod", "lantern huts": "lantern Pods", "Lantern huts": "lantern Pods", "Lantern Levies": "lantern Pods",
	"Chronicle of Woe": "postmortem", "Chronicles of Woe": "postmortems", "The Nines": "The SLI", "the Nines": "the SLI", "Nines": "SLI", "the Nine": "the SLO",
	"One Nine": "a 90% SLO", "Two Nines": "a 99% SLO", "Three Nines": "a 99.9% SLO", "Scrying Wisp": "ephemeral container",
	"Knightly Order": "StatefulSet", "Sealed Scroll": "Secret", "Garrison Charter": "PodDisruptionBudget",
	"Levy": "Pod", "Levies": "Pods", "Keep": "Node", "Keeps": "Nodes",
	"Muster": "ReplicaSet", "Musters": "ReplicaSets", "Guildhall": "Service", "Guildhalls": "Services",
	"Chancery": "kube-apiserver", "Codex": "etcd", "Quartermaster": "kube-scheduler", "Stewards": "controllers",
	"Steward": "controller", "Reeve": "kubelet", "Wayfinder": "kube-proxy", "Herald": "CoreDNS",
	"Observatory": "metrics-server", "Augury": "metrics-server", "Augur": "metrics-server", "Omens": "metrics",
	"Horn": "HPA", "Horns": "HPAs", "Pledge": "request", "Pledges": "requests", "Binding": "limit", "Bindings": "limits",
	"Labor": "CPU", "Lore": "memory", "Hand": "CPU", "Hands": "CPUs", "Food": "CPU", "food": "CPU", "Housing": "memory", "housing": "memory",
	"rations": "millicores", "ration": "millicore", "hungry": "throttled", "Hungry": "Throttled", "Hunger": "CPU throttling", "hunger": "CPU throttling", "Scriptorium": "YAML editor",
	"Academy": "registry", "Soulvault": "registry", "Soulstone": "image", "Soulstones": "images",
	"Lamplighter": "lantern container", "Lamplighters": "lantern containers",
	"Unhoused": "Pending", "Guttering": "CrashLoopBackOff", "Banished": "Evicted",
	"Voidtaken": "OOMKilled", "Kindling": "ContainerCreating", "Burning": "Running", "Snuffed": "Error",
	"Ward": "taint", "Wards": "taints", "Charm": "toleration", "Charms": "tolerations",
	"Marginalia": "annotations", "Familiar": "sidecar", "Familiars": "sidecars", "Squire": "init container",
	"Squires": "init containers", "Storm": "network partition", "Storms": "network partitions", "Citadel": "control plane",
	"Banner": "Deployment", "Banners": "Deployments", "Shrine": "DaemonSet", "Shrines": "DaemonSets", "Quest": "Job",
	"Quests": "Jobs", "Ritual": "CronJob", "Rituals": "CronJobs", "Scroll": "ConfigMap", "Scrolls": "ConfigMaps",
	"Writ": "ServiceAccount", "Writs": "ServiceAccounts", "writ": "ServiceAccount", "writs": "ServiceAccounts",
	"Fief": "Namespace", "Fiefs": "Namespaces", "Warband": "PodGroup", "Vows": "finalizers", "Fealty": "ownerReferences",
	"Cellar Pledge": "ephemeral-storage request", "Cellar Pledges": "ephemeral-storage requests", "Cellar Binding": "ephemeral-storage limit", "Cellar Bindings": "ephemeral-storage limits", "Cellar Ward": "DiskPressure taint", "Cellar": "ephemeral storage", "Cellars": "ephemeral storage", "cellar": "ephemeral storage", "cellars": "ephemeral storage", "the Keep's ground": "the node's disk", "Keep's ground": "node's disk", "the Keep's common store": "the image filesystem", "Palimpsest": "compaction", "Rebinding the Codex": "etcd defragmentation",
	"Rebind the Codex": "Defragment etcd", "REBIND THE CODEX": "DEFRAGMENT ETCD", "Codex Bloat": "etcd space exhaustion",
	"Rank and Displacement": "PriorityClass and preemption", "Displacement": "preemption", "displaced": "preempted",
	"Vagrant": "BestEffort Pod", "Freeman": "Burstable", "Rank": "PriorityClass", "Ranks": "PriorityClasses",
	"Quorum": "PodDisruptionBudget", "Quorums": "PodDisruptionBudgets",
	"pledged": "requested", "pledge": "request", "pledges": "requests", "binding": "limit", "bindings": "limits",
	"guttering": "crash-looping", "banished": "evicted", "familiar": "sidecar", "familiars": "sidecars",
	"Raven": "log-shipper sidecar", "Ravens": "log-shipper sidecars", "Hound": "proxy sidecar", "Hounds": "proxy sidecars",
	"Endow": "Buy", "endow": "buy", "class of": "tag", "the latest class": "tag latest", "classes of graduates": "image tags", "graduate": "image",
	"graduates": "images", "Graduate": "Image", "Graduates": "Images", "Graduates lost at sea": "Image pull failed",
	"the graduate never arrived": "ImagePullBackOff", "Tower": "control-plane Node", "Reeves": "kubelets",
	"Omen": "Metric", "omen": "metric", "omens": "metrics", "marks": "labels",
	"Great Pilgrimage": "load test", "Pilgrimage": "load test", "Siege": "Load test", "Sieges": "Load tests",
	"SIEGE": "LOAD TEST", "siege": "load test", "Conclave Seals": "load-test badges", "Conclave Seal": "load-test badge",
	"a Conclave Seal": "a load-test badge", "Conclave": "review board", "gold": "money", "Gold": "Money", "GOLD": "MONEY",
	"Feudal Age": "Staging", "FEUDAL AGE": "STAGING", "Dark Age": "Dev", "DARK AGE": "DEV", "Castle Age": "UAT",
	"CASTLE AGE": "UAT", "Age of Empire": "Production", "The Age of Empire": "Production", "the Age of Empire": "Production", "the Feudal Age": "Staging",
	"the Dark Age": "Dev", "the Castle Age": "UAT", "Feudal": "Staging",
	"Advance to the Feudal Age": "Promote to Staging", "Advance to the Castle Age": "Promote to UAT",
	"Advance to the Age of Empire": "Promote to Production", "Age up": "Promote", "Road of Ages": "Release path",
	"Ages": "Environments", "Age": "Environment", "AGE": "PROMOTE", "Simian Imp": "chaos monkey", "the Simian Imp": "the chaos monkey",
	"The Simian Imp": "The chaos monkey", "Imp's Tremor": "chaos monkey's kill", "Imp's Tremors": "chaos monkey kills",
	"Imp": "chaos monkey", "IMP": "CHAOS MONKEY", "Reforge": "Rebuild", "Reforged": "Rebuilt", "reforge": "rebuild",
	"reforged": "rebuilt", "Loom": "Prestige tree", "the Loom": "the Prestige tree", "Tapestry": "declared state",
	"Wisdom": "Knowledge", "wisdom": "knowledge", "Golden Threads": "prestige points", "Golden Thread": "prestige point",
	"Renown": "Reputation", "renown": "reputation", "isle": "zone", "isles": "zones",
	"storm": "network partition", "storms": "network partitions", "kindle": "start", "kindles": "starts",
	"kindled": "started", "Chronicle": "postmortem", "Chronicles": "postmortems",
	"Borg's Decree": "Objective", "BORG'S DECREE": "OBJECTIVE", "Grimoire": "Handbook", "Town Crier": "production launch",
	"Town Criers": "production launches", "a Town Crier": "a production launch", "hire a Town Crier": "open production to more users",
	"Hire a Town Crier": "Open production to more users", "Hire Town Criers": "Open production to more users",
	"Crier": "launch", "Criers": "launches", "CRIERS": "LAUNCH", "silence a Crier": "pull back a launch",
	"A Town Crier falls silent": "A production launch is pulled back", "keep three Town Criers proclaiming": "keep three production launches live",
	"A Town Crier proclaims your realm at the crossroads": "Production opens to more users", "Criers at the crossroads": "Production launches",
	"Crown's decree": "compliance request", "The Crown's decree": "The compliance request", "Crown's Decrees": "compliance requests",
	"Decree of the Crown": "Compliance request", "Decrees of the Crown": "Compliance requests", "Decree": "Compliance request",
	"Decrees": "Compliance requests", "DECREE": "COMPLIANCE", "decree": "manifest", "decrees": "manifests",
	"Favor": "Goodwill", "Tribute": "Fee", "tribute": "fee", "Council": "Dashboard", "Chief Advisor": "Summary",
	"Treasurer": "Cost", "Vagrant tent": "BestEffort Pod", "Vagrant shack": "BestEffort Pod", "Vagrant shacks": "BestEffort Pods", "Freeman cottage": "Burstable Pod", "Landed manor": "Guaranteed Pod",
	"Landed manors": "Guaranteed Pods", "Vagrant tents": "BestEffort Pods", "Freeman cottages": "Burstable Pods",
	"realm": "cluster", "Realm": "Cluster", "realms": "clusters",
	"box": "request", "boxes": "requests", "Box": "Request", "Boxes": "Requests", "parcel": "web request",
	"parcels": "web requests", "Parcel": "Web request", "Parcels": "Web requests", "ledger waiting": "database request waiting", "ledgers": "database requests",
	"Ledgers": "Database requests", "crate": "upload", "crates": "uploads", "Crate": "Upload", "Crates": "Uploads",
	"chest": "large upload", "chests": "large uploads", "Chest": "Large upload", "Chests": "Large uploads",
	"dispatch": "batch job", "dispatches": "batch jobs", "Dispatch": "Batch job", "Dispatches": "Batch jobs",
	"fresh": "on-time", "Fresh": "On-time", "freshness": "timeout", "Freshness": "Timeout", "spoiled": "timed out",
	"spoils": "times out", "spoil": "time out", "Spoiled": "Timed out", "sealed": "TLS", "Sealed": "TLS", "seal": "TLS",
	"crowned": "priority", "Crowned": "Priority", "quay": "queue", "Quay": "Queue", "quays": "queues", "harbor tier": "traffic tier",
	"Harbor tier": "Traffic tier", "depot": "Service", "depots": "Services", "Depot": "Service", "Depots": "Services",
	"crew": "Deployment", "crews": "Deployments", "Crew": "Deployment", "Crews": "Deployments",
	"Porter": "porter Pod", "Porters": "porter Pods", "Hauler": "hauler Pod", "Haulers": "hauler Pods",
	"Clerk": "clerk Pod", "Clerks": "clerk Pods", "Sealwright": "sealwright Pod", "Sealwrights": "sealwright Pods",
	"Courier": "courier Pod", "Couriers": "courier Pods", "contract": "customer order", "contracts": "customer orders",
	"Contract": "Customer order", "Contracts": "Customer orders", "merchant": "customer", "merchants": "customers",
	"Merchant": "Customer", "Merchants": "Customers",
	"forest and stone": "CPU and memory", "forest": "CPU", "trees": "CPU tiles", "stones": "memory tiles",
	"stone": "memory", "border stones": "node boundary", "Keep's border stones": "Node's machine", "borders": "machine bounds",
	"Keep's land": "Node's machine", "a Keep's land": "a Node's machine", "Lone Tower's land": "control-plane Node's machine",
	"lands of two Keeps": "machines of two Nodes", "another Keep's land": "another Node's machine", "Reeve's share": "kubelet reserve",
	"balanced land": "general-purpose instance", "compute land": "compute-optimized instance", "memory land": "memory-optimized instance", "on memory land": "on a memory-optimized instance", "on compute land": "on a compute-optimized instance", "on balanced land": "on a general-purpose instance", "on rich land": "on a large instance", "rich land": "large instance", "survey": "capacity estimate", "Survey": "Capacity estimate",
	"surveyed spots": "offered instance shapes",
	"Codex-keeper":   "etcd member", "Codex-keepers": "etcd members", "spoiling": "timing out", "Spoiling": "Timing out", "SPOILING": "TIMING OUT", "TRADE": "ORDERS", "calamity": "incident", "calamities": "incidents", "Calamity": "Incident", "Calamities": "Incidents", "tide": "load test", "Tide": "Load test", "cog": "small burst", "cogs": "small bursts", "Cogs": "Small bursts", "caravel": "medium burst", "caravels": "medium bursts", "Caravels": "Medium bursts", "galleon": "large burst", "galleons": "large bursts", "Galleons": "Large bursts", "carrack": "huge burst", "carracks": "huge bursts", "Carracks": "Huge bursts", "Rush Tide": "Load test", "Rush Tides": "Load tests", "rush tide": "load test", "RUSH TIDE": "LOAD TEST",
	"Cool Holds": "longer timeouts", "the Crown": "governance", "The Crown": "Governance", "Crown": "governance",
	"worker": "container", "workers": "containers", "Worker": "container", "Workers": "containers", "door porter": "static Pod",
	"quay full": "queue full", "lost in flight": "lost when its Pod stopped",
	"Vanguard": "maxSurge", "Allowed Gaps": "maxUnavailable", "Banner Archive": "rollout history",
	"furled Banner": "paused Deployment", "Inner Hall": "ClusterIP Service", "Inner Halls": "ClusterIP Services",
	"Postern Gate": "NodePort", "Postern Gates": "NodePorts", "Letter of Referral": "ExternalName Service",
	"Hands of Labor": "CPUs", "pair of Hands": "CPU", "Kingdom": "cluster", "Tidings": "Events", "Vagrants": "BestEffort", "Freemen": "Burstable", "Landed": "Guaranteed",
	"Helmbearer": "operator", "Pilgrim": "Request", "Pilgrims": "Requests", "pilgrim": "request", "pilgrims": "requests",
	"gilded ship": "premium burst", "gilded ships": "premium bursts", "golden ship": "premium burst", "golden ships": "premium bursts",
	"gleaming box": "premium request", "gleaming boxes": "premium requests", "golden boxes": "premium requests",
	"gilded": "premium", "Gilded": "Premium", "GILDED TIDE": "PREMIUM SURGE", "Gilded Tide": "Premium surge",
	"seasoned": "at scale", "Seasoned": "At scale", "Your hands still earn": "Manual handling still pays",
	"endowed": "bought", "Endowed": "Bought", "ENDOWED": "BOUGHT", "Academy classes": "image tags", "unexamined class": "untrusted image",
	"Porter at the Door": "static Pod", "Porters at the Door": "static Pods",
	"the Stone": "kubectl", "The Stone": "kubectl", "on the Stone": "in the console", "On the Stone": "In the console",
	"the Stone Keep": "the xlarge Node", "the Stone Keeps": "the xlarge Nodes", "The Stone Keep": "The xlarge Node",
	"Seal": "badge", "Seals": "badges", "Ascension": "Fleet expansion", "ascension": "fleet expansion", "Ascend": "Expand to a fleet",
	"threads": "prestige points", "THREADS": "PRESTIGE", "weave": "spend", "Weave": "Spend",
	"Charters of the Isles": "Zone charters", "Press-Gang": "ReplicaSet adoption", "relit": "restarted",
	"Heraldic badges": "labels", "Trade": "image", "Trades": "images", "trade": "image", "trades": "images", "class": "tag", "classes": "tags", "trade, its class": "image, its tag", "the trade its image names": "the image it names", "trade:class": "image:tag", "Class": "image tag", "Badges": "labels", "Badge": "Label", "The badge it looks for": "label selector", "the badge it looks for": "label selector", "shift": "ReplicaSet", "shifts": "ReplicaSets", "Shift": "ReplicaSet", "Shifts": "ReplicaSets", "SHIFT": "REPLICASET", "hut": "Pod", "huts": "Pods", "Hut": "Pod", "Huts": "Pods",
	"tally": "ready count", "Knock down": "Delete", "speak it on the Stone": "run it in the console", "Speak it on the Stone": "Run it in the console", "knock down": "delete", "UPGRADE": "SET IMAGE", "Mark": "label", "study Mark": "debug label",
	"Soft Ward": "PreferNoSchedule taint", "Harsh Ward": "NoExecute taint", "SURVIVED": "RESOLVED", "Tidecraft": "Load-test mastery", "SUMMON": "RUN", "HIRE": "RUN", "HIRE (free)": "RUN", "HIRE BY HAND": "RUN A POD", "Hire a": "Run a", "Hire another": "Run another", "Hire one": "Run one", "Hire them": "Run them", "you hire": "you run", "Keep shifts": "Cluster API MachineSets", "Crew Floor and Ceiling": "minReplicas and maxReplicas", "the Commons": "the shared ground (no Node)", "The Commons": "The shared ground (no Node)", "Commons": "shared ground", "wait in the Forum (Pending)": "stay Pending", "wait in the Forum for land": "stay Pending until a Node has room", "will wait in the Forum": "will be Pending", "waiting in the Forum": "Pending", "wait in the Forum": "stay Pending", "the bigger ships outrun them": "the traffic outgrows them", "A Keep is on its way: they land when it joins": "A Node is joining: they get scheduled once it is Ready", "eat about": "use about", "really eat": "really use", "can feed": "can hold", "eating furthest": "using furthest", "seated more of them": "placed more of them", "can carry more ships": "can serve more traffic", "the Forum's edge": "the shared ground's edge", "the Forum": "the shared ground (no Node)", "The Forum": "The shared ground (no Node)", "Forum": "shared ground", "Tabularium": "etcd", "every worker without its own now asks": "every container without its own requests now asks", "default pledges": "default requests", "of new land": "of new capacity", "Keep land": "Node capacity", "The waiting huts move in": "The Pending Pods get scheduled", "plaque": "Service", "the Tabularium": "etcd", "Curia": "kube-apiserver", "the Curia": "the kube-apiserver", "Rostra": "Events", "EMPIRE AGE": "PRODUCTION", "a ledger": "a database request", "sealed ledger": "TLS database request", "every ledger": "every database request", "ledger work": "database request work", "CHAIN": "OWNERS", "Clerks' class": "clerk image", "Porters' class": "porter image", "Haulers' class": "hauler image", "Couriers' class": "courier image", "Sealwrights' class": "sealwright image", "The Unanswered Summons": "ImagePullBackOff", "Unanswered Summons": "ImagePullBackOff", "The Archive": "The database Service", "the Archive": "the database Service", "Archive": "database Service", "RAISE A CREW": "CREATE A DEPLOYMENT", "then RAISE": "then CREATE", "click the Stone in the": "click the kubectl stone in the", "in rooms": "in mebibytes", "He decrees": "He declares", "realms, decrees": "clusters, declares", "four halls": "four components", "every hall": "every component", "Its four halls": "Its four components", "its four halls": "its four components", "treasure chest": "bonus", "a port for": "a Gateway for", "PORTS": "GATEWAYS", "a port on the": "a Gateway on the", "Upgrade a port": "Upgrade a Gateway", "upgrade a port": "upgrade a Gateway", "or a port": "or a Gateway", "to this port": "to this Gateway", "Deeper ports": "Deeper Gateways", "deeper ports": "deeper Gateways", "deeper port": "deeper Gateway", "Click a port": "Click a Gateway", "closed port": "closed Gateway", "second port": "second Gateway", "deepest port": "deepest Gateway", "a port at level": "a Gateway at level", "A port at level": "A Gateway at level", "RESEAT": "REWRITE MANIFEST", "RETUNE": "PATCH HPA", "CLASS": "IMAGE", "WORKERS": "REPLICAS", "Its class is broken": "Its image is broken", "its class": "its image", "changes class": "changes image", "Empire Age": "Production", "the Empire Age": "Production",
	"berth": "traffic ramp", "berths": "traffic ramps", "Berth": "Traffic ramp", "Berths": "Traffic ramps", "OPEN BERTH": "RAMP UP", "CLOSE BERTH": "RAMP DOWN",
	"harbor berth": "traffic ramp", "harbor berths": "traffic ramps", "Harbor berth": "Traffic ramp", "Harbor berths": "Traffic ramps",
	"Tremor": "Chaos strike", "The Crowding": "Node-pressure eviction", "Crowding": "Node-pressure eviction", "island": "zone", "islands": "zones",
	"Island": "Zone", "Islands": "Zones", "thread": "prestige point", "CURSED": "UNTRUSTED", "Cursed": "Untrusted", "cursed": "untrusted",
	"hound": "proxy sidecar", "hounds": "proxy sidecars", "readings": "metrics",
	"Open a harbor berth": "Ramp traffic up", "A berth closes: fewer ships will come": "Traffic ramps down: fewer requests will come",
	"harbor master": "release manager", "The harbor master": "The release manager", "Gate Harbor": "gate-harbor Gateway", "the Muniment Room": "the archive StatefulSet", "The Muniment Room": "The archive StatefulSet", "Muniment Room": "archive StatefulSet", "Vault": "PersistentVolume", "Vaults": "PersistentVolumes", "vault": "PersistentVolume", "vaults": "PersistentVolumes", "Deed": "PersistentVolumeClaim", "Deeds": "PersistentVolumeClaims", "deed": "PersistentVolumeClaim", "deeds": "PersistentVolumeClaims", "shelves": "storage", "a scroll on its own shelf": "its manifest folder", "a scroll on its shelf": "its manifest folder", "the kubelet's shelf": "the kubelet's manifest folder", "Reeve's shelf": "kubelet's manifest folder", "its own shelf": "its manifest folder", "its shelf": "its manifest folder", "dig a Vault": "create a PersistentVolume", "Dig a Vault": "Create a PersistentVolume", "write a Deed": "create a PersistentVolumeClaim", "The Vault-wrights' Guild": "The vault-wrights StorageClass", "the Vault-wrights' Guild": "the vault-wrights StorageClass", "Vault-wrights": "StorageClass vault-wrights", "Stateful Crews": "StatefulSets", "Stateful Crew": "StatefulSet", "archivist": "archive Pod", "archivists": "archive Pods", "post-warden": "post endpoint", "post-wardens": "post endpoints", "Post-warden": "Post endpoint", "Post-Warden": "Post endpoint", "warden": "post Pod", "wardens": "post Pods", "The post opens": "The post Service opens", "strands its couriers": "leaves its courier Pods without a local post endpoint", "satchel": "batch", "satchels": "batches", "Daemons": "DaemonSets", "Daemon": "DaemonSet", "Place a warden there": "Run a post Pod on that Node", "Frostwater Quay": "Frostwater Quay", "Morrow Quay": "Morrow Quay", "A rumour at the Academy": "An untrusted image in the registry", "rumour": "untrusted image", "rumours": "untrusted images", "Rumour": "Untrusted image", "next class": "next tag", "new class": "new tag", "latest class": "latest tag", "last class": "last tag", "newest class": "newest tag", "same class": "same tag", "owned class": "pulled tag", "known class": "known tag", "leaner class": "leaner tag", "bad class": "bad tag", "shown class": "shown tag", "porter class": "porter tag", "Better classes": "Newer tags", "land tier": "node price tier", "Land tier": "Node price tier", "carry the seals": "carry the TLS requests", "harbor ladder": "traffic ladder", "gutters": "is in CrashLoopBackOff", "relights and falls again and again": "restarts again and again", "its worker stops and is relit again and again": "it restarts again and again", "their workers stop and are relit again and again": "they restart again and again", "guttered": "crashed", "the class it runs": "the image it runs", "a class that does not crash": "an image that does not crash", "in the camp": "Pending", "never housed": "never scheduled", "Mend it": "Fix it", "still unpacks its tools": "still warms up", "unpacks its tools": "warms up", "its shutters are open": "it is Ready", "its shutters are closed": "it is not Ready", "freezes": "hangs", "frozen": "hung", "the Watch": "the probes", "The Watch": "The probes", "the Shutters": "the readinessProbe", "The Shutters": "The readinessProbe", "the Watchman's Bell": "the livenessProbe", "The Watchman's Bell": "The livenessProbe", "the watchman": "the liveness probe", "The watchman": "The liveness probe", "and ring after": "and restart after",
	"raze": "delete", "razes": "deletes", "razed": "deleted", "razing": "deleting", "Raze": "Delete", "Razed": "Deleted",
	"relight": "restart", "relights": "restarts", "relighting": "restarting", "Relight": "Restart",
	"shutters": "readiness probes", "Shutters": "Readiness probes", "shutter": "readiness probe",
	"stopped answering the watchman's bell and will be relit": "failed liveness probe, will be restarted",
	"the watchman's bell": "the liveness probe", "The watchman's bell": "The liveness probe",
	"watchman's bell": "liveness probe", "Watchman's Bell": "Liveness probe", "watchman": "liveness probe", "Watchman": "Liveness probe", "bell": "liveness probe", "Bell": "Liveness probe", "BELL": "LIVENESS",
	"Charter of Keeps": "Nodes", "Masonry": "Instance types", "The Vaults": "PersistentVolumes", "The Scriptorium": "YAML editor",
	"Ceiling": "maxReplicas", "Pitch": "target utilization", "toppled": "killed", "warded": "tainted", "Warded": "Tainted", "lit the candle": "reported the Node Ready", "lights its candle": "reports the Node Ready", "lights the candle": "reports the Node Ready", "turned out": "evicted",
	"rune by rune": "field by field", "rune": "field", "runes": "fields", "reads its scroll": "reads its manifest", "re-reads its scroll": "re-reads its manifest", "the Scrolls": "the ConfigMaps", "The Scrolls": "The ConfigMaps", "fief": "namespace", "fiefs": "namespaces",
	"badge": "label", "badges": "labels", "rooms": "memory",
	"ships": "request bursts", "ship": "request burst", "sails in": "arrives", "sailing in": "arriving", "sail elsewhere": "go elsewhere",
	"sail in": "arrive", "docks": "arrives", "merchant's charter": "customer order", "get keeps": "get nodes", "get keep": "get node", "GROUND": "DISK",
	"caravel masters": "medium bursts", "galleon masters": "large bursts", "carrack masters": "huge bursts", "cog masters": "small bursts",
	"Masons' Guild": "cloud provider", "chronicle line to the rookery": "log line to the log store", "chronicle lines to the rookery": "log lines to the log store", "rookery": "log store",
}

var (
	trueWords     = buildTrueWords()
	truePattern   = compileTrue(trueWords)
	parenthesis   = regexp.MustCompile(` \(([^()]+)\)`)
	articles      = []string{"a ", "an ", "the "}
	bareNames     = map[string]bool{"kubectl": true, "etcd": true, "coredns": true, "dev": true, "staging": true, "uat": true, "production": true, "xlarge": true}
	trueValues    = valuesOf(trueTable)
	spokenLetters = map[string]bool{"HPA": true, "HPAs": true, "HPAS": true, "SLO": true}
	tradeNames    = map[string]bool{"porter": true, "clerk": true, "hauler": true, "courier": true, "sealwright": true, "lantern": true}
	statusTerms   = map[string]bool{"Pending": true, "CrashLoopBackOff": true, "ImagePullBackOff": true, "OOMKilled": true, "Evicted": true, "DiskPressure": true, "Terminating": true, "ContainerCreating": true}
)

func buildTrueWords() map[string]string {
	out := make(map[string]string, len(trueTable)*2)
	keys := make([]string, 0, len(trueTable))
	for from, to := range trueTable {
		out[from] = to
		keys = append(keys, from)
	}
	sort.Strings(keys)
	for _, from := range keys {
		upper := strings.ToUpper(from)
		if _, taken := out[upper]; taken || plainWords[from] || plainWords[capitalized(from)] {
			continue
		}
		out[upper] = strings.ToUpper(trueTable[from])
	}
	return out
}

func compileTrue(words map[string]string) *regexp.Regexp {
	keys := make([]string, 0, len(words))
	for k := range words {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) != len(keys[j]) {
			return len(keys[i]) > len(keys[j])
		}
		return keys[i] < keys[j]
	})
	quoted := make([]string, len(keys))
	for i, k := range keys {
		quoted[i] = regexp.QuoteMeta(k)
	}
	return regexp.MustCompile(`\b(?:` + strings.Join(quoted, "|") + `)\b`)
}

var hiredCount = regexp.MustCompile(`\bhired (\d)`)

var nestedGloss = regexp.MustCompile(` \(([^()]+) \(([^()]+)\)\)`)

var (
	rationCount = regexp.MustCompile(`\b(\d[\d,]*) rations?(?: of (?:food|Food|Labor))?\b`)
	roomCount   = regexp.MustCompile(`\b(\d[\d,]*) rooms?(?: of (?:housing|Housing|Lore))?\b`)
	shelfCount  = regexp.MustCompile(`\b(\d[\d,]*) (?:shelf|shelves)\b`)
	barrelCount = regexp.MustCompile(`\b(\d[\d,]*) barrels?(?: of (?:cellar|Cellar))?\b`)
)

func unitNames(text string) string {
	text = rationCount.ReplaceAllStringFunc(text, func(m string) string {
		n := countIn(m)
		if n > 0 && n%1000 == 0 {
			return plural(n/1000, "CPU", "CPUs")
		}
		return strconv.Itoa(n) + "m of CPU"
	})
	text = shelfCount.ReplaceAllStringFunc(text, func(m string) string { return strconv.Itoa(countIn(m)) + " Gi" })
	text = barrelCount.ReplaceAllStringFunc(text, func(m string) string { return mebiNames(countIn(m), "ephemeral storage") })
	return roomCount.ReplaceAllStringFunc(text, func(m string) string { return mebiNames(countIn(m), "memory") })
}

func mebiNames(n int, of string) string {
	if n > 0 && n%1024 == 0 {
		return strconv.Itoa(n/1024) + "Gi of " + of
	}
	return strconv.Itoa(n) + "Mi of " + of
}

func countIn(match string) int {
	digits, _, _ := strings.Cut(match, " ")
	n, _ := strconv.Atoi(strings.ReplaceAll(digits, ",", ""))
	return n
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}

var money = regexp.MustCompile(`([+-]?)((?:\d[\d.,]*[KMBGTPE]?)|(?:%[-+ #0-9.]*[a-z])) gold\b`)

var depotTitle = regexp.MustCompile(`\b(Parcels|PARCELS|Ledgers|LEDGERS|Crates|CRATES|Seals|SEALS|Dispatches|DISPATCHES) (?:Depot|DEPOT)\b`)

func depotTitles(text string) string {
	return depotTitle.ReplaceAllStringFunc(text, func(m string) string {
		name, depot, _ := strings.Cut(m, " ")
		if depot == "DEPOT" {
			return strings.ToLower(name) + " SERVICE"
		}
		return strings.ToLower(name) + " Service"
	})
}

func TrueNames(text string) string {
	return keepingNames(text, trueNames)
}

func trueNames(text string) string {
	if text == "" {
		return text
	}
	text = depotTitles(text)
	text = money.ReplaceAllString(text, "$1$$$2")
	text = hiredCount.ReplaceAllString(text, "run $1")
	text = unitNames(text)
	var b strings.Builder
	last := 0
	for _, m := range truePattern.FindAllStringIndex(text, -1) {
		word := text[m[0]:m[1]]
		if KeepAsVerb(text, m[0], m[1], word) || rollAsVerb(text, m[0], m[1], word) || scrollAsVerb(text, m[1], word) || shipAsVerb(text, m[1], word) || TradeAsCommerce(text, m[0], m[1], word) || OtherClass(text, m[0], m[1], word) || inIdentifier(text, m[0], m[1]) || keyChord(text, m[1], word) || ServiceName(text, m[0], m[1], word) {
			continue
		}
		to, gap := classOf(text, m[1], word), text[last:m[0]]
		if alreadyNamed(text[m[1]:], word, to) {
			continue
		}
		to = openSentence(text, m[0], word, to)
		if fixed := fitArticle(b.String()+gap, to); len(fixed) >= b.Len() {
			gap = fixed[b.Len():]
		}
		if repeatsBefore(b.String()+gap, to) {
			b.WriteString(strings.TrimRight(gap, " "))
			last = m[1]
			continue
		}
		b.WriteString(gap)
		b.WriteString(to)
		last = m[1]
		if extra, n := agreeing(to, text[m[1]:]); n > 0 {
			b.WriteString(extra)
			last = m[1] + n
		}
	}
	b.WriteString(text[last:])
	return collapse(sloEcho.ReplaceAllStringFunc(b.String(), dropSLOEcho))
}

var sloEcho = regexp.MustCompile(`(\d+(?:\.\d+)?)% SLO \((\d+(?:\.\d+)?)%\)`)

func dropSLOEcho(m string) string {
	parts := sloEcho.FindStringSubmatch(m)
	if parts[1] != parts[2] {
		return m
	}
	return parts[1] + "% SLO"
}

var pluralToSingular = map[string]string{
	"are": "is", "have": "has", "hold": "holds", "ask": "asks", "fall": "falls", "count": "counts", "show": "shows", "stay": "stays",
	"drop": "drops", "dip": "dips", "read": "reads", "measure": "measures", "need": "needs", "wait": "waits", "do": "does",
	"say": "says", "tell": "tells", "rise": "rises", "miss": "misses", "grade": "grades", "rent": "rents", "charter": "charters",
	"offer": "offers", "buy": "buys", "sell": "sells", "lend": "lends", "give": "gives", "own": "owns", "decide": "decides", "judge": "judges",
}

var singularSubjects = map[string]bool{"SLI": true, "The SLI": true, "the SLI": true, "SLO": true, "a 90% SLO": true, "a 99% SLO": true, "a 99.9% SLO": true, "cloud provider": true, "the cloud provider": true, "The cloud provider": true}

func agreeing(to, rest string) (string, int) {
	if !singularSubjects[to] {
		return "", 0
	}
	skip := 0
	if strings.HasPrefix(rest, "]]") {
		skip = 2
	}
	for _, adverb := range []string{" now", " still", " also", " always"} {
		if strings.HasPrefix(rest[skip:], adverb+" ") {
			skip += len(adverb)
			break
		}
	}
	verb := wordAfter(rest, skip)
	one, ok := pluralToSingular[verb]
	if !ok {
		return "", 0
	}
	return rest[:skip+1] + one, skip + 1 + len(verb)
}

func repeatsBefore(before, to string) bool {
	if strings.ContainsRune(to, ' ') || len(to) < 3 {
		return false
	}
	return strings.EqualFold(wordBefore(before, len(before)), to)
}

var cargoNames = map[string]bool{"parcels": true, "ledgers": true, "crates": true, "seals": true, "dispatches": true}

func ServiceName(text string, start, end int, word string) bool {
	if !cargoNames[strings.ToLower(word)] {
		return false
	}
	switch strings.ToLower(wordAfter(text, end)) {
	case "depot", "depots", "service", "services":
		return true
	}
	switch strings.ToLower(wordBefore(text, start)) {
	case "named", "name", "service", "services", "svc":
		return true
	}
	if strings.HasSuffix(strings.TrimRight(text[:start], " "), "service-name:") {
		return true
	}
	return start > 0 && text[start-1] == '='
}

func fitArticle(before, to string) string {
	head, article := splitArticle(before)
	switch {
	case article == "":
		return before
	case strings.EqualFold(article, "the"):
		first, _, _ := strings.Cut(to, " ")
		if bareNames[strings.ToLower(to)] || statusTerms[first] {
			return head
		}
		return before
	case vowelSound(to):
		return head + matchCase(article, "an") + " "
	}
	return head + matchCase(article, "a") + " "
}

func splitArticle(before string) (string, string) {
	if !strings.HasSuffix(before, " ") {
		return before, ""
	}
	trimmed := before[:len(before)-1]
	cut := strings.LastIndexAny(trimmed, " \n(") + 1
	word := trimmed[cut:]
	for _, a := range []string{"a", "an", "the"} {
		if strings.EqualFold(word, a) {
			return trimmed[:cut], word
		}
	}
	return before, ""
}

func matchCase(model, word string) string {
	if model != "" && unicode.IsUpper(rune(model[0])) {
		return strings.ToUpper(word[:1]) + word[1:]
	}
	return word
}

func vowelSound(word string) bool {
	if word == "" {
		return false
	}
	first, _, _ := strings.Cut(word, " ")
	if first == "xlarge" {
		return true
	}
	if spokenLetters[first] {
		return strings.ContainsRune("AEFHILMNORSX", rune(word[0]))
	}
	return strings.ContainsRune("aeiouAEIOU", rune(word[0]))
}

var verbObjects = map[string]bool{
	"the": true, "a": true, "an": true, "your": true, "its": true, "their": true, "our": true, "every": true, "each": true, "all": true,
	"any": true, "some": true, "both": true, "no": true, "one": true, "two": true, "three": true, "four": true, "five": true, "six": true,
	"seven": true, "eight": true, "nine": true, "ten": true, "them": true, "it": true, "this": true, "that": true, "these": true,
	"those": true, "what": true, "more": true, "up": true,
}

var nounLeaders = map[string]bool{
	"a": true, "an": true, "the": true, "each": true, "every": true, "whatever": true, "whichever": true, "which": true, "another": true,
	"any": true, "its": true, "your": true, "their": true, "our": true, "per": true, "new": true, "other": true, "same": true, "of": true,
	"this": true, "that": true, "no": true, "one": true, "two": true, "three": true, "second": true, "first": true, "on": true, "in": true,
}

func KeepAsVerb(text string, start, end int, word string) bool {
	plural := false
	switch word {
	case "Keeps", "KEEPS":
		plural = true
	case "Keep", "KEEP":
	default:
		return false
	}
	next := strings.ToLower(wordAfter(text, end))
	if next == "" || nounLeaders[strings.ToLower(wordBefore(text, start))] {
		return false
	}
	if !plural && !sentenceStart(text, start) && strings.ToLower(wordBefore(text, start)) != "to" {
		return false
	}
	if verbObjects[next] || unicode.IsDigit(rune(next[0])) {
		return true
	}
	return sentenceStart(text, start) && strings.HasSuffix(next, "ing")
}

var scrollMoves = map[string]bool{"the": true, "up": true, "down": true, "back": true, "to": true, "it": true}

var (
	rollHolders = map[string]bool{"hand": true, "hand-kept": true, "own": true}
	rollMoves   = map[string]bool{"out": true, "over": true, "back": true}
)

func rollAsVerb(text string, start, end int, word string) bool {
	switch word {
	case "roll", "rolls":
	default:
		return false
	}
	if end < len(text) && text[end] == '-' || rollMoves[strings.ToLower(wordAfter(text, end))] {
		return true
	}
	before := wordBefore(text, start)
	return !nounLeaders[strings.ToLower(before)] && !rollHolders[before] && !strings.HasSuffix(before, "'s")
}

func scrollAsVerb(text string, end int, word string) bool {
	switch word {
	case "Scroll", "SCROLL":
		return scrollMoves[strings.ToLower(wordAfter(text, end))]
	}
	return false
}

var shipObjects = map[string]bool{"the": true, "its": true, "their": true, "every": true, "each": true}

func shipAsVerb(text string, end int, word string) bool {
	switch word {
	case "ship", "ships", "Ship", "Ships":
		return shipObjects[strings.ToLower(wordAfter(text, end))]
	}
	return false
}

var classOwners = map[string]bool{"qos": true, "quality": true, "storage": true, "priority": true, "ingress": true, "gateway": true,
	"runtime": true, "device": true, "volume": true, "instance": true}

var classValues = map[string]bool{"nginx": true, "besteffort": true, "burstable": true, "guaranteed": true}

func OtherClass(text string, start, end int, word string) bool {
	if word != "class" && word != "classes" {
		return false
	}
	return classOwners[strings.ToLower(wordBefore(text, start))] || classValues[strings.ToLower(wordAfter(text, end))] ||
		strings.HasPrefix(text[end:], "=") || start > 0 && text[start-1] == '('
}

var commerceNeighbours = map[string]bool{"route": true, "routes": true, "your": true, "more": true}

func TradeAsCommerce(text string, start, end int, word string) bool {
	return word == "trade" && (commerceNeighbours[wordAfter(text, end)] || commerceNeighbours[strings.ToLower(wordBefore(text, start))])
}

func classOf(text string, end int, word string) string {
	if next := wordAfter(text, end); word == "class of" && next != "" && !unicode.IsDigit(rune(next[0])) {
		return "tag of"
	}
	return trueWords[word]
}

func wordAfter(text string, end int) string {
	if end >= len(text) || text[end] != ' ' {
		return ""
	}
	rest := text[end+1:]
	stop := strings.IndexFunc(rest, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '\'' })
	if stop < 0 {
		return rest
	}
	return rest[:stop]
}

func wordBefore(text string, start int) string {
	head := strings.TrimRight(text[:start], " ")
	cut := strings.LastIndexFunc(head, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '\'' }) + 1
	return head[cut:]
}

func sentenceStart(text string, at int) bool {
	head := strings.TrimRight(text[:at], " ")
	return head == "" || strings.HasSuffix(head, ".") || strings.HasSuffix(head, ":") || strings.HasSuffix(head, "\n") ||
		strings.HasSuffix(head, "!") || strings.HasSuffix(head, "?")
}

func collapse(text string) string {
	text = unnest(text)
	matches := parenthesis.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text
	}
	var b strings.Builder
	last := 0
	for _, m := range matches {
		inner := text[m[2]:m[3]]
		if head, rest, ok := strings.Cut(inner, ": "); ok && echoes(text[:m[0]], head) {
			b.WriteString(text[last:m[0]] + " (" + rest + ")")
			last = m[1]
			continue
		}
		if !echoes(text[:m[0]], inner) && !restates(text[:m[0]], inner) && !echoesAlternative(text[:m[0]], inner) {
			continue
		}
		b.WriteString(text[last:m[0]])
		last = m[1]
	}
	b.WriteString(text[last:])
	return b.String()
}

func echoes(before, inner string) bool {
	inner = strings.ToLower(strings.TrimSpace(inner))
	for _, a := range articles {
		inner = strings.TrimPrefix(inner, a)
	}
	words := strings.Fields(inner)
	if len(words) == 0 {
		return false
	}
	prior := strings.Fields(strings.ToLower(before))
	if len(prior) < len(words) {
		return false
	}
	tail := strings.Join(prior[len(prior)-len(words):], " ")
	return singular(tail) == singular(strings.Join(words, " "))
}

func singular(s string) string {
	return strings.TrimSuffix(strings.TrimRight(s, ".,:;"), "s")
}

func valuesOf(table map[string]string) map[string]bool {
	out := make(map[string]bool, len(table))
	for _, to := range table {
		out[strings.ToLower(to)] = true
	}
	return out
}

func restates(before, inner string) bool {
	term := strings.ToLower(strings.TrimSpace(inner))
	for _, a := range articles {
		term = strings.TrimPrefix(term, a)
	}
	if !trueValues[term] && !trueValues[singular(term)] {
		return false
	}
	padded := " " + strings.ToLower(strings.NewReplacer(",", " ", ":", " ", "(", " ", ")", " ", ".", " ").Replace(before)) + " "
	return strings.Contains(padded, " "+term+" ") || strings.Contains(padded, " "+singular(term)+" ") ||
		strings.Contains(padded, " "+singular(term)+"s ")
}

func capitalized(word string) string {
	if word == "" {
		return word
	}
	return strings.ToUpper(word[:1]) + word[1:]
}

func inIdentifier(text string, start, end int) bool {
	joins := "-_="
	if start > 1 && text[start-1] == '/' && isPathChar(text[start-2]) {
		return true
	}
	return (start > 0 && strings.IndexByte(joins, text[start-1]) >= 0) || (end < len(text) && strings.IndexByte(joins, text[end]) >= 0)
}

func isPathChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.'
}

func alreadyNamed(rest, word, to string) bool {
	if len(to) <= len(word) || !strings.EqualFold(to[:len(word)], word) {
		return false
	}
	tail := to[len(word):]
	return len(rest) >= len(tail) && strings.EqualFold(rest[:len(tail)], tail)
}

func openSentence(text string, at int, word, to string) string {
	if to == "" || !unicode.IsUpper(rune(word[0])) || !unicode.IsLower(rune(to[0])) || identifierLike(to) {
		return to
	}
	if !fullStop(text, at) {
		return to
	}
	return capitalized(to)
}

func identifierLike(to string) bool {
	first, _, _ := strings.Cut(to, " ")
	return bareNames[strings.ToLower(to)] || bareNames[first] || strings.Contains(first, "-") || tradeNames[first]
}

func fullStop(text string, at int) bool {
	head := strings.TrimRight(text[:at], " ")
	return head == "" || strings.HasSuffix(head, ".") || strings.HasSuffix(head, "!") || strings.HasSuffix(head, "?") || strings.HasSuffix(head, "\n")
}

func unnest(text string) string {
	var b strings.Builder
	last := 0
	for _, m := range nestedGloss.FindAllStringSubmatchIndex(text, -1) {
		if !echoes(text[:m[0]], text[m[2]:m[3]]) {
			continue
		}
		b.WriteString(text[last:m[0]] + " (" + text[m[4]:m[5]] + ")")
		last = m[1]
	}
	b.WriteString(text[last:])
	return b.String()
}

func echoesAlternative(before, inner string) bool {
	parts := strings.Split(inner, " / ")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if echoes(before, part) {
			return true
		}
	}
	return false
}

func keyChord(text string, end int, word string) bool {
	return (word == "Shift" || word == "SHIFT") && end < len(text) && (text[end] == '+' || strings.HasPrefix(text[end:], ", Ctrl"))
}
