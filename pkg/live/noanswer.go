package live

import (
	"errors"
	"fmt"
	"net"
)

func missingName(err error) (string, bool) {
	var dns *net.DNSError
	if errors.As(err, &dns) && dns.IsNotFound {
		return dns.Name, true
	}
	return "", false
}

func noRecord(name string, err error) error {
	return fmt.Errorf("no DNS record for %s: the cluster may be stopped, deleted or recreated (fetch its kubeconfig again), or reachable only inside its own network or VPN: %w", name, err)
}

func NoAnswer(err error) error {
	if name, ok := missingName(err); ok {
		return noRecord(name, err)
	}
	return fmt.Errorf("the cluster does not answer: %w", err)
}

func withDNSHint(err error) error {
	if name, ok := missingName(err); ok {
		return noRecord(name, err)
	}
	return err
}
