package lexicon

import (
	"strings"
	"testing"
)

func TestTrueNamesSpeaksKubernetes(t *testing.T) {
	cases := map[string]string{
		"Summon your first hut.":                                           "Summon your first Pod.",
		"Cordon the Lone Tower: a Keep (a Node).":                          "Cordon the control-plane Node: a Node.",
		"Restore the Codex (etcd), the realm's memory.":                    "Restore etcd, the cluster's memory.",
		"Raise the Chancery (kube-apiserver).":                             "Raise the kube-apiserver.",
		"Build a Guildhall (Service) on the plaza.":                        "Build a Service on the plaza.",
		"Build the Stewards' Hall (kube-controller-manager).":              "Build the kube-controller-manager.",
		"Keep three huts burning at once.":                                 "Keep three Pods burning at once.",
		"Done. Keep the ledger flowing.":                                   "Done. Keep the ledger flowing.",
		"A Keep never grows: drain one Stone Keep first.":                  "A Node never grows: drain one xlarge Node first.",
		"The Keep's Reeve lifts the Ward.":                                 "The Node's kubelet lifts the taint.",
		"Serve 5 parcels by hand. Parcels wait on the quay.":               "Serve 5 web requests by hand. Web requests wait on the queue.",
		"Place a hut at the door.":                                         "Place a static Pod.",
		"Set the Labor Pledge: 8 Hands and 16Gi of Lore.":                  "Set the CPU request: 8 CPUs and 16Gi of memory.",
		"Raise a Muster of at least 3 Lantern Levies.":                     "Raise a ReplicaSet of at least 3 lantern Pods.",
		"Sound a War Horn (HorizontalPodAutoscaler) on the Muster.":        "Sound a HorizontalPodAutoscaler on the ReplicaSet.",
		"Ask the Speaking Stone (kubectl) what stands.":                    "Ask kubectl what stands.",
		"HUT AT THE DOOR":                                                  "STATIC POD",
		"PORTER AT THE DOOR":                                               "STATIC POD",
		"MUSTER":                                                           "REPLICASET",
		"LAMPLIGHTERS":                                                     "LANTERN PODS",
		"EDGE SCROLL: OFF":                                                 "EDGE SCROLL: OFF",
		"The Cellar rises. Rebind the Codex before the Palimpsest.":        "The ephemeral storage rises. Defragment etcd before the compaction.",
		"Rank and Displacement: a Vagrant is displaced first.":             "PriorityClass and preemption: a BestEffort Pod is preempted first.",
		"Freeman cottages and Landed manors":                               "Burstable Pods and Guaranteed Pods",
		"3 of 4 Pods pledged; guttering Pods":                              "3 of 4 Pods requested; crash-looping Pods",
		"Bind a Raven and a Hound to the familiar slots":                   "Bind a log-shipper sidecar and a proxy sidecar to the sidecar slots",
		"Pull the class of 1.1 and summon new graduates":                   "Pull the tag 1.1 and summon new images",
		"Graduates lost at sea":                                            "Image pull failed",
		"If the Tower starves, the omens say so":                           "If the control-plane Node starves, the metrics say so",
		"BANISHED KINDLING BURNING":                                        "EVICTED CONTAINERCREATING RUNNING",
		"Heraldic badges on every Pod":                                     "Labels on every Pod",
		"The outer isle is chartered: ships reach it":                      "The outer zone is chartered: request bursts reach it",
		"The Reeves kindle it":                                             "The kubelets start it",
		"Watch the Keep's Reeve":                                           "Watch the Node's kubelet",
		"THE STEWARDS' HALL":                                               "THE KUBE-CONTROLLER-MANAGER",
		"Caravels dock now, then galleons":                                 "Medium bursts dock now, then large bursts",
		"kubectl drain stone-keep-3 and app=crew stay literal":             "kubectl drain stone-keep-3 and app=crew stay literal",
		"Too little forest here: a crew needs 1 (6 trees)":                 "Too little CPU here: a Deployment needs 1 (6 CPU tiles)",
		"build within a Keep's border stones, on a Keep's land":            "build within a Node's machine, on a Node's machine",
		"+4 Hands and 8Gi of Lore (balanced land) after the Reeve's share": "+4 CPUs and 8Gi of memory (general-purpose instance) after the kubelet reserve",
		"The Stewards watch the quay":                                      "The controllers watch the queue",
		"Cool Holds keep the quay full of boxes longer":                    "Longer timeouts keep the queue full of requests longer",
		"Porters, class of 1.1 answer the parcels depot":                   "porter Pods, tag 1.1 answer the parcels Service",
		"The Vanguard and Allowed Gaps of a furled Banner":                 "The maxSurge and maxUnavailable of a paused Deployment",
		"Only Inner Halls may be raised here":                              "Only ClusterIP Services may be raised here",
		"Deliver 120 fresh sealed crates before they spoil":                "Deliver 120 on-time TLS uploads before they time out",
		"Boxes wait on the quay at Frostwater Quay":                        "Requests wait on the queue at Frostwater Quay",
		"Open a depot for parcels and raise a crew of Porters":             "Open a Service for web requests and raise a Deployment of porter Pods",
		"A merchant sends a burst: 40 huge dispatches":                     "A customer sends a burst: 40 huge batch jobs",
		"Traffic: ledgers and crates arrive":                               "Traffic: database requests and uploads arrive",
		"Add a Keep for 150 rations; +1 ration each":                       "Add a Node for 150m of CPU; +1m of CPU each",
		"Needs %.0f rations (you have %.0f rations). FOOD":                 "Needs %.0f millicores (you have %.0f millicores). CPU",
		"Raise a Great Keep after the Stone Keep":                          "Raise a 2xlarge Node after the xlarge Node",
		"STONE KEEP, then the Great Keep":                                  "XLARGE NODE, then the 2xlarge Node",
		"The Stewards struck: the Reeve's Ward":                            "The controllers struck: the kubelet's taint",
		"Raise the realm at the Chancery with Wards":                       "Raise the cluster at the kube-apiserver with taints",
		"Survive a night of drains. Hands and Lore":                        "Survive a night of drains. CPUs and memory",
		"A decree lapsed: the Chancery waits":                              "A manifest lapsed: the kube-apiserver waits",
		"Write a decree in the Scriptorium":                                "Write a manifest in the YAML editor",
		"Ask the Council's Treasurer":                                      "Ask the Dashboard's Cost",
		"A Vagrant tent, a Freeman cottage and two Landed manors":          "A BestEffort Pod, a Burstable Pod and two Guaranteed Pods",
		"A Vagrant shack (BestEffort)":                                     "A BestEffort Pod",
		"HOW THE REALM WORKS in your realm":                                "HOW THE CLUSTER WORKS in your cluster",
		"CHANCERY PAGES":                                                   "KUBE-APISERVER PAGES",
		"Open the Chancery and call a Reeve":                               "Open the kube-apiserver and call a kubelet",
		"THE CODEX AND THE CHANCERY":                                       "ETCD AND THE KUBE-APISERVER",
		"Keep three Keeps ready.":                                          "Keep three Nodes ready.",
		"A Keep falls silent: fewer huts will land":                        "A Node falls silent: fewer Pods will land",
		"cordon a Keep, or wait":                                           "cordon a Node, or wait",
		"THE CHANCERY":                                                     "THE KUBE-APISERVER",
		"THE SPEAKING STONE":                                               "KUBECTL",
		"A Muster keeps huts burning (ReplicaSet)":                         "A ReplicaSet keeps Pods burning",
		"THE STORM BURNING":                                                "THE STORM RUNNING",
		"Put a hut at the Lone Tower's door (a static Pod).":               "Put a static Pod.",
		"Huts (Pods) and Levies (Pods)":                                    "Pods and Pods",
		"The Guildhall's huts are Unhoused.":                               "The Service's Pods are Pending.",
		"kubectl get pods -o wide":                                         "kubectl get pods -o wide",
		"Kubernites":                                                       "Kubernites",
		"":                                                                 "",
		"A War Horn sounds. An Augur reads. A Horn calls.":                 "A HorizontalPodAutoscaler sounds. A metrics-server reads. An HPA calls.",
		"Restore the Codex. The Speaking Stone awakens.":                   "Restore etcd. kubectl awakens.",
		"Add land: a Keep adds 4 Hands of Labor.":                          "Add land: a Node adds 4 CPUs.",
		"Banners (Deployments: rolling updates)":                           "Deployments (rolling updates)",
		"A large ship docks: 3 boxes wait on its quay":                     "A large request burst arrives: 3 requests wait on its queue",
		"BERTHS x3 0:42":                                                   "TRAFFIC RAMPS x3 0:42",
		"x4 SEASONED, next x2 at 25":                                       "x4 AT SCALE, next x2 at 25",
		"The parcels crew is seasoned: 10 ready huts":                      "The web requests Deployment is at scale: 10 ready Pods",
		"Your hands still help: click the pile any time.":                  "Your hands still help: click the pile any time.",
		"The big box holds 40 rations.":                                    "The big request holds 40m of CPU.",
		"PULLED; Pulled with registry classes":                             "PULLED; Pulled with registry tags",
		"Press PORTER AT THE DOOR":                                         "Press STATIC POD",
		"Speak it on the Stone. The Stone opens.":                          "Run it in the console. kubectl opens.",
		"Raise the Stone Keep beside the Stone Keeps":                      "Raise the xlarge Node beside the xlarge Nodes",
		"Three LEDGERS and a ledger; read the books":                       "Three DATABASE REQUESTS and a database request; read the books",
		"The Reeve relit it; +1.5B rations":                                "The kubelet restarted it; +1.5B millicores",
		"a chronicle line to the rookery":                                  "a log line to the log store",
		"PORTER PODS, TAG 1.0":                                             "PORTER PODS, TAG 1.0",
		"Knock down a porter hut and watch the shift":                      "Delete a porter Pod and watch the ReplicaSet",
		"One shift per revision; the newest keeps the huts.":               "One ReplicaSet per revision; the newest keeps the Pods.",
		"Lay a Soft Ward or a Harsh Ward":                                  "Lay a PreferNoSchedule taint or a NoExecute taint",
		"The post opens: couriers hand their satchels only to the post-warden on their own Keep.": "The post Service opens: couriers hand their batches only to the post endpoint on their own Node.",
		"A Keep with no warden strands its couriers.":                                             "A Node with no post Pod leaves its courier Pods without a local post endpoint.",
		"A rumour in the harbor: courier:2.1":                                                     "An untrusted image in the harbor: courier:2.1",
		"the last class: the ladder has nothing more to teach":                                    "the last tag: the ladder has nothing more to teach",
		"on their latest class, the next class waits":                                             "on their latest tag, the next tag waits",
		"land tier 2":                      "node price tier 2",
		"The carracks carry the seals":     "The huge bursts carry the TLS requests",
		"QoS class: Burstable":             "QoS class: Burstable",
		"Open the Daemons page":            "Open the DaemonSets page",
		"the Muniment Room already stands": "the archive StatefulSet already stands", "Dig a Vault of 10 shelves, then write a Deed for its archivist.": "Create a PersistentVolume of 10 Gi, then create a PersistentVolumeClaim for its archive Pod.", "Open Stateful Crews": "Open StatefulSets",
		"Click the Gate Harbor and press OPEN BERTH.":                                      "Click the gate-harbor Gateway and press RAMP UP.",
		"Frostwater Quay and Morrow Quay: the quay is full":                                "Frostwater Quay and Morrow Quay: the queue is full",
		"FROSTWATER QUAY and MORROW QUAY":                                                  "FROSTWATER QUAY and MORROW QUAY",
		"give them a Keep on memory land":                                                  "give them a Node on a memory-optimized instance",
		"give them a Keep on compute land.":                                                "give them a Node on a compute-optimized instance.",
		"The harbor master closes a berth: 2/3 berths":                                     "The release manager closes a traffic ramp: 2/3 traffic ramps",
		"Open a harbor berth":                                                              "Ramp traffic up",
		"GATE HARBOR: BERTHS":                                                              "GATE-HARBOR GATEWAY: TRAFFIC RAMPS",
		"Press SUMMON to call one hut from the harbor.":                                    "Press RUN to call one Pod from the harbor.",
		"Pledges II: Lore Pledge. Pledge honestly.":                                        "Requests II: memory request. Request honestly.",
		"The Unhoused (Pending (FailedScheduling)) struck":                                 "Pending (FailedScheduling) struck",
		"Builders begin the Keep (Node (a machine running a kubelet)).":                    "Builders begin the Node (a machine running a kubelet).",
		"The Keep is full. The Crowding (Node-pressure eviction)":                          "The Node is full. Node-pressure eviction",
		"The hound leads boxes back; charter an island":                                    "The proxy sidecar leads requests back; charter a zone",
		"too close to the Forum: a Keep's borders stop at its edge":                        "too close to the shared ground (no Node): a Node's machine bounds stop at its edge",
		"the Tabularium beside the Curia":                                                  "etcd beside the kube-apiserver",
		"You carry a sealed ledger from the quay by hand (+12 rations).":                   "You carry a TLS database request from the queue by hand (+12m of CPU).",
		"You carry a ledger from the quay by hand.":                                        "You carry a database request from the queue by hand.",
		"Opens RAISE A CREW: pick a trade, its class and how many huts.":                   "Opens CREATE A DEPLOYMENT: pick an image, its tag and how many Pods.",
		"CLASS porter:1.1, WORKERS 3, RETUNE or RESEAT":                                    "IMAGE porter:1.1, REPLICAS 3, PATCH HPA or REWRITE MANIFEST",
		"PORTERS, CLASS OF 1.0":                                                            "PORTER PODS, TAG 1.0",
		"The Unhoused (Pending: FailedScheduling) struck. The guttering hut fell.":         "Pending (FailedScheduling) struck. CrashLoopBackOff fell.",
		"The Cellar Ward and The Unanswered Summons":                                       "DiskPressure taint and ImagePullBackOff",
		"Pull the Clerks' class":                                                           "Pull the clerk image",
		"The Unanswered Summons (ErrImagePull / ImagePullBackOff) struck":                  "ImagePullBackOff struck",
		"Builders begin The Archive (Service).":                                            "Builders begin The database Service.",
		"Done. Keep the ledger flowing. F1 opens the ledger":                               "Done. Keep the ledger flowing. F1 opens the ledger",
		"Keep shifts (Cluster API MachineSets)":                                            "Cluster API MachineSets",
		"Services stand on the Commons: build the Guildhall on its paving":                 "Services stand on the shared ground (no Node): build the Service on its paving",
		"Depots stand along the Forum's edge: build the Archive in a free spot on its rim": "Services stand along the shared ground's edge: build the database Service in a free spot on its rim",
		"Next plot: memory-xlarge (memory land), 3 Hands (CPU) and 16Gi Lore (memory)":     "Next plot: memory-xlarge (memory-optimized instance), 3 CPUs and 16Gi memory",
		"+8 Hands and 4Gi of Lore (compute land); 2.4 rations/s":                           "+8 CPUs and 4Gi of memory (compute-optimized instance); 2.4m of CPU/s",
		"a rich land plot": "a large instance plot",
		"Scale the porters to 4: the bigger ships outrun them":                          "Scale the porters to 4: the traffic outgrows them",
		"The porters wait in the Forum. A Keep is on its way: they land when it joins.": "The porters stay Pending. A Node is joining: they get scheduled once it is Ready.",
		"2 Unhoused: ready, press BUILD":                                                "2 Pending: ready, press BUILD",
		"4/6 porters declared: the new huts will be Unhoused":                           "4/6 porters declared: the new Pods will be Pending",
		"The Lone Tower has no more to give, so new huts are Unhoused (Pending).":       "The control-plane Node has no more to give, so new Pods are Pending.",
		"New huts wait in the Forum for land.":                                          "New Pods stay Pending until a Node has room.",
		"The clerks eat about 330Mi of Lore each but pledge only 64Mi, so the Quartermaster seated more of them on one Keep than its Lore can feed; when it ran short, the kubelet banished the huts eating furthest beyond their pledge.": "The clerks use about 330Mi of memory each but request only 64Mi, so the kube-scheduler placed more of them on one Node than its memory can hold; when it ran short, the kubelet evicted the Pods using furthest beyond their request.",
		"Pledge the Lore the clerks really eat: 448Mi":      "Request the memory the clerks really use: 448Mi",
		"Open a berth: the idle Keeps can carry more ships": "Open a traffic ramp: the idle Nodes can serve more traffic",
		"The porters-web plaque":                            "The porters-web Service",
		"The Stewards write the realm's default pledges (LimitRange resource-defaults): every worker without its own now asks 200m of CPU": "The controllers write the cluster's default requests (LimitRange resource-defaults): every container without its own requests now asks 200m of CPU",
		"Raise a Stone Keep for the porters: 4 Hands and 8Gi Lore of new land. The waiting huts move in.":                                  "Raise an xlarge Node for the porters: 4 CPUs and 8Gi memory of new capacity. The Pending Pods get scheduled.",
		"Every Hand and Lore a hut reserves takes Keep land, and a Keep has only so much.":                                                 "Every CPU and memory a Pod reserves takes Node capacity, and a Node has only so much.",
		"Each porter pledges 200 rations and 64 rooms.":                                                                                    "Each porter requests 200m of CPU and 64Mi of memory.",
		"A Stone Keep holds 4,000 rations and 8,192 rooms":                                                                                 "An xlarge Node holds 4 CPUs and 8Gi of memory",
		"Set the Food Pledge, then the Housing Binding.":                                                                                   "Set the CPU request, then the memory limit.",
		"Hungry huts work slower: food is short.":                                                                                          "Throttled Pods work slower: CPU is short.",
		"Housing cannot be shared out; 1 ration is tiny.":                                                                                  "Memory cannot be shared out; 1m of CPU is tiny.",
	}
	for in, want := range cases {
		if got := TrueNames(in); got != want {
			t.Errorf("TrueNames(%q)\n got %q\nwant %q", in, got, want)
		}
	}
}

func TestTrueNamesIsIdempotent(t *testing.T) {
	for _, in := range []string{
		"Cordon the Lone Tower: a Keep (a Node).",
		"A Horn measures Labor against Pledges.",
		"The Stewards lift the not-ready Ward from Keeps, keep Guild Rolls, and raise Musters.",
		"HUT AT THE DOOR",
		"PORTERS, CLASS OF 1.0",
		"HAULERS and CLERKS and COURIERS and SEALWRIGHTS",
		"Porters carry for the Stone Keep",
	} {
		once := TrueNames(in)
		if twice := TrueNames(once); twice != once {
			t.Errorf("not idempotent: %q -> %q -> %q", in, once, twice)
		}
	}
}

func TestTrueNamesCoversTheLexicon(t *testing.T) {
	lx, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range lx.Entries {
		if storyOnly[e.Name] {
			continue
		}
		if got := TrueNames(e.Name); got == e.Name {
			t.Errorf("lexicon name %q (%s) has no true name", e.Name, e.K8s)
		}
	}
}

func TestTrueNamesLeavesNoStoryTermBehind(t *testing.T) {
	for from, to := range trueTable {
		if from == to {
			continue
		}
		got := TrueNames("the " + from + " stands")
		if strings.Contains(got, " "+from+" ") {
			t.Errorf("%q survived: %q", from, got)
		}
	}
}

func TestIncidentLinesReadInKubernetesTerms(t *testing.T) {
	cases := map[string]string{
		"a hut of the porters gutters: it relights and falls again and again":         "a Pod of the porters is in CrashLoopBackOff: it restarts again and again",
		"a hut of the porters gutters: its worker stops and is relit again and again": "a Pod of the porters is in CrashLoopBackOff: it restarts again and again",
		"new clerks are Unhoused: no Keep takes them in":                              "new clerks are Pending: no Node takes them in",
		"Mend it: undo the rollout, or set a class that does not crash":               "Fix it: undo the rollout, or set an image that does not crash",
		"Mend it: add or grow a Keep, pledge less, or scale the crew down":            "Fix it: add or grow a Node, request less, or scale the Deployment down",
		"The hut relit and guttered again and again.":                                 "The Pod restarted and crashed again and again.",
		"Its own logs end in a crash line: the class it runs fails on start.":         "Its own logs end in a crash line: the image it runs fails on start.",
		"The new huts stayed Unhoused: no Keep took them in.":                         "The new Pods stayed Pending: no Node took them in.",
		"Write the postmortem: the calamity waits":                                    "Write the postmortem: the incident waits",
		"Look at what the Keeps have left to pledge":                                  "Look at what the Nodes have left to request",
	}
	for story, want := range cases {
		if got := TrueNames(story); got != want {
			t.Errorf("TrueNames(%q)\n got %q\nwant %q", story, got, want)
		}
	}
}

func TestWatchLinesReadInKubernetesTerms(t *testing.T) {
	cases := map[string]string{
		"Requests are sent to porters-x while its worker still unpacks its tools":  "Requests are sent to porters-x while its container still warms up",
		"porters-x reads Running and its shutters are open, yet nothing leaves it": "porters-x reads Running and it is Ready, yet nothing leaves it",
		"Will you do this every time a worker freezes?":                            "Will you do this every time a container hangs?",
		"ready: set the Watch on porters (PROBES)":                                 "ready: set the probes on porters (PROBES)",
		"The watchman restarted 3 busy porters in the last 120 s":                  "The liveness probe restarted 3 busy porters in the last 120 s",
		"Let the watchman wait 5 s and ring after 3 misses":                        "Let the liveness probe wait 5 s and restart after 3 misses",
		"Set the Watch on this crew: the Shutters and the Watchman's Bell.":        "Set the probes on this Deployment: the readinessProbe and the livenessProbe.",
		"The Shutters: no box is sent to a hut until it answers that it is ready.": "The readinessProbe: no request is sent to a Pod until it answers that it is ready.",
	}
	for story, want := range cases {
		if got := TrueNames(story); got != want {
			t.Errorf("TrueNames(%q)\n got %q\nwant %q", story, got, want)
		}
	}
}

func TestNamesModeSaysEndpointSliceWhereTheStorySaysRoll(t *testing.T) {
	for story, want := range map[string]string{
		"Put each porter on the depot's roll":                                            "EndpointSlice",
		"Let the depot look for a badge: the Stewards keep its roll":                     "its EndpointSlice",
		"You keep this roll yourself: the depot never checks it (kube-proxy trusts it).": "this EndpointSlice",
		"A roll: the list of ready huts a depot hands its boxes to.":                     "An EndpointSlice:",
		"TEAR UP the hand roll: the Stewards keep only the roll they write":              "only the EndpointSlice they write",
		"on the roll: 3 of 5, 2 boxes waiting":                                           "on the EndpointSlice:",
		"This one hands boxes only to the huts on the roll you keep (an EndpointSlice).": "on the EndpointSlice you keep.",
		"A fallen hut stays on a hand-kept roll, and the boxes sent there spoil":         "hand-kept EndpointSlice",
		"the Stewards write their own roll":                                              "their own EndpointSlice",
		"write each [[depot]]'s roll":                                                    "'s EndpointSlice",
	} {
		if got := TrueNames(story); !strings.Contains(got, want) || strings.Contains(got, " roll") {
			t.Errorf("NAMES mode reads %q for %q, want %q", got, story, want)
		}
	}
	for _, verb := range []string{
		"The crew rolls the change out to new huts, one by one.",
		" (while it rolls, the Nines dip)",
		"a shift keeps no revisions to roll: use Crews",
		"Adding one changes the template, so the Pods roll over.",
		"Crews roll a class out hut by hut.",
		"which roll out hut by hut",
		"the roll-out needs room for 4 more huts",
	} {
		if got := TrueNames(verb); strings.Contains(got, "EndpointSlice") {
			t.Errorf("the verb 'roll' became a noun: %q reads %q", verb, got)
		}
	}
}
