package ui

type Scroll struct {
	Offset int
}

func MaxOffset(totalLines, visibleLines int) int {
	return max(totalLines-visibleLines, 0)
}

func (s Scroll) By(delta, totalLines, visibleLines int) Scroll {
	next := s.Offset + delta
	return Scroll{Offset: min(max(next, 0), MaxOffset(totalLines, visibleLines))}
}

func (s Scroll) Clamp(totalLines, visibleLines int) Scroll {
	return s.By(0, totalLines, visibleLines)
}

func (s Scroll) Window(lines []string, visible int) []string {
	start := min(max(s.Offset, 0), len(lines))
	end := min(start+visible, len(lines))
	return lines[start:end]
}

func VisibleLines(height, scale int) int {
	return max(height/(LineHeight*max(scale, 1)), 1)
}
