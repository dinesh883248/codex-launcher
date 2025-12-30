package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"os/signal"
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
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatalf("db open failed: %v", err)
	}
	defer db.Close()

	store := api.NewStore(db)
	if err := store.Init(ctx); err != nil {
		log.Fatalf("db init failed: %v", err)
	}

	cfg := core.Config{
		PollInterval: *poll,
		CodexBin:     *codexBin,
		CodexModel:   *codexModel,
		Reasoning:    *reasoning,
		WorkDir:      *workDir,
	}
	core.StartWorker(ctx, store, cfg)
}
