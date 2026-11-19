package kubectltree

import (
	"errors"
	"strings"
)

var (
	ErrOpenQuote = errors.New("a quote is left open")
	ErrShell     = errors.New("the Stone runs one kubectl command: pipes, redirections and command lists belong to a shell")
)

func Words(line string) ([]string, error) {
	var out []string
	var cur strings.Builder
	inWord := false
	quote := rune(0)
	escaped, dqEscaped := false, false
	for _, r := range line {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case dqEscaped:
			if !strings.ContainsRune("\"\\$`", r) {
				cur.WriteRune('\\')
			}
			cur.WriteRune(r)
			dqEscaped = false
		case quote == '\'':
			if r == '\'' {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case quote == '"':
			switch r {
			case '"':
				quote = 0
			case '\\':
				dqEscaped = true
			default:
				cur.WriteRune(r)
			}
		case r == '\\':
			escaped, inWord = true, true
		case r == '\'' || r == '"':
			quote, inWord = r, true
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if inWord {
				out = append(out, cur.String())
				cur.Reset()
				inWord = false
			}
		case strings.ContainsRune("|;&<>`", r):
			return nil, ErrShell
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 || escaped || dqEscaped {
		return nil, ErrOpenQuote
	}
	if inWord {
		out = append(out, cur.String())
	}
	return out, nil
}
