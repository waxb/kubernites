package kubectl

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/sim/clock"
	"github.com/waxb/kubernites/pkg/sim/store"
)

type Usage struct {
	CPUMilli float64
	MemBytes float64
}

type Cluster interface {
	Store() *store.Store
	Clock() *clock.Clock
	Unlocked(id string) bool
	Logs(uid types.UID, previous bool) ([]string, bool)
	PodUsage(uid string) (Usage, bool)
	NodeUsage(name string) (Usage, bool)
	MetricsAvailable() bool
}

type Result struct {
	Output   string
	OK       bool
	Verb     string
	Resource string
}

type Invocation struct {
	Verb      string
	Args      []string
	Flags     map[string]string
	Bools     map[string]bool
	Namespace string
	allNS     bool
}

var valueFlags = map[string]string{
	"n": "namespace", "namespace": "namespace", "o": "output", "output": "output", "l": "selector", "selector": "selector",
	"image": "image", "labels": "labels", "replicas": "replicas", "requests": "requests", "limits": "limits",
	"grace-period": "grace-period", "c": "container", "container": "container", "containers": "container", "tail": "tail", "restart": "restart",
	"port": "port", "f": "filename", "filename": "filename", "r": "replicas", "target-port": "target-port",
	"to-revision": "to-revision", "revision": "revision", "type": "type", "protocol": "protocol",
	"min": "min", "max": "max", "cpu": "cpu", "cpu-percent": "cpu-percent", "memory": "memory", "name": "name",
	"patch": "patch", "overrides": "overrides", "override-type": "override-type", "field-selector": "field-selector",
	"types": "types", "for": "for", "L": "label-columns", "label-columns": "label-columns",
	"clusterip": "clusterip", "tcp": "tcp",
	"timeout": "timeout", "pod-selector": "pod-selector", "skip-wait-for-delete-timeout": "skip-wait-for-delete-timeout",
}

var boolFlags = map[string]string{
	"A": "all-namespaces", "all-namespaces": "all-namespaces", "show-labels": "show-labels", "previous": "previous",
	"p": "previous", "all": "all", "overwrite": "overwrite", "force": "force", "h": "help", "help": "help",
	"w": "watch", "watch": "watch", "i": "stdin", "t": "tty", "it": "stdin", "stdin": "stdin", "tty": "tty",
	"ignore-daemonsets": "ignore-daemonsets", "delete-emptydir-data": "delete-emptydir-data",
}

type Handler func(Cluster, Invocation) (string, string, error)

var handlers map[string]Handler

func init() {
	handlers = map[string]Handler{
		"get": get, "describe": describe, "logs": logs, "top": top, "explain": explain,
		"version": version, "cluster-info": clusterInfo, "api-resources": apiResources, "help": help,
		"apply": NeedsScriptorium, "edit": NeedsScriptorium, "replace": NeedsScriptorium,
		"exec": NoRunningContainer, "attach": NoRunningContainer, "port-forward": NoRunningContainer, "cp": NoRunningContainer, "rollout": rollout,
		"debug": NoRunningContainer, "events": events,
	}
}

func Register(verb string, h Handler) {
	handlers[verb] = h
}

func Lookup(verb string) (Handler, bool) {
	h, ok := handlers[verb]
	return h, ok
}

var serverMessages []func(error) (string, bool)

func RegisterServerMessage(message func(error) (string, bool)) {
	serverMessages = append(serverMessages, message)
}

func Execute(c Cluster, command string) Result {
	tokens, err := tokenize(command)
	if err != nil {
		return fail("", "", err)
	}
	if len(tokens) > 0 && tokens[0] == "kubectl" {
		tokens = tokens[1:]
	}
	tokens = patchTokens(hoistGlobal(tokens))
	if len(tokens) == 0 {
		return fail("", "", errors.New("error: the Speaking Stone awaits a command. Try: kubectl help"))
	}
	inv, err := parse(tokens)
	if err != nil {
		return fail(tokens[0], "", err)
	}
	h, ok := handlers[inv.Verb]
	if !ok {
		return fail(inv.Verb, "", fmt.Errorf("error: unknown command %q for \"kubectl\"", inv.Verb))
	}
	out, resource, err := h(c, inv)
	if err != nil {
		return fail(inv.Verb, resource, err)
	}
	return Result{Output: ensureNewline(out), OK: true, Verb: inv.Verb, Resource: resource}
}

func fail(verb, resource string, err error) Result {
	return Result{Output: ensureNewline(ServerMessage(err)), Verb: verb, Resource: resource}
}

func ServerMessage(err error) string {
	switch {
	case errors.Is(err, store.ErrNoSpace):
		return "Error from server: " + store.ErrNoSpace.Error()
	}
	for _, message := range serverMessages {
		if out, ok := message(err); ok {
			return out
		}
	}
	return err.Error()
}

func ensureNewline(s string) string {
	if s == "" || strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

func parse(tokens []string) (Invocation, error) {
	inv := Invocation{Verb: tokens[0], Flags: map[string]string{}, Bools: map[string]bool{}}
	for i := 1; i < len(tokens); i++ {
		tok := tokens[i]
		if tok == "--" {
			inv.Args = append(inv.Args, tokens[i+1:]...)
			break
		}
		if !strings.HasPrefix(tok, "-") || tok == "-" {
			inv.Args = append(inv.Args, tok)
			continue
		}
		next, err := inv.flag(tokens, i)
		if err != nil {
			return inv, err
		}
		i = next
	}
	inv.Namespace = inv.Flags["namespace"]
	if inv.Namespace == "" {
		inv.Namespace = "default"
	}
	inv.allNS = inv.Bools["all-namespaces"]
	return inv, nil
}

func (inv Invocation) flag(tokens []string, i int) (int, error) {
	tok := tokens[i]
	name, value, hasValue := strings.Cut(strings.TrimLeft(tok, "-"), "=")
	if def, ok := optionalValueFlags[name]; ok {
		if !hasValue {
			value = def
		}
		inv.Flags[name] = value
		return i, nil
	}
	if canon, ok := valueFlags[name]; ok {
		if !hasValue {
			if i+1 >= len(tokens) {
				return i, fmt.Errorf("error: flag needs an argument: %s", tok)
			}
			i++
			value = tokens[i]
		}
		if prior := inv.Flags[canon]; canon == "label-columns" && prior != "" {
			value = prior + "," + value
		}
		inv.Flags[canon] = value
		return i, nil
	}
	if canon, ok := boolFlags[name]; ok {
		inv.Bools[canon] = value != "false"
		return i, nil
	}
	if canon, attached, ok := shorthand(tok); ok {
		inv.Flags[canon] = attached
		return i, nil
	}
	return i, fmt.Errorf("error: unknown flag: %s", tok)
}

func shorthand(tok string) (string, string, bool) {
	if strings.HasPrefix(tok, "--") || len(tok) < 3 {
		return "", "", false
	}
	canon, ok := valueFlags[tok[1:2]]
	return canon, tok[2:], ok
}

func hoistGlobal(tokens []string) []string {
	var lead []string
	i := 0
	for i < len(tokens) {
		n := globalWidth(tokens[i])
		if n == 0 || i+n > len(tokens) {
			break
		}
		lead = append(lead, tokens[i:i+n]...)
		i += n
	}
	if len(lead) == 0 || i >= len(tokens) {
		return tokens
	}
	out := append([]string{tokens[i]}, lead...)
	return append(out, tokens[i+1:]...)
}

func globalWidth(tok string) int {
	switch {
	case tok == "-n" || tok == "--namespace":
		return 2
	case strings.HasPrefix(tok, "--namespace="), strings.HasPrefix(tok, "-n") && !strings.HasPrefix(tok, "--"):
		return 1
	}
	return 0
}

func tokenize(s string) ([]string, error) {
	out := []string{}
	var cur strings.Builder
	var quote rune
	inToken := false
	for _, r := range s {
		switch {
		case quote != 0 && r == quote:
			quote = 0
		case quote != 0:
			cur.WriteRune(r)
		case r == '"' || r == '\'':
			quote, inToken = r, true
		case r == ' ' || r == '\t' || r == '\n':
			if inToken {
				out = append(out, cur.String())
				cur.Reset()
				inToken = false
			}
		default:
			cur.WriteRune(r)
			inToken = true
		}
	}
	if quote != 0 {
		return nil, errors.New("error: unterminated quote in command")
	}
	if inToken {
		out = append(out, cur.String())
	}
	return out, nil
}

type scriptorium interface {
	ScriptoriumOpen() bool
}

func NeedsScriptorium(c Cluster, inv Invocation) (string, string, error) {
	if !GuardsCitadel(c) {
		if file, ok := inv.Flags["filename"]; ok && file != "" {
			return "", "", fmt.Errorf("error: the path %q does not exist", file)
		}
		return "", "", fmt.Errorf("error: kubectl %s is not part of these chapters: their terminal has no text editor", inv.Verb)
	}
	if s, ok := c.(scriptorium); ok && !s.ScriptoriumOpen() {
		return "", "", fmt.Errorf("error: the Speaking Stone cannot read files from disk. Manifests are written and applied in the Scriptorium (kubectl %s), which opens in the Feudal Age", inv.Verb)
	}
	return "", "", fmt.Errorf("error: the Speaking Stone cannot read files from disk. Open the Scriptorium to write and apply manifests (kubectl %s)", inv.Verb)
}

func NoRunningContainer(_ Cluster, inv Invocation) (string, string, error) {
	return "", "", fmt.Errorf("error: kubectl %s reaches into a running container: this simulated cluster runs none", inv.Verb)
}

func WriteVerbs() []string {
	return []string{"autoscale", "cordon", "create", "delete", "drain", "expose", "label", "patch", "run", "scale", "set", "taint", "uncordon"}
}

func Verbs() []string {
	out := make([]string, 0, len(handlers))
	for v := range handlers {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
