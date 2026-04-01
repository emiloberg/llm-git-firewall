//go:build integration

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/emiloberg/llm-git-firewall/internal/config"
	"github.com/emiloberg/llm-git-firewall/internal/guard"
	"github.com/emiloberg/llm-git-firewall/internal/watcher"
)

func TestIntegrationFullFlow(t *testing.T) {
	root := t.TempDir()
	repoDir := filepath.Join(root, "org", "project")
	pendingDir := filepath.Join(repoDir, ".llm-git-firewall", "pending")
	resultsDir := filepath.Join(repoDir, ".llm-git-firewall", "results")
	os.MkdirAll(pendingDir, 0755)
	os.MkdirAll(resultsDir, 0755)

	// Initialize a real git repo
	g := &guard.Guard{
		GlobalRules: config.Rules{
			Allow: []string{"git status"},
			Deny:  []string{"git push origin main"},
		},
	}

	g.Execute("git init", repoDir)
	g.Execute("git config user.email test@test.com", repoDir)
	g.Execute("git config user.name Test", repoDir)

	events := make(chan watcher.RequestEvent, 10)
	w, err := watcher.New(root, events)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer w.Close()

	go w.Run()
	time.Sleep(100 * time.Millisecond)

	// Test 1: Allowed command
	reqFile := filepath.Join(pendingDir, "2026-04-01T15-30-00.txt")
	os.WriteFile(reqFile, []byte("git status"), 0644)

	select {
	case evt := <-events:
		if err := g.ProcessRequest(evt.FilePath, evt.RepoPath); err != nil {
			t.Fatalf("process request failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	resultFile := filepath.Join(resultsDir, "2026-04-01T15-30-00.txt")
	data, err := os.ReadFile(resultFile)
	if err != nil {
		t.Fatalf("result file not found: %v", err)
	}
	if !strings.Contains(string(data), "status: success") {
		t.Errorf("expected success, got: %s", string(data))
	}

	// Test 2: Denied command
	reqFile2 := filepath.Join(pendingDir, "2026-04-01T15-31-00.txt")
	os.WriteFile(reqFile2, []byte("git push origin main"), 0644)

	select {
	case evt := <-events:
		if err := g.ProcessRequest(evt.FilePath, evt.RepoPath); err != nil {
			t.Fatalf("process request failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	resultFile2 := filepath.Join(resultsDir, "2026-04-01T15-31-00.txt")
	data2, err := os.ReadFile(resultFile2)
	if err != nil {
		t.Fatalf("result file not found: %v", err)
	}
	if !strings.Contains(string(data2), "status: denied") {
		t.Errorf("expected denied, got: %s", string(data2))
	}
}
