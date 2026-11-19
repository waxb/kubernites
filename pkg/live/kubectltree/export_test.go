package kubectltree

import (
	"context"
	"testing"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	kubectlcmd "k8s.io/kubectl/pkg/cmd"
)

func ConfigFor(t testing.TB, tree *Tree, line string) *genericclioptions.ConfigFlags {
	t.Helper()
	args, err := Words(line)
	if err != nil {
		t.Fatal(err)
	}
	r := &run{}
	ios := genericiooptions.NewTestIOStreamsDiscard()
	cf := tree.flags(context.Background(), ios, r, nil)
	root := kubectlcmd.NewKubectlCommand(kubectlcmd.KubectlOptions{ConfigFlags: cf, IOStreams: ios, Arguments: append([]string{"kubectl"}, args...)})
	tree.prune(root, cf, ios, r)
	return cf
}
