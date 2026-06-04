package agent

import (
	"regexp"
	"strings"
)

var whitespaceRE = regexp.MustCompile(`\s+`)

func cleanSegments(segments []Segment) []Segment {
	cleaned := make([]Segment, 0, len(segments))
	lastKey := ""
	for _, seg := range segments {
		text := normalizeTranscriptText(seg.Text)
		if shouldSkipTranscriptText(text) {
			continue
		}

		key := strings.ToLower(text)
		if key == lastKey {
			continue
		}
		lastKey = key
		seg.Text = text
		cleaned = append(cleaned, seg)
	}
	return cleaned
}

func buildTranscriptFromSegments(segments []Segment, fallback string) string {
	if len(segments) == 0 {
		return normalizeTranscriptText(fallback)
	}

	parts := make([]string, 0, len(segments))
	for _, seg := range segments {
		text := normalizeTranscriptText(seg.Text)
		if shouldSkipTranscriptText(text) {
			continue
		}
		parts = append(parts, text)
	}
	if len(parts) == 0 {
		return normalizeTranscriptText(fallback)
	}
	return strings.Join(parts, "\n")
}

func normalizeTranscriptText(text string) string {
	return whitespaceRE.ReplaceAllString(strings.TrimSpace(text), " ")
}

func shouldSkipTranscriptText(text string) bool {
	if strings.TrimSpace(text) == "" {
		return true
	}
	if strings.Trim(text, ".,!?;:，。！？；：、()[]{}<>\"'`~_-") == "" {
		return true
	}
	return false
}
