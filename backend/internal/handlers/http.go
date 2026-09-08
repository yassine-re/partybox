package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/models"
	"partybox/backend/internal/realtime"
	"partybox/backend/internal/services"
)

func Router(s *services.Service, rt *realtime.Server, frontendURL string) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	_ = r.SetTrustedProxies(nil)
	r.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Header("X-Content-Type-Options", "nosniff")
		origin := c.GetHeader("Origin")
		if origin != "" && origin == frontendURL {
			c.Header("Access-Control-Allow-Origin", frontendURL)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		if isWebSocketRequest(c.Request) {
			c.Next()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.GET("/api/health", func(c *gin.Context) {
		if err := s.Repo.Pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/api/boxes/:boxId", func(c *gin.Context) {
		b, err := s.Repo.Box(c.Request.Context(), c.Param("boxId"))
		respond(c, b, err, http.StatusOK)
	})
	r.POST("/api/boxes/:boxId/games", func(c *gin.Context) {
		var body struct {
			Name       string          `json:"name"`
			PlayerName string          `json:"player_name"`
			Mode       models.GameMode `json:"mode"`
		}
		body.Mode = models.ModeSecretMissions // Preserve clients that omit mode.
		if !bind(c, &body) {
			return
		}
		session, err := s.Create(c.Request.Context(), c.Param("boxId"), body.Name, body.PlayerName, body.Mode)
		respond(c, session, err, http.StatusCreated)
	})
	r.POST("/api/games/:gameId/join", func(c *gin.Context) {
		var body struct {
			Name string `json:"name"`
		}
		if !bind(c, &body) {
			return
		}
		session, err := s.Join(c.Request.Context(), c.Param("gameId"), body.Name)
		if err == nil {
			rt.Broadcast(realtime.NewEvent(realtime.EventPlayerJoined, session.Game.ID, session.Player.ID))
		}
		respond(c, session, err, http.StatusCreated)
	})
	// Reserved contract only: no simulated hardware validation or unauthenticated effect.
	r.POST("/api/boxes/:boxId/events", func(c *gin.Context) {
		var body struct {
			Type string `json:"type"`
		}
		if !bind(c, &body) {
			return
		}
		if body.Type != "button_press" {
			respond(c, nil, models.ErrInvalid, 0)
			return
		}
		c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"code": "not_implemented", "message": "Événements ESP32 prévus pour une prochaine version."}})
	})
	auth := r.Group("/api")
	auth.Use(func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			respond(c, nil, models.ErrUnauthorized, 0)
			c.Abort()
			return
		}
		p, err := s.Authenticate(c.Request.Context(), strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			respond(c, nil, err, 0)
			c.Abort()
			return
		}
		c.Set("player", p)
		c.Next()
	})
	auth.GET("/players/me", func(c *gin.Context) { c.JSON(http.StatusOK, currentPlayer(c)) })
	auth.GET("/players/me/mission", func(c *gin.Context) {
		mission, err := s.Repo.Mission(c.Request.Context(), currentPlayer(c))
		respond(c, gin.H{"mission": mission}, err, http.StatusOK)
	})
	auth.POST("/players/me/mission/complete", func(c *gin.Context) {
		var body struct {
			AssignmentID string `json:"assignment_id"`
		}
		if !bind(c, &body) {
			return
		}
		player := currentPlayer(c)
		result, err := s.Complete(c.Request.Context(), player, body.AssignmentID)
		if err == nil && !result.AlreadyCompleted {
			rt.Broadcast(realtime.NewEvent(realtime.EventMissionCompleted, player.GameID, player.ID))
		}
		respond(c, result, err, http.StatusOK)
	})
	auth.GET("/games/:gameId", func(c *gin.Context) {
		g, err := s.Game(c.Request.Context(), c.Param("gameId"), currentPlayer(c))
		respond(c, g, err, http.StatusOK)
	})
	auth.GET("/games/:gameId/leaderboard", func(c *gin.Context) {
		g, err := s.Game(c.Request.Context(), c.Param("gameId"), currentPlayer(c))
		respond(c, gin.H{"players": g.Players}, err, http.StatusOK)
	})
	auth.POST("/games/:gameId/start", func(c *gin.Context) {
		gameID := c.Param("gameId")
		err := s.Start(c.Request.Context(), gameID, currentPlayer(c))
		if err == nil {
			rt.Broadcast(realtime.NewEvent(realtime.EventGameStarted, gameID, ""))
		}
		respond(c, gin.H{"status": "playing"}, err, http.StatusOK)
	})
	auth.POST("/games/:gameId/end", func(c *gin.Context) {
		gameID := c.Param("gameId")
		err := s.End(c.Request.Context(), gameID, currentPlayer(c))
		if err == nil {
			rt.Broadcast(realtime.NewEvent(realtime.EventGameEnded, gameID, ""))
		}
		respond(c, gin.H{"status": "ended"}, err, http.StatusOK)
	})
	auth.POST("/games/:gameId/ws-ticket", func(c *gin.Context) {
		player := currentPlayer(c)
		gameID := c.Param("gameId")
		if player.GameID != gameID {
			respond(c, nil, models.ErrForbidden, 0)
			return
		}
		ticket, expiresAt, err := rt.IssueTicket(player.ID, gameID)
		respond(c, gin.H{"ticket": ticket, "expires_at": expiresAt}, err, http.StatusCreated)
	})
	r.GET("/api/games/:gameId/ws", func(c *gin.Context) {
		rt.ServeHTTP(c.Writer, c.Request, c.Param("gameId"))
	})
	r.NoRoute(func(c *gin.Context) { respond(c, nil, models.ErrNotFound, 0) })
	return r
}

func isWebSocketRequest(request *http.Request) bool {
	return request.Method == http.MethodGet &&
		strings.HasPrefix(request.URL.Path, "/api/games/") &&
		strings.HasSuffix(request.URL.Path, "/ws")
}

func currentPlayer(c *gin.Context) models.Player { return c.MustGet("player").(models.Player) }

func bind(c *gin.Context, body any) bool {
	if err := c.ShouldBindJSON(body); err != nil {
		respond(c, nil, models.ErrInvalid, 0)
		return false
	}
	return true
}

func respond(c *gin.Context, value any, err error, status int) {
	if err == nil {
		c.JSON(status, value)
		return
	}
	code, message := "internal_error", "Une erreur est survenue. Réessaie dans un instant."
	status = http.StatusInternalServerError
	switch {
	case errors.Is(err, models.ErrInvalid):
		status, code = 400, "invalid_input"
	case errors.Is(err, models.ErrUnauthorized):
		status, code = 401, "unauthorized"
	case errors.Is(err, models.ErrForbidden):
		status, code = 403, "forbidden"
	case errors.Is(err, models.ErrNotFound):
		status, code = 404, "not_found"
	case errors.Is(err, models.ErrConflict):
		status, code = 409, "conflict"
	}
	if status < 500 {
		message = err.Error()
	} else {
		slog.Error("request failed", "error", err)
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
