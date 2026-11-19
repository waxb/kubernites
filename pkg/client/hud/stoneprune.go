package hud

import (
	"regexp"
	"strings"
)

var (
	rawField     = regexp.MustCompile(`^(\s*)([A-Za-z][A-Za-z0-9 ./_-]*):(\s+(\S.*))?\s*$`)
	describeHead = regexp.MustCompile(`^Name:\s`)
	systemBadges = map[string]bool{
		"kubernetes.io/arch": true, "kubernetes.io/hostname": true, "kubernetes.io/os": true, "kubernites.dev/keep-type": true,
	}
	emptyBadges = "<none>"
)

func isDescribe(lines []string) bool {
	for _, l := range lines {
		if describeHead.MatchString(l) {
			return true
		}
	}
	return false
}

func leadingSpaces(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

func lastDeeper(lines []string, at, indent int) int {
	j := at + 1
	for j < len(lines) && strings.TrimSpace(lines[j]) != "" && leadingSpaces(lines[j]) > indent {
		j++
	}
	return j - 1
}

func pruneDescribe(lines []string) []string {
	if !isDescribe(lines) {
		return lines
	}
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		m := rawField.FindStringSubmatch(lines[i])
		switch {
		case m == nil:
			out = append(out, lines[i])
		case storyDropped[m[2]], m[2] == "Priority" && strings.TrimSpace(m[4]) == "0":
			i = lastDeeper(lines, i, len(m[1]))
		case m[2] == "Labels":
			end := lastDeeper(lines, i, len(m[1]))
			out = append(out, keptBadges(m, lines[i+1:end+1])...)
			i = end
		default:
			out = append(out, lines[i])
		}
	}
	return dropEmptyHeaders(out)
}

func systemBadge(entry string) bool {
	key, _, _ := strings.Cut(entry, "=")
	return systemBadges[key] || strings.HasPrefix(key, "node-role.kubernetes.io/")
}

func keptBadges(m []string, rest []string) []string {
	entries := []string{}
	if first := strings.TrimSpace(m[4]); first != "" {
		entries = append(entries, first)
	}
	cont := ""
	for _, l := range rest {
		entries = append(entries, strings.TrimSpace(l))
		cont = l[:leadingSpaces(l)]
	}
	kept := make([]string, 0, len(entries))
	for _, e := range entries {
		if e != emptyBadges && !systemBadge(e) {
			kept = append(kept, e)
		}
	}
	gap := " "
	if m[3] != "" {
		gap = m[3][:len(m[3])-len(m[4])]
	}
	head := m[1] + m[2] + ":" + gap
	if len(kept) == 0 {
		return []string{head + emptyBadges}
	}
	if cont == "" {
		cont = strings.Repeat(" ", len(head))
	}
	out := []string{head + kept[0]}
	for _, e := range kept[1:] {
		out = append(out, cont+e)
	}
	return out
}

func dropEmptyHeaders(lines []string) []string {
	out := make([]string, 0, len(lines))
	for i, l := range lines {
		m := rawField.FindStringSubmatch(l)
		if m != nil && strings.TrimSpace(m[4]) == "" && (i+1 == len(lines) || strings.TrimSpace(lines[i+1]) == "" || leadingSpaces(lines[i+1]) <= len(m[1])) {
			continue
		}
		out = append(out, l)
	}
	return out
}
