package view

type RankView struct {
	Title   string   `json:"title"`
	Lesson  string   `json:"lesson"`
	Runes   []string `json:"runes"`
	Cost    Cost     `json:"cost"`
	Learned bool     `json:"learned"`
}
