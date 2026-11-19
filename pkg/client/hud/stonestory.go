package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func StoryInput(cmd string) string {
	if !ui.StoryNaming() || strings.TrimSpace(cmd) == "" {
		return cmd
	}
	story := lexicon.StoryText(kubectlVerb + cmd)
	return strings.TrimPrefix(story, lexicon.StoryCommand+" ")
}

func kubeInput(input string) string {
	if !ui.StoryNaming() || strings.TrimSpace(input) == "" {
		return input
	}
	kube := lexicon.CommandFromStory(lexicon.StoryCommand + " " + input)
	return strings.TrimPrefix(kube, kubectlVerb)
}

func storyWord(word string) string {
	if !ui.StoryNaming() {
		return word
	}
	return strings.TrimSpace(lexicon.StoryText(word))
}

func (h *HUD) stoneRunsKubectl() bool {
	return h.live != nil && h.backend.Start != nil && h.StoneKubectl()
}

func (h *HUD) stoneStory() bool {
	return ui.StoryNaming() && !h.stoneRunsKubectl()
}

func (h *HUD) stonePrompt() string {
	if h.stoneRunsKubectl() {
		return PromptFor(string(h.Stone.Input))
	}
	return StonePrompt(string(h.Stone.Input))
}

func (h *HUD) typedCommand(input string) string {
	if h.stoneStory() {
		return kubeInput(input)
	}
	return input
}

func (h *HUD) typedInput(cmd string) string {
	if h.stoneStory() {
		return StoryInput(cmd)
	}
	return cmd
}

func (h *HUD) EchoTyped(cmd string) {
	if h.stoneRunsKubectl() && !NodeCommand(cmd) {
		h.Stone.Append(LineRawEcho, "> "+ChipLabel(cmd))
		return
	}
	h.Stone.Append(LineEcho, EchoFor(cmd))
}

func (h *HUD) stoneCompletion() Completion {
	if !h.stoneStory() {
		return h.Stone.Completion(h.vocab)
	}
	if h.Stone.Cursor != len(h.Stone.Input) {
		return Completion{}
	}
	input := string(h.Stone.Input)
	kube := kubeInput(input)
	c := Complete(kube, h.vocab)
	out := Completion{}
	for _, w := range c.Candidates {
		out.Candidates = append(out.Candidates, storyWord(w))
	}
	if c.Ghost != "" {
		if full := StoryInput(kube + c.Ghost); strings.HasPrefix(full, input) {
			out.Ghost = full[len(input):]
		}
	}
	return out
}

func (h *HUD) stoneTab() {
	if !h.stoneStory() {
		h.Stone.Tab(h.vocab)
		return
	}
	kube := kubeInput(string(h.Stone.Input))
	probe := Stone{Input: []rune(kube), Cursor: len([]rune(kube))}
	probe.Tab(h.vocab)
	h.Stone.SetInput(StoryInput(string(probe.Input)))
}

func StonePrompt(input string) string {
	if !ui.StoryNaming() {
		return PromptFor(input)
	}
	if strings.HasPrefix(strings.TrimSpace(input), lexicon.CodexCommand) || strings.HasPrefix(strings.TrimSpace(input), etcdctlVerb) {
		return "$ "
	}
	return "$ " + lexicon.StoryCommand + " "
}
