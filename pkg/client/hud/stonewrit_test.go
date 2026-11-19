package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/lexicon"
)

const describeAccountRaw = `Name:                default
Namespace:           guild
Labels:              <none>
Annotations:         <none>
Image pull secrets:  <none>
Events:              <none>
`

const getAccountsRaw = `NAMESPACE     NAME      AGE
guild         default   2m
kube-system   coredns   5m
`

func TestStoryModeReadsAServiceAccountAsAWrit(t *testing.T) {
	got := storyOut(t, describeAccountRaw)
	if leak := lexicon.KubernetesWords.FindString(got); leak != "" || strings.Contains(got, "pull") {
		t.Errorf("story describe of a writ keeps %q:\n%s", leak, got)
	}
	listed := storyOut(t, getAccountsRaw)
	if leak := lexicon.KubernetesWords.FindString(listed); leak != "" || !strings.Contains(listed, "coredns") {
		t.Errorf("story get writs keeps %q:\n%s", leak, listed)
	}
}
