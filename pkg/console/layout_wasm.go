//go:build js || wasm

package console

import "strings"

func LayoutTitleBox(title string, width int) string {
	separator := strings.Repeat("=", width)
	return separator + "\n  " + title + "\n" + separator
}

func LayoutInfoSection(label, value string) string {
	return "  " + label + ": " + value
}

func LayoutEmphasisBox(content string, color interface{}) string {
	marker := strings.Repeat("!", len(content)+4)
	return marker + "\n  " + content + "\n" + marker
}

func LayoutJoinVertical(sections ...string) string {
	return strings.Join(sections, "\n")
}
