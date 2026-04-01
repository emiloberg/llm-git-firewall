package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanForPendingDirs(t *testing.T) {
	root := t.TempDir()

	// Create two repos, one with pending dir, one without
	repo1 := filepath.Join(root, "org", "repo1")
	repo2 := filepath.Join(root, "org", "repo2")
	os.MkdirAll(filepath.Join(repo1, ".git-llm-guard", "pending"), 0755)
	os.MkdirAll(repo2, 0755)

	dirs, err := ScanForPendingDirs(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dirs) != 1 {
		t.Fatalf("expected 1 pending dir, got %d", len(dirs))
	}

	expected := filepath.Join(repo1, ".git-llm-guard", "pending")
	if dirs[0] != expected {
		t.Errorf("got %q, want %q", dirs[0], expected)
	}
}

func TestScanForWatchDirs(t *testing.T) {
	root := t.TempDir()

	// Create directories at depth 1 and 2
	os.MkdirAll(filepath.Join(root, "org1"), 0755)
	os.MkdirAll(filepath.Join(root, "org1", "repo1"), 0755)
	os.MkdirAll(filepath.Join(root, "org1", "repo2"), 0755)
	os.MkdirAll(filepath.Join(root, "org2"), 0755)

	dirs, err := ScanForWatchDirs(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include org1, org2, org1/repo1, org1/repo2
	if len(dirs) < 4 {
		t.Errorf("expected at least 4 dirs, got %d: %v", len(dirs), dirs)
	}
}

func TestWatcherDetectsNewFile(t *testing.T) {
	root := t.TempDir()
	repoDir := filepath.Join(root, "org", "repo1")
	pendingDir := filepath.Join(repoDir, ".git-llm-guard", "pending")
	os.MkdirAll(pendingDir, 0755)

	events := make(chan RequestEvent, 10)
	w, err := New(root, events)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer w.Close()

	go w.Run()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Create a request file
	reqFile := filepath.Join(pendingDir, "2026-04-01T15-30-00.txt")
	os.WriteFile(reqFile, []byte("git push origin feat/test"), 0644)

	// Wait for event
	select {
	case evt := <-events:
		if evt.FilePath != reqFile {
			t.Errorf("got file %q, want %q", evt.FilePath, reqFile)
		}
		if evt.RepoPath != repoDir {
			t.Errorf("got repo %q, want %q", evt.RepoPath, repoDir)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}
