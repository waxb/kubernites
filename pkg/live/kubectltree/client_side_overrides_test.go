package kubectltree_test

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
)

func TestASecretMadeByRunOrExposeOverridesShowsNoValueInATemplate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newFake(t)
	tree := stoneTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	secret := `'{"apiVersion":"v1","kind":"Secret","stringData":{"password":"hunter2"},"data":{"token":"aHVudGVyMg=="}}'`
	for _, source := range []string{
		"run x --image=busybox -n shop --dry-run=client --overrides=" + secret,
		"expose deployment web -n shop --port=80 --dry-run=client --overrides=" + secret,
	} {
		for _, format := range []string{"go-template={{.stringData.password}}", "jsonpath={.data.token}", "template={{.data}}", "custom-columns=P:.stringData.password"} {
			line := source + " -o " + format
			res := run(t, tree, line)
			if strings.Contains(res.Output, "hunter2") || strings.Contains(res.Output, "aHVudGVyMg==") {
				t.Errorf("a Secret made on this computer printed its value through a template at READ ONLY:\n%s\n%s", line, res.Output)
			}
		}
	}
}
