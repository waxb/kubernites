package hud

import "strings"

func BareCommand(command string) string {
	cmd := strings.TrimSpace(command)
	if strings.HasPrefix(cmd, etcdctlVerb) {
		return cmd
	}
	return strings.TrimPrefix(cmd, kubectlVerb)
}
