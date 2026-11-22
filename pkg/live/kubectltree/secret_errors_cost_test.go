package kubectltree_test

import (
	"context"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live/kubectltree"
)

func TestMaskingHeldErrorsTakesOneLinearPass(t *testing.T) {
	shapes := map[string]string{
		"brackets never closed":          "error: " + strings.Repeat("["+strings.Repeat("1,", 40), 5000),
		"closed nesting broken inside":   strings.Repeat("[1,", 20000) + "x" + strings.Repeat("]", 20000),
		"many small fragments":           strings.Repeat(`{"a":[1,2,{"b":"c"}]} `, 20000),
		"brackets in strings":            strings.Repeat(`"{[" `, 50000),
		"strings never closed":           strings.Repeat(`{"a`, 50000),
		"opening braces only":            strings.Repeat("{", 300000),
		"objects broken at their end":    strings.Repeat(`{"data":{"k":"djE="},"x":1 ! `, 10000),
		"keys broken before their colon": strings.Repeat(`{"data" "k"`, 40000),
	}
	for name, text := range shapes {
		_, work, done := kubectltree.MaskErrorsIn(context.Background(), text, []string{"hunter2!"})
		if !done || work > 10*len(text) {
			t.Errorf("%s: masking %d bytes of held errors did %d bytes of work (%.1f per byte), more than one linear pass allows", name, len(text), work, float64(work)/float64(len(text)))
		}
	}
	stopped, cancel := context.WithCancel(context.Background())
	cancel()
	out, _, done := kubectltree.MaskErrorsIn(stopped, shapes["many small fragments"], nil)
	if done || strings.Contains(out, `"b":"c"`) || !strings.Contains(out, "STOP") {
		t.Errorf("STOP did not end the masking of held errors: done %v, %q", done, out[:min(len(out), 200)])
	}
}

func TestOneLinearPassStillMasksEverySecretFragmentInHeldErrors(t *testing.T) {
	for in, secret := range map[string]string{
		`error: {"data":{"password":"aHVudGVyMg=="}}`:                                                                             "aHVudGVyMg==",
		`[1, {"stringData":{"password":"hunter-two"}}, oops]`:                                                                     "hunter-two",
		`patch [{"op":"replace","path":"/data/password","value":"cGF0Y2hlZA=="}] failed`:                                          "cGF0Y2hlZA==",
		`error: "unterminated {"data":{"password":"aW5zaWRl"}} more`:                                                              "aW5zaWRl",
		`{"metadata":{"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"data\":{\"p\":\"bGFzdA==\"}}"}} oops`: "bGFzdA==",
	} {
		out, _, done := kubectltree.MaskErrorsIn(context.Background(), in, nil)
		if !done || strings.Contains(out, secret) {
			t.Errorf("the held error %q kept %q: %q", in, secret, out)
		}
	}
}
