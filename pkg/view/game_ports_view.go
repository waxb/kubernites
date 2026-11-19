package view

type PortNextView struct {
	Level      int      `json:"level"`
	Ship       string   `json:"ship"`
	Ships      string   `json:"ships"`
	Kinds      []string `json:"kinds"`
	Listeners  []string `json:"listeners"`
	Multiplier float64  `json:"multiplier"`
	Upkeep     float64  `json:"upkeep"`
	MaxBerths  int      `json:"maxBerths"`
	Opens      string   `json:"opens"`
}

type PortGates struct {
	Open    string `json:"open,omitempty"`
	Upgrade string `json:"upgrade,omitempty"`
	Berth   string `json:"berth,omitempty"`
	Close   string `json:"close,omitempty"`
}

type PortEffects struct {
	Open    string `json:"open,omitempty"`
	Upgrade string `json:"upgrade,omitempty"`
	Berth   string `json:"berth,omitempty"`
}

type PortView struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Open        bool          `json:"open"`
	Level       int           `json:"level"`
	MaxLevel    int           `json:"maxLevel"`
	Ship        string        `json:"ship"`
	Ships       string        `json:"ships"`
	Kinds       []string      `json:"kinds"`
	Listeners   []string      `json:"listeners"`
	Multiplier  float64       `json:"multiplier"`
	Upkeep      float64       `json:"upkeep"`
	Berths      int           `json:"berths"`
	MaxBerths   int           `json:"maxBerths"`
	Draw        float64       `json:"draw"`
	Share       float64       `json:"share"`
	ArrivalRate float64       `json:"arrivalRate"`
	OpenedAt    float64       `json:"openedAt"`
	LevelAt     float64       `json:"levelAt"`
	OpenCost    float64       `json:"openCost"`
	UpgradeCost float64       `json:"upgradeCost"`
	BerthCost   float64       `json:"berthCost"`
	Next        *PortNextView `json:"next,omitempty"`
	Gates       PortGates     `json:"gates"`
	Effects     PortEffects   `json:"effects"`
	K8s         string        `json:"k8s"`
}
