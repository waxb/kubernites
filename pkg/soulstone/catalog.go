package soulstone

var catalog = append(ladderProfiles(), []Profile{
	{
		Image: "raven:1.0", Name: "Ravens, class of 1.0", Profession: "Raven", Sprite: "raven", Cost: 120, Familiar: true,
		Description:         "A familiar that flies every chronicle line to the rookery, like a log shipper. Bound beside a worker, it carries no boxes itself.",
		CPUPerPilgrimMilliS: 15, ThreadMilli: 1000, IdleMilli: 25, MemBaseMi: 60, MemPerRpsMi: 4, MTBFSeconds: 3600,
		PullSeconds: 2, StartSeconds: 1, SizeMi: 24,
		StartLines: []string{"the raven settles on the eaves", "watching the chronicles for new lines"}, CrashLine: "fatal: the raven lost its way home (exit status 1)",
		DutyLine: "carried %d chronicle lines to the rookery in the last %gs",
	},
	{
		Image: "hound:1.0", Name: "Hounds, class of 1.0", Profession: "Hound", Sprite: "hound", Cost: 300, Familiar: true, Proxy: true,
		Description:         "A familiar that meets every box at the door and leads it to its master, like a proxy (envoy). If the master stops mid-task, the hound leads the box back to the queue.",
		CPUPerPilgrimMilliS: 150, ThreadMilli: 2000, IdleMilli: 10, MemBaseMi: 64, MemPerRpsMi: 36, MTBFSeconds: 3600,
		PullSeconds: 3, StartSeconds: 1, SizeMi: 56,
		StartLines: []string{"the hound takes its place at the door", "leading boxes to :8080"}, CrashLine: "fatal: the hound slipped its leash (exit status 1)",
		DutyLine: "led %d boxes to the hearth in the last %gs",
	},
	systemProfile("registry.k8s.io/kube-apiserver:v1.37.0", "The Chancery", "Chancellor", "chancellor", 90, 280),
	systemProfile("registry.k8s.io/etcd:3.7.0-0", "The Codex", "Codex-keeper", "codex-keeper", 35, 60),
	systemProfile("registry.k8s.io/kube-controller-manager:v1.37.0", "The Stewards' Hall", "Steward", "steward", 25, 55),
	systemProfile("registry.k8s.io/kube-scheduler:v1.37.0", "The Quartermaster", "Quartermaster", "quartermaster", 10, 30),
	systemProfile("registry.k8s.io/coredns/coredns:v1.14.6", "The Herald", "Herald", "herald", 4, 18),
	systemProfile("registry.k8s.io/kube-proxy:v1.37.0", "The Wayfinder", "Wayfinder", "wayfinder", 2, 20),
	systemProfile("registry.k8s.io/metrics-server/metrics-server:v0.8.0", "The Augur", "Augur", "augur", 6, 28),
	systemProfile("warden:1.0", "The Post-Warden", "Post-warden", "warden", 8, 24),
	flannelProfile(),
}...)

const FlannelImage = "ghcr.io/flannel-io/flannel:v0.28.10"

func flannelProfile() Profile {
	p := systemProfile(FlannelImage, "Flannel", "Flannel", "wayfinder", 3, 20)
	p.PullSeconds, p.SizeMi = 2, 32
	p.StartLines = []string{"Determining IP address of default interface", "Using interface with name eth0 and address 172.18.0.2", "Wrote subnet file to /run/flannel/subnet.env"}
	return p
}

var aliases = withSystemAliases(ladderAliases())

func withSystemAliases(out map[string]string) map[string]string {
	out["registry.k8s.io/etcd:3.6.0-0"] = "registry.k8s.io/etcd:3.7.0-0"
	out["registry.k8s.io/coredns/coredns:v1.12.0"] = "registry.k8s.io/coredns/coredns:v1.14.6"
	return out
}
