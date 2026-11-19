package write

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

const secretYAML = `apiVersion: v1
data:
  password: aHVudGVyMg==
  user: YWRtaW4=
kind: Secret
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: '{"data":{"password":"aHVudGVyMg=="}}'
  name: db
  namespace: shop
type: Opaque
`

func TestSecretValuesAreMaskedUntilRevealedKeyByKey(t *testing.T) {
	out, masked := MaskSecrets(secretYAML, nil)
	if !masked || strings.Contains(out, "aHVudGVyMg") || strings.Contains(out, "YWRtaW4") || !strings.Contains(out, "password: '"+Masked+"'") {
		t.Fatalf("masked %v:\n%s", masked, out)
	}
	out, _ = MaskSecrets(secretYAML, func(ns, name, key string) bool { return ns == "shop" && name == "db" && key == "user" })
	if !strings.Contains(out, "YWRtaW4=") || strings.Contains(out, "aHVudGVyMg") {
		t.Fatalf("revealing one key:\n%s", out)
	}
	list := `{"apiVersion":"v1","kind":"List","items":[{"kind":"Secret","metadata":{"name":"a"},"stringData":{"token":"abc"}},{"kind":"ConfigMap","metadata":{"name":"b"},"data":{"color":"blue"}}]}`
	out, masked = MaskSecrets(list, nil)
	if !masked || strings.Contains(out, "abc") || !strings.Contains(out, "blue") {
		t.Fatalf("a list:\n%s", out)
	}
	cm := "apiVersion: v1\nkind: ConfigMap\ndata:\n  color: blue\n"
	if out, masked := MaskSecrets(cm, nil); masked || out != cm {
		t.Fatalf("a ConfigMap was changed:\n%s", out)
	}
	if out, masked := MaskSecrets("NAME   TYPE     DATA   AGE\ndb     Opaque   2      1d\n", nil); masked || !strings.Contains(out, "Opaque") {
		t.Fatalf("a table was changed:\n%s", out)
	}
}

func TestAnAnswerUnderSecretsCarriesNoValueWhateverTheMethod(t *testing.T) {
	const secret = `{"apiVersion":"v1","kind":"Secret","metadata":{"name":"s","namespace":"shop","annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"data\":{\"password\":\"aHVudGVyMg==\"}}"}},"data":{"password":"aHVudGVyMg=="},"stringData":{"token":"hunter2"}}`
	f := &fakeServer{answer: func(r *http.Request, body string) (int, string) {
		if strings.HasSuffix(r.URL.Path, "/secrets") {
			return http.StatusCreated, `{"apiVersion":"v1","kind":"SecretList","items":[` + secret + `]}`
		}
		return http.StatusOK, secret
	}}
	g := bound(t, Seamless, f, Secrets)
	for _, r := range []Request{
		{Method: http.MethodPatch, Path: "/api/v1/namespaces/shop/secrets/s", Body: []byte(`{"metadata":{"labels":{"a":"b"}}}`)},
		{Method: http.MethodPut, Path: "/api/v1/namespaces/shop/secrets/s", Body: []byte(secret)},
		{Method: http.MethodDelete, Path: "/api/v1/namespaces/shop/secrets/s"},
		{Method: http.MethodPost, Path: "/api/v1/namespaces/shop/secrets", Body: []byte(secret)},
	} {
		o := Order{Request: r, ContentType: JSONBody}
		preview, _ := g.Preview(context.Background(), o)
		sent := g.Send(context.Background(), o)
		for _, a := range []Answer{preview, sent} {
			if !a.OK() {
				t.Fatalf("%s %s: %v", r.Method, r.Path, a.Text())
			}
			for _, v := range []string{"aHVudGVyMg==", "hunter2"} {
				if strings.Contains(string(a.Body), v) || strings.Contains(a.Text(), v) {
					t.Errorf("%s %s answered with %s: %s", r.Method, r.Path, v, a.Body)
				}
			}
		}
	}
}
