package hud

import (
	"image/color"
	"strings"

	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

const bookBridgeYAML = 12

func BookBridge(p view.Page, f Frame, cells int) []BookLine {
	b, ok := bridgeAsBuilt(p.ID, f)
	if !ok {
		return nil
	}
	v := b.View("", "")
	if own, mine := view.OwnCommand(b, saidLines(f)); mine {
		v = b.View(own.Command, own.YAML)
	}
	kube := make([]string, 0, len(v.Tokens))
	for _, t := range v.Tokens {
		kube = append(kube, t.Kube)
	}
	out := []BookLine{BlankLine(), TextLine(bridgeLabel, bookHead, 1)}
	if v.Yours {
		out = append(out, TextLine(bridgeYoursNote, bookSoft, 1))
	}
	out = append(out, bookPrefixed("Stone:   ", lexicon.StoryLine(v.Tokens), cells, bookSoft)...)
	tool, rest := toolOf(strings.Join(kube, " "))
	out = append(out, bookPrefixed(tool+": ", rest, cells, bookLink)...)
	if len(v.YAML) > 0 {
		out = append(out, BlankLine(), TextLine(v.YAMLHeading(), bookHead, 1))
		for i, l := range v.YAML {
			if i == bookBridgeYAML {
				out = append(out, TextLine("  ...", BookInk, 1))
				break
			}
			out = append(out, TextLine(clipCells(l, cells), BookInk, 1))
		}
	}
	out = append(out, BlankLine(), TextLine("THE RULE", bookHead, 1))
	out = append(out, WrapSpans([]view.TextSpan{{Text: v.Rule}}, cells, BookInk, 0)...)
	bends := v.Bends
	if f.State.Tutorial {
		bends = view.ChapterBends(bends)
	}
	for _, bend := range bends {
		out = append(out, BlankLine(), TextLine("WHERE THE STORY BENDS", bookHead, 1))
		out = append(out, WrapSpans([]view.TextSpan{{Text: bend.Bends}}, cells, bookSoft, 0)...)
	}
	return out
}

func toolOf(command string) (string, string) {
	tool, rest, _ := strings.Cut(strings.TrimSpace(command), " ")
	if tool == etcdctlVerb {
		return tool, rest
	}
	return strings.TrimSpace(kubectlVerb), strings.TrimPrefix(strings.TrimSpace(command), kubectlVerb)
}

func bookPrefixed(lead, text string, cells int, clr color.RGBA) []BookLine {
	out := []BookLine{}
	pad := strings.Repeat(" ", len(lead))
	for i, chunk := range wordChunks(text, cells-len(lead)) {
		head := pad
		if i == 0 {
			head = lead
		}
		out = append(out, TextLine(head+chunk, clr, 1))
	}
	return out
}

func clipCells(text string, cells int) string {
	runes := []rune(text)
	if len(runes) <= cells {
		return text
	}
	return string(runes[:cells])
}

func wordChunks(text string, cells int) []string {
	out := []string{}
	line := ""
	for _, word := range strings.Fields(text) {
		switch {
		case line == "":
			line = word
		case len([]rune(line))+1+len([]rune(word)) <= cells:
			line += " " + word
		default:
			out = append(out, line)
			line = word
		}
		for len([]rune(line)) > cells {
			out = append(out, string([]rune(line)[:cells]))
			line = string([]rune(line)[cells:])
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}
