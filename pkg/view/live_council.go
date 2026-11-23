package view

type LiveCouncil struct {
	NodesReady     int            `json:"nodesReady"`
	Nodes          int            `json:"nodes"`
	PodsReady      int            `json:"podsReady"`
	PodsActive     int            `json:"podsActive"`
	WorkloadsFull  int            `json:"workloadsFull"`
	Workloads      int            `json:"workloads"`
	ServicesServed int            `json:"servicesServed"`
	Services       int            `json:"services"`
	Pending        []LivePending  `json:"pending,omitempty"`
	Short          []LiveWorkload `json:"short,omitempty"`
	Unserved       []LiveService  `json:"unserved,omitempty"`
	Busiest        []LivePodUsage `json:"busiest,omitempty"`
	Unread         []string       `json:"unread,omitempty"`
}

type LivePending struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Reason    string `json:"reason,omitempty"`
}

type LiveWorkload struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Ready     int    `json:"ready"`
	Desired   int    `json:"desired"`
}

type LiveService struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Endpoints int    `json:"endpoints"`
}

type LivePodUsage struct {
	Namespace   string  `json:"namespace"`
	Name        string  `json:"name"`
	CPUMilli    float64 `json:"cpuMilli"`
	MemoryBytes float64 `json:"memoryBytes"`
}
