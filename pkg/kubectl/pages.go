package kubectl

const (
	PageCrews  = "deployments"
	PageShift  = "foremen"
	PageBadges = "marks"
)

var crewVerbs = map[string]bool{"scale": true, "rollout": true}

var readOnlyRollouts = map[string]bool{"status": true, "history": true}

func PageFor(command string) string {
	tokens, err := tokenize(command)
	if err != nil {
		return ""
	}
	if len(tokens) > 0 && tokens[0] == "kubectl" {
		tokens = tokens[1:]
	}
	tokens = patchTokens(hoistGlobal(tokens))
	if len(tokens) == 0 {
		return ""
	}
	inv, err := parse(tokens)
	if err != nil {
		return ""
	}
	return pageOf(inv)
}

func pageOf(inv Invocation) string {
	switch inv.Verb {
	case "create":
		return createPage(inv.Args)
	case "expose":
		return PageBadges
	case "set":
		return setPage(inv.Args)
	case "rollout":
		if len(inv.Args) > 0 && readOnlyRollouts[inv.Args[0]] {
			return ""
		}
		return workloadPage(inv.Args[min(1, len(inv.Args)):])
	}
	if crewVerbs[inv.Verb] {
		return workloadPage(inv.Args)
	}
	return ""
}

func createPage(args []string) string {
	switch {
	case len(args) > 0 && (args[0] == "deployment" || args[0] == "deploy"):
		return PageCrews
	case len(args) > 0 && (args[0] == "service" || args[0] == "svc"):
		return PageBadges
	}
	return ""
}

func setPage(args []string) string {
	if len(args) == 0 {
		return ""
	}
	if args[0] == "selector" {
		return PageBadges
	}
	return workloadPage(args[1:])
}

func workloadPage(args []string) string {
	typeTok, _, err := SplitTypeName(args)
	if err != nil {
		return ""
	}
	r, err := ResolveType(typeTok)
	if err != nil {
		return ""
	}
	switch r.Kind {
	case "Deployment":
		return PageCrews
	case "ReplicaSet":
		return PageShift
	}
	return ""
}

func VerbOf(command string) string {
	tokens, err := tokenize(command)
	if err != nil {
		return ""
	}
	if len(tokens) > 0 && tokens[0] == "kubectl" {
		tokens = tokens[1:]
	}
	if len(tokens) > 1 && (tokens[0] == "create" || tokens[0] == "set" || tokens[0] == "rollout") {
		return tokens[0] + " " + tokens[1]
	}
	if len(tokens) > 0 {
		return tokens[0]
	}
	return ""
}
