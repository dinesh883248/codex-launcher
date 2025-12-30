package core

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"almono/api"
)

// codex JSON event types
type codexEvent struct {
	Type   string          `json:"type"`
	Item   json.RawMessage `json:"item,omitempty"`
	Usage  *usageInfo      `json:"usage,omitempty"`
}

type itemInfo struct {
	Type             string `json:"type"`
	Text             string `json:"text,omitempty"`
	Message          string `json:"message,omitempty"`
	Command          string `json:"command,omitempty"`
	AggregatedOutput string `json:"aggregated_output,omitempty"`
	ExitCode         *int   `json:"exit_code,omitempty"`
	Status           string `json:"status,omitempty"`
}

type usageInfo struct {
	InputTokens       int `json:"input_tokens"`
	CachedInputTokens int `json:"cached_input_tokens"`
	OutputTokens      int `json:"output_tokens"`
}

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
		"--json",
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

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	// parse JSON events and store relevant output
	lineNum := 1
	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			// parse JSON event
			var event codexEvent
			if jsonErr := json.Unmarshal([]byte(line), &event); jsonErr != nil {
				continue
			}

			// process relevant events
			content := processEvent(event)
			if content != "" {
				log.Printf("[%d] %s", requestID, content)
				if storeErr := store.AddOutputLine(ctx, requestID, lineNum, content); storeErr != nil {
					log.Printf("failed to store output line: %v", storeErr)
				}
				lineNum++
			}
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

// processEvent extracts display content from codex JSON events
func processEvent(event codexEvent) string {
	switch event.Type {
	case "item.completed":
		var item itemInfo
		if err := json.Unmarshal(event.Item, &item); err != nil {
			return ""
		}
		switch item.Type {
		case "reasoning":
			return fmt.Sprintf("Thinking: %s", truncate(item.Text, 100))
		case "command_execution":
			if item.Status == "completed" {
				result := fmt.Sprintf("$ %s", item.Command)
				if item.AggregatedOutput != "" {
					result += fmt.Sprintf(" -> %s", truncate(item.AggregatedOutput, 100))
				}
				if item.ExitCode != nil && *item.ExitCode != 0 {
					result += fmt.Sprintf(" (exit %d)", *item.ExitCode)
				}
				return result
			}
		case "agent_message":
			// don't truncate final response - it's important
			text := strings.ReplaceAll(item.Text, "\n", " ")
			return fmt.Sprintf("Response: %s", strings.TrimSpace(text))
		case "error":
			return fmt.Sprintf("Error: %s", truncate(item.Message, 100))
		}
	case "turn.completed":
		if event.Usage != nil {
			return fmt.Sprintf("Tokens: %d input, %d output", event.Usage.InputTokens, event.Usage.OutputTokens)
		}
	}
	return ""
}

// truncate limits string length and adds ellipsis
func truncate(s string, maxLen int) string {
	// replace newlines with spaces for single-line display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func responseFor(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
