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
	"partybox/backend/internal/vision"
)

type Handler struct {
	Service  *services.Service
	Realtime *realtime.Server
}

func Router(s *services.Service, rt *realtime.Server, frontendURL string) *gin.Engine {
	h := &Handler{Service: s, Realtime: rt}
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
		bodyLimit := int64(8192)
		timeout := 10 * time.Second
		if c.Request.Method == http.MethodPost && c.Request.URL.Path == proofPath {
			bodyLimit = vision.MaxUploadBytes
			timeout = 60 * time.Second
			controller := http.NewResponseController(c.Writer)
			_ = controller.SetReadDeadline(time.Now().Add(30 * time.Second))
			_ = controller.SetWriteDeadline(time.Now().Add(65 * time.Second))
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, bodyLimit)
		if strings.HasSuffix(c.Request.URL.Path, "/ai-missions/generate") {
			timeout = 45 * time.Second
			// The server keeps a short global WriteTimeout. Gin exposes the
			// underlying writer, so only this long-running route gets more time.
			_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(50 * time.Second))
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.GET("/api/health", func(c *gin.Context) {
		if err := h.Service.Repo.Pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	public := r.Group("/api")
	auth := r.Group("/api")
	auth.Use(h.authenticate)
	h.registerBoxRoutes(public)
	h.registerGameRoutes(public, auth)
	h.registerPlayerRoutes(auth)
	h.registerRealtimeRoutes(public, auth)
	h.registerChaosRoutes(auth)
	h.registerAIRoutes(auth)
	h.registerProofRoutes(auth)
	r.NoRoute(func(c *gin.Context) { respond(c, nil, models.ErrNotFound, 0) })
	return r
}

func (h *Handler) authenticate(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		respond(c, nil, models.ErrUnauthorized, 0)
		c.Abort()
		return
	}
	p, err := h.Service.Authenticate(c.Request.Context(), strings.TrimPrefix(header, "Bearer "))
	if err != nil {
		respond(c, nil, err, 0)
		c.Abort()
		return
	}
	c.Set("player", p)
	c.Next()
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
	case errors.Is(err, services.ErrProofLimit):
		status, code = 429, "proof_limit"
	case errors.Is(err, services.ErrProofProvider):
		status, code, message = 503, "vision_unavailable", services.ErrProofProvider.Error()
	}
	if status < 500 {
		message = err.Error()
	} else {
		slog.Error("request failed", "error", err)
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
