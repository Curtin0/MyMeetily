package cmd

import (
	"fmt"
)

// Run dispatches CLI subcommands (init-engine, init-model).
// The default GUI mode is launched directly from main.go via Wails.
func Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("请使用 Wails GUI 模式启动（直接运行 mymeetily.exe）或指定子命令: init-engine | init-model")
	}

	switch args[0] {
	case "init-engine":
		return runInitEngineCommand(args[1:])
	case "init-model":
		return runInitModelCommand(args[1:])
	default:
		return fmt.Errorf("未知子命令: %s (可用: init-engine, init-model)", args[0])
	}
}
