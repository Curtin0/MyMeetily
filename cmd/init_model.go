package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mymeetily/mymeetily/internal/config"
	"github.com/mymeetily/mymeetily/internal/ollama"
)

const defaultWhisperModelURL = "https://hf-mirror.com/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo-q5_0.bin"

const defaultSummaryModelURL = "https://hf-mirror.com/Qwen/Qwen2.5-1.5B-Instruct-GGUF/resolve/main/qwen2.5-1.5b-instruct-q4_k_m.gguf"

const defaultSummaryModelPath = "./llmmodels/summary/qwen2.5-1.5b-instruct-q4_k_m.gguf"

type initModelOptions struct {
	force         bool
	whisperURL    string
	whisperSource string
}

func runInitModelCommand(args []string) error {
	opts := initModelOptions{
		whisperURL: defaultWhisperModelURL,
	}

	fs := flag.NewFlagSet("init-model", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.BoolVar(&opts.force, "force", false, "强制重新下载 Whisper 模型")
	fs.StringVar(&opts.whisperURL, "whisper-url", opts.whisperURL, "覆盖默认 Whisper 模型下载地址")
	fs.StringVar(&opts.whisperSource, "whisper-source", "", "使用本地 Whisper 模型文件或自定义源路径，优先于 --whisper-url")
	fs.Usage = func() {
		fmt.Println("用法:")
		fmt.Println("  go run . init-model [flags]")
		fmt.Println()
		fmt.Println("示例:")
		fmt.Println("  go run . init-model")
		fmt.Println("  go run . init-model --force")
		fmt.Println()
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("未知参数: %s", strings.Join(fs.Args(), " "))
	}
	if strings.TrimSpace(opts.whisperSource) != "" {
		fmt.Printf("Whisper 模型源: %s\n", strings.TrimSpace(opts.whisperSource))
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	ctx := context.Background()

	fmt.Println("========================================")
	fmt.Println("  MyMeetily — 模型初始化")
	fmt.Println("========================================")
	fmt.Println()

	fmt.Println("--- Whisper 模型 ---")
	if err := ensureWhisperModel(cfg.ASR.ModelPath, opts.whisperURL, opts.whisperSource, opts.force); err != nil {
		return err
	}
	fmt.Println()

	fmt.Println("--- Ollama 总结模型 ---")
	ollamaClient := ollama.NewClient(cfg.Ollama.Endpoint, cfg.Ollama.Model, cfg.Ollama.Temperature, cfg.Ollama.MaxTokens)
	if err := ollamaClient.Ping(ctx); err != nil {
		return fmt.Errorf("请先启动 Ollama 服务: %w", err)
	}
	if err := ollamaClient.EnsureModelAvailable(ctx, defaultSummaryModelPath, defaultSummaryModelURL, false); err != nil {
		return fmt.Errorf("总结模型准备失败: %w", err)
	}
	fmt.Println()

	fmt.Println("========================================")
	fmt.Println("  模型初始化完成")
	fmt.Println("========================================")
	return nil
}

func ensureWhisperModel(modelPath, modelURL, modelSource string, force bool) error {
	if !force {
		if _, err := os.Stat(modelPath); err == nil {
			fmt.Printf("Whisper 模型已就绪: %s ✓\n", modelPath)
			return nil
		}
	}

	if force {
		fmt.Printf("正在重新下载 Whisper 模型到: %s\n", modelPath)
	} else {
		fmt.Printf("未找到 Whisper 模型: %s\n", modelPath)
		fmt.Println("正在自动准备 Whisper Large-v3 Turbo (Q5_0, ~548MB)...")
		fmt.Println("建议优先使用国内镜像或本地文件，以获得更稳定的下载体验。")
	}
	if strings.TrimSpace(modelSource) != "" {
		fmt.Printf("模型来源: %s\n", modelSource)
	} else {
		fmt.Printf("下载地址: %s\n", modelURL)
	}
	fmt.Println()

	modelDir := filepath.Dir(modelPath)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		return fmt.Errorf("创建模型目录失败: %w", err)
	}

	source := modelURL
	if strings.TrimSpace(modelSource) != "" {
		source = modelSource
	}
	if err := copyOrDownloadFile(modelPath, source); err != nil {
		_ = os.Remove(modelPath)
		return fmt.Errorf("Whisper 模型下载失败: %w\n请手动运行: curl -L -o %s %s", err, modelPath, source)
	}

	fmt.Println()
	fmt.Println("Whisper 模型下载完成！")
	return nil
}
