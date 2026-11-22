package live

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"k8s.io/client-go/rest"

	"github.com/waxb/kubernites/pkg/live/write"
)

func entriesKubeconfig(t *testing.T, path, cluster, user string) {
	t.Helper()
	text := "apiVersion: v1\nkind: Config\nclusters:\n- name: prod\n  cluster:\n    server: https://127.0.0.1:6443\n" + cluster +
		"users:\n- name: prod\n  user:\n" + user +
		"contexts:\n- name: prod\n  context:\n    cluster: prod\n    user: prod\n    namespace: shop\ncurrent-context: prod\n"
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestTheIdentityFollowsEveryFieldThatDecidesWhereRequestsGoOrWhomTheyActAs(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{"ca": "ca-one", "cert": "cert-one", "key": "key-one", "token": "token-one"}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	f := func(name string) string { return filepath.Join(dir, name) }
	baseCluster := "    certificate-authority: " + f("ca") + "\n"
	baseUser := "    client-certificate: " + f("cert") + "\n    client-key: " + f("key") + "\n    tokenFile: " + f("token") + "\n"
	execUser := "    exec:\n      apiVersion: client.authentication.k8s.io/v1\n      command: get-token\n      args: [--cluster, prod]\n      env:\n      - name: PROFILE\n        value: dev\n      provideClusterInfo: false\n      interactiveMode: Never\n"
	path := filepath.Join(dir, "config")
	identity := func(cluster, user string) string {
		t.Helper()
		entriesKubeconfig(t, path, cluster, user)
		list, err := ListContexts([]string{path})
		if err != nil || len(list) != 1 {
			t.Fatalf("%v %v", list, err)
		}
		return list[0].Identity
	}
	base := identity(baseCluster, baseUser)
	execBase := identity(baseCluster, execUser)
	changes := []struct {
		what, cluster, user string
		exec                bool
	}{
		{what: "tls-server-name", cluster: baseCluster + "    tls-server-name: other.internal\n", user: baseUser},
		{what: "insecure-skip-tls-verify", cluster: "    insecure-skip-tls-verify: true\n", user: baseUser},
		{what: "certificate-authority-data in place of the file", cluster: "    certificate-authority-data: Y2EtdHdv\n", user: baseUser},
		{what: "proxy-url", cluster: baseCluster + "    proxy-url: http://proxy.internal:3128\n", user: baseUser},
		{what: "client-certificate-data in place of the file", cluster: baseCluster, user: "    client-certificate-data: Y2VydC10d28=\n    client-key: " + f("key") + "\n    tokenFile: " + f("token") + "\n"},
		{what: "client-key-data in place of the file", cluster: baseCluster, user: "    client-certificate: " + f("cert") + "\n    client-key-data: a2V5LXR3bw==\n    tokenFile: " + f("token") + "\n"},
		{what: "a token beside the token file", cluster: baseCluster, user: baseUser + "    token: inline\n"},
		{what: "username", cluster: baseCluster, user: baseUser + "    username: alice\n"},
		{what: "password", cluster: baseCluster, user: baseUser + "    username: \"\"\n    password: secret\n"},
		{what: "act-as", cluster: baseCluster, user: baseUser + "    as: system:admin\n"},
		{what: "act-as-uid", cluster: baseCluster, user: baseUser + "    as: alice\n    as-uid: \"42\"\n"},
		{what: "act-as-groups", cluster: baseCluster, user: baseUser + "    as-groups: [system:masters]\n"},
		{what: "act-as-user-extra", cluster: baseCluster, user: baseUser + "    as-user-extra:\n      scopes: [admin]\n"},
		{what: "auth-provider", cluster: baseCluster, user: baseUser + "    auth-provider:\n      name: oidc\n      config:\n        client-id: kubernetes\n"},
		{what: "exec apiVersion", cluster: baseCluster, user: strings.Replace(execUser, "client.authentication.k8s.io/v1", "client.authentication.k8s.io/v1beta1", 1), exec: true},
		{what: "exec command", cluster: baseCluster, user: strings.Replace(execUser, "get-token", "get-admin-token", 1), exec: true},
		{what: "exec args", cluster: baseCluster, user: strings.Replace(execUser, "--cluster, prod", "--cluster, prod, --admin", 1), exec: true},
		{what: "exec env", cluster: baseCluster, user: strings.Replace(execUser, "value: dev", "value: prod", 1), exec: true},
		{what: "exec provideClusterInfo", cluster: baseCluster, user: strings.Replace(execUser, "provideClusterInfo: false", "provideClusterInfo: true", 1), exec: true},
		{what: "the cluster's exec extension", cluster: baseCluster + "    extensions:\n    - name: client.authentication.k8s.io/exec\n      extension:\n        audience: admin\n", user: execUser, exec: true},
	}
	for _, c := range changes {
		want := base
		if c.exec {
			want = execBase
		}
		if identity(c.cluster, c.user) == want {
			t.Errorf("%s changed and the identity did not", c.what)
		}
	}
	for name, body := range map[string]string{"ca": "ca-two", "cert": "cert-two", "key": "key-two", "token": "token-two"} {
		if err := os.WriteFile(f(name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if identity(baseCluster, baseUser) == base {
			t.Errorf("the %s file now holds other bytes and the identity did not change", name)
		}
		if err := os.WriteFile(f(name), []byte(files[name]), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if identity(baseCluster, baseUser) != base {
		t.Fatal("the same entries gave another identity")
	}
	moved := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(moved, []byte(files["token"]), 0o600); err != nil {
		t.Fatal(err)
	}
	if identity(baseCluster+"    disable-compression: true\n", strings.Replace(baseUser, f("token"), moved, 1)) != base {
		t.Error("the same credentials in a token file at another path, or compression turned off, changed the identity: nothing about where requests go or whom they act as changed")
	}
}

func TestAConnectionHoldsItsCredentialsAsDataReadOnceAtConnect(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("Authorization"))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"major":"1","minor":"37","gitVersion":"v1.37.0"}`))
	}))
	t.Cleanup(srv.Close)
	dir := t.TempDir()
	ca := filepath.Join(dir, "ca.crt")
	if err := os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw}), 0o600); err != nil {
		t.Fatal(err)
	}
	tokenFile := filepath.Join(dir, "token")
	if err := os.WriteFile(tokenFile, []byte("reader-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config")
	text := "apiVersion: v1\nkind: Config\nclusters:\n- name: c\n  cluster:\n    server: " + srv.URL + "\n    certificate-authority: " + ca + "\n" +
		"users:\n- name: u\n  user:\n    tokenFile: " + tokenFile + "\n" +
		"contexts:\n- name: c\n  context:\n    cluster: c\n    user: u\ncurrent-context: c\n"
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	listed, err := ListContexts([]string{path})
	if err != nil || len(listed) != 1 {
		t.Fatalf("%v %v", listed, err)
	}
	gate, err := write.New(write.Choice{Context: listed[0].Remembered(), Level: write.Writes})
	if err != nil {
		t.Fatal(err)
	}
	read, err := Dial([]string{path}, "c")
	if err != nil {
		t.Fatal(err)
	}
	gated, err := DialGated([]string{path}, "c", gate)
	if err != nil {
		t.Fatal(err)
	}
	if gated.Context.Identity != listed[0].Identity {
		t.Errorf("the gated connection carries identity %q, not %q", gated.Context.Identity, listed[0].Identity)
	}
	if err := os.WriteFile(tokenFile, []byte("admin-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, conn := range map[string]*Conn{"READ ONLY": read, "WRITES": gated} {
		for _, rc := range []*rest.Config{conn.rest, PinnedTo(conn.rest, func(c *rest.Config) *rest.Config { return c })(rest.CopyConfig(conn.rest))} {
			if rc.BearerTokenFile != "" || rc.CertFile != "" || rc.KeyFile != "" || rc.CAFile != "" {
				t.Errorf("%s: the connected config still names files a later rewrite would change: token %q cert %q key %q ca %q", name, rc.BearerTokenFile, rc.CertFile, rc.KeyFile, rc.CAFile)
			}
			if rc.BearerToken != "reader-token" || len(rc.CAData) == 0 {
				t.Errorf("%s: the connected config does not hold the credentials read at connect", name)
			}
		}
		if _, err := conn.read(context.Background(), "/version", nil, "application/json"); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	for _, h := range seen {
		if h != "Bearer reader-token" {
			t.Errorf("a request after the token file was rewritten carried %q", h)
		}
	}
}
