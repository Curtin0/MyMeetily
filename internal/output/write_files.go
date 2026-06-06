package output

import (
	"fmt"
	"os"
	"path/filepath"
)

func WriteHTML(markdown, transcript string, segments []TranscriptSegment, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	return os.WriteFile(outputPath, []byte(RenderHTML(markdown, transcript, segments)), 0o644)
}

func WriteMarkdown(content, outputPath string) error {
	return writeTextFile(content, outputPath)
}

func WriteText(content, outputPath string) error {
	return writeTextFile(content, outputPath)
}

func writeTextFile(content, outputPath string) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}

	return nil
}
