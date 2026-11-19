package app

import (
	"fmt"
	"testing"

	"github.com/waxb/kubernites/pkg/lexicon"
)

var storyCorpus = []string{
	"+0.56/S", "DAY 1 10:33", "4.03K", "Hut porters-12 is Unhoused: no Keep has room",
	"3 Pods are Pending on the Node", "250m CPU and 64Mi memory", "2 CPU, 4 GiB memory",
	"+3 for each new word spoken at the Stone, +15 for a question answered well,",
	"The Lone Tower has 0.6 free.", "kubectl scale deployment porters --replicas=10",
	"Raise a Stone Keep (a second Node).", "12 of 20 huts ready, 3 waiting",
	"clerks-f778f8b48-2478h is Unhoused", "Nines 96.5%", "1,300 of 1,900 rations pledged",
	"Hut kube-proxy-8htt4 gains its worker", "Shift metrics-server-6d94bc8694 keeps two huts running",
	"endpointslice.discovery.k8s.io/parcels-by-hand patched", "horizontalpodautoscaler.autoscaling/porters-2 patched",
	"Finished defragmenting etcd member[127.0.0.1:2379]. took 1.76s",
}

func TestTheStoryNamerAgreesWithStoryText(t *testing.T) {
	name := newStoryNamer()
	for round := range 2 {
		for _, s := range storyCorpus {
			if got, want := name(s), lexicon.StoryText(s); got != want {
				t.Errorf("round %d: %q named %q, StoryText says %q", round, s, got, want)
			}
		}
	}
}

func TestTheStoryNamerReusesItsTemplatesForChangingNumbers(t *testing.T) {
	n := &storyNamer{templates: map[string]string{}}
	for i := range 50 {
		n.name(fmt.Sprintf("Hut porters-%d waits, %d.5 requests a second", i, i))
	}
	if len(n.templates) != 1 {
		t.Fatalf("%d templates for one sentence with changing numbers", len(n.templates))
	}
}

func BenchmarkStoryNamerChangingNumbers(b *testing.B) {
	name := newStoryNamer()
	for i := 0; i < b.N; i++ {
		name(fmt.Sprintf("+%d.%d/S  hut porters-%d waits", i%100, i%10, i))
	}
}
