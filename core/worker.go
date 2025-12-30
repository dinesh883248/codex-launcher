package core

import (
	"bufio"
	"context"
	"io"
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
		err = runCodex(ctx, store, cfg, req.ID, req.Prompt)
		if err != nil {
			status = "error"
		}
		if err := store.UpdateRequest(ctx, req.ID, status, responseFor(err)); err != nil {
			log.Printf("worker update failed: %v", err)
		}
	}
}

func runCodex(ctx context.Context, store *api.Store, cfg Config, requestID int64, prompt string) error {
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
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}

	// capture stdout and stderr combined
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout // combine stderr with stdout

	if err := cmd.Start(); err != nil {
		return err
	}

	// read output line by line and store in DB
	lineNum := 1
	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			// also print to stdout for visibility
			os.Stdout.WriteString(line)
			// store in DB (strip trailing newline)
			content := line
			if len(content) > 0 && content[len(content)-1] == '\n' {
				content = content[:len(content)-1]
			}
			if storeErr := store.AddOutputLine(ctx, requestID, lineNum, content); storeErr != nil {
				log.Printf("failed to store output line: %v", storeErr)
			}
			lineNum++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}

	return cmd.Wait()
}

func responseFor(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
