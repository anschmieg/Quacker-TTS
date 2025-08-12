package util

import (
	"regexp"
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
