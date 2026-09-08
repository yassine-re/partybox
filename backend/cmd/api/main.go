package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"partybox/backend/internal/database"
	"partybox/backend/internal/handlers"
	"partybox/backend/internal/repositories"
	"partybox/backend/internal/services"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	if err := run(); err != nil {
		slog.Error("PartyBox stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startup, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	pool, err := database.Open(startup, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer pool.Close()
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			return database.Migrate(startup, pool, env("MIGRATIONS_DIR", "../database/migrations"))
		case "seed":
			return database.Seed(startup, pool, env("SEED_FILE", "../database/seed.sql"))
		default:
			return errors.New("usage: api [migrate|seed]")
		}
	}
	s := &services.Service{Repo: &repositories.Repository{Pool: pool}}
	server := &http.Server{
		Addr:              ":" + env("BACKEND_PORT", "8080"),
		Handler:           handlers.Router(s, env("FRONTEND_URL", "http://localhost:3000")),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
	}
	errorsCh := make(chan error, 1)
	go func() { slog.Info("PartyBox API listening", "addr", server.Addr); errorsCh <- server.ListenAndServe() }()
	select {
	case err = <-errorsCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
	return nil
}
