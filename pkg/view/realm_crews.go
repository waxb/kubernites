package view

import (
	"fmt"
	"github.com/waxb/kubernites/pkg/soulstone"
)

type CommandOffer struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Command string `json:"command"`
	Action  Action `json:"action"`
}

func CrewCommand(name, image string, replicas int) string {
	cmd := fmt.Sprintf("kubectl create deployment %s --image=%s --replicas=%d", name, image, replicas)
	if p, ok := soulstone.Lookup(image); ok && p.Port > 0 {
		cmd += fmt.Sprintf(" --port=%d", p.Port)
	}
	return cmd
}

func DepotCommand(kind string) string {
	d, ok := soulstone.DepotFor(kind)
	if !ok {
		return ""
	}
	return fmt.Sprintf("kubectl expose deployment %s --name=%s --port=%d --target-port=%d", d.Crew, d.Service, d.Port, d.TargetPort)
}
