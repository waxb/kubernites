package kubectltree_test

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestTheStoneAtWritesOnKind(t *testing.T) {
	cfg, bin := os.Getenv("KUBERNITES_STONE_KUBECONFIG"), os.Getenv("KUBERNITES_STONE_KUBECTL")
	if cfg == "" || bin == "" {
		t.Skip("KUBERNITES_STONE_KUBECONFIG and KUBERNITES_STONE_KUBECTL name a kind cluster this VM made and the kubectl binary of the same version")
	}
	raw, err := clientcmd.LoadFromFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	p := parityKind{t: t, config: cfg, context: raw.CurrentContext, bin: bin, home: t.TempDir()}
	p.setup()
	server := raw.Clusters[raw.Contexts[p.context].Cluster].Server
	gate, err := write.New(write.Choice{Context: write.Context{Name: p.context, Server: server, User: raw.Contexts[p.context].AuthInfo}, Level: write.Writes})
	if err != nil {
		t.Fatal(err)
	}
	rc, err := clientcmd.NewNonInteractiveClientConfig(*raw, p.context, &clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gate.Client(rc); err != nil {
		t.Fatal(err)
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{cfg}, Context: p.context, Wrap: gate.WrapConfig})
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Close()
	var asks []live.StoneAsk
	answer := false
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, gate, server, h, func(a live.StoneAsk) bool {
			asks = append(asks, a)
			return answer
		})
	})
	pod, _, _ := p.binary("get", "pods", "-n", "parity", "-l", "app=web-b", "-o", "jsonpath={.items[0].metadata.name}")
	line := "delete pod " + pod + " -n parity --wait=false"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	res := tree.Run(ctx, line)
	if len(asks) != 1 || asks[0].Why != write.NeedsConfirm || !asks[0].Preview.OK() || asks[0].Order.Request.Method != http.MethodDelete || len(asks[0].Order.Request.Body) == 0 {
		t.Fatalf("the Stone asked %+v", asks)
	}
	if res.Code == 0 || len(res.Writes) != 0 {
		t.Fatalf("CANCEL answered %d with writes %+v:\n%s", res.Code, res.Writes, res.Output)
	}
	if out, errOut, code := p.binary("get", "pod", pod, "-n", "parity", "-o", "jsonpath={.metadata.deletionTimestamp}"); code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("the delete's dry run touched the Pod: %q %s (exit %d)", out, errOut, code)
	}

	answer = true
	res = tree.Run(ctx, line)
	if res.Code != 0 || !strings.Contains(res.Output, `pod "`+pod+`" deleted`) {
		t.Fatalf("CONFIRM answered %d:\n%s", res.Code, res.Output)
	}
	if len(res.Writes) != 1 || res.Writes[0].Method != http.MethodDelete || res.Writes[0].Path != "/api/v1/namespaces/parity/pods/"+pod || res.Writes[0].Code != http.StatusOK {
		t.Fatalf("the tree reported %+v", res.Writes)
	}
	if out, _, _ := p.binary("get", "pod", pod, "-n", "parity", "--ignore-not-found", "-o", "jsonpath={.metadata.name}/{.metadata.deletionTimestamp}"); strings.TrimSpace(out) == pod+"/" {
		t.Fatalf("the confirmed delete left %s standing", pod)
	}
	t.Logf("asked %d times; CANCEL left %s, CONFIRM deleted it", len(asks), pod)
}
