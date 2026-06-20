package mods

import (
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher notifies when the mods folder changes.
type Watcher struct {
	mu       sync.Mutex
	watcher  *fsnotify.Watcher
	modsRoot string
	onChange func()
}

func NewWatcher(modsRoot string, onChange func()) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	watcher := &Watcher{watcher: w, modsRoot: modsRoot, onChange: onChange}
	if modsRoot != "" {
		_ = w.Add(modsRoot)
	}
	go watcher.loop()
	return watcher, nil
}

func (w *Watcher) loop() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				if w.onChange != nil {
					w.onChange()
				}
			}
		case _, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

func (w *Watcher) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}

func (w *Watcher) SetRoot(modsRoot string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.modsRoot = modsRoot
	_ = w.watcher.Close()
	nw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.watcher = nw
	if modsRoot != "" {
		_ = w.watcher.Add(modsRoot)
	}
	return nil
}
