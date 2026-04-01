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
