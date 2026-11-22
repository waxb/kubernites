package kubectltree

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

type TreeFlag struct {
	Command string
	Name    string
	Locked  bool
}

func TreeFlags(t testing.TB, tree *Tree) []TreeFlag {
	t.Helper()
	r := &run{}
	ios := genericiooptions.NewTestIOStreamsDiscard()
	ctx := context.Background()
	cf := tree.flags(ctx, ios, r, nil)
	root := kubectlcmd.NewKubectlCommand(kubectlcmd.KubectlOptions{ConfigFlags: cf, IOStreams: ios, Arguments: []string{"kubectl"}})
	tree.prune(root, cf, ios, r)
	tree.reachInto(root, cf, ios, r, ctx)
	tree.guardFiles(root, r)
	var out []TreeFlag
	walk(root, func(c *cobra.Command) {
		if c.DisableFlagParsing {
			return
		}
		c.Flags().VisitAll(func(f *pflag.Flag) {
			_, locked := f.Value.(lockedValue)
			out = append(out, TreeFlag{Command: c.CommandPath(), Name: f.Name, Locked: locked})
		})
	})
	return out
}

var FileFlags, TemplateFiles = fileFlags, templateFiles

func Queued(t *Terminal) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.keys)
}

func CopyBounds(askBytes int64, askFiles int, showBytes int64, showFiles int) func() {
	a, b, c, d := copyAskBytes, copyAskFiles, copyShowBytes, copyShowFiles
	copyAskBytes, copyAskFiles, copyShowBytes, copyShowFiles = askBytes, askFiles, showBytes, showFiles
	return func() { copyAskBytes, copyAskFiles, copyShowBytes, copyShowFiles = a, b, c, d }
}

var CopyAskBytes, CopyAskFiles = copyAskBytes, copyAskFiles

func QueuedKeys(t *Terminal) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.keys)
}

var MaskErrors = maskErrors

var MaskErrorsIn = maskErrorsIn

func (t *Tree) Protected(p string) bool {
	return t.protected(p)
}

func SetProtectedWalk(n int) func() {
	was := protectedWalk
	protectedWalk = n
	return func() { protectedWalk = was }
}
