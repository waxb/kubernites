package view

type CodexView struct {
	SizeBytes       int64   `json:"sizeBytes"`
	InUseBytes      int64   `json:"inUseBytes"`
	QuotaBytes      int64   `json:"quotaBytes"`
	Alarm           bool    `json:"alarm"`
	LastRebind      float64 `json:"lastRebind"`
	Revision        int64   `json:"revision"`
	CompactRevision int64   `json:"compactRevision"`
	WriteDelay      float64 `json:"writeDelay"`
	LateWrites      int     `json:"lateWrites"`
}
