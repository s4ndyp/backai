package main

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/s4ndyp/backai/internal/api"
	"github.com/s4ndyp/backai/internal/config"
	"github.com/s4ndyp/backai/internal/db"
	"github.com/s4ndyp/backai/internal/store"
	"github.com/s4ndyp/backai/internal/version"
)

//go:embed all:static
var static embed.FS

func main() {
	_ = godotenv.Load()
	cfg := config.LoadServer()
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx := context.Background()
	pool, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if cfg.MigrateOnStart {
		if err := db.Migrate(ctx, pool); err != nil {
			log.Error("migrate failed", "err", err)
			os.Exit(1)
		}
	}

	st := store.New(pool)
	webRoot, _ := fs.Sub(static, "static")
	srv := api.New(cfg, st, webRoot, log)
	sched := api.NewScheduler(st, log)

	httpServer := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      srv.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sched.Start(runCtx)

	go func() {
		log.Info("backai server listening", "addr", cfg.HTTPAddr, "version", version.Version)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)
}
