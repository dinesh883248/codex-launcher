package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"almono/api"
	"almono/core"
	"almono/web"

	_ "modernc.org/sqlite"
)

func main() {
	addr := flag.String("addr", ":55136", "listen address")
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

	// start worker in background
	cfg := core.Config{
		PollInterval: *poll,
		CodexBin:     *codexBin,
		CodexModel:   *codexModel,
		Reasoning:    *reasoning,
		WorkDir:      *workDir,
	}
	go core.StartWorker(ctx, store, cfg)

	// start web server
	svc := api.NewService(store)
	webServer, err := web.NewServer(svc)
	if err != nil {
		log.Fatalf("template init failed: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/requests", api.NewRequestHandler(svc))
	mux.HandleFunc("/requests/new", webServer.HandleCreate)
	mux.HandleFunc("/requests/", webServer.HandleRequests)
	mux.HandleFunc("/requests", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/requests/", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/", webServer.HandleRequests)

	srv := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("Codex Launcher running at http://127.0.0.1%s", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
