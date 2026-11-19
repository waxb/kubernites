package view

import (
	"github.com/waxb/kubernites/pkg/lexicon"
	"strings"
)

type BridgeView struct {
	Page      string
	Title     string
	K8sTitle  string
	Tokens    []lexicon.CommandToken
	Yours     bool
	YAML      []string
	StoryYAML []string
	Changed   bool
	Rule      string
	Bends     []Liberty
	Doc       string
}

func (b Bridge) View(own, ownYAML string) BridgeView {
	page, _ := FindPage(b.Page)
	command, yours := b.Command, strings.TrimSpace(own) != ""
	if yours {
		command = strings.TrimSpace(own)
	}
	yaml, changed := b.YAML, !readVerbs[firstWord(command)]
	switch generated, ok := YAMLOf(command); {
	case strings.TrimSpace(ownYAML) != "":
		yaml = ownYAML
	case ok:
		yaml = generated
	}
	v := BridgeView{Page: b.Page, Title: page.Title, K8sTitle: page.K8sTitle, Tokens: lexicon.AlignCommand(command), Yours: yours,
		Changed: changed && yaml != "", Rule: b.Rule, Doc: b.Doc}
	if yaml != "" {
		v.YAML = strings.Split(yaml, "\n")
		v.StoryYAML = strings.Split(lexicon.StoryManifest(yaml), "\n")
	}
	for _, area := range b.Bends {
		if l, ok := LibertyOf(area); ok {
			v.Bends = append(v.Bends, l)
		}
	}
	return v
}

func (v BridgeView) YAMLHeading() string {
	if v.Changed {
		return "THE YAML IT CHANGED"
	}
	return "WHERE IT LIVES IN THE YAML"
}

func OwnCommand(b Bridge, said []SaidLine) (SaidLine, bool) {
	for i := len(said) - 1; i >= 0; i-- {
		if b.Matches(said[i].Command) {
			return said[i], true
		}
	}
	return SaidLine{}, false
}

type SaidLine struct {
	Command string
	YAML    string
	Typed   bool
	At      float64
}
