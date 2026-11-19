package view

type Cost struct {
	Gold   float64 `json:"gold"`
	Wisdom float64 `json:"wisdom"`
}

type GrimoirePage struct {
	ID          string     `json:"id"`
	School      string     `json:"school"`
	Name        string     `json:"name"`
	K8s         string     `json:"k8s"`
	Description string     `json:"description"`
	Lesson      string     `json:"lesson"`
	Cost        Cost       `json:"cost"`
	State       string     `json:"state"`
	Requires    []string   `json:"requires"`
	Rank        int        `json:"rank"`
	MaxRank     int        `json:"maxRank"`
	Ranks       []RankView `json:"ranks"`
	Branch      string     `json:"branch"`
	Era         string     `json:"era"`
	Earn        string     `json:"earn,omitempty"`
	Now         string     `json:"now,omitempty"`
	PriceRule   string     `json:"priceRule,omitempty"`
	Buys        string     `json:"buys,omitempty"`
	Opens       string     `json:"opens,omitempty"`
}

type SoulstoneEntry struct {
	Image       string  `json:"image"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Cost        float64 `json:"cost"`
	Owned       bool    `json:"owned"`
	Visible     bool    `json:"visible"`
	Cursed      bool    `json:"cursed"`
	Effect      string  `json:"effect"`
}

type KeepOffer struct {
	KeepType     string  `json:"keepType"`
	Name         string  `json:"name"`
	K8s          string  `json:"k8s"`
	CPU          string  `json:"cpu"`
	Memory       string  `json:"memory"`
	Cost         float64 `json:"cost"`
	Value        float64 `json:"value"`
	UpkeepPerSec float64 `json:"upkeepPerSec"`
	MinPower     uint8   `json:"minPower,omitempty"`
	Territory    int     `json:"territory"`
	Class        string  `json:"class,omitempty"`
	Instance     string  `json:"instance,omitempty"`
	Available    bool    `json:"available"`
	Reason       string  `json:"reason,omitempty"`
	Effect       string  `json:"effect"`
	PriceRule    string  `json:"priceRule,omitempty"`
	UpkeepRule   string  `json:"upkeepRule,omitempty"`
	Idle         string  `json:"idle,omitempty"`
	Net          string  `json:"net,omitempty"`
}

type ChronicleOption struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type ChronicleView struct {
	ID          string            `json:"id"`
	Kind        string            `json:"kind"`
	Calamity    string            `json:"calamity"`
	K8s         string            `json:"k8s"`
	Question    string            `json:"question"`
	Options     []ChronicleOption `json:"options"`
	Resolved    bool              `json:"resolved"`
	Correct     bool              `json:"correct"`
	Ravens      bool              `json:"ravens,omitempty"`
	Explanation string            `json:"explanation"`
	Truth       string            `json:"truth,omitempty"`
	Incident    string            `json:"incident,omitempty"`
	Timeline    *IncidentTimeline `json:"timeline,omitempty"`
	Plain       bool              `json:"plain,omitempty"`
	Subject     string            `json:"subject,omitempty"`
	Struck      string            `json:"struck,omitempty"`
	Healed      string            `json:"healed,omitempty"`
	MendLine    string            `json:"mendLine,omitempty"`
}

type Requirement struct {
	Text    string  `json:"text"`
	Kind    string  `json:"kind,omitempty"`
	Met     bool    `json:"met"`
	Latched bool    `json:"latched,omitempty"`
	Gold    float64 `json:"gold,omitempty"`
	Set     bool    `json:"set,omitempty"`
}

type AgeUpView struct {
	Available    bool          `json:"available"`
	NextAge      string        `json:"nextAge"`
	Requirements []Requirement `json:"requirements"`
	Blocker      string        `json:"blocker,omitempty"`
}

type Snapshot struct {
	Age           string            `json:"age"`
	Gold          float64           `json:"gold"`
	GoldPerSec    float64           `json:"goldPerSec"`
	Wisdom        float64           `json:"wisdom"`
	Renown        float64           `json:"renown"`
	Berths        BerthView         `json:"berths"`
	Ports         []PortView        `json:"ports"`
	Unlocks       map[string]bool   `json:"unlocks"`
	Grimoire      []GrimoirePage    `json:"grimoire"`
	Soulstones    []SoulstoneEntry  `json:"soulstones"`
	KeepShop      []KeepOffer       `json:"keepShop"`
	Trades        []TradeView       `json:"trades"`
	Land          LandView          `json:"land"`
	Objectives    []ObjectiveView   `json:"objectives"`
	Quests        []QuestView       `json:"quests"`
	Proclamations []Proclamation    `json:"proclamations"`
	Chronicles    []ChronicleView   `json:"chronicles"`
	Incidents     []IncidentReport  `json:"incidents"`
	Treasure      []TreasureView    `json:"treasure"`
	SealsUnpaid   float64           `json:"sealsUnpaid,omitempty"`
	AgeUp         AgeUpView         `json:"ageUp"`
	AgeMoment     *AgeMomentView    `json:"ageMoment,omitempty"`
	Fluency       []string          `json:"fluency"`
	Reveals       []string          `json:"reveals"`
	Buildables    []Buildable       `json:"buildables"`
	Reforge       ReforgeView       `json:"reforge"`
	Siege         SiegeView         `json:"siege"`
	Mastery       MasteryView       `json:"mastery"`
	Road          RoadView          `json:"road"`
	Storm         StormView         `json:"storm"`
	Charter       CharterView       `json:"charter"`
	Almanac       []string          `json:"almanac"`
	Routes        []RouteView       `json:"routes"`
	Decrees       []DecreeView      `json:"decrees"`
	Contracts     []ContractView    `json:"contracts"`
	Ratings       RatingsView       `json:"ratings"`
	Intro         *IntroView        `json:"intro,omitempty"`
	IntroQueue    []string          `json:"introQueue,omitempty"`
	Introduced    []string          `json:"introduced"`
	IntroTargets  []string          `json:"introducedTargets"`
	Try           *ObjectiveView    `json:"try,omitempty"`
	Effects       map[string]string `json:"effects"`
	Autonomy      []AutonomyView    `json:"autonomy"`
	Study         StudyView         `json:"study"`
	Golden        GoldenView        `json:"golden"`
	Calm          float64           `json:"calm,omitempty"`
	Guidance      string            `json:"guidance"`
	CalmRealm     bool              `json:"calmRealm,omitempty"`
	CalmOffer     bool              `json:"calmOffer,omitempty"`
	Soaking       bool              `json:"soaking,omitempty"`
	Enjoying      bool              `json:"enjoying,omitempty"`
	Met           string            `json:"met,omitempty"`
	Held          bool              `json:"held,omitempty"`
	Pause         float64           `json:"pause,omitempty"`
	RaiseKeeper   string            `json:"raiseKeeper,omitempty"`
	Kube          KubeView          `json:"kube"`
	Tutorial      bool              `json:"tutorial,omitempty"`
}
