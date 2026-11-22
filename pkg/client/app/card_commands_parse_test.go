package app

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	kubectlcmd "k8s.io/kubectl/pkg/cmd"

	"github.com/waxb/kubernites/pkg/client/ui"
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

func TestEveryCommandAHutCardBuildsForTheStoneParsesInKubectlsOwnTree(t *testing.T) {
	t.Setenv("KUBECTL_KUBERC", "false")
	t.Setenv("HOME", t.TempDir())
	withPorts := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "web-1", Namespace: "shop"}, Spec: corev1.PodSpec{Containers: []corev1.Container{
		{Name: "web", Ports: []corev1.ContainerPort{{ContainerPort: 8080}, {ContainerPort: 9090, Protocol: corev1.ProtocolTCP}}},
		{Name: "side", Ports: []corev1.ContainerPort{{ContainerPort: 53, Protocol: corev1.ProtocolUDP}}},
	}}}
	noPorts := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "job-1", Namespace: "batch"}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "work"}}}}
	defer ui.SetNamer(nil)
	for _, names := range []bool{false, true} {
		if names {
			ui.SetNamer(func(s string) string { return s })
		} else {
			ui.SetStoryNamer(func(s string) string { return s })
		}
		for _, p := range []*corev1.Pod{withPorts, noPorts} {
			for _, worker := range []string{"", p.Spec.Containers[0].Name} {
				for _, sp := range hutReachSpecs(p, worker) {
					if sp.stone == "" {
						continue
					}
					line := sp.stone
					if !sp.stoneRun {
						line += "8080"
					}
					if err := kubectlParses(line); err != nil {
						t.Errorf("NAMES %v, worker %q: the hut card's %s builds %q for the Stone, which kubectl v0.37.0 does not accept: %v", names, worker, sp.label, line, err)
					}
				}
			}
		}
	}
}
