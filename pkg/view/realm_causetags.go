package view

const TagOOMKilled = "oomkilled"

const TagThrottled = "throttled"

const TagEvicted = "evicted"

const TagCrashLoop = "crashloop"

const TagImagePull = "imagepull"

const TagUnhoused = "unhoused"

type CauseTag struct {
	Pod  string `json:"pod"`
	Node string `json:"node,omitempty"`
	Crew string `json:"crew,omitempty"`
	Tag  string `json:"tag"`
	Text string `json:"text"`
	Why  string `json:"why,omitempty"`
}
