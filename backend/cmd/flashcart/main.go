package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"flashcart/backend/internal/config"
	"flashcart/backend/internal/httpapi"
	"flashcart/backend/internal/jobs"
	"flashcart/backend/internal/storage"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if command == "seed" {
		return runSeed(ctx)
	}

	if command == "serve" {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		pool, err := storage.Open(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
		if err != nil {
			return err
		}
		defer pool.Close()
		mux := httpapi.New(pool, cfg)
		server := &http.Server{Addr: ":" + cfg.Port, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
		done := make(chan error, 1)
		go func() { logger.Info("server listening", "addr", server.Addr); done <- server.ListenAndServe() }()
		select {
		case err := <-done:
			if !errors.Is(err, http.ErrServerClosed) {
				return err
			}
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdown); err != nil {
				return err
			}
		}
		return nil
	}
	if command == "maintenance" || command == "maintenance-loop" {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		pool, err := storage.Open(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
		if err != nil {
			return err
		}
		defer pool.Close()
		interval := time.Duration(0)
		if command == "maintenance-loop" {
			interval = 5 * time.Second
		}
		if interval == 0 {
			n, err := jobs.ExpireBatch(ctx, pool, 50)
			if err != nil {
				return err
			}
			logger.Info("maintenance complete", "expired_orders", n)
			return nil
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			n, err := jobs.ExpireBatch(ctx, pool, 50)
			if err != nil {
				return err
			}
			if n > 0 {
				logger.Info("maintenance complete", "expired_orders", n)
			}
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			}
		}
	}
	if command == "publish-outbox" || command == "publish-outbox-loop" {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		pool, err := storage.Open(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
		if err != nil {
			return err
		}
		defer pool.Close()
		interval := time.Duration(0)
		if command == "publish-outbox-loop" {
			interval = 2 * time.Second
		}
		if interval == 0 {
			n, err := jobs.PublishOutbox(ctx, pool, cfg.PubSubAPIEndpoint, cfg.CloudProject, cfg.PubSubTopic, cfg.PubSubEmulatorHost, 50)
			if err != nil {
				return err
			}
			logger.Info("outbox publish complete", "published", n)
			return nil
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			n, err := jobs.PublishOutbox(ctx, pool, cfg.PubSubAPIEndpoint, cfg.CloudProject, cfg.PubSubTopic, cfg.PubSubEmulatorHost, 50)
			if err != nil {
				return err
			}
			if n > 0 {
				logger.Info("outbox publish complete", "published", n)
			}
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			}
		}
	}
	if command == "consume-push" {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		pool, err := storage.Open(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
		if err != nil {
			return err
		}
		defer pool.Close()
		port := cfg.Port
		if port == "8080" && cfg.AppEnv != "production" {
			port = "8081"
		}
		server := &http.Server{Addr: ":" + port, Handler: jobs.PubSubPushHandler(pool), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 20 * time.Second}
		done := make(chan error, 1)
		go func() {
			logger.Info("private event consumer listening", "addr", server.Addr)
			done <- server.ListenAndServe()
		}()
		select {
		case err := <-done:
			if !errors.Is(err, http.ErrServerClosed) {
				return err
			}
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdown); err != nil {
				return err
			}
		}
		return nil
	}
	if command == "migrate" {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		pool, err := storage.Open(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
		if err != nil {
			return err
		}
		defer pool.Close()
		db, err := sql.Open("pgx", cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer db.Close()
		return goose.UpContext(ctx, db, "migrations")
	}
	return fmt.Errorf("usage: flashcart [serve|migrate|seed|maintenance|maintenance-loop|publish-outbox|publish-outbox-loop|consume-push]")
}
