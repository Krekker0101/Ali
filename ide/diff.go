package ide

import (
	"fmt"
	"strings"
)

func BuildChange(path string, action string, before string, after string) FileChange {
	change := FileChange{
		Path:                 path,
		Action:               action,
		Before:               before,
		After:                after,
		RequiresConfirmation: action == actionDelete,
	}
	change.Diff = UnifiedDiff(path, action, before, after)
	return change
}

func UnifiedDiff(path string, action string, before string, after string) string {
	var b strings.Builder
	switch action {
	case actionCreate:
		fmt.Fprintf(&b, "--- /dev/null\n+++ b/%s\n", path)
		writeHunkHeader(&b, 0, 0, 1, lineCount(after))
		for _, line := range splitDiffLines(after) {
			b.WriteString("+")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	case actionDelete:
		fmt.Fprintf(&b, "--- a/%s\n+++ /dev/null\n", path)
		writeHunkHeader(&b, 1, lineCount(before), 0, 0)
		for _, line := range splitDiffLines(before) {
			b.WriteString("-")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	default:
		fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", path, path)
		writeHunkHeader(&b, 1, lineCount(before), 1, lineCount(after))
		for _, line := range splitDiffLines(before) {
			b.WriteString("-")
			b.WriteString(line)
			b.WriteByte('\n')
		}
		for _, line := range splitDiffLines(after) {
			b.WriteString("+")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func writeHunkHeader(b *strings.Builder, oldStart int, oldCount int, newStart int, newCount int) {
	fmt.Fprintf(b, "@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount)
}

func lineCount(s string) int {
	lines := splitDiffLines(s)
	if len(lines) == 0 {
		return 0
	}
	return len(lines)
}

func splitDiffLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return []string{""}
	}
	return strings.Split(s, "\n")
}
