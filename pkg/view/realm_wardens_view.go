package view

type DaemonOnKeep struct {
	Name  string `json:"name"`
	Ready bool   `json:"ready"`
}

type DaemonSetView struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Image     string `json:"image"`
	Desired   int32  `json:"desired"`
	Current   int32  `json:"current"`
	Ready     int32  `json:"ready"`
	Updated   int32  `json:"updated"`
	Selector  string `json:"selector"`
}
