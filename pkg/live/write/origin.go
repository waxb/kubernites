package write

import (
	"net/url"
	"strings"
)

func SameOrigin(u *url.URL, server *url.URL) bool {
	if u == nil || server == nil {
		return false
	}
	return scheme(u.Scheme) == scheme(server.Scheme) && strings.EqualFold(u.Host, server.Host)
}

func scheme(s string) string {
	switch strings.ToLower(s) {
	case "wss":
		return "https"
	case "ws":
		return "http"
	}
	return strings.ToLower(s)
}
