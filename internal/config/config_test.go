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
