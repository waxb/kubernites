package write

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"k8s.io/client-go/rest"
)

var userinfo = regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.\-]*://)[A-Za-z0-9\-._:~!$&'()*+,;=%@]*@`)

func Redact(text string) string {
	return userinfo.ReplaceAllString(text, "$1")
}

var authorityStart = regexp.MustCompile(`^\s*(?:[A-Za-z][A-Za-z0-9+.\-]*:)?//`)

func splitServer(server string) (string, string, string) {
	head := authorityStart.FindString(server)
	tail := server[len(head):]
	end := strings.IndexAny(tail, "/?#")
	if end < 0 {
		end = len(tail)
	}
	at := strings.LastIndex(tail[:end], "@") + 1
	return head, tail[:at], tail[at:]
}

func RedactServer(server string) string {
	head, _, rest := splitServer(server)
	return head + rest
}

func ServerUserinfo(server string) string {
	_, info, _ := splitServer(server)
	return info
}

func ServerURL(c *rest.Config) (*url.URL, string, error) {
	u, versioned, err := rest.DefaultServerUrlFor(c)
	if err != nil {
		return nil, "", fmt.Errorf("the server address %q is not one Kubernetes can use", RedactServer(c.Host))
	}
	return u, versioned, nil
}
