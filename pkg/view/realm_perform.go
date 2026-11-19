package view

import (
	"fmt"
	"github.com/waxb/kubernites/pkg/soulstone"
)

type Performed struct {
	Message  string   `json:"message"`
	Commands []string `json:"commands,omitempty"`
}

func DoorExposeCommand(kind, pod string) string {
	d, ok := soulstone.DepotFor(kind)
	if !ok {
		return ""
	}
	return fmt.Sprintf("kubectl expose pod %s --name=%s --port=%d --target-port=%d", pod, d.Service, d.Port, d.TargetPort)
}
