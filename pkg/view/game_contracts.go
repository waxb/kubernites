package view

const ContractsFlag = "contracts"

type ContractReward struct {
	Gold   float64 `json:"gold"`
	Wisdom float64 `json:"wisdom"`
	Favor  float64 `json:"favor"`
	Boost  float64 `json:"boost,omitempty"`
}
