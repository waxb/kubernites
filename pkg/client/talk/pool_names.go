package talk

var namesLines = []Line{
	{Text: "ContainerCreating. Pulling myself together.", When: SitKindling, Tone: Joker, Pool: PoolNames},
	{Text: "Waiting: ContainerCreating. Nearly there.", When: SitKindling, Tone: Cheerful, Pool: PoolNames},
	{Text: "Image already present on machine. Let's work.", When: SitKindling, Tone: Proud, Pool: PoolNames, If: imageReady},
	{Text: "Not started yet. Is that normal? It's normal.", When: SitKindling, Tone: Anxious, Pool: PoolNames},

	{Text: "Restart count? Let's not talk about the restart count.", When: SitGuttering, Tone: Grumpy, Pool: PoolNames},
	{Text: "CrashLoopBackOff. Back-off doubles, up to five minutes.", When: SitGuttering, Tone: Dreamy, Pool: PoolNames},
	{Text: "Exit code {exit}. The kubelet will try me again.", When: SitGuttering, Tone: Anxious, Pool: PoolNames, Needs: []string{"keep"}, If: restarts},
	{Text: "Same image, same me. Every restart.", When: SitGuttering, Tone: Proud, Pool: PoolNames},
	{Text: "Crash, back off, restart. I know this dance.", When: SitGuttering, Tone: Joker, Pool: PoolNames},
	{Text: "Restarting! Third time lucky! Or tenth.", When: SitGuttering, Tone: Cheerful, Pool: PoolNames},

	{Text: "Hit my memory limit once. OOMKilled.", When: SitVoidtaken, Tone: Joker, Pool: PoolNames, Needs: []string{"binding"}, If: oomed, Then: "Woke up fresh."},
	{Text: "OOMKilled. Exit code {exit}. Back now.", When: SitVoidtaken, Tone: Grumpy, Pool: PoolNames, If: oomed},
	{Text: "OOMKilled! Is my memory limit too low?", When: SitVoidtaken, Tone: Anxious, Pool: PoolNames, Needs: []string{"binding"}, If: oomed},

	{Text: "Still waiting: my image could not be pulled.", When: SitNever, Tone: AnyTone, Pool: PoolNames},

	{Text: "Throttled again. The CPU limit is a strict teacher.", When: SitHungry, Tone: Grumpy, Pool: PoolNames, Needs: []string{"binding"}, If: Facts.AtCPULimit},
	{Text: "Requests: {cpu}. Reality: whatever the CPU gives me.", When: SitHungry, Tone: Joker, Pool: PoolNames, Needs: []string{"pledge"}, If: pledged},
	{Text: "Short of CPU. Every request takes longer.", When: SitHungry, Tone: Anxious, Pool: PoolNames},
	{Text: "Less CPU than I want. I make do.", When: SitHungry, Tone: Proud, Pool: PoolNames},

	{Text: "Near my memory limit. One step more is OOMKilled.", When: SitCramped, Tone: Anxious, Pool: PoolNames, Needs: []string{"binding"}},
	{Text: "Working set near my limit. Tight squeeze.", When: SitCramped, Tone: Grumpy, Pool: PoolNames, Needs: []string{"binding"}},
	{Text: "So much memory. All of it mine. Nearly all.", When: SitCramped, Tone: Dreamy, Pool: PoolNames},

	{Text: "SIGTERM? Give me my grace period, I'm mid-request.", When: SitRites, Tone: Grumpy, Pool: PoolNames, If: busy},
	{Text: "Our Pod is being deleted. I stop when SIGTERM says.", When: SitRites, Tone: Dreamy, Pool: PoolNames},
	{Text: "SIGTERM received. Finishing within my grace period.", When: SitRites, Tone: Proud, Pool: PoolNames},
	{Text: "Being deleted already? I only just started!", When: SitRites, Tone: Anxious, Pool: PoolNames, If: shortlived},

	{Text: "Liveness probe failing. Restart me, kubelet.", When: SitFrozen, Tone: AnyTone, Pool: PoolNames, Needs: []string{"bell"}, If: func(f Facts) bool { return f.Worker.Liveness }},

	{Text: "Not Ready. The Service sends me nothing yet.", When: SitShutters, Tone: AnyTone, Pool: PoolNames, Needs: []string{"shutters", "depot"}},
	{Text: "Readiness probe not passing yet. Patience.", When: SitShutters, Tone: Proud, Pool: PoolNames, Needs: []string{"shutters"}},

	{Text: "Restarted. Last State: Terminated, exit code {exit}.", When: SitRelit, Tone: Proud, Pool: PoolNames},
	{Text: "Same image, same me. Every restart.", When: SitRelit, Tone: Dreamy, Pool: PoolNames},
	{Text: "Restart count went up. Nobody saw. Good.", When: SitRelit, Tone: Joker, Pool: PoolNames},

	{Text: "Queue full. New requests are turned away.", When: SitProblem, Tone: AnyTone, Pool: PoolNames, If: problem("quayfull")},

	{Text: "Our Pod made the endpoints! Send requests!", When: SitJoined, Tone: Cheerful, Pool: PoolNames, Needs: []string{"depot"}},
	{Text: "Our Pod left the Service's endpoints. Quiet now.", When: SitLeft, Tone: AnyTone, Pool: PoolNames, Needs: []string{"depot"}},

	{Text: "A rolling update is on. Be kind to the new Pods.", When: SitRollout, Tone: Cheerful, Pool: PoolNames, Needs: []string{"crew"}},
	{Text: "New ReplicaSet, new Pods. Ours scales down.", When: SitRollout, Tone: Dreamy, Pool: PoolNames, Needs: []string{"crew"}, If: retiring},

	{Text: "My Pod is a static Pod. The kubelet runs it from a file.", When: SitDoor, Tone: Proud, Pool: PoolNames, Needs: []string{"door", "keep"}},
	{Text: "My Pod has no owner. Delete it and nothing recreates it.", When: SitHand, Tone: Grumpy, Pool: PoolNames},
	{Text: "A ReplicaSet keeps the count. My Pod is one replica.", When: SitCrew, Tone: Proud, Pool: PoolNames, Needs: []string{"crew"}},
	{Text: "My Pod's labels say app={app}.", When: SitRoll, Tone: Joker, Pool: PoolNames, Needs: []string{"badge", "depot"}, If: onRollApp, Then: "That's how the Service finds it."},
	{Text: "The Service sends my Pod requests. I never ask from where.", When: SitRoll, Tone: Dreamy, Pool: PoolNames, Needs: []string{"depot"}},
	{Text: "Restart count? Let's not talk about the restart count.", When: SitShaken, Tone: Grumpy, Pool: PoolNames},

	{Text: "If I crash, the kubelet starts me again.", When: SitGeneral, Tone: Cheerful, Pool: PoolNames, Needs: []string{"keep"}, If: restarts},
	{Text: "My readiness probe says yes. My heart says maybe.", When: SitGeneral, Tone: Anxious, Pool: PoolNames, Needs: []string{"shutters"}, If: readyProbe},
	{Text: "I live in a Pod. The view is mostly localhost.", When: SitGeneral, Tone: Joker, Pool: PoolNames},
	{Text: "Requests: {cpu}. Reality: whatever the CPU gives me.", When: SitGeneral, Tone: Joker, Pool: PoolNames, Needs: []string{"pledge"}, If: pledged},
	{Text: "I log to stdout. Someone reads it, right?", When: SitGeneral, Tone: Anxious, Pool: PoolNames, Needs: []string{"chronicle"}},
	{Text: "My Pod is ephemeral. I try not to dwell on it.", When: SitGeneral, Tone: Dreamy, Pool: PoolNames},
	{Text: "Liveness probe passed. Officially alive.", When: SitGeneral, Tone: Cheerful, Pool: PoolNames, Needs: []string{"bell"}, If: alive},
	{Text: "When the Node drains, my Pod is evicted. I pack light.", When: SitGeneral, Tone: Joker, Pool: PoolNames, Needs: []string{"drain"}, If: drainable},
	{Text: "Stateless, they call me. I don't remember yesterday.", When: SitGeneral, Tone: Dreamy, Pool: PoolNames, If: stateless},
	{Text: "The scheduler picked this Node for my Pod. Good choice.", When: SitGeneral, Tone: Proud, Pool: PoolNames, Needs: []string{"quartermaster"}, If: scheduled},
	{Text: "Killed for OOM? Not me. I stay well under my limit.", When: SitGeneral, Tone: Proud, Pool: PoolNames, Needs: []string{"binding"}, If: roomy},
	{Text: "kubectl exec into me? Please wipe your feet.", When: SitGeneral, Tone: Grumpy, Pool: PoolNames, Needs: []string{"stone"}, If: Facts.running},
	{Text: "Image already present on machine. Let's work.", When: SitGeneral, Tone: Cheerful, Pool: PoolNames, If: imageReady},
}

var systemLines = []Line{
	{Text: "Every decree passes through me.", When: SitSystem, Trade: "Chancellor", Tone: Proud},
	{Text: "I write nothing down myself. The Codex does.", When: SitSystem, Trade: "Chancellor", Tone: Joker},
	{Text: "Every request to the cluster goes through me.", When: SitSystem, Trade: "Chancellor", Tone: Proud, Pool: PoolNames},
	{Text: "I'm kube-apiserver. etcd stores what I accept.", When: SitSystem, Trade: "Chancellor", Tone: Joker, Pool: PoolNames},
	{Text: "I keep the one book of truth.", When: SitSystem, Trade: "Codex-keeper", Tone: Proud},
	{Text: "Every decree the Chancery accepts, I write down.", When: SitSystem, Trade: "Codex-keeper", Tone: Dreamy},
	{Text: "I'm etcd. I keep the cluster's state.", When: SitSystem, Trade: "Codex-keeper", Tone: Proud, Pool: PoolNames},
	{Text: "Every write the API server accepts lands in me.", When: SitSystem, Trade: "Codex-keeper", Tone: Dreamy, Pool: PoolNames},
	{Text: "I nudge the realm toward what is written.", When: SitSystem, Trade: "Steward", Tone: Proud},
	{Text: "A shift one hut short? I raise another.", When: SitSystem, Trade: "Steward", Tone: Cheerful},
	{Text: "I run the controllers. Desired state is my job.", When: SitSystem, Trade: "Steward", Tone: Proud, Pool: PoolNames},
	{Text: "A ReplicaSet one Pod short? I create one.", When: SitSystem, Trade: "Steward", Tone: Cheerful, Pool: PoolNames},
	{Text: "I give every hut its quarters.", When: SitSystem, Trade: "Quartermaster", Tone: Proud},
	{Text: "I read only pledges, never appetite.", When: SitSystem, Trade: "Quartermaster", Tone: Grumpy},
	{Text: "I'm kube-scheduler. I pick a Node for each Pod.", When: SitSystem, Trade: "Quartermaster", Tone: Proud, Pool: PoolNames},
	{Text: "I look at requests, not at actual usage.", When: SitSystem, Trade: "Quartermaster", Tone: Grumpy, Pool: PoolNames},
	{Text: "I announce every depot by name.", When: SitSystem, Trade: "Herald", Tone: Proud},
	{Text: "I'm CoreDNS. I answer for every Service's name.", When: SitSystem, Trade: "Herald", Tone: Proud, Pool: PoolNames},
	{Text: "I keep the roads to every depot open.", When: SitSystem, Trade: "Wayfinder", Tone: Proud},
	{Text: "I'm kube-proxy. I route Service traffic to Pods.", When: SitSystem, Trade: "Wayfinder", Tone: Proud, Pool: PoolNames},
	{Text: "I read how much food and rooms each worker uses.", When: SitSystem, Trade: "Augur", Tone: Dreamy},
	{Text: "I'm metrics-server. kubectl top reads from me.", When: SitSystem, Trade: "Augur", Tone: Proud, Pool: PoolNames},
	{Text: "I'm Flannel. Pods talk across Nodes through me.", When: SitSystem, Trade: "Flannel", Tone: Proud, Pool: PoolNames},
}

func Pool() []Line {
	out := make([]Line, 0, len(storyLines)+len(namesLines)+len(systemLines)+len(petLines)+len(annoyedLines))
	out = append(out, storyLines...)
	out = append(out, namesLines...)
	out = append(out, systemLines...)
	out = append(out, petLines...)
	return append(out, annoyedLines...)
}
