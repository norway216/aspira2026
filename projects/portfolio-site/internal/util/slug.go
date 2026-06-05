package util

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	nonSlugRe       = regexp.MustCompile(`[^a-z0-9-]+`)
	multiDashRe     = regexp.MustCompile(`-+`)
	leadingTrailing = regexp.MustCompile(`^-|-$`)
)

// Slugify converts a string to a URL-friendly slug.
func Slugify(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	// Replace common separators with dashes.
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")

	// Remove all non-slug characters.
	s = nonSlugRe.ReplaceAllString(s, "")

	// Collapse multiple dashes.
	s = multiDashRe.ReplaceAllString(s, "-")

	// Trim leading and trailing dashes.
	s = leadingTrailing.ReplaceAllString(s, "")

	if s == "" {
		return "untitled"
	}
	return s
}

// TruncateWords truncates a string to n words.
func TruncateWords(s string, n int) string {
	words := strings.Fields(s)
	if len(words) <= n {
		return s
	}
	return strings.Join(words[:n], " ") + "..."
}

// TruncateRunes truncates a string to n characters (runes).
func TruncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

// WordCount counts the number of words in a string.
func WordCount(s string) int {
	return len(strings.Fields(s))
}

// ReadingTime estimates reading time in minutes.
// Assumes average reading speed of 200 words per minute for technical content.
func ReadingTime(wordCount int) int {
	if wordCount <= 0 {
		return 1
	}
	minutes := wordCount / 200
	if wordCount%200 > 0 {
		minutes++
	}
	if minutes < 1 {
		minutes = 1
	}
	return minutes
}

// Ellipsis is the Unicode ellipsis character.
const Ellipsis = "…"

// SanitizeString removes non-printable characters.
func SanitizeString(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, s)
}
