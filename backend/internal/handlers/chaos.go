package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) registerChaosRoutes(auth *gin.RouterGroup) {
	auth.GET("/games/:gameId/chaos", func(c *gin.Context) {
		state, err := h.Service.ChaosState(
			c.Request.Context(), c.Param("gameId"), currentPlayer(c),
		)
		respond(c, state, err, http.StatusOK)
	})
}
