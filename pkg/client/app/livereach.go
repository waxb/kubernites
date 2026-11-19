package app

import (
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/live/write"
)

const shellCheck = "command -v bash >/dev/null && exec bash || exec sh"

func reachAccess(sub, verb string, p *corev1.Pod) write.Access {
	return write.Access{Verb: verb, Version: "v1", Resource: "pods", Subresource: sub, Namespace: p.Namespace, Name: p.Name}
}

func forwardPortsOf(p *corev1.Pod) []string {
	var out []string
	for _, c := range p.Spec.Containers {
		for _, port := range c.Ports {
			if port.Protocol != "" && port.Protocol != corev1.ProtocolTCP {
				continue
			}
			spec := ":" + strconv.Itoa(int(port.ContainerPort))
			if !strings.Contains(" "+strings.Join(out, " ")+" ", " "+spec+" ") {
				out = append(out, spec)
			}
		}
	}
	return out
}

func hutReachSpecs(p *corev1.Pod, worker string) []orderSpec {
	ns := " -n " + p.Namespace
	if worker != "" {
		ns += " -c " + worker
	}
	quiet := ""
	if !ui.Naming() {
		quiet = " -q"
	}
	logs := orderSpec{key: "logs", label: "LOGS", icon: "icon/logs", cell: hud.CentreCell, chord: "V", replaces: []string{"sel:logs"}, read: true,
		stone: "kubectl logs -f " + p.Name + ns + " --tail=200", stoneRun: true, verb: "follow its logs",
		tip:    ui.Bridge("Follow what this hut's worker says, as it says it, in the Speaking Stone. Esc stops.", "kubectl logs -f: follow the container's log in the Stone. Esc stops it."),
		access: reachAccess("log", "get", p)}
	shell := orderSpec{key: "shell", label: "SHELL", icon: "icon/stone", cell: 3, chord: "Y", stoneRun: true, verb: "open a shell in it",
		stone:  "kubectl exec -it " + p.Name + ns + quiet + " -- sh -c \"" + shellCheck + "\"",
		tip:    ui.Bridge("Open a shell inside this hut's worker, in the Speaking Stone. ctrl-p ctrl-q leaves.", "kubectl exec -it: a shell in the container (bash, else sh), in the Stone's terminal. ctrl-p ctrl-q leaves."),
		access: reachAccess("exec", "create", p)}
	more := orderSpec{key: orderMore, label: "MORE", icon: "icon/tech", cell: 7, tip: ui.Bridge("More orders for this hut", "More orders for this Pod")}
	attach := orderSpec{key: "attach", label: "ATTACH", icon: "icon/thread", cell: 0, page: 1, chord: "J", stoneRun: true, verb: "attach to it",
		stone:  "kubectl attach -it " + p.Name + ns + quiet,
		tip:    ui.Bridge("Join this hut's worker where it already runs, in the Speaking Stone. ctrl-p ctrl-q leaves; the worker carries on.", "kubectl attach -it: join the container's running process in the Stone's terminal. ctrl-p ctrl-q leaves; the container runs on."),
		access: reachAccess("attach", "create", p)}
	ports := forwardPortsOf(p)
	forward := orderSpec{key: "port-forward", label: "FORWARD", icon: "icon/key-gate", cell: 1, page: 1, chord: "F", verb: "forward a port to it",
		stone:  "kubectl port-forward " + p.Name + ns + " " + strings.Join(ports, " "),
		tip:    ui.Bridge("Forward this hut's ports to 127.0.0.1 on this computer. The live card lists the forward with STOP.", "kubectl port-forward: listen on 127.0.0.1 only, forwarding to the Pod's ports. The live card lists it with STOP."),
		access: reachAccess("portforward", "create", p), stoneRun: len(ports) > 0}
	if len(ports) == 0 {
		forward.stone = strings.TrimSpace(forward.stone) + " "
		forward.tip += ui.Bridge(" This hut names no port: type one in the Stone.", " This Pod declares no port: type one in the Stone.")
	}
	return []orderSpec{logs, shell, more, attach, forward}
}

func (s *Session) runStoneOrder(sp orderSpec) []hud.Intent {
	if s.Hud.StoneRunning() {
		s.Hud.Toast(ui.Bridge("The Speaking Stone is busy: stop it first (Esc).", "The Stone is running a command: stop it first (Esc)."), true)
		return nil
	}
	s.Hud.RunStone(sp.stone, sp.stoneRun)
	return []hud.Intent{{Kind: hud.IntentSound, Command: "click"}}
}
