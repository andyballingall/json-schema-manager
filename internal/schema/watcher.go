package schema

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchEvent describes a file change event in the registry.
// Either the user changed a schema file, in which case we will run the
// tests for that schema matching the test scope, or the user changed a test
// document, in which case we will rerun the single test with its schema.
type WatchEvent struct {
	Key      Key    // The Key of the schema that will be tested
	TestPath string // If set, only this specific test document changed
}

// Watcher monitors the registry for file changes and triggers validation.
type Watcher struct {
	registry *Registry
	logger   *slog.Logger
	Ready    chan struct{}

	newWatcher func() (*fsnotify.Watcher, error)
}

// NewWatcher creates a new Watcher for the given registry.
func NewWatcher(r *Registry, logger *slog.Logger) *Watcher {
	return &Watcher{
		registry:   r,
		logger:     logger.With("component", "watcher"),
		Ready:      make(chan struct{}),
		newWatcher: fsnotify.NewWatcher,
	}
}

// Watch starts monitoring the registry for changes. It calls the provided callback
// whenever a relevant change is detected. It blocks until the context is cancelled.
func (w *Watcher) Watch(ctx context.Context, callback func(WatchEvent)) error {
	watcher, err := w.newWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := w.addRecursive(watcher, w.registry.RootDirectory()); err != nil {
		return err
	}

	w.logger.Info("Watching for changes", "root", w.registry.RootDirectory())
	if w.Ready != nil {
		close(w.Ready)
	}

	var timer *time.Timer
	const debounceDuration = 100 * time.Millisecond
	var pendingEvent *WatchEvent

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-watcher.Errors:
			w.logger.Error("Watcher error", "error", err)
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if ev := w.handleEvent(watcher, event); ev != nil {
				if timer != nil {
					timer.Stop()
				}
				pendingEvent = ev
				timer = time.AfterFunc(debounceDuration, func() {
					callback(*pendingEvent)
				})
			}
		}
	}
}

// handleEvent processes a single fsnotify event. If it's a new directory, it adds it to the watcher.
// If it's a relevant file change, it returns a pointer to a WatchEvent.
func (w *Watcher) handleEvent(watcher *fsnotify.Watcher, event fsnotify.Event) *WatchEvent {
	if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
		return nil
	}

	if event.Has(fsnotify.Create) {
		info, err := os.Stat(event.Name)
		if err == nil && info.IsDir() {
			if err := w.addRecursive(watcher, event.Name); err != nil {
				w.logger.Error("Failed to watch new directory", "path", event.Name, "error", err)
			}
			return nil
		}
	}

	return w.mapToWatchEvent(event.Name)
}

// addRecursive adds the given path and all its subdirectories to the watcher.
func (w *Watcher) addRecursive(watcher *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(filepath.Base(path), ".") && path != root {
				return filepath.SkipDir
			}
			return watcher.Add(path)
		}
		return nil
	})
}

// mapToWatchEvent maps a file path to a WatchEvent. Returns nil if the file is not relevant.
func (w *Watcher) mapToWatchEvent(path string) *WatchEvent {
	if strings.HasSuffix(path, SchemaSuffix) {
		key, err := w.registry.KeyFromSchemaPath(path)
		if err == nil {
			return &WatchEvent{Key: key}
		}
	}

	if filepath.Ext(path) == ".json" && !strings.HasSuffix(path, SchemaSuffix) {
		return w.mapTestDocToWatchEvent(path)
	}

	return nil
}

func (w *Watcher) mapTestDocToWatchEvent(path string) *WatchEvent {
	dir := filepath.Dir(path)
	parentDirName := filepath.Base(dir)
	if parentDirName != string(TestDocTypePass) && parentDirName != string(TestDocTypeFail) {
		return nil
	}

	homeDir := filepath.Dir(dir)
	entries, err := os.ReadDir(homeDir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), SchemaSuffix) {
			schemaPath := filepath.Join(homeDir, entry.Name())
			key, err := w.registry.KeyFromSchemaPath(schemaPath)
			if err == nil {
				return &WatchEvent{Key: key, TestPath: path}
			}
		}
	}
	return nil
}
