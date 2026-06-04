package cmd

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mymeetily/mymeetily/internal/layout"
)

const (
	defaultWhisperVersion = "v1.8.5"
	whisperRepo           = "ggml-org/whisper.cpp"
	engineMetadataName    = ".engine-metadata.json"
)

type initEngineOptions struct {
	force          bool
	whisperVersion string
	whisperVariant string
	whisperZip     string
}

type engineMetadata struct {
	Engine      string `json:"engine"`
	Version     string `json:"version"`
	Asset       string `json:"asset"`
	SourceURL   string `json:"source_url"`
	InstalledAt string `json:"installed_at"`
}

func runInitEngineCommand(args []string) error {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return fmt.Errorf("init-engine 当前仅支持 Windows amd64")
	}

	opts := initEngineOptions{
		whisperVersion: defaultWhisperVersion,
		whisperVariant: "cuda-12.4",
	}

	fs := flag.NewFlagSet("init-engine", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.BoolVar(&opts.force, "force", false, "强制覆盖已存在的引擎文件")
	fs.StringVar(&opts.whisperVersion, "whisper-version", opts.whisperVersion, "whisper.cpp 版本，如 v1.8.5 或 latest")
	fs.StringVar(&opts.whisperVariant, "whisper-variant", opts.whisperVariant, "whisper.cpp 变体: cpu | blas | cuda-11.8 | cuda-12.4")
	fs.StringVar(&opts.whisperZip, "whisper-zip", "", "使用本地 whisper.cpp zip 包，跳过在线下载")
	fs.Usage = func() {
		fmt.Println("用法:")
		fmt.Println("  go run . init-engine [flags]")
		fmt.Println()
		fmt.Println("示例:")
		fmt.Println("  go run . init-engine --whisper-variant cuda-12.4")
		fmt.Println("  go run . init-engine --whisper-version v1.7.6 --whisper-variant cuda-12.4")
		fmt.Println("  go run . init-engine --whisper-zip llmengine/whispercpp/Release/whisper-cublas-12.4.0-bin-x64.zip")
		fmt.Println()
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("未知参数: %s", strings.Join(fs.Args(), " "))
	}

	appRoot := findAppRoot()

	fmt.Println("========================================")
	fmt.Println("  MyMeetily — 引擎初始化")
	fmt.Println("========================================")
	fmt.Println()

	if err := installWhisperEngine(appRoot, opts); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  引擎初始化完成")
	return nil
}

func installWhisperEngine(appRoot string, opts initEngineOptions) error {
	version := normalizeReleaseVersion(opts.whisperVersion)
	asset, err := whisperAssetName(opts.whisperVariant)
	if err != nil {
		return err
	}
	url := releaseDownloadURL(whisperRepo, version, asset)
	destDir := layout.WhisperEngineDir(appRoot)
	metadataPath := filepath.Join(destDir, engineMetadataName)
	autoZipPath := filepath.Join(destDir, asset)
	source, err := resolveEngineArchiveSource(url, opts.whisperZip, autoZipPath)
	if err != nil {
		return err
	}
	requiredFiles := []string{
		filepath.Join(destDir, "whisper-cli.exe"),
		filepath.Join(destDir, "whisper.dll"),
		filepath.Join(destDir, "ggml.dll"),
	}

	fmt.Println("=== 初始化 whisper.cpp 引擎 ===")
	fmt.Printf("目标版本: %s\n", version)
	fmt.Printf("下载变体: %s\n", asset)
	fmt.Printf("安装来源: %s\n", source.display)
	fmt.Printf("安装目录: %s\n", destDir)

	status, meta, err := inspectEngine(metadataPath, version, asset, requiredFiles...)
	if err != nil {
		return err
	}
	if !opts.force {
		switch status {
		case engineStatusExact:
			fmt.Println("已检测到目标版本 whisper.cpp，引擎初始化跳过。")
			return nil
		case engineStatusUnknown:
			fmt.Println("检测到现有 whisper.cpp，但版本未知；为避免意外覆盖，本次跳过。")
			fmt.Println("如需按指定版本重新下载，请追加 --force。")
			return nil
		case engineStatusDifferent:
			fmt.Printf("检测到现有 whisper.cpp 版本为 %s，将覆盖为 %s。\n", meta.Version, version)
		}
	}

	if err := installFromZip(destDir, func(tmpZip, stageDir string) error {
		if err := source.populate(tmpZip); err != nil {
			return err
		}
		if err := extractAllFilesFromZip(tmpZip, stageDir); err != nil {
			return err
		}
		if err := normalizeWhisperStageLayout(stageDir); err != nil {
			return err
		}
		if err := ensureRequiredFiles("whisper.cpp",
			filepath.Join(stageDir, "whisper-cli.exe"),
			filepath.Join(stageDir, "whisper.dll"),
			filepath.Join(stageDir, "ggml.dll"),
		); err != nil {
			return err
		}
		return writeEngineMetadata(filepath.Join(stageDir, engineMetadataName), engineMetadata{
			Engine:      "whisper.cpp",
			Version:     version,
			Asset:       asset,
			SourceURL:   source.metadataSource,
			InstalledAt: time.Now().Format(time.RFC3339),
		})
	}); err != nil {
		return fmt.Errorf("安装 whisper.cpp 引擎失败: %w", err)
	}

	fmt.Println("whisper.cpp 引擎已就绪。")
	return nil
}

type engineInstallStatus int

type engineArchiveSource struct {
	display        string
	metadataSource string
	populate       func(dst string) error
}

const (
	engineStatusMissing engineInstallStatus = iota
	engineStatusUnknown
	engineStatusDifferent
	engineStatusExact
)

func inspectEngine(metadataPath, expectedVersion, expectedAsset string, requiredFiles ...string) (engineInstallStatus, engineMetadata, error) {
	for _, file := range requiredFiles {
		if !fileExists(file) {
			return engineStatusMissing, engineMetadata{}, nil
		}
	}

	meta, err := readEngineMetadata(metadataPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return engineStatusUnknown, engineMetadata{}, nil
		}
		return engineStatusMissing, engineMetadata{}, err
	}

	if meta.Version == "" || meta.Asset == "" {
		return engineStatusUnknown, meta, nil
	}

	if meta.Version == expectedVersion && meta.Asset == expectedAsset {
		return engineStatusExact, meta, nil
	}

	return engineStatusDifferent, meta, nil
}

func readEngineMetadata(path string) (engineMetadata, error) {
	var meta engineMetadata
	data, err := os.ReadFile(path)
	if err != nil {
		return meta, err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

func writeEngineMetadata(path string, meta engineMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func installFromZip(destDir string, install func(tmpZip, stageDir string) error) error {
	parentDir := filepath.Dir(destDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return err
	}

	tmpZip, err := os.CreateTemp(parentDir, "engine-*.zip")
	if err != nil {
		return err
	}
	tmpZipPath := tmpZip.Name()
	_ = tmpZip.Close()
	defer os.Remove(tmpZipPath)

	stageDir, err := os.MkdirTemp(parentDir, filepath.Base(destDir)+"-staging-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stageDir)

	if err := install(tmpZipPath, stageDir); err != nil {
		return err
	}

	if err := os.RemoveAll(destDir); err != nil {
		return err
	}

	return os.Rename(stageDir, destDir)
}

func resolveEngineArchiveSource(downloadURL, localZipPath, autoZipPath string) (engineArchiveSource, error) {
	localZipPath = strings.TrimSpace(localZipPath)
	if localZipPath != "" {
		return newLocalEngineArchiveSource(localZipPath)
	}

	if strings.TrimSpace(autoZipPath) != "" && fileExists(autoZipPath) {
		return newLocalEngineArchiveSource(autoZipPath)
	}

	return engineArchiveSource{
		display:        downloadURL,
		metadataSource: downloadURL,
		populate: func(dst string) error {
			return downloadFile(dst, downloadURL)
		},
	}, nil
}

func newLocalEngineArchiveSource(localZipPath string) (engineArchiveSource, error) {
	absPath, err := filepath.Abs(strings.TrimSpace(localZipPath))
	if err != nil {
		return engineArchiveSource{}, fmt.Errorf("解析本地 zip 路径失败: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return engineArchiveSource{}, fmt.Errorf("访问本地 zip 失败: %w", err)
	}
	if info.IsDir() {
		return engineArchiveSource{}, fmt.Errorf("本地 zip 路径不能是目录: %s", absPath)
	}

	return engineArchiveSource{
		display:        absPath,
		metadataSource: absPath,
		populate: func(dst string) error {
			return copyFile(absPath, dst)
		},
	}, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("复制本地文件失败: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("关闭目标文件失败: %w", err)
	}

	return nil
}

func ensureRequiredFiles(engineName string, requiredFiles ...string) error {
	missing := make([]string, 0, len(requiredFiles))
	for _, path := range requiredFiles {
		if !fileExists(path) {
			missing = append(missing, filepath.Base(path))
		}
	}
	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf("%s 解压后缺少关键文件: %s", engineName, strings.Join(missing, ", "))
}

func normalizeWhisperStageLayout(stageDir string) error {
	nestedReleaseDir := filepath.Join(stageDir, "Release")
	if !dirExists(nestedReleaseDir) {
		return nil
	}
	if fileExists(filepath.Join(stageDir, "whisper-cli.exe")) || !fileExists(filepath.Join(nestedReleaseDir, "whisper-cli.exe")) {
		return nil
	}

	entries, err := os.ReadDir(nestedReleaseDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		src := filepath.Join(nestedReleaseDir, entry.Name())
		dst := filepath.Join(stageDir, entry.Name())
		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}

	return os.RemoveAll(nestedReleaseDir)
}

func extractSingleFileFromZip(zipPath, entryBaseName, destPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.EqualFold(filepath.Base(f.Name), entryBaseName) {
			if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
				return err
			}
			return extractZipFile(f, destPath)
		}
	}

	return fmt.Errorf("压缩包中未找到 %s", entryBaseName)
}

func extractAllFilesFromZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target, err := zipEntryTargetPath(destDir, f.Name)
		if err != nil {
			return err
		}
		if target == "" {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := extractZipFile(f, target); err != nil {
			return err
		}
	}

	return nil
}

func zipEntryTargetPath(destDir, entryName string) (string, error) {
	name := filepath.Clean(filepath.FromSlash(entryName))
	if name == "." || name == "" {
		return "", nil
	}
	if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("压缩包条目路径非法: %s", entryName)
	}

	target := filepath.Join(destDir, name)
	cleanDest := filepath.Clean(destDir)
	cleanTarget := filepath.Clean(target)
	if cleanTarget != cleanDest && !strings.HasPrefix(cleanTarget, cleanDest+string(filepath.Separator)) {
		return "", fmt.Errorf("压缩包条目越界: %s", entryName)
	}

	return cleanTarget, nil
}

func extractZipFile(file *zip.File, destPath string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

func whisperAssetName(variant string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(variant)) {
	case "", "cpu":
		return "whisper-bin-x64.zip", nil
	case "blas":
		return "whisper-blas-bin-x64.zip", nil
	case "cuda-11.8", "cuda11.8", "cublas-11.8":
		return "whisper-cublas-11.8.0-bin-x64.zip", nil
	case "cuda-12.4", "cuda12.4", "cublas-12.4":
		return "whisper-cublas-12.4.0-bin-x64.zip", nil
	default:
		return "", fmt.Errorf("不支持的 whisper 变体: %s", variant)
	}
}

func normalizeReleaseVersion(version string) string {
	v := strings.TrimSpace(version)
	if v == "" || strings.EqualFold(v, "latest") {
		return "latest"
	}
	if strings.HasPrefix(strings.ToLower(v), "v") {
		return "v" + v[1:]
	}
	return "v" + v
}

func releaseDownloadURL(repo, version, asset string) string {
	if version == "latest" {
		return fmt.Sprintf("https://github.com/%s/releases/latest/download/%s", repo, asset)
	}
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, version, asset)
}
