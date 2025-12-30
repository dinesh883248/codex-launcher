package core

import (
	"context"
	"log"
	"os"
	"os/exec"
	"time"

	"almono/api"
)

type Config struct {
	PollInterval time.Duration
	CodexBin     string
	CodexModel   string
	Reasoning    string
	WorkDir      string
}

func StartWorker(ctx context.Context, store *api.Store, cfg Config) {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.CodexBin == "" {
		cfg.CodexBin = "codex"
	}
	if cfg.CodexModel == "" {
		cfg.CodexModel = "gpt-5.2-codex"
	}
	if cfg.Reasoning == "" {
		cfg.Reasoning = "high"
	}

	log.Printf("worker ready; polling every %s", cfg.PollInterval)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		req, ok, err := store.ClaimNextPending(ctx)
		if err != nil {
			log.Printf("worker claim failed: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			continue
		}
		if !ok {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			continue
		}

		log.Printf("processing request %d", req.ID)
		status := "processed"
		err = runCodex(ctx, cfg, req.Prompt)
		if err != nil {
			status = "error"
		}
		if err := store.UpdateRequest(ctx, req.ID, status, responseFor(err)); err != nil {
			log.Printf("worker update failed: %v", err)
		}
	}
}

func runCodex(ctx context.Context, cfg Config, prompt string) error {
	args := []string{
		"exec",
		"-m",
		cfg.CodexModel,
		"--config",
		"model_reasoning_effort=" + cfg.Reasoning,
		"--dangerously-bypass-approvals-and-sandbox",
		"--skip-git-repo-check",
		prompt,
	}
	cmd := exec.CommandContext(ctx, cfg.CodexBin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}
	return cmd.Run()
}

func responseFor(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
