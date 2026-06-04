package cmd

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/mymeetily/mymeetily/internal/layout"
)

func TestNormalizeReleaseVersion(t *testing.T) {
	tests := map[string]string{
		"":         "latest",
		"latest":   "latest",
		"v1.2.3":   "v1.2.3",
		"1.2.3":    "v1.2.3",
		" V0.9.0 ": "v0.9.0",
	}

	for input, want := range tests {
		if got := normalizeReleaseVersion(input); got != want {
			t.Fatalf("normalizeReleaseVersion(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestReleaseDownloadURL(t *testing.T) {
	got := releaseDownloadURL("ggml-org/whisper.cpp", "latest", "whisper-cublas-12.4.0-bin-x64.zip")
	want := "https://github.com/ggml-org/whisper.cpp/releases/latest/download/whisper-cublas-12.4.0-bin-x64.zip"
	if got != want {
		t.Fatalf("releaseDownloadURL(latest) = %q, want %q", got, want)
	}

	got = releaseDownloadURL("ggml-org/whisper.cpp", "v1.8.5", "whisper-bin-x64.zip")
	want = "https://github.com/ggml-org/whisper.cpp/releases/download/v1.8.5/whisper-bin-x64.zip"
	if got != want {
		t.Fatalf("releaseDownloadURL(versioned) = %q, want %q", got, want)
	}
}

func TestWhisperAssetName(t *testing.T) {
	tests := map[string]string{
		"cpu":         "whisper-bin-x64.zip",
		"blas":        "whisper-blas-bin-x64.zip",
		"cuda-11.8":   "whisper-cublas-11.8.0-bin-x64.zip",
		"cuda-12.4":   "whisper-cublas-12.4.0-bin-x64.zip",
		"cublas-12.4": "whisper-cublas-12.4.0-bin-x64.zip",
	}

	for input, want := range tests {
		got, err := whisperAssetName(input)
		if err != nil {
			t.Fatalf("whisperAssetName(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("whisperAssetName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestExtractAllFilesFromZipPreservesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "bundle.zip")
	destDir := filepath.Join(tmpDir, "dest")

	createZipArchive(t, zipPath, map[string]string{
		"nested/bin/whisper-cli.exe": "binary",
		"nested/lib/ggml.dll":        "library",
	})

	if err := extractAllFilesFromZip(zipPath, destDir); err != nil {
		t.Fatalf("extractAllFilesFromZip: %v", err)
	}

	for name, want := range map[string]string{
		"nested/bin/whisper-cli.exe": "binary",
		"nested/lib/ggml.dll":        "library",
	} {
		data, err := os.ReadFile(filepath.Join(destDir, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("read extracted file %q: %v", name, err)
		}
		if string(data) != want {
			t.Fatalf("content for %q = %q, want %q", name, string(data), want)
		}
	}
}

func TestInstallWhisperEngineFromLocalZip(t *testing.T) {
	appRoot := t.TempDir()
	archivePath := filepath.Join(layout.WhisperEngineDir(appRoot), "whisper-cublas-12.4.0-bin-x64.zip")
	createZipArchive(t, archivePath, map[string]string{
		"whisper-cli.exe": "cli",
		"whisper.dll":     "dll",
		"ggml.dll":        "ggml",
	})

	if err := installWhisperEngine(appRoot, initEngineOptions{
		whisperVersion: defaultWhisperVersion,
		whisperVariant: "cuda-12.4",
	}); err != nil {
		t.Fatalf("installWhisperEngine(local zip): %v", err)
	}

	destDir := layout.WhisperEngineDir(appRoot)
	for _, name := range []string{"whisper-cli.exe", "whisper.dll", "ggml.dll"} {
		if !fileExists(filepath.Join(destDir, name)) {
			t.Fatalf("expected file %q to be installed", name)
		}
	}

	meta, err := readEngineMetadata(filepath.Join(destDir, engineMetadataName))
	if err != nil {
		t.Fatalf("readEngineMetadata: %v", err)
	}
	archiveAbs, err := filepath.Abs(archivePath)
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	if meta.Engine != "whisper.cpp" {
		t.Fatalf("meta.Engine = %q", meta.Engine)
	}
	if meta.Version != defaultWhisperVersion {
		t.Fatalf("meta.Version = %q, want %q", meta.Version, defaultWhisperVersion)
	}
	if meta.Asset != "whisper-cublas-12.4.0-bin-x64.zip" {
		t.Fatalf("meta.Asset = %q", meta.Asset)
	}
	if meta.SourceURL != archiveAbs {
		t.Fatalf("meta.SourceURL = %q, want %q", meta.SourceURL, archiveAbs)
	}
}

func TestResolveEngineArchiveSourceLocalZip(t *testing.T) {
	tmpDir := t.TempDir()
	localZip := filepath.Join(tmpDir, "whisper.zip")
	if err := os.WriteFile(localZip, []byte("zip"), 0o644); err != nil {
		t.Fatalf("write local zip: %v", err)
	}

	source, err := resolveEngineArchiveSource("https://example.invalid/download.zip", localZip, "")
	if err != nil {
		t.Fatalf("resolveEngineArchiveSource: %v", err)
	}
	if source.display == "" {
		t.Fatalf("expected display to be set")
	}
}

func createZipArchive(t *testing.T, zipPath string, entries map[string]string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		t.Fatalf("mkdir zip dir: %v", err)
	}

	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}

	zw := zip.NewWriter(file)
	for name, content := range entries {
		writer, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %q: %v", name, err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}
}