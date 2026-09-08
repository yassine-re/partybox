package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/services"
)

func (h *Handler) registerAIRoutes(auth *gin.RouterGroup) {
	auth.POST("/games/:gameId/ai-missions/generate", func(c *gin.Context) {
		var body services.AIGenerationOptions
		if !bind(c, &body) {
			return
		}
		generated, err := h.Service.GenerateAIMissions(c.Request.Context(), c.Param("gameId"), currentPlayer(c), body)
		if err != nil {
			respond(c, nil, err, 0)
			return
		}
		respond(c, gin.H{"generated": generated}, nil, http.StatusOK)
	})

	auth.GET("/games/:gameId/ai-missions/status", func(c *gin.Context) {
		status, err := h.Service.AIMissionsStatus(c.Request.Context(), c.Param("gameId"), currentPlayer(c))
		respond(c, status, err, http.StatusOK)
	})
}
