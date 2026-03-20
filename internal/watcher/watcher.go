package watcher

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type Event struct {
	Path string
}

type Watcher struct {
	fsw       *fsnotify.Watcher
	events    chan Event
	done      chan struct{}
	closeOnce sync.Once
	filter    *Filter
}

func New(dir string, excludePatterns []string) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	filter, err := NewFilter(excludePatterns)
	if err != nil {
		fsw.Close()
		return nil, err
	}

	w := &Watcher{
		fsw:    fsw,
		events: make(chan Event, 100),
		done:   make(chan struct{}),
		filter: filter,
	}

	if err := w.addRecursive(dir); err != nil {
		fsw.Close()
		return nil, err
	}

	go w.loop(dir)
	return w, nil
}

func (w *Watcher) Events() <-chan Event {
	return w.events
}

func (w *Watcher) Stop() error {
	w.closeOnce.Do(func() {
		close(w.done)
	})
	return w.fsw.Close()
}

func (w *Watcher) loop(baseDir string) {
	defer close(w.events)
	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			rel, err := filepath.Rel(baseDir, event.Name)
			if err != nil {
				continue
			}
			if w.filter.ShouldExclude(rel) {
				continue
			}
			if isHidden(rel) {
				continue
			}
			// Add new directories to watch
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					w.addRecursive(event.Name)
				}
			}
			select {
			case w.events <- Event{Path: event.Name}:
			default:
			}
		case _, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
		}
	}
}

func (w *Watcher) addRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if isHiddenName(name) || name == "node_modules" {
				return filepath.SkipDir
			}
			return w.fsw.Add(path)
		}
		return nil
	})
}

func isHidden(path string) bool {
	for _, part := range strings.Split(path, string(filepath.Separator)) {
		if isHiddenName(part) {
			return true
		}
	}
	return false
}

func isHiddenName(name string) bool {
	return len(name) > 1 && name[0] == '.'
}
