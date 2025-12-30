package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"almono/api"
	"almono/core"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := flag.String("db", "db.sqlite3", "sqlite database path")
	poll := flag.Duration("poll", 2*time.Second, "worker poll interval")
	codexBin := flag.String("codex", "codex", "codex binary")
	codexModel := flag.String("model", "gpt-5.2-codex", "codex model")
	reasoning := flag.String("reasoning", "high", "codex reasoning effort")
	workDir := flag.String("workdir", "", "codex workdir")
	asciinemaBin := flag.String("asciinema", "", "asciinema binary (defaults to .venv/bin/asciinema if present)")
	tmuxBin := flag.String("tmux", "tmux", "tmux binary")
	session := flag.String("session", "almono-worker", "tmux session name")
	cols := flag.Int("cols", 80, "terminal columns for recording")
	rows := flag.Int("rows", 72, "terminal rows for recording")
	child := flag.Bool("child", false, "run worker loop (internal)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *child {
		runWorker(ctx, *dbPath, core.Config{
			PollInterval: *poll,
			CodexBin:     *codexBin,
			CodexModel:   *codexModel,
			Reasoning:    *reasoning,
			WorkDir:      *workDir,
		})
		return
	}

	baseDir := filepath.Dir(*dbPath)
	castDir := filepath.Join(baseDir, "casts")
	if err := os.MkdirAll(castDir, 0o755); err != nil {
		log.Fatalf("cast dir failed: %v", err)
	}
	asciinemaPath := *asciinemaBin
	if asciinemaPath == "" {
		local := filepath.Join(baseDir, ".venv", "bin", "asciinema")
		if _, err := os.Stat(local); err == nil {
			asciinemaPath = local
		} else {
			asciinemaPath = "asciinema"
		}
	}

	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("resolve worker path failed: %v", err)
	}
	childArgs := []string{
		shellQuote(exe),
		"--child",
		"--db",
		shellQuote(*dbPath),
		"--poll",
		shellQuote(poll.String()),
		"--codex",
		shellQuote(*codexBin),
		"--model",
		shellQuote(*codexModel),
		"--reasoning",
		shellQuote(*reasoning),
	}
	if *workDir != "" {
		childArgs = append(childArgs, "--workdir", shellQuote(*workDir))
	}
	childCmd := strings.Join(childArgs, " ")
	castPath := filepath.Join(castDir, api.LiveCastName())
	recordCmd := exec.Command(
		*tmuxBin,
		"new-session",
		"-d",
		"-s",
		*session,
		"-x",
		strconv.Itoa(*cols),
		"-y",
		strconv.Itoa(*rows),
		asciinemaPath,
		"rec",
		"-q",
		"--overwrite",
		"--cols",
		strconv.Itoa(*cols),
		"--rows",
		strconv.Itoa(*rows),
		"-c",
		childCmd,
		castPath,
	)
	recordCmd.Stdout = os.Stdout
	recordCmd.Stderr = os.Stderr

	if tmuxHasSession(*tmuxBin, *session) {
		log.Printf("tmux session already running: %s", *session)
		return
	}
	if err := recordCmd.Run(); err != nil {
		log.Fatalf("tmux launch failed: %v", err)
	}
	_ = exec.Command(
		*tmuxBin,
		"resize-window",
		"-t",
		*session,
		"-x",
		strconv.Itoa(*cols),
		"-y",
		strconv.Itoa(*rows),
	).Run()
	log.Printf("worker tmux session started: %s", *session)
}

func runWorker(ctx context.Context, dbPath string, cfg core.Config) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("db open failed: %v", err)
	}
	defer db.Close()

	store := api.NewStore(db)
	if err := store.Init(ctx); err != nil {
		log.Fatalf("db init failed: %v", err)
	}

	core.StartWorker(ctx, store, cfg)
}

func tmuxHasSession(tmuxBin, name string) bool {
	err := exec.Command(tmuxBin, "has-session", "-t", name).Run()
	return err == nil
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
