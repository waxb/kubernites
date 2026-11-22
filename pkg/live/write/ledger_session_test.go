package write

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestNoPasswordOrTokenTypedInAContainerCommandReachesTheLedger(t *testing.T) {
	dir := t.TempDir()
	ctx := Context{Name: "prod", Server: "https://prod.example:6443", User: "admin"}
	ledger := OpenLedger(dir, ctx)
	object := Ref{GVR: GVR{Version: "v1", Resource: "pods"}, Kind: "Pod", Namespace: "shop", Name: "db-0"}.String()
	cases := map[string]string{
		"mysql -uroot -pHUNTER2SECRET":                               "HUNTER2SECRET",
		"vault login hvs.CAESIJTOKENVALUE":                           "hvs.CAESIJTOKENVALUE",
		"curl -H 'Authorization: Bearer eyJhbGciOiJSUzI1NiJ9.TOKEN'": "eyJhbGciOiJSUzI1NiJ9.TOKEN",
	}
	for command := range cases {
		e := Entry{At: time.Unix(0, 0), Context: ctx.Name, Level: "WRITES", Order: "kubectl exec -it db-0 -n shop -- " + command, Object: object, Answer: "exited", Container: "db", Command: command, Ended: "exited"}
		if err := ledger.Append(e); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(ledger.Path())
	if err != nil {
		t.Fatal(err)
	}
	for command, value := range cases {
		if strings.Contains(string(data), value) {
			t.Errorf("the ledger file holds the credential %q typed in the container command %q", value, command)
		}
	}
}

func TestASessionsLedgerLineKeepsTheKubectlCommandAndTheProgramOnly(t *testing.T) {
	dir := t.TempDir()
	ctx := Context{Name: "prod", Server: "https://prod.example:6443", User: "admin"}
	ledger := OpenLedger(dir, ctx)
	object := Ref{GVR: GVR{Version: "v1", Resource: "pods"}, Kind: "Pod", Namespace: "shop", Name: "db-0"}.String()
	sessions := []Entry{
		{Order: "kubectl exec -it db-0 -n shop -- mysql -uroot -pHUNTER2SECRET", Container: "db", Command: "mysql -uroot -pHUNTER2SECRET", Ended: "exited 0"},
		{Order: `kubectl exec db-0 -n shop -c db -- sh -c "vault login hvs.SECRETVALUE"`, Container: "db", Command: "sh -c vault login hvs.SECRETVALUE", Ended: "exited 0"},
		{Order: "kubectl run tmp -it --rm --image=mysql -n shop -- mysql -pRUNSECRET", Container: "tmp", Command: "mysql -pRUNSECRET", Ended: "exited 0"},
		{Order: "kubectl debug db-0 -it --image=busybox -n shop -- wget --header=Authorization:DEBUGSECRET", Container: "debugger", Command: "wget --header=Authorization:DEBUGSECRET", Ended: "exited 0"},
		{Order: "kubectl exec -it db-0 -n shop -- sh -c 'unclosed SPLITSECRET", Container: "db", Command: "sh", Ended: "exited 0"},
	}
	for _, e := range sessions {
		e.At, e.Context, e.Level, e.Object, e.Answer = time.Unix(0, 0), ctx.Name, "WRITES", object, e.Ended
		if err := ledger.Append(e); err != nil {
			t.Fatal(err)
		}
	}
	forward := Entry{At: time.Unix(0, 0), Context: ctx.Name, Level: "WRITES", Order: "kubectl port-forward db-0 -n shop :8080 :9090", Object: object, Command: ":8080 :9090", Ended: "open on 127.0.0.1:41000, 127.0.0.1:41001"}
	if err := ledger.Append(forward); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(ledger.Path())
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"HUNTER2SECRET", "SECRETVALUE", "RUNSECRET", "DEBUGSECRET", "SPLITSECRET", "-uroot", "vault"} {
		if strings.Contains(string(data), secret) {
			t.Errorf("the ledger keeps %q, an argument of a container's program", secret)
		}
	}
	for _, kept := range []string{`kubectl exec -it db-0 -n shop -- mysql"`, `"command":"mysql"`, `-c db -- sh"`, `"command":"sh"`, `-- wget"`, `"order":"kubectl port-forward db-0 -n shop :8080 :9090"`, `"command":":8080 :9090"`} {
		if !strings.Contains(string(data), kept) {
			t.Errorf("the ledger lost %s:\n%s", kept, data)
		}
	}
}
