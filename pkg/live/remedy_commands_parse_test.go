package live

import (
	"fmt"
	"strings"
	"testing"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	kubectlcmd "k8s.io/kubectl/pkg/cmd"

	"github.com/waxb/kubernites/pkg/live/kubectltree"
)

func kubectlParses(line string) error {
	args, err := kubectltree.Words(line)
	if err != nil {
		return err
	}
	if len(args) > 0 && args[0] == "kubectl" {
		args = args[1:]
	}
	root := kubectlcmd.NewKubectlCommand(kubectlcmd.KubectlOptions{ConfigFlags: genericclioptions.NewConfigFlags(true), IOStreams: genericiooptions.NewTestIOStreamsDiscard(), Arguments: append([]string{"kubectl"}, args...)})
	cmd, rest, err := root.Find(args)
	if err != nil {
		return err
	}
	if cmd == root {
		return fmt.Errorf("no kubectl command is named %v", args)
	}
	if err := cmd.ParseFlags(rest); err != nil {
		return err
	}
	if cmd.Args != nil {
		return cmd.Args(cmd, cmd.Flags().Args())
	}
	return nil
}

func TestEveryCommandACalamityCardShowsOrRunsParsesInKubectlsOwnTree(t *testing.T) {
	t.Setenv("KUBECTL_KUBERC", "false")
	t.Setenv("HOME", t.TempDir())
	seen := 0
	for _, c := range signalCases {
		s := calmSnapshot()
		c.break_(&s)
		for _, v := range Signals(&s) {
			for _, r := range v.Remedies {
				if r.Command == "" {
					continue
				}
				seen++
				if err := kubectlParses(strings.ReplaceAll(r.Command, "<image>", "busybox:1.37")); err != nil {
					t.Errorf("%s: the card's remedy %q is %q, which kubectl v0.37.0 does not accept: %v", c.name, r.Label, r.Command, err)
				}
			}
		}
	}
	if seen < len(signalCases) {
		t.Fatalf("only %d remedies for %d signals", seen, len(signalCases))
	}
}
