package hud

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

var (
	notFoundError   = regexp.MustCompile(`(?m)^Error from server \(NotFound\): (\S+) "([^"]+)" not found$`)
	noKindError     = regexp.MustCompile(`(?m)^error: the server doesn't have a resource type "([^"]+)"`)
	badFieldError   = regexp.MustCompile(`(?m)^Error from server \(BadRequest\): unable to filter by the field "([^"]+)": try (.+)$`)
	unreadableField = regexp.MustCompile(`(?m)^error: unable to read the field selector`)
	unknownRune     = regexp.MustCompile(`(?m)^error: the rune "([^"]+)" is unknown to you\. Learn (.+) in the Grimoire first`)
	existsError     = regexp.MustCompile(`(?m)^(?:Error from server \(AlreadyExists\)|error: failed to create [^:]+): (\S+) "([^"]+)" already exists$`)
)

func StoryError(text string) string {
	if !ui.StoryNaming() {
		return text
	}
	if told := storyRefusal(text); told != nil {
		return strings.Join(told, " ")
	}
	return text
}

func storyRefusal(output string) []string {
	if m := notFoundError.FindStringSubmatch(output); m != nil {
		return []string{fmt.Sprintf("No %s named %q stands here. List them first, then ask by name.", singularKind(m[1]), m[2])}
	}
	if m := noKindError.FindStringSubmatch(output); m != nil {
		return []string{fmt.Sprintf("The Stone knows no kind called %q. api-resources lists every kind it can read.", lexicon.StoryText(m[1]))}
	}
	if m := badFieldError.FindStringSubmatch(output); m != nil {
		return []string{fmt.Sprintf("The Stone cannot filter by %q. It can filter by: %s", m[1], m[2])}
	}
	if m := unknownRune.FindStringSubmatch(output); m != nil {
		return []string{fmt.Sprintf("The word %q is not yours to speak yet. Learn %s in the Grimoire first.", lexicon.StoryKeyOf(m[1]), m[2])}
	}
	if m := existsError.FindStringSubmatch(output); m != nil {
		return []string{fmt.Sprintf("A %s named %q already stands.", singularKind(m[1]), m[2])}
	}
	if unreadableField.MatchString(output) {
		return []string{"The Stone could not read that field filter. Write it as field=value, for example status.phase=Running."}
	}
	return nil
}

func singularKind(token string) string {
	return strings.TrimSuffix(storyNoun(token), "s")
}
