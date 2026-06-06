package output

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "subdir", "test.md")

	content := "# Test\n\nHello World!"
	if err := WriteMarkdown(content, outputPath); err != nil {
		t.Fatalf("WriteMarkdown failed: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if string(data) != content {
		t.Errorf("content mismatch: got %q, want %q", string(data), content)
	}
}
