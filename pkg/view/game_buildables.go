package view

const PlaceAnywhere = "anywhere"

const PlaceControlPlane = "control-plane"

const PlaceForum = "forum"

const TowerCost = 10.0

type Buildable struct {
	Kind        string  `json:"kind"`
	Name        string  `json:"name"`
	K8s         string  `json:"k8s"`
	Description string  `json:"description"`
	Lore        string  `json:"lore,omitempty"`
	Cost        float64 `json:"cost"`
	W           int     `json:"w"`
	H           int     `json:"h"`
	Placement   string  `json:"placement"`
	MinPower    uint8   `json:"minPower,omitempty"`
	Territory   int     `json:"territory,omitempty"`
	Survey      bool    `json:"survey,omitempty"`
	Available   bool    `json:"available"`
	Reason      string  `json:"reason,omitempty"`
	Locked      bool    `json:"locked,omitempty"`
	Effect      string  `json:"effect"`
	Reserve     bool    `json:"reserve,omitempty"`
}

type BuildDef struct {
	Kind        string
	Name        string
	K8s         string
	Description string
	Lore        string
	Cost        float64
	W, H        int
	Placement   string
	Seconds     float64
	Requires    []string
	Once        bool
	MinPower    uint8
	Rune        string
	Age         string
}

var BuildDefs = []BuildDef{
	{Kind: "keep", Name: "Keep", K8s: "Node (a machine running a kubelet)", Cost: TowerCost, W: 3, H: 3, Placement: PlaceAnywhere, Seconds: 8,
		Description: "Holds your huts and gives them food and housing; more Keeps hold more huts (a Node, run by its kubelet).",
		Lore:        "Land and walls for your huts; its Reeve (the Keep's overseer) looks after whatever lives there."},
	{Kind: "great-keep", Name: "Great Keep", K8s: "Node (a larger instance type: borders of 7 tiles, so more forest and stone in reach)", Cost: 4000, W: 3, H: 3, Placement: PlaceAnywhere, Seconds: 30, Rune: "great-keeps", Age: AgeFeudal,
		Description: "A bigger Keep: more food and housing under one roof, if its land fits. Keeps never grow: raise a greater one (an instance type).",
		Lore:        "Borders reaching 7 tiles, taking in more forest and stone."},
	{Kind: "citadel-keep", Name: "Citadel Keep", K8s: "Node (the largest instance type: borders of 9 tiles, the most forest and stone in reach)", Cost: 40000, W: 3, H: 3, Placement: PlaceAnywhere, Seconds: 40, Rune: "citadel-keeps", Age: AgeFeudal,
		Description: "The biggest Keep: the most food and housing under one roof, on the most land (the largest instance type).",
		Lore:        "Borders reaching 9 tiles; the largest Keep the Sky Barons rent."},
	{Kind: "codex", Name: "The Codex", K8s: "etcd", Cost: 10, W: 2, H: 2, Placement: PlaceControlPlane, Seconds: 10, Requires: []string{"keep"}, Once: true,
		Description: "Records every order you give, every crew, depot and count, so the realm can rebuild whatever is lost (etcd).",
		Lore:        "The realm's book: every hut, crew and order you give is written in it; what is not written did not happen."},
	{Kind: "chancery", Name: "The Chancery", K8s: "kube-apiserver", Cost: 11, W: 2, H: 2, Placement: PlaceControlPlane, Seconds: 12, Requires: []string{"codex"}, Once: true,
		Description: "Takes your orders, checks them and writes them into the Codex; it also wakes the Speaking Stone (the API server).",
		Lore:        "The only way into the Codex: you and every hall speak through it."},
	{Kind: "quartermaster", Name: "The Quartermaster", K8s: "kube-scheduler", Cost: 12, W: 2, H: 2, Placement: PlaceControlPlane, Seconds: 10, Requires: []string{"chancery"}, Once: true,
		Description: "Places every new hut on a Keep where it fits: food and housing enough, and every Ward on it matched by a Charm of the hut. Without it, new huts wait (the scheduler).",
		Lore:        "It places each hut only where it fits and no Ward turns it away."},
	{Kind: "stewards-hall", Name: "The Stewards' Hall", K8s: "kube-controller-manager", Cost: 13, W: 2, H: 2, Placement: PlaceControlPlane, Seconds: 12, Requires: []string{"chancery"}, Once: true,
		Description: "Hold the realm as you declared it: each crew at its count, each depot with its list of ready huts, silent Keeps fenced off (the controllers).",
		Lore:        "The Stewards keep the realm's promises, day and night."},
	{Kind: "archive", Name: "The Archive", K8s: "Service (ledgers)", Cost: 300, W: 1, H: 2, Placement: PlaceForum, Seconds: 25, Requires: []string{"stewards-hall"}, Once: true, Age: AgeFeudal,
		Description: "Sends each ledger to any ready clerk, so ledgers reach the crew without your hands. Ledgers pay twelve times a parcel (a Service).",
		Lore:        "The ledgers' door: the clerks behind it keep the books."},
	{Kind: "guildhall", Name: "Parcels Depot", K8s: "Service (parcels)", Cost: 0, W: 1, H: 2, Placement: PlaceForum, Seconds: 10, Requires: []string{"chancery"}, Once: true,
		Description: "Sends each parcel to any ready porter, so parcels reach the crew without your hands (a Service).",
		Lore:        "One name on the quay for every parcel; the porters behind it answer."},
}

func FindBuildDef(kind string) (BuildDef, bool) {
	for _, d := range BuildDefs {
		if d.Kind == kind {
			return d, true
		}
	}
	return BuildDef{}, false
}

const LadderDepotGate = "The parcels depot opens from its own tile while you keep its roll by hand"
