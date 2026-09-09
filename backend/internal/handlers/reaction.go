package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/realtime"
	"partybox/backend/internal/services"
)

func (h *Handler) registerReactionRoutes(auth *gin.RouterGroup) {
	auth.GET("/games/:gameId/reaction", func(c *gin.Context) {
		state, err := h.Service.ReactionState(c.Request.Context(), c.Param("gameId"), currentPlayer(c))
		respond(c, state, err, http.StatusOK)
	})
	auth.POST("/games/:gameId/reaction/:challengeId/assign", func(c *gin.Context) {
		var body services.ReactionAssignment
		if !bind(c, &body) {
			return
		}
		gameID := c.Param("gameId")
		challenge, err := h.Service.AssignReaction(
			c.Request.Context(), gameID, c.Param("challengeId"), currentPlayer(c), body,
		)
		if err == nil {
			h.Realtime.Broadcast(realtime.NewEvent(realtime.EventReactionChallengeChanged, gameID, ""))
		}
		respond(c, challenge, err, http.StatusOK)
	})
}
