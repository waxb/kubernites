package live

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live/write"
)

func TestTheLedgerFileFollowsTheClusterAndKeepsItsFileWhenACredentialRotates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	listed := func(cluster, user string) write.Context {
		t.Helper()
		identityKubeconfig(t, path, cluster, user)
		list, err := ListContexts([]string{path})
		if err != nil || len(list) != 1 {
			t.Fatalf("%v %v", list, err)
		}
		return list[0].Remembered()
	}
	base := "    server: https://127.0.0.1:6443\n    certificate-authority-data: Y2Etb25l\n"
	first := listed(base, "    token: reader-token\n")
	ledgers := t.TempDir()
	old := write.LedgerFile(ledgers, write.Context{Name: first.Name, Server: first.Server, User: first.User})
	if err := os.WriteFile(old, []byte("{\"order\":\"earlier\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(old)
	if err := write.OpenLedger(ledgers, first).Append(write.Entry{At: time.Unix(0, 0), Context: first.Name, Order: "kubectl scale", Object: "deployment shop/web", Code: 200}); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(old); string(after) != string(before) {
		t.Errorf("the ledger file under the old key was touched:\n%s", after)
	}
	rotated := listed(base, "    token: rotated-token\n")
	if write.LedgerFile(ledgers, rotated) != write.LedgerFile(ledgers, first) {
		t.Error("a rotated token moved the cluster's ledger to another file")
	}
	for what, cluster := range map[string]string{
		"another certificate authority":   "    server: https://127.0.0.1:6443\n    certificate-authority-data: Y2EtdHdv\n",
		"another server certificate name": base + "    tls-server-name: other.internal\n",
	} {
		if write.LedgerFile(ledgers, listed(cluster, "    token: reader-token\n")) == write.LedgerFile(ledgers, first) {
			t.Errorf("a cluster with %s behind the same names shares the ledger file", what)
		}
	}
}
