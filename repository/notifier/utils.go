package notifier

import (
	"html"
	"strings"
)

// escapeMarkdown escapes Discord Markdown special characters.
func escapeMarkdown(in string) string {
	// Escape special characters used in Markdown: *, _, `, ~, |
	replacer := strings.NewReplacer(
		"*", "\\*",
		"_", "\\_",
		"`", "\\`",
		"~", "\\~",
		"|", "\\|",
	)
	return replacer.Replace(replaceNewlines(in))
}

func escapeHtml(in string) string {
	return html.EscapeString(replaceNewlines(in))
}

func replaceNewlines(in string) string {
	return strings.ReplaceAll(in, "\n", "\\n")
}
