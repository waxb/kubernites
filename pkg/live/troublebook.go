package live

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	MendSettle  = 30.0
	MendedFor   = 60.0
	woeAnswerAt = 140
)

type troubleLife struct {
	sig      signal
	seen     time.Time
	quiet    bool
	struck   float64
	gone     float64
	mended   float64
	mendLine string
}

type troubleBook struct {
	lives      map[string]troubleLife
	chronicles []view.ChronicleView
	written    int
}

func newTroubleBook(sigs []signal) troubleBook {
	b := troubleBook{lives: map[string]troubleLife{}}
	for _, g := range sigs {
		b.lives[g.view.ID] = troubleLife{sig: g, quiet: true, gone: -1, mended: -1}
	}
	return b
}

func (b *troubleBook) reckon(sigs []signal, now float64, s *Snapshot, wall time.Time) {
	present := map[string]bool{}
	for _, g := range sigs {
		id := g.view.ID
		present[id] = true
		life, ok := b.lives[id]
		if !ok || life.mended >= 0 {
			b.lives[id] = troubleLife{sig: g, struck: now, gone: -1, mended: -1, seen: wall}
			continue
		}
		life.sig, life.gone = g, -1
		b.lives[id] = life
	}
	for id, life := range b.lives {
		if present[id] || life.mended >= 0 || life.gone >= 0 {
			continue
		}
		life.gone = now
		life.mendLine = mendedLine(s, life.sig)
		b.lives[id] = life
	}
	b.age(now)
}

func (b *troubleBook) ids() []string {
	out := make([]string, 0, len(b.lives))
	for id := range b.lives {
		out = append(out, id)
	}
	slices.SortFunc(out, func(x, y string) int {
		return cmp.Or(cmp.Compare(b.lives[x].struck, b.lives[y].struck), cmp.Compare(x, y))
	})
	return out
}

func (b *troubleBook) age(now float64) {
	for _, id := range b.ids() {
		life := b.lives[id]
		if life.gone >= 0 && life.mended < 0 && now-life.gone >= MendSettle {
			life.mended = now
			b.lives[id] = life
			if !life.quiet {
				b.write(life)
			}
		}
		if life.mended >= 0 && now-life.mended >= MendedFor {
			delete(b.lives, id)
		}
	}
}

func (b *troubleBook) views() []view.CalamityView {
	out := []view.CalamityView{}
	for _, id := range b.ids() {
		life := b.lives[id]
		v := life.sig.view
		v.Remedies = slices.Clone(v.Remedies)
		v.Quiet, v.Struck = life.quiet, life.struck
		if life.mended >= 0 {
			v.Mended = life.mendLine
		}
		out = append(out, v)
	}
	return out
}

func (b *troubleBook) pending(id string) bool {
	life, ok := b.lives[id]
	return ok && life.gone >= 0 && life.mended < 0
}

func (b *troubleBook) quiet() []view.CalamityView {
	var out []view.CalamityView
	for _, v := range b.views() {
		if v.Quiet && v.Mended == "" {
			out = append(out, v)
		}
	}
	return out
}

func subjectOf(g signal) string {
	switch g.crew.kind {
	case "":
		return "Pod " + g.crew.ns + "/" + g.crew.name
	case "Node":
		return "Node " + g.crew.name
	case "Job", "PersistentVolumeClaim", "HorizontalPodAutoscaler":
		return g.crew.kind + " " + g.crew.ns + "/" + g.crew.name
	}
	return "A Pod of " + g.crew.words()
}

func titleOf(g signal) string {
	switch g.crew.kind {
	case "", "Node":
		return subjectOf(g)
	}
	return g.crew.words()
}

func clipWords(text string, n int) string {
	if len(text) <= n {
		return text
	}
	cut := strings.LastIndex(text[:n], " ")
	if cut < n/2 {
		cut = n
	}
	return strings.TrimRight(text[:cut], " ,;:") + "..."
}

func (b *troubleBook) write(life troubleLife) {
	b.written++
	g := life.sig
	id := fmt.Sprintf("woe-%d", b.written)
	at := g.at
	if at.IsZero() {
		at = life.seen
	}
	healed := life.seen.Add(time.Duration((life.mended - life.struck) * float64(time.Second)))
	if healed.Before(at) {
		healed = at
	}
	reason, _, _ := strings.Cut(g.view.Cause, ":")
	b.chronicles = append(b.chronicles, view.ChronicleView{ID: id, Kind: g.view.Kind, Calamity: titleOf(g), K8s: reason,
		Explanation: strings.TrimSpace(g.view.Cause + " " + g.view.Why), Truth: clipWords(g.view.Cause, woeAnswerAt),
		Resolved: true, Correct: true, Plain: true, Subject: subjectOf(g), Struck: at.UTC().Format("15:04 UTC"), Healed: healed.UTC().Format("15:04 UTC"), MendLine: life.mendLine})
}

func podsOfCrew(s *Snapshot, crew crewKey) (int, int) {
	ix := indexMirror(s)
	total, ready := 0, 0
	for i := range s.Pods {
		p := &s.Pods[i]
		if p.Namespace != crew.ns || ix.crewOf(p) != crew || p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed {
			continue
		}
		total++
		if podReady(*p) {
			ready++
		}
	}
	return total, ready
}

func findNamed[T any](list []T, meta func(*T) *metav1.ObjectMeta, ns, name string) *T {
	for i := range list {
		if m := meta(&list[i]); m.Namespace == ns && m.Name == name {
			return &list[i]
		}
	}
	return nil
}

func mendedLine(s *Snapshot, g signal) string {
	c := g.crew
	switch {
	case g.view.Kind == view.TagEvicted && c.kind != "":
		total, ready := podsOfCrew(s, c)
		return fmt.Sprintf("%d of %d Pods of %s are Ready.", ready, total, c.words())
	case c.kind == "Node":
		n := findNamed(s.Nodes, func(n *corev1.Node) *metav1.ObjectMeta { return &n.ObjectMeta }, "", c.name)
		if n == nil {
			return "Node " + c.name + " is gone."
		}
		return "Node " + c.name + " is Ready and schedulable."
	case c.kind == "Job":
		if findNamed(s.Jobs, func(j *batchv1.Job) *metav1.ObjectMeta { return &j.ObjectMeta }, c.ns, c.name) == nil {
			return "Job " + c.ns + "/" + c.name + " is gone."
		}
		return "Job " + c.ns + "/" + c.name + " no longer reports Failed."
	case c.kind == "PersistentVolumeClaim":
		pvc := findNamed(s.PVCs, func(p *corev1.PersistentVolumeClaim) *metav1.ObjectMeta { return &p.ObjectMeta }, c.ns, c.name)
		if pvc == nil {
			return "PersistentVolumeClaim " + c.ns + "/" + c.name + " is gone."
		}
		return "PersistentVolumeClaim " + c.ns + "/" + c.name + " is " + string(pvc.Status.Phase) + "."
	case c.kind == "HorizontalPodAutoscaler":
		return "HorizontalPodAutoscaler " + c.ns + "/" + c.name + " is no longer held at its maximum."
	case c.kind == "":
		p := findNamed(s.Pods, func(p *corev1.Pod) *metav1.ObjectMeta { return &p.ObjectMeta }, c.ns, c.name)
		switch {
		case p == nil:
			return "Pod " + c.ns + "/" + c.name + " is gone."
		case podReady(*p):
			return "Pod " + c.ns + "/" + c.name + " is Running and Ready."
		}
		return "Pod " + c.ns + "/" + c.name + " is " + string(p.Status.Phase) + "."
	}
	total, ready := podsOfCrew(s, c)
	if total == 0 {
		return strings.ToUpper(c.words()[:1]) + c.words()[1:] + " has no Pods left."
	}
	return fmt.Sprintf("%d of %d Pods of %s are Ready.", ready, total, c.words())
}
