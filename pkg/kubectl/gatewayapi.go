package kubectl

import (
	"fmt"
	"strings"
)

type GatewayRow struct {
	Name       string
	Class      string
	Programmed bool
	Age        string
}

type RouteRow struct {
	Kind string
	Name string
	Age  string
}

type GatewayView struct {
	ClassAge string
	Gateways []GatewayRow
	Routes   []RouteRow
}

type GatewayReader interface {
	GatewayAPI() GatewayView
}

const (
	gatewayGroup      = "gateway.networking.k8s.io"
	gatewayClassName  = "harbor"
	gatewayController = "kubernites.dev/harbor-controller"
)

type gatewayKind struct {
	kind, plural, singular, short, version string
	namespaced                             bool
}

var gatewayKinds = []gatewayKind{
	{"GatewayClass", "gatewayclasses", "gatewayclass", "gc", "v1", false},
	{"Gateway", "gateways", "gateway", "gtw", "v1", true},
	{"HTTPRoute", "httproutes", "httproute", "", "v1", true},
	{"TCPRoute", "tcproutes", "tcproute", "", "v1alpha2", true},
	{"TLSRoute", "tlsroutes", "tlsroute", "", "v1alpha2", true},
}

func gatewayKindOf(token string) (gatewayKind, bool) {
	lower := strings.ToLower(token)
	lower = strings.TrimSuffix(lower, "."+gatewayGroup)
	for _, k := range gatewayKinds {
		if lower == k.plural || lower == k.singular || (k.short != "" && lower == k.short) {
			return k, true
		}
	}
	return gatewayKind{}, false
}

func (k gatewayKind) apiVersion() string { return gatewayGroup + "/" + k.version }

func gatewayTable(c Cluster, k gatewayKind, inv Invocation, names []string, prefix bool) (string, error) {
	reader, ok := c.(GatewayReader)
	if !ok {
		return "", fmt.Errorf("error: the server doesn't have a resource type %q", k.plural)
	}
	view := reader.GatewayAPI()
	headers, rows := gatewayRows(k, view)
	if len(names) > 0 {
		kept := [][]string{}
		for _, n := range names {
			found := false
			for _, r := range rows {
				if r[0] == n {
					kept, found = append(kept, r), true
				}
			}
			if !found {
				return "", fmt.Errorf("Error from server (NotFound): %s.%s %q not found", k.plural, gatewayGroup, n)
			}
		}
		rows = kept
	}
	if len(rows) == 0 {
		if k.namespaced {
			return fmt.Sprintf("No resources found in %s namespace.", inv.Namespace), nil
		}
		return "No resources found", nil
	}
	if inv.Flags["output"] == "name" {
		out := make([]string, len(rows))
		for i, r := range rows {
			out[i] = k.singular + "." + gatewayGroup + "/" + r[0]
		}
		return strings.Join(out, "\n"), nil
	}
	if o := inv.Flags["output"]; o != "" && o != "wide" {
		return "", fmt.Errorf("error: the Stone reads Gateway API objects as a table or -o name; %q is not offered here", o)
	}
	if prefix {
		rows = append([][]string(nil), rows...)
		for i := range rows {
			rows[i] = append([]string{k.singular + "." + gatewayGroup + "/" + rows[i][0]}, rows[i][1:]...)
		}
	}
	if inv.allNS && k.namespaced {
		headers = append([]string{"NAMESPACE"}, headers...)
		for i := range rows {
			rows[i] = append([]string{defaultNamespace}, rows[i]...)
		}
	}
	return renderTable(headers, rows), nil
}

const defaultNamespace = "default"

func gatewayRows(k gatewayKind, v GatewayView) ([]string, [][]string) {
	rows := [][]string{}
	switch k.kind {
	case "GatewayClass":
		if len(v.Gateways) > 0 {
			rows = append(rows, []string{gatewayClassName, gatewayController, "True", v.ClassAge})
		}
		return []string{"NAME", "CONTROLLER", "ACCEPTED", "AGE"}, rows
	case "Gateway":
		for _, g := range v.Gateways {
			rows = append(rows, []string{g.Name, g.Class, "<none>", fmt.Sprint(g.Programmed), g.Age})
		}
		for i := range rows {
			rows[i][3] = map[string]string{"true": "True", "false": "False"}[rows[i][3]]
		}
		return []string{"NAME", "CLASS", "ADDRESS", "PROGRAMMED", "AGE"}, rows
	}
	for _, r := range v.Routes {
		if r.Kind == k.kind {
			rows = append(rows, []string{r.Name, "<none>", r.Age})
		}
	}
	return []string{"NAME", "HOSTNAMES", "AGE"}, rows
}
