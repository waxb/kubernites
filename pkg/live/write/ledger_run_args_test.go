package write

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestRunArgumentsNeverReachTheLedgerHoweverTheyAreTyped(t *testing.T) {
	dir := t.TempDir()
	ctx := Context{Name: "prod", Server: "https://prod.example:6443", User: "admin"}
	ledger := OpenLedger(dir, ctx)
	object := Ref{GVR: GVR{Version: "v1", Resource: "pods"}, Kind: "Pod", Namespace: "shop", Name: "vault"}.String()
	typed := map[string]string{
		"kubectl run -i --rm vault --image=hashicorp/vault:1.17 --restart=Never -n shop login hvs.NODASHTOKEN":                                                                                                            "hvs.NODASHTOKEN",
		"kubectl run -it --rm vault --image=hashicorp/vault:1.17 --restart=Never -n shop login hvs.BEFOREDASHTOKEN -- -method=token":                                                                                      "hvs.BEFOREDASHTOKEN",
		`kubectl run -it --rm db --image=mysql:8 --restart=Never -n shop --overrides='{"spec":{"containers":[{"name":"db","image":"mysql:8","stdin":true,"tty":true,"command":["mysql","-uroot","-pOVERRIDESECRET"]}]}}'`: "OVERRIDESECRET",
	}
	for order := range typed {
		session := Entry{At: time.Unix(0, 0), Context: ctx.Name, Level: "WRITES", Order: order, Object: object, Answer: "exited", Container: "vault", Ended: "exited"}
		if err := ledger.Append(session); err != nil {
			t.Fatal(err)
		}
		created := Entry{At: time.Unix(0, 0), Context: ctx.Name, Level: "WRITES", Order: order, Object: object, Code: 201, Answer: "OK"}
		if err := ledger.Append(created); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(ledger.Path())
	if err != nil {
		t.Fatal(err)
	}
	for order, secret := range typed {
		if strings.Contains(string(data), secret) {
			t.Errorf("the ledger keeps %q, an argument of the program kubectl run starts in the container, typed as %q", secret, order)
		}
	}
}
