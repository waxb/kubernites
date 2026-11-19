package view

type RouteView struct {
	Kind     string `json:"kind"`
	Service  string `json:"service"`
	Route    string `json:"route"`
	Listener string `json:"listener"`
	Port     int    `json:"port"`
	Open     bool   `json:"open"`
	Ready    int    `json:"ready"`
}
