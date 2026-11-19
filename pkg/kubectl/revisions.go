package kubectl

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func revisionRow(c Cluster, o store.Object, _ bool) []string {
	cr := o.(*appsv1.ControllerRevision)
	controller := "<none>"
	if ref := metav1.GetControllerOf(cr); ref != nil {
		gv, _ := schema.ParseGroupVersion(ref.APIVersion)
		controller = fmt.Sprintf("%s/%s", strings.ToLower(schema.GroupKind{Group: gv.Group, Kind: ref.Kind}.String()), ref.Name)
	}
	return []string{cr.Name, controller, fmt.Sprint(cr.Revision), age(c, cr.CreationTimestamp)}
}
