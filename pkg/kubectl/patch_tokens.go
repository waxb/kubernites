package kubectl

import (
	"strings"
)

func patchTokens(tokens []string) []string {
	if len(tokens) == 0 || tokens[0] != "patch" {
		return tokens
	}
	out := make([]string, len(tokens))
	for i, tok := range tokens {
		switch {
		case tok == "-p":
			out[i] = "--patch"
		case strings.HasPrefix(tok, "-p="):
			out[i] = "--patch=" + strings.TrimPrefix(tok, "-p=")
		default:
			out[i] = tok
		}
	}
	return out
}
