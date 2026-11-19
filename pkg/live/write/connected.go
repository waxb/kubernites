package write

import (
	"bytes"
	"net/url"
	"reflect"
	"strings"

	"k8s.io/client-go/rest"
)

func Elsewhere(cfg, connected *rest.Config) string {
	if !sameServer(cfg.Host, connected.Host) || cfg.APIPath != connected.APIPath && cfg.APIPath != "" && connected.APIPath != "" {
		return "the server " + cfg.Host + " is not " + connected.Host + ", the one resolved at connect"
	}
	a, b := cfg.TLSClientConfig, connected.TLSClientConfig
	if a.Insecure != b.Insecure || a.ServerName != b.ServerName || a.CAFile != b.CAFile || !bytes.Equal(a.CAData, b.CAData) {
		return "the server's TLS settings are not the ones resolved at connect"
	}
	if cfg.BearerToken != connected.BearerToken || cfg.BearerTokenFile != connected.BearerTokenFile || cfg.Username != connected.Username || cfg.Password != connected.Password ||
		a.CertFile != b.CertFile || a.KeyFile != b.KeyFile || !bytes.Equal(a.CertData, b.CertData) || !bytes.Equal(a.KeyData, b.KeyData) ||
		!reflect.DeepEqual(cfg.Impersonate, connected.Impersonate) || !reflect.DeepEqual(cfg.AuthProvider, connected.AuthProvider) || !reflect.DeepEqual(cfg.ExecProvider, connected.ExecProvider) {
		return "the credentials are not the ones resolved at connect"
	}
	return ""
}

func sameServer(a, b string) bool {
	ua, errA := url.Parse(strings.TrimRight(a, "/"))
	ub, errB := url.Parse(strings.TrimRight(b, "/"))
	if errA != nil || errB != nil || a == "" || b == "" {
		return false
	}
	return strings.EqualFold(ua.Scheme, ub.Scheme) && strings.EqualFold(ua.Host, ub.Host) && ua.Path == ub.Path
}
