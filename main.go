package main

import (
	"embed"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"junimohut/internal/app"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

// FilesDroppedPayload is emitted when OS files land on a data-file-drop-target zone.
type FilesDroppedPayload struct {
	Files    []string `json:"files"`
	TargetID string   `json:"targetId"`
}

func init() {
	application.RegisterEvent[bool]("mods-changed")
	application.RegisterEvent[string]("nxm-url")
	application.RegisterEvent[string]("nexus-download-ready")
	application.RegisterEvent[string]("config-editor-open-mod")
	application.RegisterEvent[bool]("config-editor-reload")
	application.RegisterEvent[FilesDroppedPayload]("files-dropped")
}

func main() {
	startSMAPI := flag.Bool("start-smapi", false, "Launch SMAPI after startup")
	flag.Parse()

	services := app.NewServices(app.CoreOptions{
		StartSMAPI: *startSMAPI || app.ParseStartSMAPIFlag(),
		CLIArgs:    os.Args[1:],
	})

	var mainWindow *application.WebviewWindow

	wailsApp := application.New(application.Options{
		Name:        "Junimo Hut",
		Description: "Cross-platform mod manager for Stardew Valley",
		Services:    services.Register(),
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
						services.System.EmitNXMURL(arg)
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

	services.Core.SetApplication(wailsApp)

	mainWindow = wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:          "Junimo Hut — Mod manager for Stardew Valley",
		Width:          1430,
		Height:         900,
		EnableFileDrop: true,
		Frameless:      true,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 72,
			Backdrop:                application.MacBackdropTranslucent,
		},
		BackgroundColour: application.NewRGB(22, 23, 28),
		URL:              "/",
	})

	mainWindow.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		files := event.Context().DroppedFiles()
		if len(files) == 0 {
			return
		}
		details := event.Context().DropTargetDetails()
		wailsApp.Event.Emit("files-dropped", FilesDroppedPayload{
			Files:    files,
			TargetID: details.ElementID,
		})
	})

	wailsApp.Event.OnApplicationEvent(events.Common.ApplicationLaunchedWithUrl, func(e *application.ApplicationEvent) {
		url := e.Context().URL()
		if strings.HasPrefix(url, "nxm://") {
			services.System.EmitNXMURL(url)
			if mainWindow != nil {
				mainWindow.Restore()
				mainWindow.Focus()
			}
		}
	})

	go func() {
		time.Sleep(400 * time.Millisecond)
		services.System.ProcessCommandLineArgs(os.Args[1:])
	}()

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
