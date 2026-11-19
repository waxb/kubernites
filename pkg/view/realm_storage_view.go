package view

type VaultView struct {
	Name    string `json:"name"`
	Isle    string `json:"isle"`
	Zone    string `json:"zone"`
	Shelves int    `json:"shelves"`
	Phase   string `json:"phase"`
	Claim   string `json:"claim"`
	Reclaim string `json:"reclaim"`
	Dynamic bool   `json:"dynamic"`
}

type DeedView struct {
	Name    string `json:"name"`
	Shelves int    `json:"shelves"`
	Phase   string `json:"phase"`
	Volume  string `json:"volume"`
	Class   string `json:"class"`
	Waiting string `json:"waiting"`
}

type StorageClassView struct {
	Name        string `json:"name"`
	BindingMode string `json:"bindingMode"`
	Reclaim     string `json:"reclaim"`
	Default     bool   `json:"default"`
}

type StorageView struct {
	Vaults  []VaultView        `json:"vaults"`
	Deeds   []DeedView         `json:"deeds"`
	Classes []StorageClassView `json:"classes"`
}

type StorageState struct {
	Storage      StorageView       `json:"storage"`
	StatefulSets []StatefulSetView `json:"statefulSets"`
	Archive      *ArchiveView      `json:"archive,omitempty"`
}

type StatefulPodView struct {
	Name    string `json:"name"`
	Ordinal int    `json:"ordinal"`
	Claim   string `json:"claim"`
	Keep    string `json:"keep"`
	Phase   string `json:"phase"`
}

type StatefulSetView struct {
	Name     string            `json:"name"`
	Replicas int               `json:"replicas"`
	Ready    int               `json:"ready"`
	Pods     []StatefulPodView `json:"pods"`
	Updating bool              `json:"updating"`
}
