package guard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/emiloberg/llm-git-firewall/internal"
	"github.com/emiloberg/llm-git-firewall/internal/config"
)

type Guard struct {
	GlobalRules config.Rules
}

func (g *Guard) Validate(cmd string, repoPath string) (bool, string) {
	rules := g.GlobalRules

	if repoPath != "" {
		repoCfgPath := filepath.Join(repoPath, internal.DirName, "config.yaml")
		repoCfg, err := config.LoadRepo(repoCfgPath)
		if err == nil {
			rules = config.MergeRules(g.GlobalRules, repoCfg.Rules)
		}
	}

	for _, pattern := range rules.Deny {
		if MatchPattern(pattern, cmd) {
			return false, fmt.Sprintf("matches deny rule %q", pattern)
		}
	}

	for _, pattern := range rules.Allow {
		if MatchPattern(pattern, cmd) {
			return true, ""
		}
	}

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
	resultsDir := filepath.Join(repoPath, internal.DirName, "results")
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

	if err := os.Remove(reqFile); err != nil {
		return fmt.Errorf("removing request file: %w", err)
	}

	return nil
}
