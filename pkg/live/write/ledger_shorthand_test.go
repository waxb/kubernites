package write

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestTheLedgerKeepsNoSecretValueBehindACombinedShorthand(t *testing.T) {
	dir := t.TempDir()
	ctx := Context{Name: "prod", Server: "https://prod.example:6443", User: "admin"}
	ledger := OpenLedger(dir, ctx)
	object := Ref{GVR: GVR{Version: "v1", Resource: "secrets"}, Kind: "Secret", Namespace: "shop", Name: "db"}.String()
	cases := map[string]string{
		"kubectl patch secret db -n shop -Rp 'stringData: {password: hunter2A}'":                            "hunter2A",
		"kubectl patch secret db -n shop -Rp'stringData: {password: hunter2B}'":                             "hunter2B",
		"kubectl patch secret db -n shop --type=json -Rp '[{op: add, path: /data/k, value: aHVudGVyQw==}]'": "aHVudGVyQw==",
	}
	for order := range cases {
		if err := ledger.Append(Entry{At: time.Unix(0, 0), Context: ctx.Name, Level: "WRITES", Order: order, Object: object, Code: 200, Answer: "OK"}); err != nil {
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

func TestEverySpellingPflagAcceptsForAPatchIsMaskedInTheLedgerLine(t *testing.T) {
	bodies := []string{`{"data":{"k":"VALUE-ONE"}}`, `stringData: {k: VALUE-ONE}`, "stringData:\n  k: VALUE-ONE", `[{op: add, path: /data/k, value: VALUE-ONE}]`, `[{"op":"add","path":"/data/k","value":"VALUE-ONE"}]`}
	spellings := []func(string) string{
		func(b string) string { return "-p " + quoteWord(b) },
		func(b string) string { return "-p" + quoteWord(b) },
		func(b string) string { return "-p=" + quoteWord(b) },
		func(b string) string { return "--patch " + quoteWord(b) },
		func(b string) string { return "--patch=" + quoteWord(b) },
		func(b string) string { return "-Rp " + quoteWord(b) },
		func(b string) string { return "-Rp" + quoteWord(b) },
		func(b string) string { return "-Rp=" + quoteWord(b) },
		func(b string) string { return "-wRp " + quoteWord(b) },
		func(b string) string { return "-patch=" + quoteWord(b) },
	}
	for _, body := range bodies {
		for _, spell := range spellings {
			typed := "kubectl patch secret db -n shop --type=json " + spell(body)
			if got := SecretCommand(typed); strings.Contains(got, "VALUE-ONE") {
				t.Errorf("%s\nreached the ledger as %s", typed, got)
			}
		}
	}
	for typed, want := range map[string]string{
		"kubectl patch secret db -n shop -o yaml":             "kubectl patch secret db -n shop -o yaml",
		"kubectl patch secret db -nshop -oyaml -p x":          "kubectl patch secret db -nshop -oyaml -p ****",
		"kubectl label secret db -n shop tier=front":          "kubectl label secret db -n shop tier=front",
		"kubectl create secret generic db --from-literal=a=b": "kubectl create secret generic db --from-literal=a=****",
	} {
		if got := SecretCommand(typed); got != want {
			t.Errorf("%s\n got %s\nwant %s", typed, got, want)
		}
	}
}
