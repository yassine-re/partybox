package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/models"
)

func (h *Handler) registerBoxRoutes(public *gin.RouterGroup) {
	public.GET("/boxes/:boxId", func(c *gin.Context) {
		box, err := h.Service.Repo.Box(c.Request.Context(), c.Param("boxId"))
		respond(c, box, err, http.StatusOK)
	})
	public.POST("/boxes/:boxId/games", func(c *gin.Context) {
		var body struct {
			Name                  string          `json:"name"`
			PlayerName            string          `json:"player_name"`
			Mode                  models.GameMode `json:"mode"`
			DurationMinutes       *int            `json:"duration_minutes"`
			LeaderboardVisibility string          `json:"leaderboard_visibility"`
			ValidationMode        string          `json:"validation_mode"`
		}
		body.Mode = models.ModeSecretMissions // Preserve clients that omit mode.
		if !bind(c, &body) {
			return
		}
		options := models.GameOptions{DurationMinutes: 30, LeaderboardVisibility: "visible", ValidationMode: "trust"}
		if body.DurationMinutes != nil {
			options.DurationMinutes = *body.DurationMinutes
		}
		if body.LeaderboardVisibility != "" {
			options.LeaderboardVisibility = body.LeaderboardVisibility
		}
		if body.ValidationMode != "" {
			options.ValidationMode = body.ValidationMode
		}
		session, err := h.Service.CreateConfigured(
			c.Request.Context(), c.Param("boxId"), body.Name, body.PlayerName, body.Mode, options,
		)
		respond(c, session, err, http.StatusCreated)
	})
	// Reserved contract only: no simulated hardware validation or unauthenticated effect.
	public.POST("/boxes/:boxId/events", func(c *gin.Context) {
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
		c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{
			"code": "not_implemented", "message": "Événements ESP32 prévus pour une prochaine version.",
		}})
	})
}
