package view

import (
	"strings"
)

type TextSpan struct {
	Text string `json:"text"`
	Link string `json:"link,omitempty"`
}

func Parse(text string) []TextSpan {
	out := []TextSpan{}
	for text != "" {
		open := strings.Index(text, "[[")
		if open < 0 {
			break
		}
		end := strings.Index(text[open:], "]]")
		if end < 0 {
			break
		}
		if open > 0 {
			out = append(out, TextSpan{Text: text[:open]})
		}
		inner := text[open+2 : open+end]
		id, label, labelled := strings.Cut(inner, "|")
		if !labelled {
			label = afterArticle(text[:open], linkLabel(id))
		}
		out = append(out, TextSpan{Text: label, Link: id})
		text = text[open+end+2:]
	}
	if text != "" {
		out = append(out, TextSpan{Text: text})
	}
	return out
}

func Plain(text string) string {
	var b strings.Builder
	for _, s := range Parse(text) {
		b.WriteString(s.Text)
	}
	return b.String()
}

func linkLabel(id string) string {
	if p, ok := FindPage(id); ok {
		return p.Title
	}
	return id
}

func afterArticle(before, label string) string {
	rest, titled := strings.CutPrefix(label, "The ")
	if !titled || !strings.HasSuffix(strings.ToLower(before), "the ") {
		return label
	}
	return rest
}
