package main

import (
	"embed"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	application.RegisterEvent[bool]("mods-changed")
	application.RegisterEvent[string]("nxm-url")
	application.RegisterEvent[string]("nexus-download-ready")
}

func main() {
	startSMAPI := flag.Bool("start-smapi", false, "Launch SMAPI after startup")
	_ = startSMAPI
	flag.Parse()

	appService := NewApp()

	var mainWindow *application.WebviewWindow

	app := application.New(application.Options{
		Name:        "Junimo Hut",
		Description: "Cross-platform mod manager for Stardew Valley",
		Services: []application.Service{
			application.NewService(appService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.junimohut.app",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				for _, arg := range data.Args {
					if strings.HasPrefix(arg, "nxm://") {
						appService.EmitNXMURL(arg)
						if mainWindow != nil {
							mainWindow.Restore()
							mainWindow.Focus()
						}
						return
					}
				}
			},
		},
	})

	appService.SetApplication(app)

	mainWindow = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Junimo Hut — Mod manager for Stardew Valley",
		Width:  1430,
		Height: 900,
		Frameless: true,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 72,
			Backdrop:                application.MacBackdropTranslucent,
		},
		BackgroundColour: application.NewRGB(22, 23, 28),
		URL:              "/",
	})

	app.Event.OnApplicationEvent(events.Common.ApplicationLaunchedWithUrl, func(e *application.ApplicationEvent) {
		url := e.Context().URL()
		if strings.HasPrefix(url, "nxm://") {
			appService.EmitNXMURL(url)
			if mainWindow != nil {
				mainWindow.Restore()
				mainWindow.Focus()
			}
		}
	})

	go func() {
		time.Sleep(400 * time.Millisecond)
		appService.ProcessCommandLineArgs(os.Args[1:])
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
