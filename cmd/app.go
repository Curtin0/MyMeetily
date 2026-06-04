package cmd

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mymeetily/mymeetily/internal/config"
	"github.com/mymeetily/mymeetily/internal/tui"
	"github.com/mymeetily/mymeetily/internal/workflow"
)

func Run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "init-engine":
			return runInitEngineCommand(args[1:])
		case "init-model":
			return runInitModelCommand(args[1:])
		}
	}

	return runInteractive()
}

func runInteractive() error {
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		fmt.Fprintln(os.Stderr, "将使用默认配置继续。")
		cfg, _ = config.Load("")
	}

	fmt.Println("========================================")
	fmt.Println("  MyMeetily - 环境检查")
	fmt.Println("========================================")
	fmt.Println()

	fmt.Println("--- 语音识别模型 (whisper.cpp) ---")
	if err := ensureWhisperModel(cfg.ASR.ModelPath, defaultWhisperModelURL, "", false); err != nil {
		return fmt.Errorf("语音模型准备失败: %w", err)
	}
	fmt.Println()

	fmt.Println("--- Ollama 服务与总结模型 ---")
	startupStatus, err := workflow.DetectSummaryAvailability(context.Background(), cfg, defaultSummaryModelPath, defaultSummaryModelURL)
	if err != nil {
		return fmt.Errorf("Ollama 模型准备失败: %w", err)
	}
	fmt.Println(startupStatus.SummaryMessage)
	fmt.Println()

	fmt.Println("========================================")
	if startupStatus.SummaryEnabled {
		fmt.Println("  语音转写与自动总结已就绪，正在启动...")
	} else {
		fmt.Println("  语音转写已就绪，当前将跳过自动总结并直接输出报告...")
	}
	fmt.Println("========================================")
	fmt.Println()

	program := tea.NewProgram(
		tui.New(cfg, startupStatus.SummaryEnabled),
		tea.WithAltScreen(),
	)

	if _, err := program.Run(); err != nil {
		return err
	}

	return nil
}
