package write

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestTheLedgerKeepsNoSecretValueTypedInAnyQuotingForm(t *testing.T) {
	dir := t.TempDir()
	ctx := Context{Name: "prod", Server: "https://prod.example:6443", User: "admin"}
	ledger := OpenLedger(dir, ctx)
	object := Ref{GVR: GVR{Version: "v1", Resource: "secrets"}, Kind: "Secret", Namespace: "shop", Name: "db"}.String()
	cases := map[string]string{
		"kubectl create secret generic db -n shop --from-literal=password='S0me$ecretA'":                  "S0me$ecretA",
		`kubectl create secret generic db -n shop --from-literal=password="hunter two B"`:                 "hunter two B",
		"kubectl create secret generic db -n shop --from-literal password='S0me$ecretC'":                  "S0me$ecretC",
		"kubectl patch secret db -n shop -p'data: {password: aHVudGVyRA==}'":                              "aHVudGVyRA==",
		"kubectl patch secret db -n shop --type=json -p'[{op: add, path: /data/k, value: aHVudGVyRQ==}]'": "aHVudGVyRQ==",
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

func TestASecretWritesLedgerLineIsBuiltFromItsWords(t *testing.T) {
	for typed, want := range map[string]string{
		`kubectl create secret generic db -n shop --from-literal=password='S0me$ecret' --from-literal "user=it's me"`: "kubectl create secret generic db -n shop --from-literal=password=**** --from-literal user=****",
		`kubectl create secret generic db --from-literal="a b=c d"`:                                                   "kubectl create secret generic db '--from-literal=a b=****'",
		`kubectl patch secret db -p'data: {password: aHVudGVy}'`:                                                      "kubectl patch secret db -p****",
		`kubectl patch secret db -p='{"data":{"k":"aHVudGVy"}}'`:                                                      "kubectl patch secret db -p=****",
		`kubectl patch secret db --patch "stringData: {k: hunter}"`:                                                   "kubectl patch secret db --patch ****",
		`kubectl patch secret db --type=json --patch='[{"op":"add","path":"/data/k","value":"aHVudGVy"}]'`:            "kubectl patch secret db --type=json --patch=****",
		`kubectl create secret generic db --from-file=key=./id_rsa --from-env-file .env`:                              "kubectl create secret generic db --from-file=key=**** --from-env-file ****",
		`kubectl create secret docker-registry r --docker-password='p w' --docker-username=me`:                        "kubectl create secret docker-registry r --docker-password=**** --docker-username=me",
		`kubectl create secret generic db --from-literal=k='open`:                                                     "****",
	} {
		if got := SecretCommand(typed); got != want {
			t.Errorf("%s\n got %s\nwant %s", typed, got, want)
		}
	}
}

func TestASecretCommandIsSplitIntoWordsAndNeverMatchedAsRawText(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "ledger.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	split := false
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !slices.Contains([]string{"SecretCommand", "maskedValue", "quoteWord"}, fn.Name.Name) {
			continue
		}
		ast.Inspect(fn, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "Words" {
					split = true
				}
				if sel, ok := x.Fun.(*ast.SelectorExpr); ok && (strings.Contains(sel.Sel.Name, "Match") || strings.Contains(sel.Sel.Name, "Find") || strings.HasPrefix(sel.Sel.Name, "ReplaceAll") && sel.Sel.Name != "ReplaceAll") {
					t.Errorf("%s matches raw text with %s", fn.Name.Name, sel.Sel.Name)
				}
			}
			return true
		})
	}
	if !split {
		t.Error("SecretCommand does not split the command with the Stone's word splitter")
	}
}
