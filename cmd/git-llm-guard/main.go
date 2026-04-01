package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"path/filepath"

	"github.com/git-llm-guard/git-llm-guard/internal/config"
	"github.com/git-llm-guard/git-llm-guard/internal/guard"
	"github.com/git-llm-guard/git-llm-guard/internal/watcher"
)

func main() {
	homeDir, _ := os.UserHomeDir()
	defaultConfig := homeDir + "/.git-llm-guard.yaml"

	configPath := flag.String("config", defaultConfig, "path to config file")
	initFlag := flag.Bool("init", false, "create default config file at ~/.git-llm-guard.yaml")
	flag.Parse()

	if *initFlag {
		if err := createDefaultConfig(defaultConfig); err != nil {
			fmt.Fprintf(os.Stderr, "error creating config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Config created at %s\nEdit the 'root' field to point to your shared directory.\n", defaultConfig)
		return
	}

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

const defaultConfigTemplate = `root: %s

rules:
  allow:
    - "git pull *"
    - "git fetch *"
    - "git push origin *"
  deny:
    - "git push origin main"
    - "git push origin master"
    - "git push origin develop"
    - "*--force*"
    - "* -f"
    - "* -f *"
`

func createDefaultConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config already exists at %s", path)
	}

	homeDir, _ := os.UserHomeDir()
	defaultRoot := filepath.Join(homeDir, "code", "shared")

	content := fmt.Sprintf(defaultConfigTemplate, defaultRoot)
	return os.WriteFile(path, []byte(content), 0644)
}
