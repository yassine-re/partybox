package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"partybox/backend/internal/ai"
	"partybox/backend/internal/database"
	"partybox/backend/internal/handlers"
	mlranker "partybox/backend/internal/ml"
	"partybox/backend/internal/realtime"
	"partybox/backend/internal/repositories"
	"partybox/backend/internal/services"
	"partybox/backend/internal/vision"
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
	var generator ai.MissionGenerator
	key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if key != "" && model != "" {
		generator = ai.NewOpenAIResponsesGenerator(key, model, os.Getenv("OPENAI_BASE_URL"))
	} else if key != "" {
		slog.Warn("AI mission generation disabled: OPENAI_MODEL is not configured")
	}
	repository := &repositories.Repository{Pool: pool}
	rankerPath := strings.TrimSpace(os.Getenv("ML_RANKER_MODEL"))
	if rankerPath != "" {
		ranker, loadErr := mlranker.Load(rankerPath)
		if loadErr != nil {
			slog.Warn("ML mission ranking disabled", "error", loadErr)
		} else {
			repository.MissionScorer = ranker
			slog.Info("ML mission ranking enabled")
		}
	}
	s := &services.Service{
		Repo: repository,
		AI:   generator,
	}
	s.Reaction, err = services.ParseReactionConfig(os.Getenv)
	if err != nil {
		return err
	}
	visionModel := strings.TrimSpace(os.Getenv("OPENAI_VISION_MODEL"))
	if key != "" && visionModel != "" {
		s.Vision, err = vision.NewOpenAIValidator(key, visionModel, os.Getenv("OPENAI_VISION_DETAIL"), os.Getenv("OPENAI_BASE_URL"))
		if err != nil {
			return err
		}
	}
	frontendURL := env("FRONTEND_URL", "http://localhost:3000")
	realtimeServer := realtime.NewServer(frontendURL, s.AuthorizeRealtime)
	defer realtimeServer.Close()
	if s.Reaction.Enabled {
		go runReactionScheduler(ctx, s, realtimeServer)
	}
	server := &http.Server{
		Addr:              ":" + env("BACKEND_PORT", "8080"),
		Handler:           handlers.Router(s, realtimeServer, frontendURL),
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

func runReactionScheduler(ctx context.Context, service *services.Service, server *realtime.Server) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changes, err := service.TickReactions(ctx)
			if err != nil {
				if ctx.Err() == nil {
					slog.Error("reaction scheduler tick failed", "error", err)
				}
				continue
			}
			for _, change := range changes {
				eventType := realtime.EventReactionChallengeChanged
				if change.Resolved {
					eventType = realtime.EventReactionChallengeResolved
				}
				server.Broadcast(realtime.NewEvent(eventType, change.GameID, ""))
			}
		}
	}
}
