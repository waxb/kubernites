package view

type LedgerView struct {
	Open          bool    `json:"open"`
	Level         float64 `json:"level"`
	Cap           float64 `json:"cap"`
	RecordedRate  float64 `json:"recordedRate"`
	Scribes       int     `json:"scribes"`
	TotalRecorded float64 `json:"totalRecorded"`
}
