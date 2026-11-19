package view

type CrewView struct {
	Name       string           `json:"name"`
	Image      string           `json:"image"`
	Kind       string           `json:"kind,omitempty"`
	Keeper     string           `json:"keeper,omitempty"`
	Depot      string           `json:"depot,omitempty"`
	Ready      int              `json:"ready"`
	Replicas   int              `json:"replicas"`
	Horn       bool             `json:"horn,omitempty"`
	PerReplica ReplicaCost      `json:"perReplica"`
	Fits       int              `json:"fits"`
	Room       RoomFit          `json:"room"`
	Owed       int              `json:"owed,omitempty"`
	Pending    int              `json:"pending"`
	KeepOffer  *CrewKeepOffer   `json:"keepOffer,omitempty"`
	Rollout    *RolloutRoomView `json:"rollout,omitempty"`
	Takes      ResourcePair     `json:"takes"`
	TakesUse   bool             `json:"takesUse,omitempty"`
	Use        ResourcePair     `json:"use"`
	Work       Work             `json:"work"`
	Peak       UsePeakView      `json:"peak"`
	CharterAt  int              `json:"charterAt,omitempty"`
	MilestoneView
}
