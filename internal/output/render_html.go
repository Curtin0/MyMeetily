package output

import (
	"fmt"
	"regexp"
	"strings"
)

type TranscriptSegment struct {
	Start float64
	End   float64
	Text  string
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>会议纪要</title>
<style>
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    max-width: 760px;
    margin: 48px auto;
    padding: 0 24px;
    background: #fff;
    color: #333;
    line-height: 1.85;
    font-size: 15px;
  }
  h1 {
    font-size: 22px;
    font-weight: 600;
    color: #222;
    border-bottom: 1px solid #e0e0e0;
    padding-bottom: 12px;
    margin-bottom: 24px;
  }
  h2 { font-size: 17px; font-weight: 600; color: #333; margin-top: 36px; }
  h3 { font-size: 15px; font-weight: 600; color: #444; }
  strong { color: #222; }
  blockquote {
    border-left: 3px solid #d0d0d0;
    margin: 16px 0;
    padding: 8px 16px;
    background: #fafafa;
    color: #666;
    border-radius: 0 4px 4px 0;
  }
  table { width: 100%%; border-collapse: collapse; margin: 16px 0; font-size: 14px; }
  th, td { border: 1px solid #e8e8e8; padding: 8px 12px; text-align: left; }
  th { background: #f5f5f5; color: #333; font-weight: 600; }
  hr { border: none; border-top: 1px solid #e8e8e8; margin: 28px 0; }
  ul, ol { padding-left: 24px; }
  li { margin: 4px 0; }
  code { background: #f5f5f5; padding: 2px 6px; border-radius: 3px; font-size: 13px; }
  pre { background: #f5f5f5; color: #333; padding: 16px; border-radius: 6px; overflow-x: auto; font-size: 13px; }
  .decision { background: #f9fafb; border-left: 3px solid #555; padding: 8px 12px; margin: 8px 0; border-radius: 0 4px 4px 0; }
  .todo { background: #fafafa; border-left: 3px solid #999; padding: 8px 12px; margin: 8px 0; border-radius: 0 4px 4px 0; }
  .insight { background: #f9f9f9; border-left: 3px solid #666; padding: 8px 12px; margin: 8px 0; border-radius: 0 4px 4px 0; }
  details.transcript {
    margin-top: 32px;
    border: 1px solid #e8e8e8;
    border-radius: 6px;
    background: #fcfcfc;
    overflow: hidden;
  }
  details.transcript summary {
    cursor: pointer;
    list-style: none;
    padding: 14px 16px;
    font-weight: 600;
    color: #222;
    background: #f7f7f7;
  }
  details.transcript summary::-webkit-details-marker { display: none; }
  details.transcript summary::after {
    content: "展开";
    float: right;
    color: #666;
    font-weight: 400;
    font-size: 13px;
  }
  details.transcript[open] summary::after { content: "收起"; }
  .transcript-body {
    padding: 16px;
    border-top: 1px solid #e8e8e8;
    background: #fff;
  }
  .transcript-segments {
    display: grid;
    gap: 12px;
  }
  .transcript-segment {
    display: grid;
    grid-template-columns: 88px minmax(0, 1fr);
    gap: 12px;
    align-items: start;
  }
  .transcript-time {
    color: #666;
    font-size: 12px;
    line-height: 1.6;
    font-variant-numeric: tabular-nums;
  }
  .transcript-text {
    margin: 0;
    white-space: pre-wrap;
    word-break: break-word;
    line-height: 1.85;
  }
  .transcript-raw {
    margin: 0;
    white-space: pre-wrap;
    word-break: break-word;
    line-height: 1.85;
  }
</style>
</head>
<body>
%s
</body>
</html>`

func RenderHTML(markdown, transcript string, segments []TranscriptSegment) string {
	html := markdownToHTML(markdown) + renderTranscriptDetails(transcript, segments)
	return fmt.Sprintf(htmlTemplate, html)
}

func markdownToHTML(md string) string {
	var b strings.Builder
	lines := strings.Split(md, "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		switch {
		case strings.HasPrefix(line, "# "):
			b.WriteString("<h1>" + escapeHTML(line[2:]) + "</h1>\n")
		case strings.HasPrefix(line, "## "):
			b.WriteString("<h2>" + escapeHTML(line[3:]) + "</h2>\n")
		case strings.HasPrefix(line, "### "):
			b.WriteString("<h3>" + escapeHTML(line[4:]) + "</h3>\n")
		case strings.HasPrefix(line, "> "):
			b.WriteString("<blockquote>" + processInline(line[2:]) + "</blockquote>\n")
		case strings.HasPrefix(line, "---"):
			b.WriteString("<hr>\n")
		case strings.HasPrefix(line, "- **D"):
			b.WriteString("<div class=\"decision\">" + processInline(line[2:]) + "</div>\n")
		case strings.HasPrefix(line, "- **TODO"):
			b.WriteString("<div class=\"todo\">" + processInline(line[2:]) + "</div>\n")
		case strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* "):
			b.WriteString("<li>" + processInline(line[2:]) + "</li>\n")
		case strings.TrimSpace(line) == "":
			b.WriteString("<br>\n")
		default:
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				b.WriteString("<p>" + processInline(trimmed) + "</p>\n")
			}
		}
	}

	return b.String()
}

func processInline(text string) string {
	reBold := regexp.MustCompile(`\*\*(.+?)\*\*`)
	text = reBold.ReplaceAllString(text, "<strong>$1</strong>")

	reCode := regexp.MustCompile("`(.+?)`")
	text = reCode.ReplaceAllString(text, "<code>$1</code>")

	return text
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func renderTranscriptDetails(transcript string, segments []TranscriptSegment) string {
	transcript = strings.TrimSpace(transcript)
	if transcript == "" && len(segments) == 0 {
		return ""
	}

	return "<details class=\"transcript\">\n" +
		"<summary>原文转写</summary>\n" +
		renderTranscriptBody(transcript, segments) +
		"</details>\n"
}

func renderTranscriptBody(transcript string, segments []TranscriptSegment) string {
	if len(segments) == 0 {
		return "<div class=\"transcript-body\"><pre class=\"transcript-raw\">" + escapeHTML(transcript) + "</pre></div>\n"
	}

	var b strings.Builder
	b.WriteString("<div class=\"transcript-body\">\n")
	b.WriteString("<div class=\"transcript-segments\">\n")
	for _, seg := range segments {
		text := strings.TrimSpace(seg.Text)
		if text == "" {
			continue
		}
		b.WriteString("<div class=\"transcript-segment\">\n")
		b.WriteString("<div class=\"transcript-time\">")
		b.WriteString(escapeHTML(formatSegmentTime(seg.Start, seg.End)))
		b.WriteString("</div>\n")
		b.WriteString("<p class=\"transcript-text\">")
		b.WriteString(escapeHTML(text))
		b.WriteString("</p>\n")
		b.WriteString("</div>\n")
	}
	b.WriteString("</div>\n")
	b.WriteString("</div>\n")
	return b.String()
}

func formatSegmentTime(start, end float64) string {
	if end <= start || end <= 0 {
		return formatClock(start)
	}
	return formatClock(start) + " - " + formatClock(end)
}

func formatClock(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	total := int(seconds + 0.5)
	hours := total / 3600
	minutes := (total % 3600) / 60
	secs := total % 60
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}
