package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"almono/api"
	"almono/web"

	_ "modernc.org/sqlite"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "db.sqlite3", "sqlite database path")
	screenshotPath := flag.String("screenshot", "/dev/shm/almono-livestream.png", "livestream screenshot path")
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

	svc := api.NewService(store)
	webServer, err := web.NewServer(svc)
	if err != nil {
		log.Fatalf("template init failed: %v", err)
	}
	webServer.SetScreenshotPath(*screenshotPath)

	baseDir := filepath.Dir(*dbPath)
	castDir := filepath.Join(baseDir, "casts")
	if err := os.MkdirAll(castDir, 0o755); err != nil {
		log.Fatalf("cast dir failed: %v", err)
	}
	castPath := filepath.Join(castDir, api.LiveCastName())
	webServer.SetCastPath(castPath)

	mux := http.NewServeMux()
	mux.Handle("/api/requests", api.NewRequestHandler(svc))
	mux.HandleFunc("/requests/new", webServer.HandleCreate)
	mux.HandleFunc("/requests/", webServer.HandleRequests)
	mux.HandleFunc("/livestream/", webServer.HandleLivestream)
	mux.HandleFunc("/ls-ss", webServer.HandleLivestreamScreenshot)
	mux.HandleFunc("/stream", webServer.HandleStream)
	mux.HandleFunc("/livestream", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/livestream/", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/requests", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/requests/", http.StatusMovedPermanently)
	})
	mux.Handle("/static/", http.StripPrefix("/static/", webServer.StaticHandler()))
	mux.Handle("/asciinema-player.css", webServer.StaticHandler())
	mux.Handle("/asciinema-player.min.js", webServer.StaticHandler())
	mux.Handle("/casts/", http.StripPrefix("/casts/", http.FileServer(http.Dir(castDir))))
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

	log.Printf("listening on %s", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
