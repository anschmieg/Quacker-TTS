package util

import (
	"regexp"
	"strings"
)

// SanitizeFilenameWord cleans a single word for use in a filename.
// It removes disallowed characters and truncates if necessary.
func SanitizeFilenameWord(word string) string {
	// Allow letters, numbers, underscore, dot, hyphen
	reg := regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
	sanitized := reg.ReplaceAllString(word, "_")
	// Limit length to avoid excessively long filenames
	const maxLen = 28
	if len(sanitized) > maxLen {
		return sanitized[:maxLen]
	}
	return sanitized
}

// CleanJSONString prepares a string for embedding within a JSON payload.
// It escapes backslashes, double quotes, and newlines.
func CleanJSONString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)   // Escape backslashes first
	s = strings.ReplaceAll(s, `"`, `\"`)   // Escape double quotes
	s = strings.ReplaceAll(s, "\n", "\\n") // Escape newlines
	// Note: Single quotes don't need escaping in standard JSON
	return s
}
