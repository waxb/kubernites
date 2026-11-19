package kubectl

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func withAccounts(st *store.Store) *store.Store {
	accounts := []store.Object{}
	for _, ns := range []string{"default", "kube-system"} {
		accounts = append(accounts, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: ns}})
	}
	if err := st.Restore(accounts, 0, 0); err != nil {
		panic(err)
	}
	return st
}
