package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emiloberg/llm-git-firewall/internal/config"
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
	guardDir := filepath.Join(repoDir, ".llm-git-firewall")
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
	pendingDir := filepath.Join(repoDir, ".llm-git-firewall", "pending")
	resultsDir := filepath.Join(repoDir, ".llm-git-firewall", "results")
	os.MkdirAll(pendingDir, 0755)
	os.MkdirAll(resultsDir, 0755)

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
	pendingDir := filepath.Join(repoDir, ".llm-git-firewall", "pending")
	resultsDir := filepath.Join(repoDir, ".llm-git-firewall", "results")
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
