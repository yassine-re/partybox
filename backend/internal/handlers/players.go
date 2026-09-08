package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/realtime"
)

func (h *Handler) registerPlayerRoutes(auth *gin.RouterGroup) {
	auth.GET("/players/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, currentPlayer(c))
	})
	auth.GET("/players/me/mission", func(c *gin.Context) {
		mission, err := h.Service.Repo.Mission(c.Request.Context(), currentPlayer(c))
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
		result, err := h.Service.Complete(c.Request.Context(), player, body.AssignmentID)
		if err == nil && !result.AlreadyCompleted {
			h.Realtime.Broadcast(realtime.NewEvent(
				realtime.EventMissionCompleted, player.GameID, player.ID,
			))
		}
		respond(c, result, err, http.StatusOK)
	})
}
