package kubectltree

import (
	"regexp"
	"strings"

	"github.com/waxb/kubernites/pkg/live/write"
)

const ReasonClientSecret = "a client-side run prints a Secret only as YAML or JSON, with its values masked: use -o yaml or -o json"

var secretKind = regexp.MustCompile(`(?m)(^\s*kind:\s*["']?Secret["']?\s*$|"kind"\s*:\s*"Secret")`)

func clientSide(args []string) bool {
	for _, a := range args {
		switch {
		case a == "--":
			return false
		case a == "--dry-run=client", a == "--local", a == "--local=true":
			return true
		}
	}
	return false
}

func templated(args []string) bool {
	for i, a := range args {
		if a == "--" {
			return false
		}
		format := ""
		switch {
		case a == "-o" || a == "--output":
			if i+1 < len(args) {
				format = args[i+1]
			}
		case strings.HasPrefix(a, "--output="):
			format = strings.TrimPrefix(a, "--output=")
		case strings.HasPrefix(a, "-o"):
			format = strings.TrimPrefix(strings.TrimPrefix(a, "-o"), "=")
		}
		for _, t := range []string{"jsonpath", "go-template", "template", "custom-columns"} {
			if strings.HasPrefix(format, t) {
				return true
			}
		}
	}
	return false
}

func makesSecrets(args []string) bool {
	for i, a := range args {
		switch {
		case a == "--":
			return false
		case a == "-f" || a == "--filename" || strings.HasPrefix(a, "--filename=") || strings.HasPrefix(a, "-f="):
			return true
		case a == "create" && i+1 < len(args) && strings.HasPrefix(args[i+1], "secret"):
			return true
		}
	}
	return false
}

func maskLocal(text string) string {
	if !secretKind.MatchString(text) {
		return text
	}
	if masked, ok := write.MaskSecrets(text, nil); ok {
		return masked
	}
	return "(a Secret, not shown: the Stone could not mask it)\n"
}
