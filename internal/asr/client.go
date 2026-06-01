package asr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Client runs whisper.cpp as a subprocess for speech recognition.
type Client struct {
	binary   string // path to whisper-cli executable
	model    string // path to GGUF model file
	language string // default language
}

// Segment represents a timed transcript fragment.
type Segment struct {
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
}

// TranscribeResult holds the full ASR output.
type TranscribeResult struct {
	Text     string    `json:"text"`
	Language string    `json:"language"`
	Duration float64   `json:"duration"`
	Segments []Segment `json:"segments"`
}

// whisperJSON is the JSON output from `whisper-cli -oj`.
type whisperJSON struct {
	Text     string            `json:"text"`
	Language string            `json:"language"`
	Segments []whisperSeg      `json:"segments"`
}

type whisperSeg struct {
	Start float64 `json:"t0"`
	End   float64 `json:"t1"`
	Text  string  `json:"text"`
}

// NewClient creates an ASR client backed by whisper.cpp.
func NewClient(binary, model, language string) *Client {
	return &Client{
		binary:   binary,
		model:    model,
		language: language,
	}
}

// Transcribe runs whisper-cli as a subprocess and parses the output.
func (c *Client) Transcribe(ctx context.Context, audioPath, language string) (*TranscribeResult, error) {
	if language == "" {
		language = c.language
	}

	args := []string{
		"-m", c.model,
		"-f", audioPath,
		"-l", language,
		"-oj", // JSON output
	}
	cmd := exec.CommandContext(ctx, c.binary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("whisper-cli: %w\nstderr: %s", err, stderr.String())
	}

	raw := strings.TrimSpace(stdout.String())
	raw = stripMarkdownFence(raw)

	// Try JSON first, fall back to timestamp format.
	var wj whisperJSON
	if err := json.Unmarshal([]byte(raw), &wj); err == nil && wj.Text != "" {
		// JSON format (whisper.cpp -oj in some versions).
		result := &TranscribeResult{
			Text:     strings.TrimSpace(wj.Text),
			Language: wj.Language,
		}
		if result.Language == "" {
			result.Language = language
		}
		for _, s := range wj.Segments {
			result.Segments = append(result.Segments, Segment{
				Start:      s.Start / 100.0,
				End:        s.End / 100.0,
				Text:       strings.TrimSpace(s.Text),
				Confidence: 1.0,
			})
		}
		if len(result.Segments) > 0 {
			result.Duration = result.Segments[len(result.Segments)-1].End
		}
		return result, nil
	}

	// Timestamp format: [HH:MM:SS.mmm --> HH:MM:SS.mmm]  text
	lines := strings.Split(raw, "\n")
	result := &TranscribeResult{Language: language}
	var fullText strings.Builder
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		seg := parseTimestampLine(line)
		if seg.Text == "" {
			continue
		}
		result.Segments = append(result.Segments, seg)
		if fullText.Len() > 0 {
			fullText.WriteString(" ")
		}
		fullText.WriteString(seg.Text)
	}
	result.Text = fullText.String()
	if len(result.Segments) > 0 {
		result.Duration = result.Segments[len(result.Segments)-1].End
	}
	return result, nil
}

// parseTimestampLine extracts a Segment from a whisper timestamp line like
// "[00:00:00.000 --> 00:00:02.500]  some text".
func parseTimestampLine(line string) Segment {
	var seg Segment
	// Find "]  " or "] " separator.
	idx := strings.Index(line, "]  ")
	if idx < 0 {
		idx = strings.Index(line, "] ")
	}
	if idx < 0 {
		return seg
	}
	text := strings.TrimSpace(line[idx+1:])
	// Trim the bracket prefix.
	if strings.HasPrefix(text, " ") {
		text = strings.TrimSpace(text)
	}
	seg.Text = text

	// Parse timestamps [HH:MM:SS.mmm --> HH:MM:SS.mmm]
	ts := line[1:idx] // "HH:MM:SS.mmm --> HH:MM:SS.mmm"
	parts := strings.Split(ts, " --> ")
	if len(parts) == 2 {
		seg.Start = parseTimestamp(parts[0])
		seg.End = parseTimestamp(parts[1])
	}
	seg.Confidence = 1.0
	return seg
}

// parseTimestamp converts "HH:MM:SS.mmm" to seconds as float64.
func parseTimestamp(s string) float64 {
	var h, m int
	var sec float64
	fmt.Sscanf(s, "%d:%d:%f", &h, &m, &sec)
	return float64(h*3600+m*60) + sec
}

// stripMarkdownFence removes ```json ... ``` wrapping if present.
func stripMarkdownFence(s string) string {
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
