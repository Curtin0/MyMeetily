package main

import (
	"fmt"
	"os"

	app "github.com/mymeetily/mymeetily/cmd/mymeetily"
)

func main() {
	args := os.Args[1:]
	if err := app.Run(args); err != nil {
		if len(args) > 0 && args[0] == "init-engine" {
			fmt.Fprintf(os.Stderr, "init-engine 失败: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "运行错误: %v\n", err)
		}
		os.Exit(1)
	}
}
