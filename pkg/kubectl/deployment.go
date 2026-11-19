package kubectl

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func GetDeployment(c Cluster, namespace, name string) (*appsv1.Deployment, error) {
	d, ok := store.Get[*appsv1.Deployment](c.Store(), store.KindDeployment, namespace, name)
	if !ok {
		return nil, fmt.Errorf("Error from server (NotFound): deployments.apps %q not found", name)
	}
	return d, nil
}
