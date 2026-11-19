package app

import (
	"strings"
	"testing"
	"time"

	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestTheDrainSpeaksOfTheQuorumAndOfSendingAwayInStory(t *testing.T) {
	defer ui.SetNamer(nil)
	two, half := intstr.FromInt32(2), intstr.FromString("50%")
	least := policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Namespace: "market", Name: "storefront"},
		Spec: policyv1.PodDisruptionBudgetSpec{MinAvailable: &two}, Status: policyv1.PodDisruptionBudgetStatus{CurrentHealthy: 2, DesiredHealthy: 2}}
	most := policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Namespace: "market", Name: "kv"},
		Spec: policyv1.PodDisruptionBudgetSpec{MaxUnavailable: &half}, Status: policyv1.PodDisruptionBudgetStatus{CurrentHealthy: 4, DesiredHealthy: 2, DisruptionsAllowed: 2}}
	at := time.Date(2026, 11, 16, 14, 9, 0, 0, time.UTC)
	evict := ledgerLine{at: at, verb: write.OrderEvict, target: write.Ref{GVR: write.Pods, Kind: "Pod", Namespace: "market", Name: "kv-0"}, code: 201, answer: "OK"}
	image := ledgerLine{at: at, verb: write.OrderSetImage, target: write.Ref{GVR: write.Deployments, Kind: "Deployment", Namespace: "market", Name: "web"}, code: 200, answer: "OK"}

	ui.SetStoryNamer(lexicon.StoryText)
	for got, want := range map[string]string{
		budgetWords(least):          "QUORUM market/storefront: at least 2 must stand, 2 of 2 standing, 0 may leave now",
		budgetWords(most):           "QUORUM market/kv: at most 50% may be away, 4 of 2 standing, 2 may leave now",
		answerWords(429, "x"):       "the Quorum holds it",
		evict.text():                "14:09 SEND AWAY hut market/kv-0: done",
		verbWords(write.OrderScale): "SCALE",
	} {
		if got != want {
			t.Errorf("story: %q, want %q", got, want)
		}
	}
	if got := image.text(); strings.Contains(got, "IMAGE") || !strings.Contains(got, "SET THE CLASS") {
		t.Errorf("story: %q names the image", got)
	}

	ui.SetNamer(names.Kubernetes(64))
	for got, want := range map[string]string{
		budgetWords(least): "PodDisruptionBudget market/storefront: minAvailable 2, 2 healthy of 2 desired, 0 disruptions allowed",
		evict.text():       "14:09 EVICT Pod market/kv-0: OK",
		image.text():       "14:09 SET IMAGE Deployment market/web: OK",
	} {
		if got != want {
			t.Errorf("names: %q, want %q", got, want)
		}
	}
}
