package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	app "github.com/mymeetily/mymeetily/cmd"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	args := os.Args[1:]

	// CLI subcommands (init-engine, init-model) run without GUI
	if len(args) > 0 {
		switch args[0] {
		case "init-engine":
			if err := app.Run(args); err != nil {
				fmt.Fprintf(os.Stderr, "init-engine 失败: %v\n", err)
				os.Exit(1)
			}
			return
		case "init-model":
			if err := app.Run(args); err != nil {
				fmt.Fprintf(os.Stderr, "init-model 失败: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	// Default: launch Wails desktop GUI
	myApp := NewApp()

	err := wails.Run(&options.App{
		Title:  "MyMeetily - 本地离线 AI 会议助手",
		Width:  1280,
		Height: 860,
		MinWidth:  960,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 22, B: 28, A: 1},
		OnStartup:        myApp.startup,
		OnShutdown:       myApp.shutdown,
		Bind: []interface{}{
			myApp,
			myApp.appService,
			myApp.deviceService,
			myApp.recordService,
			myApp.pipelineService,
			myApp.resultService,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
