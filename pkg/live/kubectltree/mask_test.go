package kubectltree_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func secretsTree(t *testing.T) (*kubectltree.Tree, *fakeServer) {
	t.Helper()
	f := newFake(t)
	path := kubeconfig(t, f.URL, "stone", "shop")
	g, err := write.New(write.Choice{Context: write.Context{Name: "stone", Server: f.URL}, Level: write.Seamless, Classes: []write.Class{write.Secrets}})
	if err != nil {
		t.Fatal(err)
	}
	return newTree(t, path, g.WrapConfig), f
}

func TestSecretValuesNeverReachTheStone(t *testing.T) {
	tree, f := secretsTree(t)
	for _, line := range []string{
		"get secret s -o yaml", "get secret s -o json", "get secrets -o yaml", "get secret s -o jsonpath={.data.password}",
		"get secret s -o go-template={{.data.password}}", "describe secret s", "get secret s -o jsonpath={.metadata.annotations}",
	} {
		res := run(t, tree, line)
		if res.Code != 0 {
			t.Fatalf("%s answered %d:\n%s", line, res.Code, res.Output)
		}
		for _, value := range []string{"aHVudGVyMg==", "hunter2", "Y2VydA=="} {
			if strings.Contains(res.Output, value) {
				t.Errorf("%s shows %s:\n%s", line, value, res.Output)
			}
		}
		if strings.Contains(line, "-o") && !strings.Contains(res.Output, kubectltree.MaskedData) && !strings.Contains(res.Output, kubectltree.Masked) {
			t.Errorf("%s shows no mask:\n%s", line, res.Output)
		}
	}
	for _, a := range f.accepts {
		if strings.Contains(a, "protobuf") {
			t.Errorf("a Secret was asked for as %s", a)
		}
	}
	j := tree.Start("get secrets -w -o json", nil)
	deadline := time.Now().Add(10 * time.Second)
	text := ""
	for !strings.Contains(text, kubectltree.MaskedData) && time.Now().Before(deadline) {
		chunks, _ := j.Take()
		for _, c := range chunks {
			text += c.Text
		}
		time.Sleep(10 * time.Millisecond)
	}
	j.Stop()
	if strings.Contains(text, "aHVudGVyMg==") || !strings.Contains(text, kubectltree.MaskedData) {
		t.Fatalf("the watch showed:\n%s", text)
	}
}

func TestRevealShowsOneKey(t *testing.T) {
	tree, _ := secretsTree(t)
	v, err := tree.Reveal(context.Background(), "shop", "s", "tls.crt")
	if err != nil || v != "Y2VydA==" {
		t.Fatalf("reveal tls.crt: %q %v", v, err)
	}
	if _, err := tree.Reveal(context.Background(), "shop", "s", "nope"); err == nil {
		t.Fatal("a missing key was revealed")
	}
	if res := run(t, tree, "get secret s -o yaml"); strings.Contains(res.Output, "Y2VydA==") {
		t.Fatal("a revealed key stays unmasked in later commands")
	}
}

func TestTheStonesMaskIsTheLedgers(t *testing.T) {
	if kubectltree.Masked != write.Masked {
		t.Fatalf("the Stone masks with %q, the write ledger with %q", kubectltree.Masked, write.Masked)
	}
}
