package app

import (
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// EventPublisher emits Wails frontend events.
type EventPublisher interface {
	EmitModsChanged()
	EmitNXMURL(url string)
	EmitDownloadReady(path string)
}

// EventBridge publishes events via the Wails application instance.
type EventBridge struct {
	app *application.App
}

// NewEventBridge creates an event bridge without an app reference.
func NewEventBridge() *EventBridge {
	return &EventBridge{}
}

// SetApp attaches the Wails application for event emission.
func (b *EventBridge) SetApp(app *application.App) {
	b.app = app
}

// EmitModsChanged notifies the frontend that the mod list changed.
func (b *EventBridge) EmitModsChanged() {
	if b.app != nil {
		b.app.Event.Emit("mods-changed", true)
	}
}

// EmitNXMURL notifies the frontend to handle an nxm:// link.
func (b *EventBridge) EmitNXMURL(url string) {
	if b.app == nil || !strings.HasPrefix(url, "nxm://") {
		return
	}
	b.app.Event.Emit("nxm-url", url)
}

// EmitDownloadReady notifies the frontend that a Nexus download completed.
func (b *EventBridge) EmitDownloadReady(path string) {
	if b.app == nil || path == "" {
		return
	}
	b.app.Event.Emit("nexus-download-ready", path)
}
