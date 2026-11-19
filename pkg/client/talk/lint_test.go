package talk

import (
	"regexp"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/lexicon"
)

var capital = regexp.MustCompile(`[A-Z][a-z']+`)

func storyNames(t *testing.T) map[string]bool {
	t.Helper()
	lx, err := lexicon.Load()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, e := range lx.Entries {
		for _, w := range capital.FindAllString(e.Name, -1) {
			out[w] = true
		}
	}
	for _, p := range lx.Plain {
		for _, w := range capital.FindAllString(p.Term, -1) {
			out[w] = true
		}
	}
	return out
}

func storyTexts() []string {
	out := []string{}
	for _, l := range Pool() {
		if l.Pool == PoolNames {
			continue
		}
		f := burning()
		out = append(out, l.Say(f), l.Rest(f))
	}
	return out
}

func sentenceStarts(text string) map[int]bool {
	out := map[int]bool{0: true}
	for i := 0; i < len(text); i++ {
		if strings.ContainsRune(".!?(", rune(text[i])) {
			j := i + 1
			for j < len(text) && (text[j] == ' ' || text[j] == '(') {
				j++
			}
			out[j] = true
		}
	}
	return out
}

func TestStoryLinesSpeakNoKubernetes(t *testing.T) {
	for _, text := range storyTexts() {
		if w := lexicon.KubernetesWords.FindString(text); w != "" {
			t.Errorf("story line %q says %q", text, w)
		}
	}
}

var plainCapitals = map[string]bool{"I": true, "I'm": true, "I'd": true, "I'll": true, "I've": true, "FINE": true, "Ow": true, "Ooh": true, "Hmph": true, "Kraa": true, "Kra": true, "Hrrmm": true, "Hrrmmm": true, "Hrmf": true, "Wuff": true}

func TestStoryLinesUseOnlyTheRealmsNames(t *testing.T) {
	names := storyNames(t)
	for _, extra := range []string{"Old", "Borg", "Imp", "Academy", "Void", "Age"} {
		names[extra] = true
	}
	for _, text := range storyTexts() {
		starts := sentenceStarts(text)
		for _, loc := range capital.FindAllStringIndex(text, -1) {
			w := text[loc[0]:loc[1]]
			if starts[loc[0]] || plainCapitals[w] || names[w] || names[strings.TrimSuffix(w, "s")] {
				continue
			}
			t.Errorf("story line %q names %q, which is no story name in the lexicon", text, w)
		}
	}
}

var fallWords = regexp.MustCompile(`(?i)\b(fall|falls|fell|fallen|falling)\b`)

func TestAWorkersStopIsNeverCalledAFall(t *testing.T) {
	texts := storyTexts()
	gutter := burning()
	gutter.Worker.State = model.ContainerState{Phase: model.ContainerWaiting, Reason: "CrashLoopBackOff"}
	gutter.Worker.Last = model.ContainerState{Phase: model.ContainerTerminated, Reason: "Error", ExitCode: 1}
	ended := burning()
	ended.Worker.State = model.ContainerState{Phase: model.ContainerTerminated, Reason: "Error", ExitCode: 1}
	texts = append(texts, Status(gutter), Status(ended), LastState(gutter),
		thoughtText(gutter, Happening{Kind: HapRelit, Reason: "Error", Exit: 1}), thoughtText(gutter, Happening{Kind: HapBackOff}))
	for _, text := range texts {
		if w := fallWords.FindString(text); w != "" {
			t.Errorf("%q calls a worker's stop a %q: a hut falls, a worker stops and is relit", text, w)
		}
	}
}
