package kubectl

type teacher interface {
	Teaching() bool
}

func GuardsCitadel(c Cluster) bool {
	t, ok := c.(teacher)
	return !ok || !t.Teaching()
}
