package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/emiloberg/llm-git-firewall/internal"
	"github.com/fsnotify/fsnotify"
)

type RequestEvent struct {
	FilePath string
	RepoPath string
}

type Watcher struct {
	root   string
	events chan<- RequestEvent
	fsw    *fsnotify.Watcher
}

func New(root string, events chan<- RequestEvent) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		root:   root,
		events: events,
		fsw:    fsw,
	}

	watchDirs, err := ScanForWatchDirs(root)
	if err != nil {
		fsw.Close()
		return nil, err
	}

	pendingDirs := scanForPendingDirs(watchDirs)

	for _, dir := range append(watchDirs, pendingDirs...) {
		fsw.Add(dir)
	}

	return w, nil
}

func (w *Watcher) Run() {
	for {
		select {
		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	if event.Op&fsnotify.Create == 0 {
		return
	}

	path := event.Name

	dir := filepath.Dir(path)
	if filepath.Base(dir) == "pending" && strings.Contains(dir, internal.DirName) {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			return
		}
		repoPath := filepath.Dir(filepath.Dir(dir))
		w.events <- RequestEvent{
			FilePath: path,
			RepoPath: repoPath,
		}
		return
	}

	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return
	}

	rel, err := filepath.Rel(w.root, path)
	if err != nil {
		return
	}
	depth := strings.Count(rel, string(filepath.Separator)) + 1
	if depth <= 2 {
		w.fsw.Add(path)
	}

	pendingDir := filepath.Join(path, internal.DirName, "pending")
	if info, err := os.Stat(pendingDir); err == nil && info.IsDir() {
		w.fsw.Add(pendingDir)
	}

	if filepath.Base(path) == "pending" && strings.Contains(path, internal.DirName) {
		w.fsw.Add(path)
	}
}

func (w *Watcher) Close() error {
	return w.fsw.Close()
}

// ScanForWatchDirs returns all directories up to 2 levels deep under root.
func ScanForWatchDirs(root string) ([]string, error) {
	var dirs []string

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		level1 := filepath.Join(root, e.Name())
		dirs = append(dirs, level1)

		subEntries, err := os.ReadDir(level1)
		if err != nil {
			continue
		}
		for _, se := range subEntries {
			if !se.IsDir() || strings.HasPrefix(se.Name(), ".") {
				continue
			}
			dirs = append(dirs, filepath.Join(level1, se.Name()))
		}
	}

	return dirs, nil
}

func scanForPendingDirs(watchDirs []string) []string {
	var dirs []string
	for _, dir := range watchDirs {
		pending := filepath.Join(dir, internal.DirName, "pending")
		if info, err := os.Stat(pending); err == nil && info.IsDir() {
			dirs = append(dirs, pending)
		}
	}
	return dirs
}

// ScanForPendingDirs finds all existing pending directories under root.
func ScanForPendingDirs(root string) ([]string, error) {
	watchDirs, err := ScanForWatchDirs(root)
	if err != nil {
		return nil, err
	}
	return scanForPendingDirs(watchDirs), nil
}
