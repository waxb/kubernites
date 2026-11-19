package view

const ArchiveSettleSeconds = 300.0

const LossEmptyDir = "emptyDir"

const LossVolume = "persistentVolume"

type StorageFacts struct {
	ArchiveRaised   bool
	ArchiveBooks    []string
	ArchiveLost     int
	AttachWaiting   int
	ArchiveStateful bool
	ArchiveReplicas int
	BooksInVault    bool
	VaultsFree      int
	DeedsWaiting    int
	GuildFounded    bool
}

type ArchiveView struct {
	Unsettled float64 `json:"unsettled"`
	Seals     float64 `json:"seals"`
	SettleIn  float64 `json:"settleIn"`
	LastLoss  float64 `json:"lastLoss"`
	LostSeals float64 `json:"lostSeals"`
	LostAt    float64 `json:"lostAt"`
	LostCause string  `json:"lostCause,omitempty"`
	Pace      float64 `json:"pace"`
	Writers   int     `json:"writers"`
	Books     int     `json:"books"`
}
