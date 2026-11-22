package kubectltree

import "github.com/waxb/kubernites/pkg/live/write"

var (
	ErrOpenQuote = write.ErrOpenQuote
	ErrShell     = write.ErrShell
)

func Words(line string) ([]string, error) {
	return write.Words(line)
}
