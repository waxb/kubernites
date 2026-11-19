package view

type Page struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	K8sTitle string `json:"k8sTitle"`
	Purpose  string `json:"purpose"`
	How      string `json:"how"`
	Why      string `json:"why"`
	Game     string `json:"game"`
	K8s      string `json:"k8s"`
	Doc      string `json:"doc"`
	Sprite   string `json:"sprite"`
	Unlock   string `json:"unlock"`
	Stage    string `json:"stage"`
	Branch   string `json:"branch"`
	Category string `json:"category"`
}

const StageDev = "dev"

const StageStaging = "staging"

var Branches = []string{"networking", "scheduling", "reliability", "upkeep"}

var Categories = []string{"buildings", "workers", "resources", "powers", "troubles", "realm"}

var FindPage = func(string) (Page, bool) { return Page{}, false }
