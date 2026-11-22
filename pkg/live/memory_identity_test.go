package live

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/waxb/kubernites/pkg/live/write"
)

func sameNamesKubeconfig(t *testing.T, dir, name, ca, cert string) string {
	t.Helper()
	text := "apiVersion: v1\nkind: Config\nclusters:\n- name: default\n  cluster:\n    server: https://127.0.0.1:6443\n    certificate-authority-data: " + ca + "\n" +
		"users:\n- name: default\n  user:\n    client-certificate-data: " + cert + "\n    client-key-data: " + cert + "\n" +
		"contexts:\n- name: default\n  context:\n    cluster: default\n    user: default\ncurrent-context: default\n"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestARememberedLevelNeverCarriesToAnotherClusterBehindTheSameNames(t *testing.T) {
	dir := t.TempDir()
	laptop := sameNamesKubeconfig(t, dir, "laptop.yaml", "bGFwdG9wLWNh", "bGFwdG9wLWNlcnQ=")
	prod := sameNamesKubeconfig(t, dir, "prod.yaml", "cHJvZC1jYQ==", "cHJvZC1jZXJ0")
	first, err := ListContexts([]string{laptop})
	if err != nil || len(first) != 1 {
		t.Fatalf("laptop contexts %v %v", first, err)
	}
	second, err := ListContexts([]string{prod})
	if err != nil || len(second) != 1 {
		t.Fatalf("prod contexts %v %v", second, err)
	}
	asWrite := ContextInfo.Remembered
	var mem write.Memory
	mem.Keep(write.Remembered{Context: asWrite(first[0]), Level: write.Seamless, Warned: write.Seamless, Classes: write.AllClasses})
	if err := write.SaveMemory(dir, mem); err != nil {
		t.Fatal(err)
	}
	loaded, err := write.LoadMemory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if r, _, ok := loaded.Recall(asWrite(second[0])); ok {
		t.Errorf("a kubeconfig for another cluster (other CA and client certificate) recalls the laptop's choice: level %s, classes %v, no warning on untick", r.Level, r.Classes)
	}
}

func TestTheIdentityKeyFollowsTheCertificateAuthorityAndEveryCredentialSource(t *testing.T) {
	dir := t.TempDir()
	caFile := filepath.Join(dir, "ca.crt")
	if err := os.WriteFile(caFile, []byte("first-ca"), 0o600); err != nil {
		t.Fatal(err)
	}
	cluster := "    server: https://127.0.0.1:6443\n    certificate-authority-data: Y2E=\n"
	user := "    token: first-token\n"
	config := func(name, cluster, user string) string {
		text := "apiVersion: v1\nkind: Config\nclusters:\n- name: default\n  cluster:\n" + cluster +
			"users:\n- name: default\n  user:\n" + user +
			"contexts:\n- name: default\n  context:\n    cluster: default\n    user: default\ncurrent-context: default\n"
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	identity := func(path string) string {
		list, err := ListContexts([]string{path})
		if err != nil || len(list) != 1 {
			t.Fatalf("%s: %v %v", path, list, err)
		}
		if list[0].Identity == "" {
			t.Fatalf("%s has no identity", path)
		}
		return list[0].Identity
	}
	base := identity(config("base.yaml", cluster, user))
	if again := identity(config("copy.yaml", cluster, user)); again != base {
		t.Errorf("the same cluster and credentials in another file gave another identity")
	}
	viaFile := identity(config("ca-file.yaml", "    server: https://127.0.0.1:6443\n    certificate-authority: "+caFile+"\n", user))
	if err := os.WriteFile(caFile, []byte("second-ca"), 0o600); err != nil {
		t.Fatal(err)
	}
	if identity(config("ca-file.yaml", "    server: https://127.0.0.1:6443\n    certificate-authority: "+caFile+"\n", user)) == viaFile {
		t.Error("a certificate authority file that now holds another authority kept the identity")
	}
	for what, c := range map[string][2]string{
		"another certificate authority": {"    server: https://127.0.0.1:6443\n    certificate-authority-data: b3RoZXI=\n", user},
		"no TLS verification":           {"    server: https://127.0.0.1:6443\n    insecure-skip-tls-verify: true\n", user},
		"another token":                 {cluster, "    token: second-token\n"},
		"a token file":                  {cluster, "    tokenFile: /var/run/token\n"},
		"a client certificate":          {cluster, "    client-certificate-data: Y2VydA==\n    client-key-data: a2V5\n"},
		"an exec command":               {cluster, "    exec:\n      apiVersion: client.authentication.k8s.io/v1\n      command: aws\n      interactiveMode: Never\n"},
		"other exec arguments":          {cluster, "    exec:\n      apiVersion: client.authentication.k8s.io/v1\n      command: aws\n      args: [eks, get-token, --cluster-name, prod]\n      interactiveMode: Never\n"},
		"another exec environment":      {cluster, "    exec:\n      apiVersion: client.authentication.k8s.io/v1\n      command: aws\n      env: [{name: AWS_PROFILE, value: prod}]\n      interactiveMode: Never\n"},
		"a basic-auth user":             {cluster, "    username: admin\n    password: secret\n"},
	} {
		if identity(config("variant.yaml", c[0], c[1])) == base {
			t.Errorf("%s kept the identity", what)
		}
	}
}
