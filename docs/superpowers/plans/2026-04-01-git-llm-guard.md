# git-llm-guard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go daemon that watches for git command requests from a guest VM, validates them against allow/deny rules, executes approved commands, and writes results back.

**Architecture:** Single binary with three packages: `config` (YAML parsing + rule merging), `guard` (glob matching, validation, execution), `watcher` (fsnotify-based directory monitoring). Entry point in `cmd/git-llm-guard/main.go`.

**Tech Stack:** Go, fsnotify, gopkg.in/yaml.v3, standard library for everything else.

---

## File Structure

```
git-llm-guard/
├── cmd/git-llm-guard/main.go    — CLI entry point, flag parsing, wiring
├── internal/config/config.go    — Config structs, YAML loading, rule merging
├── internal/config/config_test.go
├── internal/guard/match.go      — Glob pattern matching for rules
├── internal/guard/match_test.go
├── internal/guard/guard.go      — Validate + Execute + result file writing
├── internal/guard/guard_test.go
├── internal/watcher/watcher.go  — fsnotify setup, directory scanning, event dispatch
├── internal/watcher/watcher_test.go
├── go.mod
└── go.sum
```

---

### Task 1: Project scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/git-llm-guard/main.go`

- [ ] **Step 1: Initialize Go module**

Run: `go mod init github.com/git-llm-guard/git-llm-guard`

Expected: `go.mod` created.

- [ ] **Step 2: Create minimal main.go**

Create `cmd/git-llm-guard/main.go`:

```go
package main

import "fmt"

func main() {
	fmt.Println("git-llm-guard starting...")
}
```

- [ ] **Step 3: Verify it compiles**

Run: `go run ./cmd/git-llm-guard`

Expected: prints `git-llm-guard starting...`

- [ ] **Step 4: Commit**

```bash
git add go.mod cmd/
git commit -m "feat: scaffold Go project with minimal main"
```

---

### Task 2: Config loading

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests for config loading**

Create `internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGlobalConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := []byte(`root: /tmp/shared
rules:
  allow:
    - "git push origin *"
  deny:
    - "git push origin main"
    - "git push origin master"
`)
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Root != "/tmp/shared" {
		t.Errorf("root = %q, want /tmp/shared", cfg.Root)
	}
	if len(cfg.Rules.Allow) != 1 {
		t.Errorf("allow rules = %d, want 1", len(cfg.Rules.Allow))
	}
	if len(cfg.Rules.Deny) != 2 {
		t.Errorf("deny rules = %d, want 2", len(cfg.Rules.Deny))
	}
}

func TestLoadRepoConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := []byte(`rules:
  allow:
    - "git push origin feat/*"
  deny:
    - "git push origin release/*"
`)
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadRepo(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Rules.Allow) != 1 {
		t.Errorf("allow rules = %d, want 1", len(cfg.Rules.Allow))
	}
	if len(cfg.Rules.Deny) != 1 {
		t.Errorf("deny rules = %d, want 1", len(cfg.Rules.Deny))
	}
}

func TestLoadRepoConfigMissing(t *testing.T) {
	cfg, err := LoadRepo("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("missing repo config should not error: %v", err)
	}
	if len(cfg.Rules.Allow) != 0 || len(cfg.Rules.Deny) != 0 {
		t.Error("missing repo config should return empty rules")
	}
}

func TestMergeRules(t *testing.T) {
	global := Rules{
		Allow: []string{"git push origin *"},
		Deny:  []string{"git push origin main"},
	}
	repo := Rules{
		Allow: []string{"git push origin feat/*"},
		Deny:  []string{"git push origin release/*"},
	}

	merged := MergeRules(global, repo)

	if len(merged.Allow) != 2 {
		t.Errorf("merged allow = %d, want 2", len(merged.Allow))
	}
	if len(merged.Deny) != 2 {
		t.Errorf("merged deny = %d, want 2", len(merged.Deny))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -v`

Expected: compilation errors (types and functions don't exist yet).

- [ ] **Step 3: Implement config package**

Create `internal/config/config.go`:

```go
package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Rules struct {
	Allow []string `yaml:"allow"`
	Deny  []string `yaml:"deny"`
}

type Config struct {
	Root  string `yaml:"root"`
	Rules Rules  `yaml:"rules"`
}

type RepoConfig struct {
	Rules Rules `yaml:"rules"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Root == "" {
		return nil, errors.New("config: root is required")
	}

	return &cfg, nil
}

func LoadRepo(path string) (*RepoConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &RepoConfig{}, nil
		}
		return nil, err
	}

	var cfg RepoConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func MergeRules(global, repo Rules) Rules {
	return Rules{
		Allow: append(append([]string{}, global.Allow...), repo.Allow...),
		Deny:  append(append([]string{}, global.Deny...), repo.Deny...),
	}
}
```

- [ ] **Step 4: Add yaml dependency**

Run: `go get gopkg.in/yaml.v3`

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`

Expected: all 4 tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat: add config loading with global/repo merge"
```

---

### Task 3: Glob matching

**Files:**
- Create: `internal/guard/match.go`
- Create: `internal/guard/match_test.go`

- [ ] **Step 1: Write failing tests for glob matching**

Create `internal/guard/match_test.go`:

```go
package guard

import "testing"

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		// Exact match
		{"git push origin main", "git push origin main", true},
		{"git push origin main", "git push origin develop", false},

		// Wildcard at end — matches simple branch
		{"git push origin *", "git push origin feat/foo", true},

		// Wildcard at end — matches nested branch
		{"git push origin feat/*", "git push origin feat/sub/branch", true},

		// Wildcard at end — no match
		{"git push origin feat/*", "git push origin fix/bar", false},

		// Wildcard in middle
		{"git push * --force", "git push origin --force", true},
		{"git push * --force", "git push origin main", false},

		// Multiple wildcards
		{"git push * *", "git push origin feat/foo", true},

		// Empty input
		{"git push origin *", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_vs_"+tt.input, func(t *testing.T) {
			got := MatchPattern(tt.pattern, tt.input)
			if got != tt.want {
				t.Errorf("MatchPattern(%q, %q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/guard/ -v -run TestMatchPattern`

Expected: compilation error (MatchPattern not defined).

- [ ] **Step 3: Implement glob matching**

Create `internal/guard/match.go`:

```go
package guard

import "strings"

// MatchPattern checks if input matches a glob pattern where * matches any
// string including /.
func MatchPattern(pattern, input string) bool {
	// Split pattern by * to get literal segments
	parts := strings.Split(pattern, "*")

	if len(parts) == 1 {
		// No wildcard — exact match
		return pattern == input
	}

	// Check prefix (before first *)
	if !strings.HasPrefix(input, parts[0]) {
		return false
	}

	// Check suffix (after last *)
	if !strings.HasSuffix(input, parts[len(parts)-1]) {
		return false
	}

	// Walk through middle segments in order
	remaining := input[len(parts[0]):]
	for _, part := range parts[1:] {
		idx := strings.Index(remaining, part)
		if idx < 0 {
			return false
		}
		remaining = remaining[idx+len(part):]
	}

	return true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/guard/ -v -run TestMatchPattern`

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/guard/match.go internal/guard/match_test.go
git commit -m "feat: add glob pattern matching for allow/deny rules"
```

---

### Task 4: Guard — validate and execute

**Files:**
- Create: `internal/guard/guard.go`
- Create: `internal/guard/guard_test.go`

- [ ] **Step 1: Write failing tests for Validate**

Append to `internal/guard/match_test.go` (or create `internal/guard/guard_test.go`):

Create `internal/guard/guard_test.go`:

```go
package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/git-llm-guard/git-llm-guard/internal/config"
)

func TestValidateDenyFirst(t *testing.T) {
	g := &Guard{
		GlobalRules: config.Rules{
			Allow: []string{"git push origin *"},
			Deny:  []string{"git push origin main"},
		},
	}

	ok, reason := g.Validate("git push origin main", "")
	if ok {
		t.Error("expected deny for push to main")
	}
	if !strings.Contains(reason, "git push origin main") {
		t.Errorf("reason should mention the deny rule, got: %s", reason)
	}
}

func TestValidateAllowed(t *testing.T) {
	g := &Guard{
		GlobalRules: config.Rules{
			Allow: []string{"git push origin *"},
			Deny:  []string{"git push origin main"},
		},
	}

	ok, _ := g.Validate("git push origin feat/foo", "")
	if !ok {
		t.Error("expected allow for push to feat/foo")
	}
}

func TestValidateDefaultDeny(t *testing.T) {
	g := &Guard{
		GlobalRules: config.Rules{
			Allow: []string{"git push origin feat/*"},
			Deny:  []string{},
		},
	}

	ok, _ := g.Validate("git push origin fix/bar", "")
	if ok {
		t.Error("expected deny for unmatched command")
	}
}

func TestValidateWithRepoOverride(t *testing.T) {
	repoDir := t.TempDir()
	guardDir := filepath.Join(repoDir, ".git-llm-guard")
	os.MkdirAll(guardDir, 0755)

	cfgContent := []byte(`rules:
  deny:
    - "git push origin release/*"
`)
	os.WriteFile(filepath.Join(guardDir, "config.yaml"), cfgContent, 0644)

	g := &Guard{
		GlobalRules: config.Rules{
			Allow: []string{"git push origin *"},
			Deny:  []string{"git push origin main"},
		},
	}

	ok, _ := g.Validate("git push origin release/v1", repoDir)
	if ok {
		t.Error("expected deny from repo override for release branch")
	}
}

func TestExecuteSuccess(t *testing.T) {
	repoDir := t.TempDir()

	g := &Guard{}
	output, err := g.Execute("git init", repoDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if output == "" {
		t.Error("expected output from git init")
	}
}

func TestExecuteFail(t *testing.T) {
	repoDir := t.TempDir()

	g := &Guard{}
	_, err := g.Execute("git push origin main", repoDir)
	if err == nil {
		t.Error("expected error from push in empty dir")
	}
}

func TestProcessRequest(t *testing.T) {
	repoDir := t.TempDir()
	pendingDir := filepath.Join(repoDir, ".git-llm-guard", "pending")
	resultsDir := filepath.Join(repoDir, ".git-llm-guard", "results")
	os.MkdirAll(pendingDir, 0755)
	os.MkdirAll(resultsDir, 0755)

	// Initialize a git repo so git status works
	g := &Guard{
		GlobalRules: config.Rules{
			Allow: []string{"git status"},
			Deny:  []string{},
		},
	}

	// Run git init in repoDir
	g.Execute("git init", repoDir)

	reqFile := filepath.Join(pendingDir, "2026-04-01T15-30-00.txt")
	os.WriteFile(reqFile, []byte("git status"), 0644)

	err := g.ProcessRequest(reqFile, repoDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Request file should no longer exist in pending
	if _, err := os.Stat(reqFile); !os.IsNotExist(err) {
		t.Error("request file should have been moved from pending")
	}

	// Result file should exist
	resultFile := filepath.Join(resultsDir, "2026-04-01T15-30-00.txt")
	data, err := os.ReadFile(resultFile)
	if err != nil {
		t.Fatalf("result file not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "git status") {
		t.Error("result should contain original command")
	}
	if !strings.Contains(content, "---") {
		t.Error("result should contain separator")
	}
	if !strings.Contains(content, "status: success") {
		t.Errorf("result should contain success status, got: %s", content)
	}
}

func TestProcessRequestDenied(t *testing.T) {
	repoDir := t.TempDir()
	pendingDir := filepath.Join(repoDir, ".git-llm-guard", "pending")
	resultsDir := filepath.Join(repoDir, ".git-llm-guard", "results")
	os.MkdirAll(pendingDir, 0755)
	os.MkdirAll(resultsDir, 0755)

	g := &Guard{
		GlobalRules: config.Rules{
			Allow: []string{"git push origin feat/*"},
			Deny:  []string{"git push origin main"},
		},
	}

	reqFile := filepath.Join(pendingDir, "2026-04-01T15-31-00.txt")
	os.WriteFile(reqFile, []byte("git push origin main"), 0644)

	err := g.ProcessRequest(reqFile, repoDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultFile := filepath.Join(resultsDir, "2026-04-01T15-31-00.txt")
	data, err := os.ReadFile(resultFile)
	if err != nil {
		t.Fatalf("result file not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "status: denied") {
		t.Errorf("result should contain denied status, got: %s", content)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/guard/ -v`

Expected: compilation errors (Guard struct and methods not defined).

- [ ] **Step 3: Implement Guard**

Create `internal/guard/guard.go`:

```go
package guard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/git-llm-guard/git-llm-guard/internal/config"
)

type Guard struct {
	GlobalRules config.Rules
}

func (g *Guard) Validate(cmd string, repoPath string) (bool, string) {
	rules := g.GlobalRules

	// Load and merge repo-specific rules if repoPath is provided
	if repoPath != "" {
		repoCfgPath := filepath.Join(repoPath, ".git-llm-guard", "config.yaml")
		repoCfg, err := config.LoadRepo(repoCfgPath)
		if err == nil {
			rules = config.MergeRules(g.GlobalRules, repoCfg.Rules)
		}
	}

	// Check deny rules first
	for _, pattern := range rules.Deny {
		if MatchPattern(pattern, cmd) {
			return false, fmt.Sprintf("matches deny rule %q", pattern)
		}
	}

	// Check allow rules
	for _, pattern := range rules.Allow {
		if MatchPattern(pattern, cmd) {
			return true, ""
		}
	}

	// Default deny
	return false, "no allow rule matched"
}

func (g *Guard) Execute(cmd string, repoPath string) (string, error) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	c := exec.Command(parts[0], parts[1:]...)
	c.Dir = repoPath

	output, err := c.CombinedOutput()
	return string(output), err
}

func (g *Guard) ProcessRequest(reqFile string, repoPath string) error {
	data, err := os.ReadFile(reqFile)
	if err != nil {
		return fmt.Errorf("reading request file: %w", err)
	}

	cmd := strings.TrimSpace(string(data))
	if cmd == "" {
		return g.writeResult(reqFile, repoPath, cmd, "fail", "empty request file")
	}

	ok, reason := g.Validate(cmd, repoPath)
	if !ok {
		return g.writeResult(reqFile, repoPath, cmd, "denied", reason)
	}

	output, err := g.Execute(cmd, repoPath)
	if err != nil {
		return g.writeResult(reqFile, repoPath, cmd, "fail", output)
	}

	return g.writeResult(reqFile, repoPath, cmd, "success", output)
}

func (g *Guard) writeResult(reqFile string, repoPath string, cmd string, status string, detail string) error {
	resultsDir := filepath.Join(repoPath, ".git-llm-guard", "results")
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return fmt.Errorf("creating results dir: %w", err)
	}

	filename := filepath.Base(reqFile)
	resultFile := filepath.Join(resultsDir, filename)

	var content string
	switch status {
	case "denied":
		content = fmt.Sprintf("%s\n---\nstatus: denied\nreason: %s\n", cmd, detail)
	default:
		content = fmt.Sprintf("%s\n---\nstatus: %s\noutput: |\n  %s\n", cmd, status, strings.ReplaceAll(strings.TrimRight(detail, "\n"), "\n", "\n  "))
	}

	if err := os.WriteFile(resultFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing result file: %w", err)
	}

	// Remove original request file from pending
	if err := os.Remove(reqFile); err != nil {
		return fmt.Errorf("removing request file: %w", err)
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/guard/ -v`

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/guard/guard.go internal/guard/guard_test.go
git commit -m "feat: add guard with validate, execute, and result writing"
```

---

### Task 5: Watcher

**Files:**
- Create: `internal/watcher/watcher.go`
- Create: `internal/watcher/watcher_test.go`

- [ ] **Step 1: Write failing test for watcher**

Create `internal/watcher/watcher_test.go`:

```go
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

	// Should include root, org1, org2, org1/repo1, org1/repo2
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/watcher/ -v`

Expected: compilation errors.

- [ ] **Step 3: Add fsnotify dependency**

Run: `go get github.com/fsnotify/fsnotify`

- [ ] **Step 4: Implement watcher**

Create `internal/watcher/watcher.go`:

```go
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
	root    string
	events  chan<- RequestEvent
	fsw     *fsnotify.Watcher
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
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/watcher/ -v`

Expected: all 3 tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/watcher/ go.mod go.sum
git commit -m "feat: add filesystem watcher with fsnotify"
```

---

### Task 6: Wire everything together in main.go

**Files:**
- Modify: `cmd/git-llm-guard/main.go`

- [ ] **Step 1: Implement main.go**

Replace `cmd/git-llm-guard/main.go` with:

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/git-llm-guard/git-llm-guard/internal/config"
	"github.com/git-llm-guard/git-llm-guard/internal/guard"
	"github.com/git-llm-guard/git-llm-guard/internal/watcher"
)

func main() {
	homeDir, _ := os.UserHomeDir()
	defaultConfig := homeDir + "/.git-llm-guard.yaml"

	configPath := flag.String("config", defaultConfig, "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	log.Printf("watching root: %s", cfg.Root)

	g := &guard.Guard{
		GlobalRules: cfg.Rules,
	}

	events := make(chan watcher.RequestEvent, 100)

	w, err := watcher.New(cfg.Root, events)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating watcher: %v\n", err)
		os.Exit(1)
	}
	defer w.Close()

	// Handle shutdown gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go w.Run()

	log.Println("git-llm-guard is running. Press Ctrl+C to stop.")

	for {
		select {
		case evt := <-events:
			log.Printf("processing request: %s", evt.FilePath)
			if err := g.ProcessRequest(evt.FilePath, evt.RepoPath); err != nil {
				log.Printf("error processing request: %v", err)
			}
		case <-sigCh:
			log.Println("shutting down...")
			return
		}
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./cmd/git-llm-guard`

Expected: binary created with no errors.

- [ ] **Step 3: Commit**

```bash
git add cmd/git-llm-guard/main.go
git commit -m "feat: wire config, guard, and watcher together in main"
```

---

### Task 7: Integration test

**Files:**
- Create: `integration_test.go`

- [ ] **Step 1: Write integration test**

Create `integration_test.go` in the project root:

```go
//go:build integration

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-llm-guard/git-llm-guard/internal/config"
	"github.com/git-llm-guard/git-llm-guard/internal/guard"
	"github.com/git-llm-guard/git-llm-guard/internal/watcher"
)

func TestIntegrationFullFlow(t *testing.T) {
	root := t.TempDir()
	repoDir := filepath.Join(root, "org", "project")
	pendingDir := filepath.Join(repoDir, ".git-llm-guard", "pending")
	resultsDir := filepath.Join(repoDir, ".git-llm-guard", "results")
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
```

- [ ] **Step 2: Run integration test**

Run: `go test -tags integration -v -run TestIntegrationFullFlow .`

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add integration_test.go
git commit -m "test: add integration test for full request flow"
```

---

### Task 8: Final verification

- [ ] **Step 1: Run all unit tests**

Run: `go test ./... -v`

Expected: all tests PASS.

- [ ] **Step 2: Run integration tests**

Run: `go test -tags integration ./... -v`

Expected: all tests PASS.

- [ ] **Step 3: Build the binary**

Run: `go build -o git-llm-guard ./cmd/git-llm-guard`

Expected: binary created successfully.

- [ ] **Step 4: Final commit**

```bash
git add .
git commit -m "chore: final build verification"
```
