package hud

import "github.com/waxb/kubernites/pkg/view"

var controlPlaneFiles = map[string]string{
	"build:codex": "etcd", "build:chancery": "kube-apiserver", "build:quartermaster": "kube-scheduler", "build:stewards-hall": "kube-controller-manager",
}

func staticPodNoteK8s(id string) string {
	component, ok := controlPlaneFiles[id]
	if !ok {
		return ""
	}
	return "\nIt runs as a static Pod on " + view.TowerName + ": the kubelet starts it from /etc/kubernetes/manifests/" + component + ".yaml."
}
