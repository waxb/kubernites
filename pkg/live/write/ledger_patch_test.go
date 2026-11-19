package write

import (
	"strings"
	"testing"
)

func TestAJSONPatchOrAnEscapedPatchOfASecretLeavesNoValueInTheLedger(t *testing.T) {
	for _, c := range []struct {
		order  string
		secret string
		keep   string
	}{
		{`kubectl patch secret db --type=json -p [{"op":"replace","path":"/data/password","value":"aHVudGVyMg=="}]`, "aHVudGVyMg==", `"path":"/data/password"`},
		{`kubectl patch secret db --type=json -p '[{"op":"test","path":"/stringData/token","value":"hunter5"},{"op":"remove","path":"/data/old"}]'`, "hunter5", `"op":"remove","path":"/data/old"`},
		{`kubectl patch secret db --type=json -p '[{"value":"hunter6","op":"add","path":"/data/user"}]'`, "hunter6", `"op":"add"`},
		{`kubectl patch secret db --type=json -p '[{"op":"add","path":"/data","value":{"password":"hunter7"}}]'`, "hunter7", `"path":"/data"`},
		{`kubectl patch secret db --type=json -p "[{\"op\":\"replace\",\"path\":\"/data/password\",\"value\":\"hunter8\"}]"`, "hunter8", `"path":"/data/password"`},
		{`kubectl patch secret db -p "{\"stringData\":{\"password\":\"hunter9\"}}"`, "hunter9", `"stringData"`},
		{`kubectl patch deployment web --type=json -p '[{"op":"replace","path":"/spec/replicas","value":3}]'`, "", `"value":3`},
	} {
		got := Mask(c.order)
		if c.secret != "" && strings.Contains(got, c.secret) {
			t.Errorf("Mask(%s) = %s keeps the value", c.order, got)
		}
		if !strings.Contains(got, c.keep) {
			t.Errorf("Mask(%s) = %s lost %s", c.order, got, c.keep)
		}
	}
}
