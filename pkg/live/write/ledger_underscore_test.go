package write

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestTheLedgerKeepsNoSecretValueBehindAnUnderscoreFlag(t *testing.T) {
	dir := t.TempDir()
	ctx := Context{Name: "prod", Server: "https://prod.example:6443", User: "admin"}
	ledger := OpenLedger(dir, ctx)
	object := Ref{GVR: GVR{Version: "v1", Resource: "secrets"}, Kind: "Secret", Namespace: "shop", Name: "db"}.String()
	cases := map[string]string{
		"kubectl create secret generic db -n shop --from_literal=password=hunter2A":                                                 "hunter2A",
		"kubectl create secret generic db -n shop --from_literal password=hunter2B":                                                 "hunter2B",
		"kubectl create secret docker-registry db -n shop --docker-server=r.example --docker-username=u --docker_password=hunter2C": "hunter2C",
	}
	for order := range cases {
		if err := ledger.Append(Entry{At: time.Unix(0, 0), Context: ctx.Name, Level: "WRITES", Order: order, Object: object, Code: 201, Answer: "OK"}); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(ledger.Path())
	if err != nil {
		t.Fatal(err)
	}
	for order, value := range cases {
		if strings.Contains(string(data), value) {
			t.Errorf("the ledger holds the Secret value %q typed in %s", value, order)
		}
	}
	if t.Failed() {
		t.Logf("ledger file:\n%s", data)
	}
}

func TestEverySecretValueFlagIsMaskedHoweverKubectlNormalizesItsName(t *testing.T) {
	dir := t.TempDir()
	ctx := Context{Name: "prod", Server: "https://prod.example:6443", User: "admin"}
	ledger := OpenLedger(dir, ctx)
	object := Ref{GVR: GVR{Version: "v1", Resource: "secrets"}, Kind: "Secret", Namespace: "shop", Name: "db"}.String()
	values := map[string]string{}
	n := 0
	for flag := range secretValueFlags {
		for _, spelling := range []string{flag, "--" + strings.ReplaceAll(flag[2:], "-", "_")} {
			for _, form := range []string{"=", " "} {
				n++
				value := fmt.Sprintf("hunter%dvalue", n)
				order := "kubectl create secret generic db -n shop " + spelling + form + "k=" + value
				values[order] = value
			}
		}
	}
	for _, flag := range []string{"--client_key_data", "--client_certificate_data", "--docker_password", "--password", "--token"} {
		n++
		value := fmt.Sprintf("credential%dvalue", n)
		values["kubectl config set-credentials x "+flag+"="+value] = value
	}
	for order := range values {
		if err := ledger.Append(Entry{At: time.Unix(0, 0), Context: ctx.Name, Level: "WRITES", Order: order, Object: object, Code: 201, Answer: "OK"}); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(ledger.Path())
	if err != nil {
		t.Fatal(err)
	}
	for order, value := range values {
		if strings.Contains(string(data), value) {
			t.Errorf("the ledger holds %q typed in %s", value, order)
		}
	}
	if FlagName("--from_env_file") != "--from-env-file" || FlagName("-p") != "-p" {
		t.Errorf("flag names are not normalized as kubectl's root does")
	}
}
