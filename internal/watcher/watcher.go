package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"

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

	// Watch all dirs 1-2 levels deep
	watchDirs, err := ScanForWatchDirs(root)
	if err != nil {
		fsw.Close()
		return nil, err
	}
	for _, dir := range watchDirs {
		fsw.Add(dir)
	}

	// Watch existing pending dirs
	pendingDirs, err := ScanForPendingDirs(root)
	if err != nil {
		fsw.Close()
		return nil, err
	}
	for _, dir := range pendingDirs {
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

	// Check if this is a new file in a pending directory
	dir := filepath.Dir(path)
	if filepath.Base(dir) == "pending" && strings.Contains(dir, ".git-llm-guard") {
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

	// Check if a new directory was created — might need watching
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return
	}

	// If it's within 2 levels of root, watch it
	rel, err := filepath.Rel(w.root, path)
	if err != nil {
		return
	}
	depth := len(strings.Split(rel, string(filepath.Separator)))
	if depth <= 2 {
		w.fsw.Add(path)
	}

	// If this creates a .git-llm-guard/pending dir, watch it
	pendingDir := filepath.Join(path, ".git-llm-guard", "pending")
	if info, err := os.Stat(pendingDir); err == nil && info.IsDir() {
		w.fsw.Add(pendingDir)
	}

	// If this IS a pending dir inside .git-llm-guard, watch it
	if filepath.Base(path) == "pending" && strings.Contains(path, ".git-llm-guard") {
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

// ScanForPendingDirs finds all existing .git-llm-guard/pending directories.
func ScanForPendingDirs(root string) ([]string, error) {
	var dirs []string

	watchDirs, err := ScanForWatchDirs(root)
	if err != nil {
		return nil, err
	}

	for _, dir := range watchDirs {
		pending := filepath.Join(dir, ".git-llm-guard", "pending")
		if info, err := os.Stat(pending); err == nil && info.IsDir() {
			dirs = append(dirs, pending)
		}
	}

	return dirs, nil
}
