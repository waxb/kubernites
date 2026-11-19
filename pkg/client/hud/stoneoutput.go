package hud

import (
	"regexp"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	tableGap      = 3
	continuedFrom = 8
)

var (
	describeField = regexp.MustCompile(`^(\s*)([A-Za-z][A-Za-z -]*):(?:(\s+)(\S.*)?)?$`)
	wholeCores    = regexp.MustCompile(`^\d+$`)
	tableCells    = regexp.MustCompile(`\s{2,}`)
	storyDropped  = storyDroppedK8s()
)

func storyDroppedK8s() map[string]bool {
	out := map[string]bool{}
	for _, label := range []string{"Type", "IP", "IPs", "IP Family Policy", "IP Families", "NodePort",
		"Session Affinity", "Annotations", "Internal Traffic Policy", "External Traffic Policy",
		"Container ID", "Image ID", "Host Port", "Mounts", "Volumes", "Node-Selectors", "ClusterIP",
		"CreationTimestamp", "Unschedulable", "Addresses", "InternalIP", "Hostname", "PodCIDR", "PodCIDRs", "ProviderID",
		"System Info", "Machine ID", "System UUID", "Boot ID", "Kernel Version", "OS Image", "Operating System", "Architecture",
		"Container Runtime Version", "Kubelet Version", "Kube-Proxy Version", "Start Time", "Started", "Finished", "Service Account", "MinReadySeconds", "Image pull secrets"} {
		out[label] = true
	}
	return out
}

var (
	storyDroppedColumns = map[string]bool{"VERSION": true, "LastHeartbeatTime": true, "LastTransitionTime": true, "IP": true, "NOMINATED NODE": true, "READINESS GATES": true}
	ageColumns          = map[string]bool{"AGE": true, "Age": true, "LAST SEEN": true}
	agePart             = regexp.MustCompile(`(\d+)([dhms])`)
)

func columnOf(header []string, c int) string {
	if c < len(header) {
		return header[c]
	}
	return ""
}

func storyCell(header []string, column, cell string) string {
	if ageColumns[column] {
		return storyAge(cell)
	}
	if mapped := tableCell(header, column, cell); mapped != "" {
		return mapped
	}
	story := lexicon.StoryText(cell)
	if kubeLeft(story) {
		return "-"
	}
	return story
}

func storyAge(cell string) string {
	parts := agePart.FindAllStringSubmatch(cell, -1)
	if len(parts) == 0 {
		return cell
	}
	words := make([]string, 0, len(parts))
	for _, p := range parts {
		unit := p[2]
		if unit == "m" {
			unit = "min"
		}
		words = append(words, p[1]+unit)
	}
	return strings.Join(words, " ")
}

type describeRow struct {
	indent, label, value string
	cont                 bool
}

func loneField(line string) bool {
	m := describeField.FindStringSubmatch(line)
	return m != nil && !tableCells.MatchString(m[2])
}

func continued(line string) bool {
	return strings.TrimSpace(line) != "" && leadingSpaces(line) >= continuedFrom && !describeField.MatchString(line) && !tabular(line)
}

func StoryConsole(lines []string) []string {
	if !ui.StoryNaming() {
		return lines
	}
	if apiResourceTable(lines) {
		return storyAPIResources(lines)
	}
	if len(lines) > 0 && strings.HasPrefix(lines[0], "apiVersion:") {
		return strings.Split(lexicon.StoryManifest(strings.Join(lines, "\n")), "\n")
	}
	lines = pruneDescribe(lines)
	describing := isDescribe(lines)
	out := []string{}
	for i := 0; i < len(lines); {
		switch {
		case describeField.MatchString(lines[i]) && i+1 < len(lines) && (describeField.MatchString(lines[i+1]) || continued(lines[i+1])):
			j := i + 1
			for j < len(lines) && (describeField.MatchString(lines[j]) || continued(lines[j])) {
				j++
			}
			out = append(out, storyDescribe(lines[i:j])...)
			i = j
		case describing && loneField(lines[i]):
			out = append(out, storyDescribe(lines[i:i+1])...)
			i++
		case tabular(lines[i]):
			j := i
			for j < len(lines) && tabular(lines[j]) && !(describing && loneField(lines[j]) && leadingSpaces(lines[j]) == 0) {
				j++
			}
			out = append(out, StoryTable(lines[i:j])...)
			i = j
		default:
			if story := wardEffects(lexicon.StoryText(lines[i])); !kubeLeft(story) || CommandSyntax(story) {
				out = append(out, story)
			}
			i++
		}
	}
	return out
}

const stoneUnsaid = "The Chancery refused, in words the realm has no story for yet. Switch NAMES to read its own reply."

func StoryAnswer(output string, ok bool) []string {
	if !ok && ui.StoryNaming() {
		if told := storyRefusal(output); told != nil {
			return told
		}
	}
	lines := StoryConsole(SplitOutput(output))
	if len(lines) == 0 && !ok && ui.StoryNaming() && strings.TrimSpace(output) != "" {
		return []string{stoneUnsaid}
	}
	return lines
}

func CommandSyntax(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), lexicon.StoryCommand+" ")
}

func kubeLeft(text string) bool {
	return lexicon.KubernetesWords.MatchString(text) || strings.Contains(text, "ClusterIP")
}

func storyDescribe(lines []string) []string {
	rows := []describeRow{}
	width := map[string]int{}
	indent, skipping := "", false
	for _, l := range lines {
		m := describeField.FindStringSubmatch(l)
		if m == nil {
			value := wardEffects(lexicon.StoryText(strings.TrimSpace(l)))
			if !skipping && !kubeLeft(value) {
				rows = append(rows, describeRow{indent: indent, value: value, cont: true})
			}
			continue
		}
		indent, skipping = m[1], false
		if storyDropped[m[2]] {
			skipping = true
			continue
		}
		row := describeRow{indent: m[1], label: lexicon.StoryText(rowLabel(m[2])), value: wardEffects(lexicon.StoryText(storyValue(m[2], rowValue(m[2], m[4]))))}
		if kubeLeft(row.label + " " + row.value) {
			skipping = true
			continue
		}
		rows = append(rows, row)
		width[row.indent] = max(width[row.indent], len([]rune(row.label)))
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.cont {
			out = append(out, r.indent+strings.Repeat(" ", width[r.indent]+3)+r.value)
			continue
		}
		pad := strings.Repeat(" ", width[r.indent]-len([]rune(r.label))+2)
		out = append(out, strings.TrimRight(r.indent+r.label+":"+pad+r.value, " "))
	}
	return out
}

func rowLabel(label string) string {
	if to, ok := rowLabels[label]; ok {
		return to
	}
	return label
}

func storyValue(label, value string) string {
	if label == "cpu" && wholeCores.MatchString(value) {
		return view.RationsOf(value)
	}
	if label == cellarResource {
		return lexicon.StoryBarrels(value)
	}
	return value
}

const cellarResource = "ephemeral-storage"

func StoryTable(lines []string) []string {
	header := tableCells.Split(strings.TrimSpace(lines[0]), -1)
	dropped := tableDropped(header)
	rows := make([][]string, 0, len(lines))
	widths := []int{}
	for n, l := range lines {
		indent := l[:len(l)-len(strings.TrimLeft(l, " "))]
		cells := []string{}
		raw := tableCells.Split(strings.TrimSpace(l), -1)
		for c, cell := range raw {
			column := columnOf(header, c)
			if dropped[column] {
				continue
			}
			if c > 0 && raw[0] == cellarResource {
				cell = lexicon.StoryBarrels(cell)
			}
			story := storyCell(header, column, cell)
			if n == 0 {
				if to, ok := headerNames[column]; ok {
					story = to
				}
			}
			if len(cells) >= len(widths) {
				widths = append(widths, 0)
			}
			widths[len(cells)] = max(widths[len(cells)], len([]rune(story)))
			cells = append(cells, story)
		}
		rows = append(rows, append([]string{indent}, cells...))
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		var b strings.Builder
		b.WriteString(r[0])
		for c, cell := range r[1:] {
			b.WriteString(cell)
			if c < len(r)-2 {
				b.WriteString(strings.Repeat(" ", widths[c]-len([]rune(cell))+tableGap))
			}
		}
		out = append(out, b.String())
	}
	return out
}

const (
	unfoldWord  = "more"
	foldMinRows = 3
)

var (
	foldable    = map[string]bool{"Conditions:": true, "Tidings:": true}
	foldedLines = []string{"Standing Class:", "Charms:"}
	keepFolded  = []string{"Capacity:", "For huts:", "Non-terminated huts:", "Allocated resources:"}
	keepSheet   = "For huts:"
)

func foldedLine(line string, heads []string) (string, bool) {
	for _, head := range heads {
		if strings.HasPrefix(line, head) {
			return strings.TrimSuffix(head, ":"), true
		}
	}
	return "", false
}

func foldedHeads(lines []string) []string {
	for _, l := range lines {
		if strings.HasPrefix(l, keepSheet) {
			return append(append([]string{}, foldedLines...), keepFolded...)
		}
	}
	return foldedLines
}

func foldSheet(lines []string) ([]string, []string) {
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "Name:") {
		return lines, nil
	}
	heads := foldedHeads(lines)
	kept, folded, names := []string{}, []string{}, []string{}
	for i := 0; i < len(lines); i++ {
		head := strings.TrimSpace(lines[i])
		if name, one := foldedLine(lines[i], heads); one {
			end := i + 1
			for end < len(lines) && strings.HasPrefix(lines[end], " ") {
				end++
			}
			folded = append(folded, lines[i:end]...)
			names = append(names, name)
			i = end - 1
			continue
		}
		if !foldable[head] || strings.HasPrefix(lines[i], " ") {
			kept = append(kept, lines[i])
			continue
		}
		end := i + 1
		for end < len(lines) && strings.HasPrefix(lines[end], " ") {
			end++
		}
		if end-i-1 < foldMinRows {
			kept = append(kept, lines[i:end]...)
		} else {
			folded = append(folded, lines[i:end]...)
			names = append(names, strings.TrimSuffix(head, ":"))
		}
		i = end - 1
	}
	if len(folded) == 0 {
		return lines, nil
	}
	return append(kept, "Folded away: "+strings.Join(names, ", ")+". Say  "+unfoldWord+"  to read them."), folded
}

func (h *HUD) unfold() []Intent {
	if len(h.Stone.folded) == 0 {
		h.Stone.Append(LineHint, "Nothing is folded away.")
		return []Intent{{Kind: IntentSound, Command: "click"}}
	}
	h.Stone.Append(LineOutput, h.Stone.folded...)
	h.Stone.folded = nil
	return []Intent{{Kind: IntentSound, Command: "click"}}
}
