package view

const JobResources = "resources"

const JobProbes = "probes"

const JobHorn = "horn"

type SaveJob struct {
	Pain    string `json:"pain,omitempty"`
	Fix     string `json:"fix,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Crew    string `json:"crew,omitempty"`
	Command string `json:"command,omitempty"`
}

func (j SaveJob) Target() (string, string) {
	switch j.Kind {
	case JobResources:
		return "crew:default/" + j.Crew, ResButton
	case JobProbes:
		return "crew:default/" + j.Crew, WatchButton
	case JobHorn:
		return "crew:default/" + j.Crew, HornButton
	}
	return "", ""
}
