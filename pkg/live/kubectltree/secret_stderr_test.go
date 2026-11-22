package kubectltree_test

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live/kubectltree"
)

func TestNoSecretValueReachesStderrInAnyFormKubectlPrints(t *testing.T) {
	for _, text := range []string{
		"Error from server (Forbidden): error when applying patch:\n{\"data\":{\"password\":\"U0VDUkVULVZBTFVF\"},\"metadata\":{\"annotations\":{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"v1\\\",\\\"data\\\":{\\\"password\\\":\\\"U0VDUkVULVZBTFVF\\\"},\\\"kind\\\":\\\"Secret\\\"}\\n\"}}}\nto:\nResource: \"/v1, Resource=secrets\", GroupVersionKind: \"/v1, Kind=Secret\"\nName: \"s\", Namespace: \"shop\"\nfor: \"secret.yaml\": error when patching \"secret.yaml\": refused\n",
		"error: unable to apply patch [{\"op\":\"replace\",\"path\":\"/data/password\",\"value\":\"U0VDUkVULVZBTFVF\"}]: refused\n",
		"error when creating \"secret.yaml\": stringData: {\"stringData\":{\"password\":\"SECRET-VALUE\"}} refused\n",
		"warning: the value SECRET-VALUE was typed in --from-literal\n",
	} {
		got := kubectltree.MaskErrors(text, []string{"SECRET-VALUE"})
		for _, v := range []string{"SECRET-VALUE", "U0VDUkVULVZBTFVF"} {
			if strings.Contains(got, v) {
				t.Errorf("stderr still holds %s:\n%s", v, got)
			}
		}
		if !strings.Contains(got, "****") {
			t.Errorf("nothing was masked:\n%s", got)
		}
	}
}
